<script lang="ts">
  import { onMount } from "svelte";
  import { formatMoney } from "$lib/api";

  let {
    tool
  }: {
    tool: "whatif" | "drawdown" | "tax";
  } = $props();

  // ---- What-If state ----
  interface WhatIfScenario {
    name: string;
    overrides: {
      monthly_contribution?: number;
      contribution_growth_rate?: number;
      expected_return?: number;
      inflation_rate?: number;
      swr?: number;
    };
  }

  let whatifLoading = $state(false);
  let whatifBaseline = $state({
    projectionYears: 30,
    monthlyContribution: 0,
    contributionGrowthRate: 5,
    expectedReturn: 12,
    returnVolatility: 18,
    inflationRate: 6,
    swr: 4,
    iterations: 500
  });
  let whatifScenarios = $state<WhatIfScenario[]>([
    { name: "Increase SIP (+10k)", overrides: { monthly_contribution: 10000 } },
    {
      name: "Job Loss / Pause",
      overrides: { monthly_contribution: 0, contribution_growth_rate: 0, expected_return: 8 }
    }
  ]);
  let whatifData = $state<{
    profile?: {
      monthly_contribution?: number;
      income_growth_rate?: number;
      historical_return?: number;
      return_volatility?: number;
    };
    baseline?: {
      fire_year_p50?: number;
      fire_probability?: number;
      final_p50?: number;
      bands?: { p50?: Array<{ date: string; balance_amount: number }> };
    };
    scenarios?: Array<{
      name: string;
      fire_year_p50?: number;
      fire_probability?: number;
      final_p50?: number;
      bands?: { p50?: Array<{ date: string; balance_amount: number }> };
    }>;
  } | null>(null);

  const whatifPalette = ["#3b82f6", "#10b981", "#f59e0b", "#8b5cf6", "#ef4444", "#14b8a6"];

  async function runWhatIf() {
    whatifLoading = true;
    try {
      const res = await fetch("/api/projection/whatif", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          baseline: {
            iterations: whatifBaseline.iterations,
            months_to_project: whatifBaseline.projectionYears * 12,
            monthly_contribution: whatifBaseline.monthlyContribution,
            contribution_growth_rate: whatifBaseline.contributionGrowthRate,
            expected_return: whatifBaseline.expectedReturn,
            return_volatility: whatifBaseline.returnVolatility,
            inflation_rate: whatifBaseline.inflationRate,
            swr: whatifBaseline.swr
          },
          scenarios: whatifScenarios.slice(0, 5)
        })
      });
      if (res.ok) {
        const data = (await res.json()) as NonNullable<typeof whatifData>;
        whatifData = data;
        if (
          data.profile &&
          whatifBaseline.monthlyContribution === 0 &&
          (data.profile.monthly_contribution || 0) > 0
        ) {
          whatifBaseline.monthlyContribution = Math.round(data.profile.monthly_contribution || 0);
        }
      }
    } finally {
      whatifLoading = false;
    }
  }

  function addWhatIfTemplate(name: string, overrides: WhatIfScenario["overrides"]) {
    if (whatifScenarios.length >= 5) return;
    whatifScenarios = [...whatifScenarios, { name, overrides }];
  }

  // ---- Drawdown state ----
  interface DrawdownBucket {
    name: string;
    accounts: string[];
    account_glob: string;
    tax_category: string;
    override_tax_category: string;
    holding_period_months: number;
  }

  let drawdownAmount = $state(500000);
  let drawdownImpact = $state(false);
  let drawdownBuckets = $state<DrawdownBucket[]>([
    {
      name: "Equity First",
      accounts: [],
      account_glob: "Assets:Equity:*",
      tax_category: "",
      override_tax_category: "equity",
      holding_period_months: 12
    },
    {
      name: "Fallback",
      accounts: [],
      account_glob: "Assets:*",
      tax_category: "",
      override_tax_category: "",
      holding_period_months: 0
    }
  ]);
  let drawdownLoading = $state(false);
  let drawdownData = $state<{
    total_tax?: number;
    effective_tax_rate?: number;
    net_proceeds?: number;
    shortfall?: number;
    allocations?: Array<{
      account: string;
      bucket?: string;
      amount: number;
      tax?: number;
      tax_category?: string;
      gain?: number;
    }>;
  } | null>(null);

  function addBucket() {
    drawdownBuckets = [
      ...drawdownBuckets,
      {
        name: `Bucket ${drawdownBuckets.length + 1}`,
        accounts: [],
        account_glob: "Assets:*",
        tax_category: "",
        override_tax_category: "",
        holding_period_months: 0
      }
    ];
  }

  function moveBucket(index: number, dir: -1 | 1) {
    const next = index + dir;
    if (next < 0 || next >= drawdownBuckets.length) return;
    const copy = [...drawdownBuckets];
    const tmp = copy[index];
    copy[index] = copy[next];
    copy[next] = tmp;
    drawdownBuckets = copy;
  }

  async function runDrawdown() {
    drawdownLoading = true;
    try {
      const res = await fetch("/api/projection/drawdown", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          amount: drawdownAmount,
          buckets: drawdownBuckets,
          include_projection_impact: drawdownImpact
        })
      });
      if (res.ok) {
        drawdownData = await res.json();
      }
    } finally {
      drawdownLoading = false;
    }
  }

  // ---- Tax state ----
  interface HarvestableItem {
    account: string;
    tax_category?: string;
    total_units?: number;
    harvestable_units: number;
    unrealized_gain?: number;
    taxable_unrealized_gain?: number;
    current_unit_price?: number;
  }

  interface FYCapitalGainEntry {
    sell_price?: number;
    tax?: {
      gain?: number;
      taxable?: number;
      short_term?: number;
      long_term?: number;
      slab?: number;
    };
  }

  interface CapitalGainAccount {
    account?: string;
    fy?: Record<string, FYCapitalGainEntry>;
  }

  let taxSubTab = $state<"harvest" | "capital_gains" | "schedule_al">("harvest");
  let taxLoading = $state(false);
  let targetHarvestGain = $state(100000);
  let harvestables = $state<HarvestableItem[]>([]);
  let capitalGains = $state<CapitalGainAccount[]>([]);
  let selectedCgYear = $state("");
  let scheduleAls = $state<
    Record<
      string,
      {
        date?: string;
        entries?: Array<{
          section: { code: string; section: string; details: string };
          amount: number;
        }>;
      }
    >
  >({});
  let selectedScheduleYear = $state("");

  const cgYears = $derived.by(() => {
    const set = new Set<string>();
    for (const c of capitalGains) {
      for (const y of Object.keys(c.fy || {})) set.add(y);
    }
    return Array.from(set).sort().reverse();
  });

  const activeCgRows = $derived.by(() => {
    if (!selectedCgYear) return [];
    return capitalGains
      .filter((c) => Boolean(c.fy?.[selectedCgYear]))
      .map((c) => ({
        account: c.account || "Account",
        fy: c.fy![selectedCgYear]
      }));
  });

  const activeCgTotals = $derived.by(() => {
    let withdrawn = 0;
    let gain = 0;
    let taxable = 0;
    let stTax = 0;
    let ltTax = 0;
    let slab = 0;
    for (const r of activeCgRows) {
      withdrawn += r.fy.sell_price || 0;
      gain += r.fy.tax?.gain || 0;
      taxable += r.fy.tax?.taxable || 0;
      stTax += r.fy.tax?.short_term || 0;
      ltTax += r.fy.tax?.long_term || 0;
      slab += r.fy.tax?.slab || 0;
    }
    return { withdrawn, gain, taxable, stTax, ltTax, slab };
  });

  async function loadTax() {
    taxLoading = true;
    try {
      const [hRes, cgRes, salRes] = await Promise.all([
        fetch("/api/harvest").then((r) => (r.ok ? r.json() : { harvestables: {} })),
        fetch("/api/capital_gains").then((r) => (r.ok ? r.json() : { capital_gains: {} })),
        fetch("/api/schedule_al").then((r) => (r.ok ? r.json() : { schedule_als: {} }))
      ]);
      harvestables = (Object.values(hRes.harvestables || {}) as HarvestableItem[]).filter(
        (h) => (h.harvestable_units || 0) > 0
      );
      capitalGains = Object.values(cgRes.capital_gains || {}) as CapitalGainAccount[];
      const cgYrs = new Set<string>();
      for (const c of capitalGains) {
        for (const y of Object.keys(c.fy || {})) cgYrs.add(y);
      }
      const sortedCg = Array.from(cgYrs).sort().reverse();
      if (sortedCg.length > 0 && !selectedCgYear) {
        selectedCgYear = sortedCg[0];
      }
      scheduleAls = salRes.schedule_als || {};
      const yrs = Object.keys(scheduleAls).sort().reverse();
      if (yrs.length > 0 && !selectedScheduleYear) {
        selectedScheduleYear = yrs[0];
      }
    } finally {
      taxLoading = false;
    }
  }

  onMount(() => {
    if (tool === "whatif") void runWhatIf();
    if (tool === "drawdown") void runDrawdown();
    if (tool === "tax") void loadTax();
  });

  $effect(() => {
    if (tool === "whatif" && !whatifData && !whatifLoading) void runWhatIf();
    if (tool === "drawdown" && !drawdownData && !drawdownLoading) void runDrawdown();
    if (tool === "tax" && harvestables.length === 0 && capitalGains.length === 0 && !taxLoading)
      void loadTax();
  });
