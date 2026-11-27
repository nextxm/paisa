<script lang="ts">
  import { onMount } from "svelte";
  import { formatMoney } from "$lib/api";

  let {
    tool
  }: {
    tool: "whatif" | "drawdown" | "tax";
  } = $props();

  // What-If state
  let whatifLoading = $state(false);
  let whatifData = $state<{
    baseline?: { fire_year_p50?: number; fire_probability?: number; final_p50?: number };
    scenarios?: Array<{
      name: string;
      fire_year_p50?: number;
      fire_probability?: number;
      final_p50?: number;
    }>;
  } | null>(null);
  let extraSip = $state(10000);
  let jobLossMonths = $state(0);

  // Drawdown state
  let drawdownAmount = $state(500000);
  let drawdownLoading = $state(false);
  let drawdownData = $state<{
    total_tax?: number;
    effective_tax_rate?: number;
    net_proceeds?: number;
    allocations?: Array<{ account: string; amount: number; tax?: number }>;
  } | null>(null);

  // Tax state
  let taxLoading = $state(false);
  let harvestables = $state<Array<{ account?: string; unrealized_loss?: number; stcg?: number; ltcg?: number }>>([]);
  let capitalGains = $state<Array<{ account?: string; stcg?: number; ltcg?: number }>>([]);
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

  async function runWhatIf() {
    whatifLoading = true;
    try {
      const res = await fetch("/api/projection/whatif", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          baseline: {
            iterations: 500,
            months_to_project: 180,
            expected_return: 12,
            return_volatility: 16,
            inflation_rate: 6,
            swr: 4
          },
          scenarios: [
            { name: `+${formatMoney(extraSip)}/mo SIP`, overrides: { monthly_contribution: extraSip } },
            {
              name: jobLossMonths > 0 ? `Pause ${jobLossMonths}m / Slower Growth` : "Conservative 8%",
              overrides: { monthly_contribution: 0, expected_return: 8 }
            }
          ]
        })
      });
      if (res.ok) {
        whatifData = await res.json();
      }
    } finally {
      whatifLoading = false;
    }
  }

  async function runDrawdown() {
    drawdownLoading = true;
    try {
      const res = await fetch("/api/projection/drawdown", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          amount: drawdownAmount,
          buckets: [
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
          ],
          include_projection_impact: false
        })
      });
      if (res.ok) {
        drawdownData = await res.json();
      }
    } finally {
      drawdownLoading = false;
    }
  }

  async function loadTax() {
    taxLoading = true;
    try {
      const [hRes, cgRes, salRes] = await Promise.all([
        fetch("/api/harvest").then((r) => (r.ok ? r.json() : { harvestables: {} })),
        fetch("/api/capital_gains").then((r) => (r.ok ? r.json() : { capital_gains: {} })),
        fetch("/api/schedule_al").then((r) => (r.ok ? r.json() : { schedule_als: {} }))
      ]);
      harvestables = Object.values(hRes.harvestables || {});
      capitalGains = Object.values(cgRes.capital_gains || {});
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
  <div class="space-y-4">
    <div
      class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-end gap-4 text-xs"
    >
      <div>
        <label class="block text-[var(--muted)] mb-1" for="wi-sip">Extra Monthly SIP</label>
        <input
          id="wi-sip"
          type="number"
          step="1000"
          bind:value={extraSip}
          class="px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
        />
      </div>
      <button
        type="button"
        disabled={whatifLoading}
        class="px-4 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
        onclick={runWhatIf}
      >
        {whatifLoading ? "Running…" : "Compare Scenarios"}
      </button>
    </div>

    {#if whatifData}
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        {#if whatifData.baseline}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-1">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Baseline
            </div>
            <div class="text-xl font-semibold tabular-nums">
              {((whatifData.baseline.fire_probability || 0) * 100).toFixed(1)}% FIRE
            </div>
            <div class="text-xs text-[var(--muted)] tabular-nums">
              Median P50: {formatMoney(whatifData.baseline.final_p50 || 0)}
            </div>
          </div>
        {/if}
        {#each whatifData.scenarios || [] as sc}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-1">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              {sc.name}
            </div>
            <div class="text-xl font-semibold tabular-nums">
              {((sc.fire_probability || 0) * 100).toFixed(1)}% FIRE
            </div>
            <div class="text-xs text-[var(--muted)] tabular-nums">
              Median P50: {formatMoney(sc.final_p50 || 0)}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else if tool === "drawdown"}
  <div class="space-y-4">
    <div
      class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-end gap-4 text-xs"
    >
      <div>
        <label class="block text-[var(--muted)] mb-1" for="dd-amt">Withdrawal Amount</label>
        <input
          id="dd-amt"
          type="number"
          step="10000"
          bind:value={drawdownAmount}
          class="px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
        />
      </div>
      <button
        type="button"
        disabled={drawdownLoading}
        class="px-4 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
        onclick={runDrawdown}
      >
        {drawdownLoading ? "Calculating…" : "Simulate Drawdown"}
      </button>
    </div>

    {#if drawdownData}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Estimated Tax
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-1">
            {formatMoney(drawdownData.total_tax || 0)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Effective Tax Rate
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {(drawdownData.effective_tax_rate || 0).toFixed(2)}%
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Net Proceeds
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
            {formatMoney(drawdownData.net_proceeds || 0)}
          </div>
        </div>
      </div>
    {/if}
  </div>
{:else}
  <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Tax Harvesting Opportunities ({harvestables.length})
      </h3>
      {#if harvestables.length === 0}
        <p class="text-sm text-[var(--muted)]">No harvesting candidates found.</p>
      {:else}
        <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
          {#each harvestables.slice(0, 12) as h}
            <div class="py-2 flex justify-between gap-2">
              <span class="truncate">{h.account || "Holding"}</span>
              <span class="font-medium">{formatMoney(h.unrealized_loss || h.ltcg || 0)}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Capital Gains Summary ({capitalGains.length})
      </h3>
      {#if capitalGains.length === 0}
        <p class="text-sm text-[var(--muted)]">No capital gains recorded.</p>
      {:else}
        <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
          {#each capitalGains.slice(0, 12) as cg}
            <div class="py-2 flex justify-between gap-2">
              <span class="truncate">{cg.account || "Account"}</span>
              <span class="font-medium">
                {formatMoney((cg.stcg || 0) + (cg.ltcg || 0))}
              </span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <!-- Schedule AL -->
  {#if Object.keys(scheduleAls).length > 0}
    {@const activeSchedule = scheduleAls[selectedScheduleYear]}
    <div class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
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
