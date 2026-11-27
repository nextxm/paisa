<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { CheckCircle2, Search } from "lucide-svelte";
  import { apiGet, formatMoney, formatShortDate } from "$lib/api";

  type LedgerTab = "health" | "prices" | "logs";
  type HealthFilter = "fix" | "review" | "dismissed" | "all";

  interface DoctorFinding {
    id: string;
    rule_id: string;
    severity: "fix" | "review" | "info";
    title: string;
    description?: string;
    account?: string;
    dismissed?: boolean;
  }

  interface DoctorSummary {
    total: number;
    fix_count: number;
    review_count: number;
    info_count: number;
    dismissed_count: number;
  }

  interface PriceRow {
    commodity_name?: string;
    base?: string;
    quote?: string;
    price?: number;
    date?: string;
    source?: string;
  }

  interface LogEntry {
    time?: string;
    level?: string;
    message?: string;
  }

  let activeTab = $state<LedgerTab>("health");
  let healthFilter = $state<HealthFilter>("fix");
  let search = $state("");

  let loading = $state(true);
  let error = $state<string | null>(null);

  let findings = $state<DoctorFinding[]>([]);
  let summary = $state<DoctorSummary>({
    total: 0,
    fix_count: 0,
    review_count: 0,
    info_count: 0,
    dismissed_count: 0
  });
  let prices = $state<PriceRow[]>([]);
  let logs = $state<LogEntry[]>([]);

  async function loadHealth() {
    const res = await apiGet<{ findings?: DoctorFinding[]; summary?: DoctorSummary }>(
      "/api/doctor/findings"
    );
    findings = res.findings || [];
    summary = res.summary || summary;
    if (summary.fix_count === 0 && summary.review_count > 0 && healthFilter === "fix") {
      healthFilter = "review";
    } else if (summary.fix_count === 0 && summary.review_count === 0 && healthFilter === "fix") {
      healthFilter = "all";
    }
  }

  onMount(async () => {
    try {
      const [, priceRes, logRes] = await Promise.all([
        loadHealth(),
        apiGet<{ prices?: Record<string, PriceRow[]> | PriceRow[] }>("/api/price").catch(() => ({
          prices: []
        })),
        apiGet<{ logs?: LogEntry[] }>("/api/logs").catch(() => ({ logs: [] }))
      ]);

      if (Array.isArray(priceRes.prices)) {
        prices = priceRes.prices;
      } else if (priceRes.prices && typeof priceRes.prices === "object") {
        prices = Object.entries(priceRes.prices).map(([name, list]) => {
          const latest = Array.isArray(list) ? list[0] : undefined;
          return {
            commodity_name: name,
            ...latest
          };
        });
      }

      logs = logRes.logs || [];
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load ledger tools";
    } finally {
      loading = false;
    }
  });

  async function dismissFinding(f: DoctorFinding) {
    await fetch("/api/doctor/dismiss", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fingerprint: f.id, rule_id: f.rule_id, note: "" })
    });
    await loadHealth();
  }

  async function undismissFinding(f: DoctorFinding) {
    await fetch("/api/doctor/undismiss", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fingerprint: f.id })
    });
    await loadHealth();
  }

  const filteredFindings = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return findings.filter((f) => {
      if (healthFilter === "fix" && (f.dismissed || f.severity !== "fix")) return false;
      if (
        healthFilter === "review" &&
        (f.dismissed || (f.severity !== "review" && f.severity !== "info"))
      )
        return false;
      if (healthFilter === "dismissed" && !f.dismissed) return false;
      if (!q) return true;
      return (
        f.title?.toLowerCase().includes(q) ||
        f.description?.toLowerCase().includes(q) ||
        f.account?.toLowerCase().includes(q)
      );
    });
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
      <h1 class="text-xl font-semibold tracking-tight">Ledger &amp; Tools</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Data health diagnostics, market prices, and system logs
      </p>
    </div>

    <div class="relative w-full sm:w-72">
      <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
      <input
        type="search"
        bind:value={search}
        placeholder="Filter…"
        class="w-full pl-9 pr-3 py-1.5 rounded-lg text-sm bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
      />
    </div>
  </div>

  <!-- Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm">
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'health'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "health")}
    >
      Data Health ({summary.fix_count + summary.review_count})
    </button>
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'prices'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "prices")}
    >
      Market Prices ({prices.length})
    </button>
    <button
      type="button"
      class="pb-2.5 border-b-2 cursor-pointer {activeTab === 'logs'
        ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
        : 'border-transparent text-[var(--muted)]'}"
      onclick={() => (activeTab = "logs")}
    >
      System Logs ({logs.length})
    </button>
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Failed to load tools</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "health"}
    <div class="flex flex-wrap gap-2 text-xs">
      {#each [
        { id: "fix", label: `Fix Now (${summary.fix_count})` },
        { id: "review", label: `To Review (${summary.review_count + summary.info_count})` },
        { id: "dismissed", label: `Dismissed (${summary.dismissed_count})` },
        { id: "all", label: `All (${summary.total})` }
      ] as f}
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border cursor-pointer {healthFilter === f.id
            ? 'bg-[var(--surface-elevated)] border-[var(--accent)] font-semibold'
            : 'bg-[var(--surface)] border-[var(--border)] text-[var(--muted)]'}"
          onclick={() => (healthFilter = f.id as HealthFilter)}
        >
          {f.label}
        </button>
      {/each}
    </div>

    {#if filteredFindings.length === 0}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center space-y-2"
      >
        <CheckCircle2 class="w-8 h-8 text-[var(--gain)] mx-auto" />
        <div class="text-sm font-medium">No findings in this view</div>
      </div>
    {:else}
      <div class="space-y-3">
        {#each filteredFindings as f (f.id)}
          <div
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex items-start justify-between gap-4"
          >
            <div class="space-y-1 min-w-0">
              <div class="flex items-center gap-2">
                <span
                  class="px-2 py-0.5 rounded text-[10px] font-semibold uppercase {f.severity ===
                  'fix'
                    ? 'bg-[var(--loss)]/15 text-[var(--loss)]'
                    : 'bg-[var(--warning)]/15 text-[var(--warning)]'}"
                >
                  {f.severity}
                </span>
                <span class="text-sm font-semibold">{f.title}</span>
              </div>
              {#if f.description}
                <p class="text-xs text-[var(--muted)]">{f.description}</p>
              {/if}
              {#if f.account}
                <div class="text-xs text-[var(--muted)] font-mono">{f.account}</div>
              {/if}
            </div>
            <button
              type="button"
              class="px-2.5 py-1 rounded-md text-xs border border-[var(--border)] hover:bg-[var(--surface-elevated)] shrink-0 cursor-pointer"
              onclick={() => (f.dismissed ? undismissFinding(f) : dismissFinding(f))}
            >
              {f.dismissed ? "Restore" : "Dismiss"}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  {:else if activeTab === "prices"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if filteredPrices.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">No price entries found.</div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Commodity</th>
                <th class="py-2.5 px-4 text-right font-medium">Price</th>
                <th class="py-2.5 px-4 font-medium">Date</th>
                <th class="py-2.5 px-4 font-medium">Source</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each filteredPrices as p}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-4 font-medium">{p.commodity_name || p.base || "—"}</td>
                  <td class="py-2 px-4 text-right font-semibold">
                    {p.price !== undefined ? formatMoney(p.price, p.quote || "INR") : "—"}
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
  {:else}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if logs.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">No system logs available.</div>
      {:else}
        <div class="divide-y divide-[var(--border)] font-mono text-xs">
          {#each logs.slice(0, 150) as log}
            <div class="px-4 py-2 flex items-start gap-3">
              <span class="text-[var(--muted)] shrink-0">
                {log.time ? dayjs(log.time).format("HH:mm:ss") : ""}
              </span>
              <span class="uppercase font-semibold shrink-0 w-14 text-[var(--muted)]">
                {log.level || "info"}
              </span>
              <span class="break-all">{log.message || ""}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
