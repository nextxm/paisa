<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { ChevronDown, ChevronUp, Dna, Sparkles } from "lucide-svelte";
  import ProjectionChart, {
    type FanBand,
    type ProjectionPoint
  } from "$lib/components/ProjectionChart.svelte";
  import PlanningSubtools from "$lib/components/PlanningSubtools.svelte";
  import { apiGet, formatMoney, getFullConfig, runtime, updateConfig } from "$lib/api";

  type PlanningTab = "projection" | "goals" | "whatif" | "drawdown" | "tax";
  type ProjectionMode = "simple" | "montecarlo";
  type ScenarioKey = "expected" | "conservative" | "optimistic";

  interface FinancialProfile {
    current_networth: number;
    monthly_contribution: number;
    savings_rate: number;
    annual_expenses: number;
    annual_income: number;
    income_growth_rate: number;
    expense_growth_rate: number;
    historical_return: number;
    return_volatility: number;
    income_years_covered: number;
    expense_years_covered: number;
    price_months_covered: number;
  }

  interface BaseProjectionResponse {
    current_networth: number;
    annual_expenses: number;
    monthly_contribution: number;
    expected_cagr: number;
    conservative_cagr: number;
    optimistic_cagr: number;
    swr: number;
  }

  interface SimResponse {
    simulation: {
      target_corpus: number;
      fire_year_p25: number;
      fire_year_p50: number;
      fire_year_p75: number;
      fire_probability: number;
      bands: {
        p10: Array<{ date: string; balance_amount: number }>;
        p25: Array<{ date: string; balance_amount: number }>;
        p50: Array<{ date: string; balance_amount: number }>;
        p75: Array<{ date: string; balance_amount: number }>;
        p90: Array<{ date: string; balance_amount: number }>;
      };
    };
  }

  interface GoalSummary {
    id: string;
    name: string;
    type: string;
    current: number;
    target: number;
    targetDate: string;
  }

  interface LifeGoalConfig {
    name: string;
    target_date: string;
    target_amount: number;
    monthly_allocation?: number;
    inflation_rate?: number | null;
    funded_by?: string[];
  }

  let activeTab = $state<PlanningTab>("projection");
  let mode = $state<ProjectionMode>("simple");
  let activeScenario = $state<ScenarioKey>("expected");

  let loading = $state(true);
  let error = $state<string | null>(null);
  let dna = $state<FinancialProfile | null>(null);
  let base = $state<BaseProjectionResponse | null>(null);
  let goals = $state<GoalSummary[]>([]);
  let projectedGoals = $state<Array<{ name?: string; fire_probability?: number; projected_date?: string }>>([]);
  let selectedGoalDetail = $state<{ name: string; data: Record<string, unknown> } | null>(null);
  let rawConfig = $state<Record<string, unknown> | null>(null);
  let lifeGoals = $state<LifeGoalConfig[]>([]);
  let savingLifeGoals = $state(false);
  let lifeGoalStatus = $state<string | null>(null);

  // Core 3 levers
  let years = $state(15);
  let monthlyContribution = $state(0);
  let expectedCagr = $state(10);

  // Collapsible assumptions
  let swr = $state(4);
  let inflationRate = $state(6);
  let cagrSpread = $state(3);
  let adjustForInflation = $state(true);
  let volatility = $state(16);

  let showAssumptions = $state(false);
  let showDna = $state(false);

  // Monte Carlo
  let simData = $state<SimResponse | null>(null);
  let simLoading = $state(false);

  onMount(async () => {
    try {
      const [projRes, dnaRes, goalsRes, projGoalsRes, cfg] = await Promise.all([
        apiGet<BaseProjectionResponse>("/api/networth/projection"),
        apiGet<{ profile: FinancialProfile }>("/api/projection/dna").catch(() => ({
          profile: null
        })),
        apiGet<{ goals?: GoalSummary[] }>("/api/goals").catch(() => ({ goals: [] })),
        apiGet<{ goals?: typeof projectedGoals }>("/api/projection/goals").catch(() => ({ goals: [] })),
        getFullConfig().catch(() => null)
      ]);

      base = projRes;
      dna = dnaRes.profile;
      goals = goalsRes.goals || [];
      projectedGoals = projGoalsRes.goals || [];
      if (cfg) {
        rawConfig = cfg;
        const g = (cfg.goals as Record<string, unknown> | undefined) || {};
        lifeGoals = Array.isArray(g.life) ? (g.life as LifeGoalConfig[]) : [];
      }

      monthlyContribution = Math.round(projRes.monthly_contribution || 0);
      expectedCagr = projRes.expected_cagr || 10;
      swr = projRes.swr || 4;
      if (dna?.return_volatility) {
        volatility = Math.min(40, Math.max(5, Math.round(dna.return_volatility * 10) / 10));
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load planning data";
    } finally {
      loading = false;
    }
  });

  function calibrateFromDna() {
    if (!dna) return;
    monthlyContribution = Math.round(dna.monthly_contribution || 0);
    if (dna.price_months_covered >= 3 && dna.historical_return > 0) {
      expectedCagr = Math.min(35, Math.max(0, Math.round(dna.historical_return * 10) / 10));
      cagrSpread = dna.return_volatility > 0 ? Math.round(dna.return_volatility * 10) / 10 : 3;
    }
    if (dna.expense_years_covered >= 2 && dna.expense_growth_rate > 0) {
      inflationRate = Math.min(15, Math.max(0, Math.round(dna.expense_growth_rate * 10) / 10));
    }
  }

  function toReal(nominalPct: number): number {
    return ((1 + nominalPct / 100) / (1 + inflationRate / 100) - 1) * 100;
  }

  function exportProjectionCsv() {
    const headers = ["Year", "Starting", "Contribution", "Return", "Ending", "FIRE"];
    const lines = [
      headers.join(","),
      ...tableRows.map((r) =>
        [
          r.year,
          r.start.toFixed(2),
          r.contribution.toFixed(2),
          r.returnAmount.toFixed(2),
          r.end.toFixed(2),
          r.isFire ? "YES" : "NO"
        ].join(",")
      )
    ];
    const blob = new Blob([lines.join("\n")], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `paisa-projection-${activeScenario}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function addLifeGoal() {
    lifeGoals = [
      ...lifeGoals,
      {
        name: "New Milestone",
        target_date: dayjs().add(3, "year").format("YYYY-MM-DD"),
        target_amount: 500000,
        monthly_allocation: 0,
        funded_by: []
      }
    ];
  }

  function removeLifeGoal(idx: number) {
    lifeGoals = lifeGoals.filter((_, i) => i !== idx);
  }

  async function saveLifeGoals() {
    if (!rawConfig) return;
    savingLifeGoals = true;
    lifeGoalStatus = null;
    const nextConfig: Record<string, unknown> = structuredClone(rawConfig);
    const prevGoals = (nextConfig.goals as Record<string, unknown> | undefined) || {};
    nextConfig.goals = {
      retirement: prevGoals.retirement || [],
      savings: prevGoals.savings || [],
      life: lifeGoals.map((g, i) => ({
        ...g,
        priority: lifeGoals.length - i,
        target_amount: Number(g.target_amount || 0),
        monthly_allocation: Number(g.monthly_allocation || 0),
        funded_by: g.funded_by || []
      }))
    };
    const res = await updateConfig(nextConfig);
    savingLifeGoals = false;
    if (res.success) {
      rawConfig = nextConfig;
      lifeGoalStatus = "Saved life milestones.";
      try {
        const goalsRes = await apiGet<{ goals?: GoalSummary[] }>("/api/goals");
        goals = goalsRes.goals || [];
      } catch {}
    } else {
      lifeGoalStatus = res.error || "Failed to save life milestones.";
    }
  }

  function buildSeries(cagrPct: number): ProjectionPoint[] {
    if (!base) return [];
    const monthlyRate = Math.pow(1 + cagrPct / 100, 1 / 12) - 1;
    let curr = base.current_networth;
    const pts: ProjectionPoint[] = [];
    const now = dayjs();
    for (let m = 1; m <= years * 12; m++) {
      curr = curr * (1 + monthlyRate) + monthlyContribution;
      pts.push({ date: now.add(m, "month").format("YYYY-MM-DD"), balance: curr });
    }
    return pts;
  }

  const effectiveExpected = $derived(adjustForInflation ? toReal(expectedCagr) : expectedCagr);
  const effectiveConservative = $derived(
    adjustForInflation ? toReal(expectedCagr - cagrSpread) : expectedCagr - cagrSpread
  );
  const effectiveOptimistic = $derived(
    adjustForInflation ? toReal(expectedCagr + cagrSpread) : expectedCagr + cagrSpread
  );

  const expectedPoints = $derived(buildSeries(effectiveExpected));
  const conservativePoints = $derived(buildSeries(effectiveConservative));
  const optimisticPoints = $derived(buildSeries(effectiveOptimistic));

  const targetCorpus = $derived(base && swr > 0 ? base.annual_expenses / (swr / 100) : 0);

  const yearsToFire = $derived.by(() => {
    if (targetCorpus <= 0) return null;
    for (let i = 0; i < expectedPoints.length; i++) {
      const yrs = (i + 1) / 12;
      const threshold = adjustForInflation
        ? targetCorpus
        : targetCorpus * Math.pow(1 + inflationRate / 100, yrs);
      if (expectedPoints[i].balance >= threshold) {
        return Math.round(yrs * 10) / 10;
      }
    }
    return null;
  });

  const fireProgress = $derived(
    base && targetCorpus > 0 ? Math.min(100, (base.current_networth / targetCorpus) * 100) : 0
  );

  const tableRows = $derived.by(() => {
    if (!base) return [];
    const cagrPct =
      activeScenario === "expected"
        ? effectiveExpected
        : activeScenario === "conservative"
          ? effectiveConservative
          : effectiveOptimistic;
    const monthlyRate = Math.pow(1 + cagrPct / 100, 1 / 12) - 1;
    let curr = base.current_networth;
    const rows = [];
    for (let y = 1; y <= years; y++) {
      const start = curr;
      let ret = 0;
      for (let m = 0; m < 12; m++) {
        const mr = curr * monthlyRate;
        ret += mr;
        curr += mr + monthlyContribution;
      }
      const exp = adjustForInflation
        ? base.annual_expenses
        : base.annual_expenses * Math.pow(1 + inflationRate / 100, y - 1);
      const rowTarget = swr > 0 ? exp / (swr / 100) : 0;
      rows.push({
        year: y,
        start,
        contribution: monthlyContribution * 12,
        returnAmount: ret,
        end: curr,
        expenses: exp,
        isFire: rowTarget > 0 && curr >= rowTarget
      });
    }
    return rows;
  });

  async function runMonteCarlo() {
    simLoading = true;
    try {
      const res = await fetch("/api/projection/simulate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          iterations: 1000,
          months_to_project: years * 12,
          monthly_contribution: monthlyContribution,
          contribution_growth_rate: 5,
          expected_return: expectedCagr,
          return_volatility: volatility,
          inflation_rate: inflationRate,
          swr
        })
      });
      if (res.ok) {
        simData = (await res.json()) as SimResponse;
      }
    } finally {
      simLoading = false;
    }
  }

  $effect(() => {
    if (mode === "montecarlo" && base) {
      const _y = years;
      const _mc = monthlyContribution;
      const _ec = expectedCagr;
      const _v = volatility;
      const _s = swr;
      const _i = inflationRate;
      void runMonteCarlo();
    }
  });

  const fanBand = $derived.by((): FanBand | null => {
    if (!simData?.simulation?.bands) return null;
    const mapBand = (arr: Array<{ date: string; balance_amount: number }>) =>
      (arr || []).map((p) => ({ date: p.date, balance: p.balance_amount }));
    return {
      p10: mapBand(simData.simulation.bands.p10),
      p25: mapBand(simData.simulation.bands.p25),
      p50: mapBand(simData.simulation.bands.p50),
      p75: mapBand(simData.simulation.bands.p75),
      p90: mapBand(simData.simulation.bands.p90)
    };
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Planning &amp; FIRE Studio</h1>
      {#if base}
        <p class="text-xs text-[var(--muted)] mt-0.5 tabular-nums">
          Baseline: <strong>{formatMoney(base.current_networth)}</strong> net worth ·
          <strong>{formatMoney(base.annual_expenses)}</strong>/yr expenses
        </p>
      {/if}
    </div>

    {#if activeTab === "projection"}
      <div class="flex flex-wrap items-center gap-2">
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-3 py-1 rounded-md cursor-pointer {mode === 'simple'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (mode = "simple")}
          >
            Simple
          </button>
          <button
            type="button"
            class="px-3 py-1 rounded-md cursor-pointer {mode === 'montecarlo'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (mode = "montecarlo")}
          >
            Monte Carlo
          </button>
        </div>

        {#if dna}
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
            onclick={calibrateFromDna}
          >
            <Sparkles class="w-3.5 h-3.5 text-[var(--accent)]" />
            <span>Calibrate</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
            onclick={() => (showDna = !showDna)}
          >
            <Dna class="w-3.5 h-3.5" />
            <span>Financial DNA</span>
          </button>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm">
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'projection'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "projection")}
    >
      Projection
    </button>
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'goals'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "goals")}
    >
      Goals ({goals.length})
    </button>
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'whatif'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "whatif")}
    >
      What-If
    </button>
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'drawdown'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "drawdown")}
    >
      Drawdown
    </button>
    {#if runtime.defaultCurrency === "INR"}
      <button
        type="button"
        class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'tax'
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)]'}"
        onclick={() => (activeTab = "tax")}
      >
        Tax
      </button>
    {/if}
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Failed to load planning data</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "projection"}
    {#if showDna && dna}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs"
      >
        <div>
          <div class="text-[var(--muted)] uppercase font-semibold">Annual Income</div>
          <div class="text-base font-semibold tabular-nums mt-0.5">
            {formatMoney(dna.annual_income)}
          </div>
        </div>
        <div>
          <div class="text-[var(--muted)] uppercase font-semibold">Annual Expenses</div>
          <div class="text-base font-semibold tabular-nums mt-0.5">
            {formatMoney(dna.annual_expenses)}
          </div>
        </div>
        <div>
          <div class="text-[var(--muted)] uppercase font-semibold">Savings Rate</div>
          <div class="text-base font-semibold tabular-nums mt-0.5">
            {dna.savings_rate.toFixed(1)}%
          </div>
        </div>
        <div>
          <div class="text-[var(--muted)] uppercase font-semibold">Historical Return</div>
          <div class="text-base font-semibold tabular-nums text-[var(--gain)] mt-0.5">
            {dna.historical_return.toFixed(1)}%
          </div>
        </div>
      </div>
    {/if}

    <!-- Outcome KPIs -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          {mode === "montecarlo" ? "Median Years to FIRE (P50)" : "Years to FIRE"}
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">
          {#if mode === "montecarlo"}
            {simData?.simulation.fire_year_p50
              ? `${simData.simulation.fire_year_p50} years`
              : `Not in ${years}y`}
          {:else}
            {yearsToFire !== null ? `${yearsToFire} years` : `Not in ${years}y`}
          {/if}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          {mode === "montecarlo" ? "FIRE Probability" : "FIRE Progress"}
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">
          {mode === "montecarlo"
            ? `${((simData?.simulation.fire_probability || 0) * 100).toFixed(1)}%`
            : `${fireProgress.toFixed(1)}%`}
        </div>
        <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden mt-2">
          <div
            class="h-full rounded-full bg-[var(--accent)]"
            style="width: {mode === 'montecarlo'
              ? (simData?.simulation.fire_probability || 0) * 100
              : fireProgress}%"
          ></div>
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Target Corpus ({adjustForInflation && mode === "simple" ? "Today's money" : "Nominal"})
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">
          {formatMoney(
            mode === "montecarlo" ? simData?.simulation.target_corpus || targetCorpus : targetCorpus
          )}
        </div>
      </div>
    </div>

    <!-- Levers + Chart -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <div
        class="lg:col-span-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4"
      >
        <h2 class="text-sm font-semibold">Core Levers</h2>
        <div class="space-y-3.5 text-xs">
          <div>
            <div class="flex justify-between mb-1">
              <span class="font-medium">Projection Horizon</span>
              <span class="tabular-nums font-semibold">{years} years</span>
            </div>
            <input
              type="range"
              min="1"
              max="40"
              bind:value={years}
              class="w-full accent-[var(--accent)]"
            />
          </div>
          <div>
            <div class="flex justify-between mb-1">
              <span class="font-medium">Monthly Contribution</span>
              <span class="tabular-nums font-semibold">{formatMoney(monthlyContribution)}</span>
            </div>
            <input
              type="range"
              min="0"
              max={Math.max(500000, monthlyContribution * 2)}
              step="1000"
              bind:value={monthlyContribution}
              class="w-full accent-[var(--accent)]"
            />
          </div>
          <div>
            <div class="flex justify-between mb-1">
              <span class="font-medium">Expected Return</span>
              <span class="tabular-nums font-semibold">{expectedCagr.toFixed(1)}%</span>
            </div>
            <input
              type="range"
              min="0"
              max="25"
              step="0.5"
              bind:value={expectedCagr}
              class="w-full accent-[var(--accent)]"
            />
          </div>
        </div>

        <div class="pt-3 border-t border-[var(--border)]">
          <button
            type="button"
            class="w-full flex items-center justify-between text-xs font-medium text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
            onclick={() => (showAssumptions = !showAssumptions)}
          >
            <span>Assumptions ({swr}% SWR · {inflationRate}% inf)</span>
            {#if showAssumptions}
              <ChevronUp class="w-4 h-4" />
            {:else}
              <ChevronDown class="w-4 h-4" />
            {/if}
          </button>

          {#if showAssumptions}
            <div class="mt-3 space-y-3 text-xs">
              {#if mode === "simple"}
                <label class="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" bind:checked={adjustForInflation} />
                  <span>Adjust for inflation (today's money)</span>
                </label>
              {/if}
              <div>
                <div class="flex justify-between mb-1">
                  <span>Safe Withdrawal Rate</span>
                  <span class="tabular-nums">{swr}%</span>
                </div>
                <input
                  type="range"
                  min="2"
                  max="7"
                  step="0.1"
                  bind:value={swr}
                  class="w-full accent-[var(--accent)]"
                />
              </div>
              <div>
                <div class="flex justify-between mb-1">
                  <span>Inflation Rate</span>
                  <span class="tabular-nums">{inflationRate}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="15"
                  step="0.5"
                  bind:value={inflationRate}
                  class="w-full accent-[var(--accent)]"
                />
              </div>
              {#if mode === "montecarlo"}
                <div>
                  <div class="flex justify-between mb-1">
                    <span>Volatility</span>
                    <span class="tabular-nums">{volatility}%</span>
                  </div>
                  <input
                    type="range"
                    min="2"
                    max="40"
                    step="0.5"
                    bind:value={volatility}
                    class="w-full accent-[var(--accent)]"
                  />
                </div>
              {:else}
                <div>
                  <div class="flex justify-between mb-1">
                    <span>Scenario Spread</span>
                    <span class="tabular-nums">±{cagrSpread}%</span>
                  </div>
                  <input
                    type="range"
                    min="1"
                    max="10"
                    step="0.5"
                    bind:value={cagrSpread}
                    class="w-full accent-[var(--accent)]"
                  />
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>

      <div class="lg:col-span-8 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5">
        {#if mode === "montecarlo" && simLoading}
          <div class="h-64 flex items-center justify-center text-sm text-[var(--muted)]">
            Running 1,000 Monte Carlo simulations…
          </div>
        {:else}
          {#if mode === "simple"}
            <div class="grid grid-cols-3 gap-3 mb-4 text-xs">
              {#each [
                { key: "conservative", label: "Conservative", cagr: effectiveConservative, pts: conservativePoints, tone: "text-[var(--warning)]" },
                { key: "expected", label: "Expected", cagr: effectiveExpected, pts: expectedPoints, tone: "text-[var(--accent)]" },
                { key: "optimistic", label: "Optimistic", cagr: effectiveOptimistic, pts: optimisticPoints, tone: "text-[var(--gain)]" }
              ] as sc}
                {@const finalVal = sc.pts[sc.pts.length - 1]?.balance ?? 0}
                <div class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/40 px-3 py-2">
                  <div class="flex items-center justify-between gap-1">
                    <span class="font-semibold {sc.tone}">{sc.label}</span>
                    <span class="font-mono text-[11px] text-[var(--muted)]">
                      {sc.cagr.toFixed(1)}% {adjustForInflation ? "Real " : ""}CAGR
                    </span>
                  </div>
                  <div class="text-sm font-semibold tabular-nums mt-0.5">{formatMoney(finalVal)}</div>
                </div>
              {/each}
            </div>
          {/if}
          <ProjectionChart
            {mode}
            conservative={conservativePoints}
            expected={expectedPoints}
            optimistic={optimisticPoints}
            fan={fanBand}
            {targetCorpus}
          />
        {/if}
      </div>
    </div>

    <!-- Year-by-Year Table (Simple mode) -->
    {#if mode === "simple"}
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        <div
          class="px-4 py-3 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-3"
        >
          <h3 class="text-sm font-semibold">Year-by-Year Projection</h3>
          <div class="flex flex-wrap items-center gap-2">
            <div
              class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
            >
              {#each [
                { key: "expected", label: "Expected", cagr: effectiveExpected },
                { key: "conservative", label: "Conservative", cagr: effectiveConservative },
                { key: "optimistic", label: "Optimistic", cagr: effectiveOptimistic }
              ] as s}
                <button
                  type="button"
                  class="px-2.5 py-1 rounded-md cursor-pointer {activeScenario === s.key
                    ? 'bg-[var(--surface)] font-semibold'
                    : 'text-[var(--muted)]'}"
                  onclick={() => (activeScenario = s.key as ScenarioKey)}
                >
                  {s.label} ({s.cagr.toFixed(1)}%)
                </button>
              {/each}
            </div>
            <span class="px-2 py-1 rounded-md bg-[var(--surface-elevated)] border border-[var(--border)] text-xs font-mono text-[var(--muted)]">
              {(activeScenario === "expected"
                ? effectiveExpected
                : activeScenario === "conservative"
                  ? effectiveConservative
                  : effectiveOptimistic
              ).toFixed(1)}% {adjustForInflation ? "Real " : ""}CAGR
            </span>
            <button
              type="button"
              onclick={exportProjectionCsv}
              class="px-2.5 py-1 rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] text-xs font-medium hover:border-[var(--accent)] cursor-pointer"
            >
              Export CSV
            </button>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Year</th>
                <th class="py-2.5 px-4 text-right font-medium">Starting</th>
                <th class="py-2.5 px-4 text-right font-medium">Contribution</th>
                <th class="py-2.5 px-4 text-right font-medium">Return</th>
                <th class="py-2.5 px-4 text-right font-medium">Ending</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each tableRows as r}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-4 font-medium">
                    Year {r.year}
                    {#if r.isFire}
                      <span
                        class="ml-1.5 px-1.5 py-0.5 text-[10px] rounded bg-[var(--accent)]/15 text-[var(--accent)] font-semibold"
                      >
                        FIRE
                      </span>
                    {/if}
                  </td>
                  <td class="py-2 px-4 text-right">{formatMoney(r.start)}</td>
                  <td class="py-2 px-4 text-right">{formatMoney(r.contribution)}</td>
                  <td
                    class="py-2 px-4 text-right {r.returnAmount >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {formatMoney(r.returnAmount)}
                  </td>
                  <td class="py-2 px-4 text-right font-semibold">{formatMoney(r.end)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {:else if activeTab === "goals"}
    <!-- Goals Tab -->
    {#if goals.length === 0}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
      >
        No goals configured yet.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each goals as g (g.id)}
          {@const pct = g.target > 0 ? Math.min(100, (g.current / g.target) * 100) : 0}
          {@const pg = projectedGoals.find((p) => p.name === g.name)}
          <button
            type="button"
            onclick={async () => {
              if (g.type !== "retirement" && g.type !== "savings") return;
              const data = await apiGet<Record<string, unknown>>(
                `/api/goals/${g.type}/${encodeURIComponent(g.name)}`
              ).catch(() => null);
              if (data) selectedGoalDetail = { name: g.name, data };
            }}
            class="text-left rounded-xl bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)] p-5 space-y-3 cursor-pointer"
          >
            <div class="flex items-center justify-between gap-2">
              <h3 class="text-base font-semibold">{g.name}</h3>
              <div class="flex items-center gap-1.5">
                {#if pg?.fire_probability !== undefined}
                  <span class="text-[11px] px-2 py-0.5 rounded-full bg-[var(--accent)]/15 text-[var(--accent)] font-medium tabular-nums">
                    {(pg.fire_probability * 100).toFixed(0)}% prob
                  </span>
                {/if}
                <span
                  class="text-xs px-2 py-0.5 rounded-full bg-[var(--surface-elevated)] border border-[var(--border)] capitalize"
                >
                  {g.type}
                </span>
              </div>
            </div>
            <div class="h-2 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
              <div class="h-full rounded-full bg-[var(--accent)]" style="width: {pct}%"></div>
            </div>
            <div class="flex justify-between text-xs tabular-nums">
              <span
                ><strong>{formatMoney(g.current)}</strong> of {formatMoney(g.target)} ({pct.toFixed(
                  1
                )}%)</span
              >
              <span class="text-[var(--muted)]">{g.targetDate}</span>
            </div>
          </button>
        {/each}
      </div>
      {#if selectedGoalDetail}
        <div class="mt-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-2">
          <div class="flex items-center justify-between">
            <h4 class="text-sm font-semibold">Goal Detail: {selectedGoalDetail.name}</h4>
            <button
              type="button"
              onclick={() => (selectedGoalDetail = null)}
              class="text-xs text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
            >
              Close
            </button>
          </div>
          <pre class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto max-h-60">{JSON.stringify(selectedGoalDetail.data, null, 2)}</pre>
        </div>
      {/if}
    {/if}

    <!-- Life Milestones Editor -->
    {#if !runtime.readonly && rawConfig}
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4 mt-5">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 class="text-sm font-semibold">Life Milestones</h3>
            <p class="text-xs text-[var(--muted)]">
              Add or edit life goals persisted to your Paisa configuration
            </p>
          </div>
          <div class="flex items-center gap-2">
            <button
              type="button"
              onclick={addLifeGoal}
              class="px-3 py-1.5 rounded-lg border border-[var(--border)] text-xs font-medium hover:border-[var(--accent)] cursor-pointer"
            >
              + Add Milestone
            </button>
            <button
              type="button"
              disabled={savingLifeGoals}
              onclick={saveLifeGoals}
              class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-60"
            >
              {savingLifeGoals ? "Saving…" : "Save Milestones"}
            </button>
          </div>
        </div>

        {#if lifeGoalStatus}
          <p class="text-xs text-[var(--muted)]">{lifeGoalStatus}</p>
        {/if}

        {#if lifeGoals.length === 0}
          <p class="text-xs text-[var(--muted)]">No life milestones configured.</p>
        {:else}
          <div class="space-y-3">
            {#each lifeGoals as lg, idx}
              <div
                class="grid grid-cols-1 sm:grid-cols-12 gap-2.5 items-end p-3 rounded-lg bg-[var(--surface-elevated)]/60 border border-[var(--border)] text-xs"
              >
                <div class="sm:col-span-4">
                  <label class="block text-[var(--muted)] mb-1" for={`lg-name-${idx}`}>Name</label>
                  <input
                    id={`lg-name-${idx}`}
                    type="text"
                    bind:value={lg.name}
                    class="w-full px-2.5 py-1.5 rounded-md bg-[var(--surface)] border border-[var(--border)] outline-none"
                  />
                </div>
                <div class="sm:col-span-3">
                  <label class="block text-[var(--muted)] mb-1" for={`lg-date-${idx}`}>Target Date</label>
                  <input
                    id={`lg-date-${idx}`}
                    type="date"
                    bind:value={lg.target_date}
                    class="w-full px-2.5 py-1.5 rounded-md bg-[var(--surface)] border border-[var(--border)] outline-none tabular-nums"
                  />
                </div>
                <div class="sm:col-span-2">
                  <label class="block text-[var(--muted)] mb-1" for={`lg-amt-${idx}`}>Target Amount</label>
                  <input
                    id={`lg-amt-${idx}`}
                    type="number"
                    bind:value={lg.target_amount}
                    class="w-full px-2.5 py-1.5 rounded-md bg-[var(--surface)] border border-[var(--border)] outline-none tabular-nums"
                  />
                </div>
                <div class="sm:col-span-2">
                  <label class="block text-[var(--muted)] mb-1" for={`lg-sip-${idx}`}>Monthly Alloc</label>
                  <input
                    id={`lg-sip-${idx}`}
                    type="number"
                    bind:value={lg.monthly_allocation}
                    class="w-full px-2.5 py-1.5 rounded-md bg-[var(--surface)] border border-[var(--border)] outline-none tabular-nums"
                  />
                </div>
                <div class="sm:col-span-1 flex justify-end">
                  <button
                    type="button"
                    onclick={() => removeLifeGoal(idx)}
                    class="px-2 py-1.5 rounded-md border border-[var(--border)] text-[var(--loss)] hover:bg-[var(--loss)]/10 cursor-pointer"
                  >
                    Remove
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {:else}
    <PlanningSubtools tool={activeTab} />
  {/if}
</div>
