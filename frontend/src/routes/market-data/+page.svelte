<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { ChevronDown, ChevronRight, RefreshCw, Search } from "lucide-svelte";
  import LedgerToolsPanels from "$lib/components/LedgerToolsPanels.svelte";
  import { apiGet, authHeaders, formatShortDate, runtime } from "$lib/api";

  type MarketTab = "prices" | "fx";
  type HistoryMode = "latest" | "all";

  interface PriceRow {
    commodity_name?: string;
    commodity_type?: string;
    commodity_id?: string;
    base?: string;
    quote?: string;
    quote_commodity?: string;
    value?: number;
    price?: number;
    date?: string;
    source?: string;
  }

  let activeTab = $state<MarketTab>("prices");
  let search = $state("");
  let loading = $state(true);
  let error = $state<string | null>(null);

  let groupedPrices = $state<Record<string, PriceRow[]>>({});
  let expandedRows = $state<Record<string, boolean>>({});
  let loadingHistoryRows = $state<Record<string, boolean>>({});
  let loadedHistoryRows = $state<Record<string, boolean>>({});

  let filterBase = $state("");
  let filterQuote = $state("");
  let filterSource = $state("");
  let reportCurrency = $state("");
  let historyMode = $state<HistoryMode>("all");

  let availableBases = $state<string[]>([]);
  let availableQuotes = $state<string[]>([]);
  let availableSources = $state<string[]>([]);
  let providers = $state<Array<{ code?: string; label?: string; name?: string }>>([]);
  let refreshing = $state(false);

  const hasActiveFilters = $derived(
    filterBase !== "" || filterQuote !== "" || filterSource !== "" || reportCurrency !== ""
  );

  function buildPriceRoute(base = filterBase, mode: HistoryMode = historyMode) {
    const params = new URLSearchParams();
    if (base) params.set("base", base);
    if (filterQuote) params.set("quote", filterQuote);
    if (filterSource) params.set("source", filterSource);
    if (reportCurrency) params.set("report_currency", reportCurrency);
    if (mode === "all") params.set("history", "all");
    const q = params.toString();
    return q ? `/api/price?${q}` : "/api/price";
  }

  async function fetchPrices() {
    loading = true;
    error = null;
    try {
      const res = await apiGet<{ prices?: Record<string, PriceRow[]> }>(buildPriceRoute());
      const next: Record<string, PriceRow[]> = {};
      for (const [commodity, rows] of Object.entries(res.prices || {})) {
        if (Array.isArray(rows) && rows.length > 0) {
          next[commodity] = rows;
        }
      }
      groupedPrices = next;
      expandedRows = {};
      loadingHistoryRows = {};
      loadedHistoryRows =
        historyMode === "all"
          ? Object.fromEntries(Object.keys(next).map((c) => [c, true]))
          : {};
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load commodity prices";
    } finally {
      loading = false;
    }
  }

  async function loadFiltersAndProviders() {
    const [filtersRes, provRes] = await Promise.all([
      apiGet<{ bases?: string[]; quotes?: string[]; sources?: string[] }>("/api/price/filters").catch(
        () => ({ bases: [], quotes: [], sources: [] })
      ),
      apiGet<{ providers?: Array<{ code?: string; label?: string; name?: string }> }>(
        "/api/price/providers"
      ).catch(() => ({ providers: [] }))
    ]);
    availableBases = filtersRes.bases || [];
    availableQuotes = filtersRes.quotes || [];
    availableSources = filtersRes.sources || [];
    providers = provRes.providers || [];
  }

  onMount(async () => {
    await Promise.all([loadFiltersAndProviders(), fetchPrices()]);
  });

  async function toggleCommodity(commodity: string) {
    const isExpanded = !expandedRows[commodity];
    expandedRows = { ...expandedRows, [commodity]: isExpanded };
    if (
      isExpanded &&
      historyMode !== "all" &&
      !loadedHistoryRows[commodity] &&
      !loadingHistoryRows[commodity]
    ) {
      loadingHistoryRows = { ...loadingHistoryRows, [commodity]: true };
      try {
        const res = await apiGet<{ prices?: Record<string, PriceRow[]> }>(
          buildPriceRoute(commodity, "all")
        );
        groupedPrices = {
          ...groupedPrices,
          [commodity]: res.prices?.[commodity] || groupedPrices[commodity] || []
        };
        loadedHistoryRows = { ...loadedHistoryRows, [commodity]: true };
      } finally {
        loadingHistoryRows = { ...loadingHistoryRows, [commodity]: false };
      }
    }
  }

  async function forceRefreshPrices() {
    refreshing = true;
    try {
      await fetch("/api/sync", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ prices: true, force_prices: true })
      });
      await Promise.all([loadFiltersAndProviders(), fetchPrices()]);
    } finally {
      refreshing = false;
    }
  }

  function changePct(rows: PriceRow[], days: number, tolerance: number): number | null {
    const first = rows[0];
    if (!first?.date) return null;
    const firstDate = dayjs(first.date);
    const firstVal = first.value ?? first.price;
    if (firstVal === undefined || firstVal === 0) return null;

    const target = firstDate.subtract(days, "day");
    const match = rows.find((r) => r.date && !dayjs(r.date).isAfter(target, "day"));
    if (!match?.date) return null;

    const diffDays = firstDate.diff(dayjs(match.date), "day");
    if (Math.abs(diffDays - days) <= tolerance) {
      const lastVal = match.value ?? match.price;
      if (lastVal && lastVal !== 0) {
        return (firstVal - lastVal) / lastVal;
      }
    }
    return null;
  }

  function formatPct(val: number | null): string {
    if (val === null) return "—";
    const pct = val * 100;
    return `${pct >= 0 ? "+" : ""}${pct.toFixed(2)}%`;
  }

  function formatPriceNum(val: number | undefined): string {
    if (val === undefined || val === null) return "—";
    return new Intl.NumberFormat(runtime.locale || "en-IN", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 4
    }).format(val);
  }

  const priceEntries = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return Object.entries(groupedPrices)
      .filter(([commodity, rows]) => {
        if (!q) return true;
        const latest = rows[0];
        return (
          commodity.toLowerCase().includes(q) ||
          latest?.commodity_name?.toLowerCase().includes(q) ||
          latest?.commodity_id?.toLowerCase().includes(q) ||
          latest?.commodity_type?.toLowerCase().includes(q) ||
          latest?.source?.toLowerCase().includes(q)
        );
      })
      .sort(([a], [b]) => a.localeCompare(b));
  });
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Market Data</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Commodity market prices, historical returns, price providers, and foreign exchange rates
      </p>
    </div>

    {#if activeTab === "prices"}
      <div class="relative w-full sm:w-72">
        <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="search"
          bind:value={search}
          placeholder="Filter commodities, ID, or source…"
          class="w-full pl-9 pr-3 py-1.5 rounded-lg text-sm bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
        />
      </div>
    {/if}
  </div>

  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "prices", label: `Commodity Prices (${priceEntries.length})` },
      { id: "fx", label: "FX Rates" }
    ] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 whitespace-nowrap cursor-pointer {activeTab === tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)]'}"
        onclick={() => (activeTab = tab.id as MarketTab)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  {#if activeTab === "fx"}
    <LedgerToolsPanels mode="fx" />
  {:else}
    <div class="space-y-4">
      <!-- Filters & Actions -->
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5 flex flex-wrap items-center gap-2 text-xs">
        <select
          bind:value={filterBase}
          onchange={fetchPrices}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="">All Base</option>
          {#each availableBases as base}
            <option value={base}>{base}</option>
          {/each}
        </select>
        <select
          bind:value={filterQuote}
          onchange={fetchPrices}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="">All Quote</option>
          {#each availableQuotes as quote}
            <option value={quote}>{quote}</option>
          {/each}
        </select>
        <select
          bind:value={filterSource}
          onchange={fetchPrices}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="">All Sources</option>
          {#each availableSources as src}
            <option value={src}>{src}</option>
          {/each}
        </select>
        <select
          bind:value={reportCurrency}
          onchange={fetchPrices}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="">No Conversion</option>
          {#each availableQuotes as quote}
            <option value={quote}>{quote}</option>
          {/each}
        </select>
        <select
          bind:value={historyMode}
          onchange={fetchPrices}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="latest">Latest Only</option>
          <option value="all">Load History</option>
        </select>
        {#if hasActiveFilters}
          <button
            type="button"
            onclick={() => {
              filterBase = "";
              filterQuote = "";
              filterSource = "";
              reportCurrency = "";
              void fetchPrices();
            }}
            class="px-2.5 py-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
          >
            Clear Filters
          </button>
        {/if}

        <div class="ml-auto flex flex-wrap items-center gap-2">
          <a
            href="/api/price/export"
            class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] hover:border-[var(--accent)] font-medium"
          >
            Export CSV
          </a>
          {#if !runtime.readonly}
            <button
              type="button"
              disabled={refreshing}
              onclick={forceRefreshPrices}
              class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer disabled:opacity-50"
            >
              <RefreshCw class="w-3.5 h-3.5 {refreshing ? 'animate-spin' : ''}" />
              <span>Force Refresh</span>
            </button>
            <button
              type="button"
              onclick={async () => {
                const res = await fetch("/api/price/delete", { method: "POST", headers: authHeaders() });
                if (res.ok) await Promise.all([loadFiltersAndProviders(), fetchPrices()]);
              }}
              class="px-2.5 py-1.5 rounded-lg border border-[var(--border)] text-[var(--loss)] hover:bg-[var(--loss)]/10 cursor-pointer"
            >
              Clear Cache
            </button>
          {/if}
        </div>
      </div>

      {#if providers.length > 0}
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <span class="text-[var(--muted)]">Providers ({providers.length}):</span>
          {#each providers as prov}
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md bg-[var(--surface)] border border-[var(--border)]">
              <span>{prov.label || prov.name || prov.code || "Provider"}</span>
              {#if !runtime.readonly && prov.code}
                <button
                  type="button"
                  title="Clear provider cache"
                  onclick={async () => {
                    const res = await fetch(
                      `/api/price/providers/delete/${encodeURIComponent(prov.code || "")}`,
                      { method: "POST", headers: authHeaders() }
                    );
                    if (res.ok) await fetchPrices();
                  }}
                  class="text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
                >
                  ×
                </button>
              {/if}
            </span>
          {/each}
        </div>
      {/if}

      {#if error}
        <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-xs text-[var(--loss)] flex items-center justify-between gap-2">
          <span>{error}</span>
          <button
            type="button"
            onclick={() => {
              historyMode = "latest";
              void fetchPrices();
            }}
            class="px-2.5 py-1 rounded bg-[var(--surface)] border border-[var(--border)] text-[var(--text)] cursor-pointer"
          >
            Retry (Latest Only)
          </button>
        </div>
      {/if}

      {#if loading}
        <div class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"></div>
      {:else}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          {#if priceEntries.length === 0}
            <div class="p-8 text-center text-sm text-[var(--muted)]">No price entries found.</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-xs text-left">
                <thead class="uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                  <tr>
                    <th class="py-2.5 px-3 w-8"></th>
                    <th class="py-2.5 px-3 font-medium">Commodity Name</th>
                    <th class="py-2.5 px-3 font-medium">Last Date</th>
                    <th class="py-2.5 px-3 font-medium">Quote</th>
                    <th class="py-2.5 px-3 text-right font-medium">Last Price</th>
                    <th class="py-2.5 px-3 text-right font-medium">1 Day</th>
                    <th class="py-2.5 px-3 text-right font-medium">1 Week</th>
                    <th class="py-2.5 px-3 text-right font-medium">4 Weeks</th>
                    <th class="py-2.5 px-3 text-right font-medium">1 Year</th>
                    <th class="py-2.5 px-3 text-right font-medium">3 Years</th>
                    <th class="py-2.5 px-3 text-right font-medium">5 Years</th>
                    <th class="py-2.5 px-3 font-medium">Type</th>
                    <th class="py-2.5 px-3 font-medium">ID</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)] tabular-nums">
                  {#each priceEntries as [commodity, rows] (commodity)}
                    {@const latest = rows[0]}
                    {@const unitPrice = latest?.value ?? latest?.price}
                    {@const quoteCurr = latest?.quote_commodity || latest?.quote || "—"}
                    {@const d1 = changePct(rows, 1, 0)}
                    {@const w1 = changePct(rows, 7, 2)}
                    {@const w4 = changePct(rows, 28, 4)}
                    {@const y1 = changePct(rows, 365, 7)}
                    {@const y3 = changePct(rows, 365 * 3, 7)}
                    {@const y5 = changePct(rows, 365 * 5, 7)}
                    <tr
                      onclick={() => toggleCommodity(commodity)}
                      class="hover:bg-[var(--surface-elevated)]/50 cursor-pointer"
                    >
                      <td class="py-2 px-3 text-[var(--muted)]">
                        {#if loadingHistoryRows[commodity]}
                          <RefreshCw class="w-3.5 h-3.5 animate-spin" />
                        {:else if expandedRows[commodity]}
                          <ChevronDown class="w-3.5 h-3.5" />
                        {:else}
                          <ChevronRight class="w-3.5 h-3.5" />
                        {/if}
                      </td>
                      <td class="py-2 px-3 font-medium">{latest?.commodity_name || commodity}</td>
                      <td class="py-2 px-3 text-[var(--muted)] whitespace-nowrap">
                        {latest?.date ? formatShortDate(latest.date) : "—"}
                      </td>
                      <td class="py-2 px-3 font-mono text-[var(--muted)]">{quoteCurr}</td>
                      <td class="py-2 px-3 text-right font-semibold">{formatPriceNum(unitPrice)}</td>
                      {#each [d1, w1, w4, y1, y3, y5] as pct}
                        <td
                          class="py-2 px-3 text-right {pct === null
                            ? 'text-[var(--muted)]'
                            : pct >= 0
                              ? 'text-[var(--gain)]'
                              : 'text-[var(--loss)]'}"
                        >
                          {formatPct(pct)}
                        </td>
                      {/each}
                      <td class="py-2 px-3 text-[var(--muted)]">{latest?.commodity_type || "—"}</td>
                      <td class="py-2 px-3 font-mono text-[var(--muted)]">{latest?.commodity_id || "—"}</td>
                    </tr>
                    {#if expandedRows[commodity]}
                      <tr>
                        <td colspan="13" class="p-0 bg-[var(--surface-elevated)]/30">
                          <div class="max-h-72 overflow-y-auto divide-y divide-[var(--border)]/50 px-6 py-1">
                            {#each rows.slice(0, 200) as h}
                              <div class="py-1 flex items-center justify-between gap-4 text-[11px]">
                                <span class="text-[var(--muted)] w-28">
                                  {h.date ? formatShortDate(h.date) : "—"}
                                </span>
                                <span class="font-mono text-[var(--muted)] w-16">
                                  {h.quote_commodity || h.quote || ""}
                                </span>
                                <span class="text-[var(--muted)] flex-1 truncate">{h.source || ""}</span>
                                <span class="font-semibold text-right">{formatPriceNum(h.value ?? h.price)}</span>
                              </div>
                            {/each}
                            {#if rows.length > 200}
                              <div class="py-1.5 text-[11px] text-[var(--muted)] text-center">
                                Showing latest 200 of {rows.length} entries (use Export CSV for full history)
                              </div>
                            {/if}
                          </div>
                        </td>
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>
