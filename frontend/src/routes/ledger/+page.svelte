<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { CheckCircle2, Search } from "lucide-svelte";
  import LedgerEditorPanel from "$lib/components/LedgerEditorPanel.svelte";
  import LedgerToolsPanels from "$lib/components/LedgerToolsPanels.svelte";
  import { apiGet, authHeaders, formatMoney, formatShortDate, runtime, updateConfig, type Posting } from "$lib/api";

  type LedgerTab =
    | "health"
    | "editor"
    | "import"
    | "postings"
    | "prices"
    | "fx"
    | "sheets"
    | "firefly"
    | "settings"
    | "logs";
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
  let postings = $state<Posting[]>([]);
  let prices = $state<PriceRow[]>([]);
  let configJson = $state<Record<string, unknown> | null>(null);
  let configDraft = $state("");
  let configSaving = $state(false);
  let configMessage = $state<string | null>(null);
  let logs = $state<LogEntry[]>([]);
  let duplicates = $state<Array<Record<string, unknown>>>([]);
  let outliers = $state<Array<Record<string, unknown>>>([]);
  let providers = $state<Array<{ code?: string; name?: string }>>([]);
  let fireflyItems = $state<Array<Record<string, unknown>>>([]);

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
      const [, postingRes, priceRes, cfgRes, logRes, dupRes, provRes, ffRes] = await Promise.all([
        loadHealth(),
        apiGet<{ postings?: Posting[] }>("/api/posting").catch(() => ({ postings: [] })),
        apiGet<{ prices?: Record<string, PriceRow[]> | PriceRow[] }>("/api/price").catch(() => ({
          prices: []
        })),
        fetch("/connect/paisa.v1.PaisaService/GetConfig", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}"
        })
          .then((r) => (r.ok ? r.json() : { config: null }))
          .catch(() => ({ config: null })),
        apiGet<{ logs?: LogEntry[] }>("/api/logs").catch(() => ({ logs: [] })),
        apiGet<{ duplicates?: Array<Record<string, unknown>>; outliers?: Array<Record<string, unknown>> }>(
          "/api/diagnosis/duplicates"
        ).catch(() => ({ duplicates: [], outliers: [] })),
        apiGet<{ providers?: Array<{ code?: string; name?: string }> }>("/api/price/providers").catch(
          () => ({ providers: [] })
        ),
        apiGet<{ items?: Array<Record<string, unknown>> }>("/api/firefly/reconcile").catch(() => ({
          items: []
        }))
      ]);

      postings = postingRes.postings || [];
      configJson = cfgRes.config || null;
      configDraft = configJson ? JSON.stringify(configJson, null, 2) : "";
      duplicates = dupRes.duplicates || [];
      outliers = dupRes.outliers || [];
      providers = provRes.providers || [];
      fireflyItems = ffRes.items || [];

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

  const filteredPostings = $derived.by(() => {
    const q = search.trim().toLowerCase();
    const list = !q
      ? postings
      : postings.filter(
          (p) =>
            p.account?.toLowerCase().includes(q) ||
            p.payee?.toLowerCase().includes(q) ||
            p.narration?.toLowerCase().includes(q)
        );
    return list.slice(0, 200);
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
        Data health diagnostics, journal editor, postings, prices, settings, and logs
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
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "health", label: `Data Health (${summary.fix_count + summary.review_count})` },
      { id: "editor", label: "Journal Editor" },
      { id: "import", label: "Import" },
      { id: "postings", label: `Postings (${postings.length})` },
      { id: "prices", label: `Market Prices (${prices.length})` },
      { id: "fx", label: "FX Rates" },
      { id: "sheets", label: "Sheets" },
      { id: "firefly", label: `Firefly (${fireflyItems.length})` },
      { id: "settings", label: "Settings" },
      { id: "logs", label: `Logs (${logs.length})` }
    ] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 whitespace-nowrap cursor-pointer {activeTab === tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)]'}"
        onclick={() => (activeTab = tab.id as LedgerTab)}
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

    {#if duplicates.length > 0 || outliers.length > 0}
      <div class="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Potential Duplicates ({duplicates.length})
          </div>
          <pre class="text-xs font-mono bg-[var(--surface-elevated)] p-2.5 rounded-lg overflow-x-auto max-h-40">{JSON.stringify(duplicates.slice(0, 10), null, 2)}</pre>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Outlier Transactions ({outliers.length})
          </div>
          <pre class="text-xs font-mono bg-[var(--surface-elevated)] p-2.5 rounded-lg overflow-x-auto max-h-40">{JSON.stringify(outliers.slice(0, 10), null, 2)}</pre>
        </div>
      </div>
    {/if}
  {:else if activeTab === "editor"}
    <LedgerEditorPanel />
  {:else if activeTab === "import" || activeTab === "fx" || activeTab === "sheets"}
    <LedgerToolsPanels mode={activeTab} />
  {:else if activeTab === "postings"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">Date</th>
              <th class="py-2.5 px-4 font-medium">Account</th>
              <th class="py-2.5 px-4 font-medium">Payee</th>
              <th class="py-2.5 px-4 text-right font-medium">Amount</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each filteredPostings as p}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2 px-4 text-xs text-[var(--muted)] whitespace-nowrap">
                  {formatShortDate(p.date)}
                </td>
                <td class="py-2 px-4 font-medium">{p.account}</td>
                <td class="py-2 px-4 text-xs text-[var(--muted)]">{p.payee || p.narration || "—"}</td>
                <td
                  class="py-2 px-4 text-right font-semibold {p.amount < 0
                    ? 'text-[var(--loss)]'
                    : ''}"
                >
                  {formatMoney(p.amount, p.commodity)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else if activeTab === "prices"}
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
    </div>
  {:else if activeTab === "firefly"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Firefly III Reconciliation ({fireflyItems.length})
      </h3>
      {#if fireflyItems.length === 0}
        <p class="text-sm text-[var(--muted)]">No Firefly reconciliation items configured or pending.</p>
      {:else}
        <pre class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto max-h-96">{JSON.stringify(fireflyItems, null, 2)}</pre>
      {/if}
    </div>
  {:else if activeTab === "settings"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Active Configuration (`paisa.yaml`)
        </h3>
        {#if !runtime.readonly && configJson}
          <button
            type="button"
            disabled={configSaving}
            onclick={async () => {
              configSaving = true;
              configMessage = null;
              try {
                const parsed = JSON.parse(configDraft) as Record<string, unknown>;
                const res = await updateConfig(parsed);
                if (res.success) {
                  configJson = parsed;
                  configMessage = "Configuration saved.";
                } else {
                  configMessage = res.error || "Failed to save configuration.";
                }
              } catch (e) {
                configMessage = e instanceof Error ? e.message : "Invalid JSON";
              } finally {
                configSaving = false;
              }
            }}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
          >
            {configSaving ? "Saving…" : "Save Config"}
          </button>
        {/if}
      </div>
      {#if configMessage}
        <p class="text-xs text-[var(--muted)]">{configMessage}</p>
      {/if}
      {#if configJson}
        <textarea
          bind:value={configDraft}
          readonly={runtime.readonly}
          rows="20"
          class="w-full p-4 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-xs outline-none focus:border-[var(--accent)]"
        ></textarea>
      {:else}
        <p class="text-sm text-[var(--muted)]">Configuration unavailable.</p>
      {/if}
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-5 mt-5">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Backup &amp; Export
        </h3>
        <p class="text-xs text-[var(--muted)]">
          Download a ZIP archive of your ledger files and configuration.
        </p>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            onclick={async () => {
              const res = await fetch("/api/backup/export", { headers: authHeaders() });
              if (!res.ok) return;
              const blob = await res.blob();
              const url = URL.createObjectURL(blob);
              const a = document.createElement("a");
              a.href = url;
              a.download = "paisa-backup.zip";
              a.click();
              URL.revokeObjectURL(url);
            }}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer"
          >
            Export Backup (.zip)
          </button>
        </div>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-2">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          About Paisa
        </h3>
        <p class="text-sm font-medium">Paisa Personal Finance Manager</p>
        <p class="text-xs text-[var(--muted)]">
          Double-entry accounting powered by ledger-cli, Go, SQLite, and SvelteKit.
        </p>
      </div>
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
