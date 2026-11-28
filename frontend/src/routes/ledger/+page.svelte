<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import { ArrowUpRight, CheckCircle2, FileCode2, Search } from "lucide-svelte";
  import LedgerEditorPanel from "$lib/components/LedgerEditorPanel.svelte";
  import LedgerToolsPanels from "$lib/components/LedgerToolsPanels.svelte";
  import { apiGet, authHeaders } from "$lib/api";
  import {
    formatEvidenceLabel,
    getEvidenceEditorLocation,
    getPrimaryEditorLocation,
    matchesFindingSearch,
    type DoctorFinding
  } from "$lib/doctor_helpers";

  type LedgerTab = "health" | "editor" | "import" | "sheets";
  type HealthFilter = "fix" | "review" | "dismissed" | "all";

  interface DoctorSummary {
    total: number;
    fix_count: number;
    review_count: number;
    info_count: number;
    dismissed_count: number;
  }

  let activeTab = $state<LedgerTab>("import");
  let editorTargetFile = $state("");
  let editorTargetLine = $state(0);
  let editorJumpSeq = $state(0);

  $effect(() => {
    const tab = $page.url.searchParams.get("tab");
    if (tab === "health" || tab === "editor" || tab === "import" || tab === "sheets") {
      activeTab = tab;
    }
    const file = $page.url.searchParams.get("file");
    const lineParam = $page.url.searchParams.get("line");
    if (file) {
      editorTargetFile = file;
      const parsed = lineParam ? Number.parseInt(lineParam, 10) : 0;
      editorTargetLine = Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
      editorJumpSeq += 1;
    }
  });

  function openInEditor(fileName: string, lineNumber = 0) {
    editorTargetFile = fileName;
    editorTargetLine = lineNumber;
    editorJumpSeq += 1;
    activeTab = "editor";
  }
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
  let duplicates = $state<Array<Record<string, unknown>>>([]);
  let outliers = $state<Array<Record<string, unknown>>>([]);

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
      const [, dupRes] = await Promise.all([
        loadHealth(),
        apiGet<{
          duplicates?: Array<Record<string, unknown>>;
          outliers?: Array<Record<string, unknown>>;
        }>("/api/diagnosis/duplicates").catch(() => ({ duplicates: [], outliers: [] }))
      ]);
      duplicates = dupRes.duplicates || [];
      outliers = dupRes.outliers || [];
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load ledger tools";
    } finally {
      loading = false;
    }
  });

  async function dismissFinding(finding: DoctorFinding) {
    await fetch("/api/doctor/dismiss", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ fingerprint: finding.id, rule_id: finding.rule_id })
    });
    await loadHealth();
  }

  async function undismissFinding(finding: DoctorFinding) {
    await fetch("/api/doctor/undismiss", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ fingerprint: finding.id })
    });
    await loadHealth();
  }

  const filteredFindings = $derived.by(() => {
    return findings.filter((f) => {
      if (healthFilter === "fix" && (f.dismissed || f.severity !== "fix")) return false;
      if (healthFilter === "review" && (f.dismissed || f.severity !== "review")) return false;
      if (healthFilter === "dismissed" && !f.dismissed) return false;
      return matchesFindingSearch(f, search);
    });
  });
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Ledger</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Data health diagnostics, statement import, 3-pane journal editor, and sheets
      </p>
    </div>

    {#if activeTab === "health"}
      <div class="relative w-full sm:w-72">
        <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="search"
          bind:value={search}
          placeholder="Filter findings…"
          class="w-full pl-9 pr-3 py-1.5 rounded-lg text-sm bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
        />
      </div>
    {/if}
  </div>

  <!-- Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [{ id: "import", label: "Import" }, { id: "editor", label: "Journal Editor" }, { id: "sheets", label: "Sheets" }, { id: "health", label: `Data Health (${summary.fix_count + summary.review_count})` }] as tab}
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
    <div class="flex flex-wrap items-center gap-2">
      {#each [{ id: "fix", label: `Fix (${summary.fix_count})` }, { id: "review", label: `Review (${summary.review_count})` }, { id: "dismissed", label: `Dismissed (${summary.dismissed_count})` }, { id: "all", label: `All (${summary.total})` }] as f}
        <button
          type="button"
          class="px-3 py-1 rounded-full text-xs border cursor-pointer {healthFilter === f.id
            ? 'bg-[var(--accent)] text-[var(--accent-fg)] border-[var(--accent)] font-medium'
            : 'bg-[var(--surface)] border-[var(--border)] text-[var(--muted)] hover:text-[var(--text)]'}"
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
        <p class="text-sm font-medium">No findings in this view</p>
        <p class="text-xs text-[var(--muted)]">Your ledger looks healthy here.</p>
      </div>
    {:else}
      <div class="space-y-3">
        {#each filteredFindings as f (f.id)}
          {@const primaryLoc = getPrimaryEditorLocation(f)}
          {@const messageText = f.details || f.description}
          <div
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex items-start justify-between gap-4"
          >
            <div class="space-y-1.5 min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  class="px-2 py-0.5 rounded-full text-[11px] font-semibold uppercase {f.severity ===
                  'fix'
                    ? 'bg-[var(--loss)]/15 text-[var(--loss)]'
                    : f.severity === 'review'
                      ? 'bg-[var(--warning)]/15 text-[var(--warning)]'
                      : 'bg-[var(--surface-elevated)] text-[var(--muted)]'}"
                >
                  {f.severity}
                </span>
                <span class="text-sm font-semibold">{f.title}</span>
              </div>

              {#if messageText}
                {#if primaryLoc}
                  <button
                    type="button"
                    onclick={() => openInEditor(primaryLoc.fileName, primaryLoc.lineNumber)}
                    class="text-left text-xs text-[var(--accent)] hover:underline inline-flex items-center gap-1 cursor-pointer"
                  >
                    <span>{messageText}</span>
                    <ArrowUpRight class="w-3.5 h-3.5 shrink-0" />
                  </button>
                {:else}
                  <p class="text-xs text-[var(--text)]/90">{messageText}</p>
                {/if}
              {/if}

              {#if f.details && f.description && f.details !== f.description}
                <p class="text-xs text-[var(--muted)]">{f.description}</p>
              {/if}

              {#if f.evidence && f.evidence.length > 0}
                <div class="flex flex-wrap gap-1.5 pt-0.5">
                  {#each f.evidence as ev}
                    {@const evLoc = getEvidenceEditorLocation(ev)}
                    {@const label = formatEvidenceLabel(ev)}
                    {#if label}
                      {#if evLoc}
                        <button
                          type="button"
                          onclick={() => openInEditor(evLoc.fileName, evLoc.lineNumber)}
                          title="Open in Journal Editor"
                          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono bg-[var(--surface-elevated)] border border-[var(--border)] hover:border-[var(--accent)] hover:text-[var(--accent)] transition-colors cursor-pointer"
                        >
                          <FileCode2 class="w-3.5 h-3.5 shrink-0" />
                          <span>{label}</span>
                        </button>
                      {:else}
                        <span
                          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono bg-[var(--surface-elevated)] border border-[var(--border)] text-[var(--muted)]"
                        >
                          {label}
                        </span>
                      {/if}
                    {/if}
                  {/each}
                </div>
              {:else if f.account}
                <div class="text-xs text-[var(--muted)] font-mono">{f.account}</div>
              {/if}
            </div>

            <div class="flex items-center gap-2 shrink-0">
              {#if primaryLoc}
                <button
                  type="button"
                  onclick={() => openInEditor(primaryLoc.fileName, primaryLoc.lineNumber)}
                  class="px-2.5 py-1 rounded-md text-xs font-medium bg-[var(--accent)]/15 text-[var(--accent)] hover:bg-[var(--accent)]/25 cursor-pointer"
                >
                  Open in Editor
                </button>
              {/if}
              <button
                type="button"
                class="px-2.5 py-1 rounded-md text-xs border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
                onclick={() => (f.dismissed ? undismissFinding(f) : dismissFinding(f))}
              >
                {f.dismissed ? "Restore" : "Dismiss"}
              </button>
            </div>
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
          <pre
            class="text-xs font-mono bg-[var(--surface-elevated)] p-2.5 rounded-lg overflow-x-auto max-h-40">{JSON.stringify(
              duplicates.slice(0, 10),
              null,
              2
            )}</pre>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Outlier Transactions ({outliers.length})
          </div>
          <pre
            class="text-xs font-mono bg-[var(--surface-elevated)] p-2.5 rounded-lg overflow-x-auto max-h-40">{JSON.stringify(
              outliers.slice(0, 10),
              null,
              2
            )}</pre>
        </div>
      </div>
    {/if}
  {:else if activeTab === "editor"}
    <LedgerEditorPanel
      targetFile={editorTargetFile}
      targetLine={editorTargetLine}
      jumpSeq={editorJumpSeq}
    />
  {:else}
    <LedgerToolsPanels mode={activeTab} />
  {/if}
</div>