</script>

{#if tool === "whatif"}
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
    <!-- Left: Baseline + Scenarios Builder -->
    <div class="lg:col-span-4 space-y-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold uppercase tracking-wider text-[var(--muted)]">Baseline Assumptions</h3>
          <button
            type="button"
            disabled={whatifLoading}
            onclick={runWhatIf}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
          >
            {whatifLoading ? "Running…" : "Run Comparison"}
          </button>
        </div>
        <div class="grid grid-cols-2 gap-2.5">
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-yrs">Years</label>
            <input id="wi-yrs" type="number" min="5" max="50" bind:value={whatifBaseline.projectionYears} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-mc">Monthly SIP</label>
            <input id="wi-mc" type="number" step="1000" bind:value={whatifBaseline.monthlyContribution} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-ret">Expected Return %</label>
            <input id="wi-ret" type="number" step="0.5" bind:value={whatifBaseline.expectedReturn} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-inf">Inflation %</label>
            <input id="wi-inf" type="number" step="0.5" bind:value={whatifBaseline.inflationRate} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-swr">SWR %</label>
            <input id="wi-swr" type="number" step="0.25" bind:value={whatifBaseline.swr} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="wi-vol">Volatility %</label>
            <input id="wi-vol" type="number" step="1" bind:value={whatifBaseline.returnVolatility} class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
          </div>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold uppercase tracking-wider text-[var(--muted)]">
            Scenarios ({whatifScenarios.length}/5)
          </h3>
          <button
            type="button"
            disabled={whatifScenarios.length >= 5}
            onclick={() => addWhatIfTemplate(`Scenario ${whatifScenarios.length + 1}`, {})}
            class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer disabled:opacity-40"
          >
            + Add
          </button>
        </div>
        <div class="flex flex-wrap gap-1.5">
          <button type="button" onclick={() => addWhatIfTemplate("+10k SIP", { monthly_contribution: (whatifBaseline.monthlyContribution || 0) + 10000 })} class="px-2 py-0.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)] cursor-pointer">+10k SIP</button>
          <button type="button" onclick={() => addWhatIfTemplate("Job Loss (8%)", { monthly_contribution: 0, expected_return: 8 })} class="px-2 py-0.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)] cursor-pointer">Job Loss</button>
          <button type="button" onclick={() => addWhatIfTemplate("Early Retire (3.5%)", { monthly_contribution: 0, swr: 3.5, inflation_rate: 7 })} class="px-2 py-0.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)] cursor-pointer">Early Retire</button>
        </div>

        <div class="space-y-2.5">
          {#each whatifScenarios as sc, idx}
            <div class="p-3 rounded-lg bg-[var(--surface-elevated)]/60 border border-[var(--border)] space-y-2">
              <div class="flex items-center justify-between gap-2">
                <input type="text" bind:value={sc.name} class="flex-1 px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] font-medium" />
                <button type="button" onclick={() => (whatifScenarios = whatifScenarios.filter((_, i) => i !== idx))} class="text-[var(--loss)] hover:underline cursor-pointer">Remove</button>
              </div>
              <div class="grid grid-cols-3 gap-2">
                <div>
                  <span class="text-[10px] text-[var(--muted)]">SIP</span>
                  <input type="number" placeholder="—" bind:value={sc.overrides.monthly_contribution} class="w-full px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] tabular-nums" />
                </div>
                <div>
                  <span class="text-[10px] text-[var(--muted)]">Return %</span>
                  <input type="number" step="0.5" placeholder="—" bind:value={sc.overrides.expected_return} class="w-full px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] tabular-nums" />
                </div>
                <div>
                  <span class="text-[10px] text-[var(--muted)]">SWR %</span>
                  <input type="number" step="0.25" placeholder="—" bind:value={sc.overrides.swr} class="w-full px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] tabular-nums" />
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    </div>

    <!-- Right: Trajectory Comparison Chart + Table -->
    <div class="lg:col-span-8 space-y-4">
      {#if whatifData}
        {@const allSeries = [
          { name: "Baseline", color: whatifPalette[0], pts: whatifData.baseline?.bands?.p50 || [] },
          ...(whatifData.scenarios || []).map((s, i) => ({
            name: s.name,
            color: whatifPalette[(i + 1) % whatifPalette.length],
            pts: s.bands?.p50 || []
          }))
        ].filter((s) => s.pts.length > 0)}

        {#if allSeries.length > 0}
          {@const maxBal = Math.max(1, ...allSeries.flatMap((s) => s.pts.map((p) => p.balance_amount || 0)))}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
              <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">
                Median Trajectory Comparison (P50)
              </span>
              <div class="flex flex-wrap items-center gap-3">
                {#each allSeries as s}
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-3 h-0.5 inline-block" style="background: {s.color}"></span>
                    {s.name}
                  </span>
                {/each}
              </div>
            </div>
            <svg viewBox="0 0 700 220" class="w-full h-56 overflow-visible">
              {#each allSeries as s}
                {@const ptsStr = s.pts
                  .map(
                    (p, idx) =>
                      `${20 + (idx / Math.max(1, s.pts.length - 1)) * 660},${200 - ((p.balance_amount || 0) / maxBal) * 180}`
                  )
                  .join(" ")}
                <polyline fill="none" stroke={s.color} stroke-width="2" points={ptsStr} />
              {/each}
            </svg>
          </div>
        {/if}

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <table class="w-full text-sm text-left">
            <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
              <tr>
                <th class="py-2.5 px-4 font-medium">Scenario</th>
                <th class="py-2.5 px-4 text-right font-medium">FIRE Probability</th>
                <th class="py-2.5 px-4 text-right font-medium">FIRE Year (P50)</th>
                <th class="py-2.5 px-4 text-right font-medium">Median Final (P50)</th>
                <th class="py-2.5 px-4 text-right font-medium">vs Baseline</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#if whatifData.baseline}
                <tr class="bg-[var(--surface-elevated)]/30 font-semibold">
                  <td class="py-2.5 px-4">Baseline</td>
                  <td class="py-2.5 px-4 text-right">{((whatifData.baseline.fire_probability || 0) * 100).toFixed(1)}%</td>
                  <td class="py-2.5 px-4 text-right">{whatifData.baseline.fire_year_p50 || "—"}</td>
                  <td class="py-2.5 px-4 text-right">{formatMoney(whatifData.baseline.final_p50 || 0)}</td>
                  <td class="py-2.5 px-4 text-right text-[var(--muted)]">—</td>
                </tr>
              {/if}
              {#each whatifData.scenarios || [] as sc}
                {@const delta = (sc.final_p50 || 0) - (whatifData.baseline?.final_p50 || 0)}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2.5 px-4 font-medium">{sc.name}</td>
                  <td class="py-2.5 px-4 text-right">{((sc.fire_probability || 0) * 100).toFixed(1)}%</td>
                  <td class="py-2.5 px-4 text-right">{sc.fire_year_p50 || "—"}</td>
                  <td class="py-2.5 px-4 text-right font-semibold">{formatMoney(sc.final_p50 || 0)}</td>
                  <td class="py-2.5 px-4 text-right {delta >= 0 ? 'text-[var(--gain)]' : 'text-[var(--loss)]'}">
                    {delta >= 0 ? "+" : ""}{formatMoney(delta)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>
{:else if tool === "drawdown"}
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
    <!-- Left: Withdrawal + Strategy Buckets -->
    <div class="lg:col-span-4 space-y-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold uppercase tracking-wider text-[var(--muted)]">Drawdown Config</h3>
          <button
            type="button"
            disabled={drawdownLoading}
            onclick={runDrawdown}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
          >
            {drawdownLoading ? "Analyzing…" : "Analyze"}
          </button>
        </div>
        <div>
          <label class="block text-[var(--muted)] mb-1" for="dd-amt">Withdrawal Amount</label>
          <input id="dd-amt" type="number" step="10000" bind:value={drawdownAmount} class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums" />
        </div>
        <label class="inline-flex items-center gap-1.5 cursor-pointer text-[var(--muted)]">
          <input type="checkbox" bind:checked={drawdownImpact} />
          Include projection impact
        </label>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold uppercase tracking-wider text-[var(--muted)]">
            Strategy Buckets ({drawdownBuckets.length})
          </h3>
          <button type="button" onclick={addBucket} class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer">
            + Add Bucket
          </button>
        </div>
        <div class="space-y-2.5">
          {#each drawdownBuckets as b, idx}
            <div class="p-3 rounded-lg bg-[var(--surface-elevated)]/60 border border-[var(--border)] space-y-2">
              <div class="flex items-center justify-between gap-2">
                <input type="text" bind:value={b.name} class="flex-1 px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] font-medium" />
                <div class="flex items-center gap-1">
                  <button type="button" onclick={() => moveBucket(idx, -1)} class="px-1.5 py-0.5 rounded border border-[var(--border)] cursor-pointer">↑</button>
                  <button type="button" onclick={() => moveBucket(idx, 1)} class="px-1.5 py-0.5 rounded border border-[var(--border)] cursor-pointer">↓</button>
                  <button type="button" onclick={() => (drawdownBuckets = drawdownBuckets.filter((_, i) => i !== idx))} class="text-[var(--loss)] px-1 cursor-pointer">×</button>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div>
                  <span class="text-[10px] text-[var(--muted)]">Account Glob</span>
                  <input type="text" bind:value={b.account_glob} class="w-full px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] font-mono" />
                </div>
                <div>
                  <span class="text-[10px] text-[var(--muted)]">Min Holding (mo)</span>
                  <input type="number" min="0" bind:value={b.holding_period_months} class="w-full px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] tabular-nums" />
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    </div>

    <!-- Right: Summary KPIs + Allocations Table -->
    <div class="lg:col-span-8 space-y-4">
      {#if drawdownData}
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">Estimated Tax</div>
            <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-1">{formatMoney(drawdownData.total_tax || 0)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">Effective Tax Rate</div>
            <div class="text-2xl font-semibold tabular-nums mt-1">{(drawdownData.effective_tax_rate || 0).toFixed(2)}%</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">Net Proceeds</div>
            <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">{formatMoney(drawdownData.net_proceeds || 0)}</div>
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="px-4 py-3 border-b border-[var(--border)] text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Recommended Liquidation Plan ({(drawdownData.allocations || []).length} lots)
          </div>
          {#if (drawdownData.allocations || []).length === 0}
            <div class="p-6 text-center text-sm text-[var(--muted)]">No eligible lots matched your buckets.</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-sm text-left">
                <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                  <tr>
                    <th class="py-2.5 px-4 font-medium">Account</th>
                    <th class="py-2.5 px-4 font-medium">Bucket / Category</th>
                    <th class="py-2.5 px-4 text-right font-medium">Withdraw</th>
                    <th class="py-2.5 px-4 text-right font-medium">Est. Tax</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)] tabular-nums">
                  {#each drawdownData.allocations || [] as lot}
                    <tr class="hover:bg-[var(--surface-elevated)]/50">
                      <td class="py-2 px-4 font-medium">{lot.account}</td>
                      <td class="py-2 px-4 text-xs text-[var(--muted)]">{lot.bucket || lot.tax_category || "—"}</td>
                      <td class="py-2 px-4 text-right font-semibold">{formatMoney(lot.amount || 0)}</td>
                      <td class="py-2 px-4 text-right text-[var(--loss)]">{formatMoney(lot.tax || 0)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="space-y-5">
    <div class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs">
      {#each [
        { id: "harvest", label: `Tax Harvesting (${harvestables.length})` },
        { id: "capital_gains", label: `Capital Gains (${cgYears.length} FY)` },
        { id: "schedule_al", label: `Schedule AL (${Object.keys(scheduleAls).length} FY)` }
      ] as st}
        <button
          type="button"
          onclick={() => (taxSubTab = st.id as typeof taxSubTab)}
          class="px-3 py-1.5 rounded-md cursor-pointer {taxSubTab === st.id
            ? 'bg-[var(--surface-elevated)] font-semibold text-[var(--text)]'
            : 'text-[var(--muted)] hover:text-[var(--text)]'}"
        >
          {st.label}
        </button>
      {/each}
    </div>

    {#if taxSubTab === "harvest"}
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-center justify-between gap-3 text-xs">
        <span class="text-[var(--muted)]">Target Taxable Gain for Redemption Estimator:</span>
        <input
          type="number"
          step="10000"
          bind:value={targetHarvestGain}
          class="w-36 px-2.5 py-1 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
        />
      </div>

      {#if harvestables.length === 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]">
          No tax harvesting opportunities found.
        </div>
      {:else}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                <tr>
                  <th class="py-2.5 px-4 font-medium">Account</th>
                  <th class="py-2.5 px-3 font-medium">Category</th>
                  <th class="py-2.5 px-3 text-right font-medium">Harvestable Units</th>
                  <th class="py-2.5 px-3 text-right font-medium">Unit Price</th>
                  <th class="py-2.5 px-3 text-right font-medium">Unrealized Gain</th>
                  <th class="py-2.5 px-3 text-right font-medium">Taxable Gain</th>
                  <th class="py-2.5 px-4 text-right font-medium">Est. Units for Target</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each harvestables as h}
                  {@const avgTaxablePerUnit =
                    (h.harvestable_units || 0) > 0 && (h.taxable_unrealized_gain || 0) > 0
                      ? (h.taxable_unrealized_gain || 0) / h.harvestable_units
                      : 0}
                  {@const estUnits =
                    avgTaxablePerUnit > 0
                      ? Math.min(h.harvestable_units, targetHarvestGain / avgTaxablePerUnit)
                      : 0}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2.5 px-4 font-medium">{h.account}</td>
                    <td class="py-2.5 px-3 text-xs text-[var(--muted)]">{h.tax_category || "—"}</td>
                    <td class="py-2.5 px-3 text-right">{(h.harvestable_units || 0).toFixed(2)}</td>
                    <td class="py-2.5 px-3 text-right">{formatMoney(h.current_unit_price || 0)}</td>
                    <td class="py-2.5 px-3 text-right text-[var(--gain)]">
                      {formatMoney(h.unrealized_gain || 0)}
                    </td>
                    <td class="py-2.5 px-3 text-right font-semibold">
                      {formatMoney(h.taxable_unrealized_gain || 0)}
                    </td>
                    <td class="py-2.5 px-4 text-right text-xs text-[var(--accent)]">
                      {estUnits > 0 ? `${estUnits.toFixed(2)} (${formatMoney(estUnits * (h.current_unit_price || 0))})` : "—"}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    {:else if taxSubTab === "capital_gains"}
      {#if cgYears.length === 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]">
          No realized capital gains recorded.
        </div>
      {:else}
        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Financial Year Capital Gains
          </span>
          <select
            bind:value={selectedCgYear}
            class="px-2.5 py-1 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] outline-none"
          >
            {#each cgYears as yr}
              <option value={yr}>{yr}</option>
            {/each}
          </select>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">Withdrawn</div>
            <div class="text-base font-semibold tabular-nums mt-1">{formatMoney(activeCgTotals.withdrawn)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">Gain</div>
            <div class="text-base font-semibold tabular-nums mt-1 text-[var(--gain)]">{formatMoney(activeCgTotals.gain)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">Taxable Gain</div>
            <div class="text-base font-semibold tabular-nums mt-1">{formatMoney(activeCgTotals.taxable)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">STCG Tax</div>
            <div class="text-base font-semibold tabular-nums mt-1 text-[var(--loss)]">{formatMoney(activeCgTotals.stTax)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">LTCG Tax</div>
            <div class="text-base font-semibold tabular-nums mt-1 text-[var(--loss)]">{formatMoney(activeCgTotals.ltTax)}</div>
          </div>
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3.5">
            <div class="text-[11px] uppercase tracking-wider text-[var(--muted)]">Slab Tax</div>
            <div class="text-base font-semibold tabular-nums mt-1 text-[var(--loss)]">{formatMoney(activeCgTotals.slab)}</div>
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                <tr>
                  <th class="py-2.5 px-4 font-medium">Account</th>
                  <th class="py-2.5 px-3 text-right font-medium">Withdrawn</th>
                  <th class="py-2.5 px-3 text-right font-medium">Gain</th>
                  <th class="py-2.5 px-3 text-right font-medium">Taxable</th>
                  <th class="py-2.5 px-3 text-right font-medium">STCG Tax</th>
                  <th class="py-2.5 px-4 text-right font-medium">LTCG Tax</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each activeCgRows as row}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2 px-4 font-medium">{row.account}</td>
                    <td class="py-2 px-3 text-right">{formatMoney(row.fy.sell_price || 0)}</td>
                    <td class="py-2 px-3 text-right text-[var(--gain)]">{formatMoney(row.fy.tax?.gain || 0)}</td>
                    <td class="py-2 px-3 text-right">{formatMoney(row.fy.tax?.taxable || 0)}</td>
                    <td class="py-2 px-3 text-right text-[var(--loss)]">{formatMoney(row.fy.tax?.short_term || 0)}</td>
                    <td class="py-2 px-4 text-right text-[var(--loss)]">{formatMoney(row.fy.tax?.long_term || 0)}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    {:else}
      {#if Object.keys(scheduleAls).length === 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]">
          No Schedule AL data available.
        </div>
      {:else}
        {@const activeSchedule = scheduleAls[selectedScheduleYear]}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="px-5 py-3 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Schedule AL (Assets &amp; Liabilities Statement)
              </h3>
              {#if activeSchedule?.date}
                <p class="text-xs text-[var(--muted)] mt-0.5">As on {activeSchedule.date.slice(0, 10)}</p>
              {/if}
            </div>
            <select
              bind:value={selectedScheduleYear}
              class="px-2.5 py-1 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            >
              {#each Object.keys(scheduleAls).sort().reverse() as yr}
                <option value={yr}>FY {yr}</option>
              {/each}
            </select>
          </div>
          {#if activeSchedule?.entries?.length}
            <div class="overflow-x-auto">
              <table class="w-full text-sm text-left">
                <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
                  <tr>
                    <th class="py-2 px-4 font-medium">Code</th>
                    <th class="py-2 px-4 font-medium">Section</th>
                    <th class="py-2 px-4 font-medium">Details</th>
                    <th class="py-2 px-4 text-right font-medium">Amount</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)] tabular-nums">
                  {#each activeSchedule.entries as entry}
                    <tr class="hover:bg-[var(--surface-elevated)]/50">
                      <td class="py-2 px-4 font-mono text-xs text-[var(--muted)]">{entry.section?.code || "—"}</td>
                      <td class="py-2 px-4 font-medium">{entry.section?.section || "—"}</td>
                      <td class="py-2 px-4 text-xs text-[var(--muted)]">{entry.section?.details || "—"}</td>
                      <td class="py-2 px-4 text-right font-semibold">{formatMoney(entry.amount || 0)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/if}
    {/if}
  </div>
{/if}
