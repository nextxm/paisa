<script lang="ts">
  import { onMount } from "svelte";
  import LineAreaChart, { type NetworthPoint } from "$lib/components/LineAreaChart.svelte";
  import { apiGet, formatMoney, formatShortDate, type AssetBreakdown } from "$lib/api";

  type WealthTab =
    | "overview"
    | "assets"
    | "investments"
    | "allocation"
    | "income"
    | "liabilities";

  interface GainRow {
    account: string;
    investmentAmount?: number;
    marketAmount?: number;
    gainAmount?: number;
    xirr?: number;
  }

  interface LiabilityRow {
    group: string;
    drawn_amount?: number;
    repaid_amount?: number;
    interest_amount?: number;
    balance_amount?: number;
  }

  interface AllocationAggregate {
    name?: string;
    account?: string;
    market_amount: number;
  }

  interface CurrencyExposureItem {
    currency: string;
    amount: number;
    percent: number;
  }

  interface InvestmentIncomeHolding {
    name?: string;
    account?: string;
    ttm_income?: number;
    yield?: number;
  }

  interface CreditCardSummary {
    name: string;
    account: string;
    balance?: number;
    limit?: number;
    unbilled?: number;
    due_date?: string;
  }

  let activeTab = $state<WealthTab>("overview");
  let yearsWindow = $state<number>(0);
  let search = $state("");

  let loading = $state(true);
  let error = $state<string | null>(null);

  let timeline = $state<NetworthPoint[]>([]);
  let xirr = $state(0);
  let assetBreakdowns = $state<AssetBreakdown[]>([]);
  let gains = $state<GainRow[]>([]);
  let liabilities = $state<LiabilityRow[]>([]);
  let allocationItems = $state<AllocationAggregate[]>([]);
  let currencyExposure = $state<CurrencyExposureItem[]>([]);
  let investmentIncomeTotal = $state(0);
  let investmentIncomeHoldings = $state<InvestmentIncomeHolding[]>([]);
  let creditCards = $state<CreditCardSummary[]>([]);

  onMount(async () => {
    try {
      const [nwRes, balRes, gainRes, liabRes, allocRes, fxExpRes, incRes, ccRes] =
        await Promise.all([
          apiGet<{ xirr?: number; networthTimeline?: NetworthPoint[] }>("/api/networth"),
          apiGet<{ asset_breakdowns?: Record<string, AssetBreakdown> }>(
            "/api/assets/balance"
          ).catch(() => ({ asset_breakdowns: {} as Record<string, AssetBreakdown> })),
          apiGet<{ gain_breakdown?: GainRow[] }>("/api/gain").catch(() => ({
            gain_breakdown: [] as GainRow[]
          })),
          apiGet<{ liability_breakdowns?: LiabilityRow[] }>("/api/liabilities/balance").catch(
            () => ({ liability_breakdowns: [] as LiabilityRow[] })
          ),
          apiGet<{ aggregates?: Record<string, AllocationAggregate> }>("/api/allocation").catch(
            () => ({ aggregates: {} as Record<string, AllocationAggregate> })
          ),
          apiGet<{ currency_exposure?: CurrencyExposureItem[] }>("/api/currency-exposure").catch(
            () => ({ currency_exposure: [] as CurrencyExposureItem[] })
          ),
          apiGet<{ ttm_total?: number; holdings?: InvestmentIncomeHolding[] }>(
            "/api/income/investment"
          ).catch(() => ({ ttm_total: 0, holdings: [] as InvestmentIncomeHolding[] })),
          apiGet<{ creditCards?: CreditCardSummary[] }>("/api/credit_cards").catch(() => ({
            creditCards: [] as CreditCardSummary[]
          }))
        ]);

      timeline = nwRes.networthTimeline || [];
      xirr = nwRes.xirr || 0;
      assetBreakdowns = Object.values(balRes.asset_breakdowns || {}).sort(
        (a, b) => (b.marketAmount || 0) - (a.marketAmount || 0)
      );
      gains = (gainRes.gain_breakdown || []).sort(
        (a, b) => (b.marketAmount || 0) - (a.marketAmount || 0)
      );
      liabilities = liabRes.liability_breakdowns || [];
      allocationItems = Object.values(allocRes.aggregates || {})
        .filter((a) => (a.market_amount || 0) > 0)
        .sort((a, b) => (b.market_amount || 0) - (a.market_amount || 0));
      currencyExposure = fxExpRes.currency_exposure || [];
      investmentIncomeTotal = incRes.ttm_total || 0;
      investmentIncomeHoldings = incRes.holdings || [];
      creditCards = ccRes.creditCards || [];
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load wealth data";
    } finally {
      loading = false;
    }
  });

  const windowedTimeline = $derived(
    yearsWindow > 0 ? timeline.slice(-(yearsWindow * 12)) : timeline
  );
  const latestPoint = $derived(timeline.length > 0 ? timeline[timeline.length - 1] : null);
  const totalAllocation = $derived(
    allocationItems.reduce((sum, a) => sum + (a.market_amount || 0), 0)
  );

  const filteredAssets = $derived(
    assetBreakdowns.filter(
      (a) =>
        Math.abs(a.marketAmount || 0) > 0.01 &&
        a.group.toLowerCase().includes(search.trim().toLowerCase())
    )
  );
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Wealth</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Net worth, asset balances, investments, allocation, income, and debt
      </p>
    </div>

    {#if activeTab === "overview"}
      <div
        class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
      >
        {#each [1, 3, 5, 0] as y}
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {yearsWindow === y
              ? 'bg-[var(--surface-elevated)] font-semibold text-[var(--text)]'
              : 'text-[var(--muted)] hover:text-[var(--text)]'}"
            onclick={() => (yearsWindow = y)}
          >
            {y === 0 ? "All" : `${y}Y`}
          </button>
        {/each}
      </div>
    {:else if activeTab === "assets"}
      <input
        type="search"
        bind:value={search}
        placeholder="Filter asset accounts…"
        class="px-3 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
      />
    {/if}
  </div>

  <!-- Underline Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "overview", label: "Overview" },
      { id: "assets", label: "Assets" },
      { id: "investments", label: "Investments" },
      { id: "allocation", label: "Allocation" },
      { id: "income", label: "Inv. Income" },
      { id: "liabilities", label: "Debt & Cards" }
    ] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 transition-colors whitespace-nowrap cursor-pointer {activeTab ===
        tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)] hover:text-[var(--text)]'}"
        onclick={() => (activeTab = tab.id as WealthTab)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Failed to load wealth data</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "overview"}
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Net Worth
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">
          {formatMoney(latestPoint?.balanceAmount || 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Total Gain
        </div>
        <div
          class="text-2xl font-semibold tabular-nums mt-1 {(latestPoint?.gainAmount || 0) >= 0
            ? 'text-[var(--gain)]'
            : 'text-[var(--loss)]'}"
        >
          {formatMoney(latestPoint?.gainAmount || 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Portfolio XIRR
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">{xirr.toFixed(1)}%</div>
      </div>
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex items-center justify-between text-xs text-[var(--muted)]">
        <span class="font-semibold uppercase tracking-wider">Net Worth Timeline</span>
        <div class="flex items-center gap-4">
          <span class="inline-flex items-center gap-1.5">
            <span class="w-3 h-0.5 bg-[var(--accent)] inline-block"></span>
            Net Worth
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-3 h-0.5 bg-[var(--muted)] inline-block"></span>
            Invested
          </span>
        </div>
      </div>
      <LineAreaChart data={windowedTimeline} />
    </div>
  {:else if activeTab === "assets"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <table class="w-full text-sm text-left">
        <thead
          class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
        >
          <tr>
            <th class="py-2.5 px-4 font-medium">Account</th>
            <th class="py-2.5 px-4 text-right font-medium">Market Value</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--border)] tabular-nums">
          {#each filteredAssets as asset}
            <tr class="hover:bg-[var(--surface-elevated)]/50">
              <td class="py-2.5 px-4 font-medium">{asset.group}</td>
              <td class="py-2.5 px-4 text-right font-semibold">
                {formatMoney(asset.marketAmount)}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else if activeTab === "investments"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if gains.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">
          No investment holdings found.
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Holding</th>
                <th class="py-2.5 px-4 text-right font-medium">Invested</th>
                <th class="py-2.5 px-4 text-right font-medium">Market Value</th>
                <th class="py-2.5 px-4 text-right font-medium">Gain</th>
                <th class="py-2.5 px-4 text-right font-medium">XIRR</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each gains as g}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2.5 px-4 font-medium">{g.account}</td>
                  <td class="py-2.5 px-4 text-right">{formatMoney(g.investmentAmount || 0)}</td>
                  <td class="py-2.5 px-4 text-right font-semibold">
                    {formatMoney(g.marketAmount || 0)}
                  </td>
                  <td
                    class="py-2.5 px-4 text-right {(g.gainAmount || 0) >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {formatMoney(g.gainAmount || 0)}
                  </td>
                  <td class="py-2.5 px-4 text-right">{(g.xirr || 0).toFixed(1)}%</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {:else if activeTab === "allocation"}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <div class="lg:col-span-8 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Asset Allocation
        </h3>
        {#if allocationItems.length === 0}
          <p class="text-sm text-[var(--muted)]">No allocation data available.</p>
        {:else}
          <div class="space-y-3">
            {#each allocationItems.slice(0, 15) as item}
              {@const pct = totalAllocation > 0 ? (item.market_amount / totalAllocation) * 100 : 0}
              <div class="space-y-1 text-xs">
                <div class="flex justify-between">
                  <span class="font-medium truncate">{item.name || item.account || "Asset"}</span>
                  <span class="tabular-nums">
                    <strong>{formatMoney(item.market_amount)}</strong>
                    <span class="text-[var(--muted)] ml-1">({pct.toFixed(1)}%)</span>
                  </span>
                </div>
                <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                  <div class="h-full rounded-full bg-[var(--accent)]" style="width: {pct}%"></div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="lg:col-span-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Currency Exposure
        </h3>
        {#if currencyExposure.length === 0}
          <p class="text-sm text-[var(--muted)]">Single currency portfolio.</p>
        {:else}
          <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
            {#each currencyExposure as c}
              <div class="py-2 flex justify-between">
                <span class="font-medium">{c.currency}</span>
                <span>{c.percent.toFixed(1)}%</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {:else if activeTab === "income"}
    <div class="space-y-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          TTM Investment Income
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
          {formatMoney(investmentIncomeTotal)}
        </div>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if investmentIncomeHoldings.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">
            No investment income recorded.
          </div>
        {:else}
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Account</th>
                <th class="py-2.5 px-4 text-right font-medium">TTM Income</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each investmentIncomeHoldings as h}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2.5 px-4 font-medium">{h.account || h.name || "—"}</td>
                  <td class="py-2.5 px-4 text-right font-semibold text-[var(--gain)]">
                    {formatMoney(h.ttm_income || 0)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    </div>
  {:else}
    <div class="space-y-5">
      {#if creditCards.length > 0}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each creditCards as cc}
            <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2">
              <div class="flex justify-between items-center">
                <span class="font-semibold text-sm">{cc.name || cc.account}</span>
                {#if cc.due_date}
                  <span class="text-xs text-[var(--muted)]">Due {formatShortDate(cc.due_date)}</span>
                {/if}
              </div>
              <div class="flex justify-between text-xs tabular-nums">
                <span class="text-[var(--muted)]">Balance</span>
                <span class="font-semibold text-[var(--loss)]">{formatMoney(cc.balance || 0)}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if liabilities.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">
            No liabilities recorded.
          </div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
              >
                <tr>
                  <th class="py-2.5 px-4 font-medium">Liability</th>
                  <th class="py-2.5 px-4 text-right font-medium">Drawn</th>
                  <th class="py-2.5 px-4 text-right font-medium">Repaid</th>
                  <th class="py-2.5 px-4 text-right font-medium">Interest</th>
                  <th class="py-2.5 px-4 text-right font-medium">Balance</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each liabilities as l}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2.5 px-4 font-medium">{l.group}</td>
                    <td class="py-2.5 px-4 text-right">{formatMoney(l.drawn_amount || 0)}</td>
                    <td class="py-2.5 px-4 text-right text-[var(--gain)]">
                      {formatMoney(l.repaid_amount || 0)}
                    </td>
                    <td class="py-2.5 px-4 text-right text-[var(--muted)]">
                      {formatMoney(l.interest_amount || 0)}
                    </td>
                    <td class="py-2.5 px-4 text-right font-semibold text-[var(--loss)]">
                      {formatMoney(l.balance_amount || 0)}
                    </td>
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
