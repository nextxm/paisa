<script lang="ts">
  import { onMount } from "svelte";
  import { Search } from "lucide-svelte";
  import LedgerToolsPanels from "$lib/components/LedgerToolsPanels.svelte";
  import { apiGet, authHeaders, formatMoney, formatShortDate, runtime } from "$lib/api";

  type MarketTab = "prices" | "fx";

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
  let prices = $state<PriceRow[]>([]);
  let providers = $state<Array<{ code?: string; name?: string }>>([]);

  onMount(async () => {
    try {
      const [priceRes, provRes] = await Promise.all([
        apiGet<{ prices?: Record<string, PriceRow[]> | PriceRow[] }>("/api/price").catch(() => ({
          prices: []
        })),
        apiGet<{ providers?: Array<{ code?: string; name?: string }> }>("/api/price/providers").catch(
          () => ({ providers: [] })
        )
      ]);
      if (Array.isArray(priceRes.prices)) {
        prices = priceRes.prices;
      } else if (priceRes.prices && typeof priceRes.prices === "object") {
        prices = Object.entries(priceRes.prices).flatMap(([name, rows]) => {
          if (!Array.isArray(rows) || rows.length === 0) {
            return [{ commodity_name: name }];
          }
          return rows.map((r) => ({ ...r, commodity_name: r.commodity_name || name }));
        });
      }
      providers = provRes.providers || [];
    } finally {
      loading = false;
    }
  });

  const filteredPrices = $derived.by(() => {
    const q = search.trim().toLowerCase();
    if (!q) return prices.slice(0, 150);
    return prices
      .filter(
        (p) =>
          p.commodity_name?.toLowerCase().includes(q) ||
          p.base?.toLowerCase().includes(q) ||
          p.source?.toLowerCase().includes(q)
      )
      .slice(0, 150);
  });
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Market Data</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Commodity market prices, price providers, and foreign exchange rates
      </p>
    </div>

    {#if activeTab === "prices"}
      <div class="relative w-full sm:w-72">
        <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="search"
          bind:value={search}
          placeholder="Filter commodities or source…"
          class="w-full pl-9 pr-3 py-1.5 rounded-lg text-sm bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
        />
      </div>
    {/if}
  </div>

  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "prices", label: `Commodity Prices (${prices.length})` },
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
  {:else if loading}
    <div class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"></div>
  {:else}
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-[var(--muted)]">Providers ({providers.length}):</span>
          {#each providers as prov}
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md bg-[var(--surface)] border border-[var(--border)]">
              <span>{prov.name || prov.code || "Provider"}</span>
              {#if !runtime.readonly && prov.code}
                <button
                  type="button"
                  title="Delete provider"
                  onclick={async () => {
                    const res = await fetch(
                      `/api/price/providers/delete/${encodeURIComponent(prov.code || "")}`,
                      { method: "POST", headers: authHeaders() }
                    );
                    if (res.ok) providers = providers.filter((p) => p.code !== prov.code);
                  }}
                  class="text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
                >
                  ×
                </button>
              {/if}
            </span>
          {/each}
        </div>
        <div class="flex items-center gap-2">
          <a
            href="/api/price/export"
            class="px-2.5 py-1.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)] font-medium"
          >
            Export Prices CSV
          </a>
          {#if !runtime.readonly}
            <button
              type="button"
              onclick={async () => {
                const res = await fetch("/api/price/delete", { method: "POST", headers: authHeaders() });
                if (res.ok) prices = [];
              }}
              class="px-2.5 py-1.5 rounded-lg border border-[var(--border)] text-[var(--loss)] hover:bg-[var(--loss)]/10 cursor-pointer"
            >
              Clear Prices
            </button>
          {/if}
        </div>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if filteredPrices.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">No price entries found.</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                <tr>
                  <th class="py-2.5 px-4 font-medium">Commodity</th>
                  <th class="py-2.5 px-4 font-medium">Type</th>
                  <th class="py-2.5 px-4 font-medium">ID</th>
                  <th class="py-2.5 px-4 text-right font-medium">Price</th>
                  <th class="py-2.5 px-4 font-medium">Date</th>
                  <th class="py-2.5 px-4 font-medium">Source</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each filteredPrices as p}
                  {@const unitPrice = p.value ?? p.price}
                  {@const quoteCurr = p.quote_commodity || p.quote || runtime.defaultCurrency || "INR"}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2 px-4 font-medium">{p.commodity_name || p.base || "—"}</td>
                    <td class="py-2 px-4 text-xs text-[var(--muted)]">{p.commodity_type || "—"}</td>
                    <td class="py-2 px-4 text-xs font-mono text-[var(--muted)]">{p.commodity_id || "—"}</td>
                    <td class="py-2 px-4 text-right font-semibold">
                      {unitPrice !== undefined ? formatMoney(unitPrice, quoteCurr) : "—"}
                    </td>
                    <td class="py-2 px-4 text-xs text-[var(--muted)]">
                      {p.date ? formatShortDate(p.date) : "—"}
                    </td>
                    <td class="py-2 px-4 text-xs text-[var(--muted)]">{p.source || "—"}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
