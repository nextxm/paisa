<script lang="ts">
  import { onMount } from "svelte";
  import COLORS from "$lib/colors";
  import {
    buildPriorityQueue,
    countIssuesByLevel,
    filterDuplicatePairs,
    filterIssues,
    filterOutliers,
    issueTone,
    type DoctorPriorityItem,
    type DoctorSection
  } from "$lib/doctor_v2";
  import { ajax, formatCurrency } from "$lib/utils";
  import type { DuplicatePair, Issue, OutlierTransaction, Posting } from "$lib/utils";
  import { dataQualityIssueCount } from "../../../../store";

  let loading = $state(true);
  let loadError = $state("");

  let issues: Issue[] = $state([]);
  let duplicates: DuplicatePair[] = $state([]);
  let outliers: OutlierTransaction[] = $state([]);
  let suppressLoading: Record<string, boolean> = $state({});

  let activeSection: DoctorSection = $state("overview");
  let issueSearchText = $state("");
  let issuePageSize = $state(8);
  let issuePage = $state(1);
  let signalSearchText = $state("");
  let minConfidence = $state(0.5);
  let pageSize = $state(6);
  let duplicatePage = $state(1);
  let outlierPage = $state(1);

  const levelLabels: Record<string, string> = {
    danger: "Critical",
    warning: "Needs review",
    info: "Informational",
    success: "Healthy"
  };

  let issueCountsByLevel = $derived(countIssuesByLevel(issues));
  let filteredIssues = $derived(filterIssues(issues, issueSearchText));
  let filteredDuplicates = $derived(
    filterDuplicatePairs(duplicates, signalSearchText, minConfidence)
  );
  let filteredOutliers = $derived(filterOutliers(outliers, signalSearchText, minConfidence));
  let priorityQueue = $derived(buildPriorityQueue(issues, duplicates, outliers).slice(0, 6));

  let issuePageCount = $derived(Math.max(1, Math.ceil(filteredIssues.length / issuePageSize)));
  let duplicatePageCount = $derived(Math.max(1, Math.ceil(filteredDuplicates.length / pageSize)));
  let outlierPageCount = $derived(Math.max(1, Math.ceil(filteredOutliers.length / pageSize)));

  let pagedIssues = $derived.by(() => {
    const start = (issuePage - 1) * issuePageSize;
    return filteredIssues.slice(start, start + issuePageSize);
  });

  let pagedDuplicates = $derived.by(() => {
    const start = (duplicatePage - 1) * pageSize;
    return filteredDuplicates.slice(start, start + pageSize);
  });

  let pagedOutliers = $derived.by(() => {
    const start = (outlierPage - 1) * pageSize;
    return filteredOutliers.slice(start, start + pageSize);
  });

  let totalSignals = $derived(issues.length + duplicates.length + outliers.length);

  onMount(async () => {
    try {
      const [diagnosis, dataQuality] = await Promise.all([
        ajax("/api/diagnosis"),
        ajax("/api/diagnosis/duplicates")
      ]);

      issues = diagnosis.issues || [];
      duplicates = dataQuality.duplicates || [];
      outliers = dataQuality.outliers || [];
    } catch (error) {
      console.error(error);
      loadError = "Doctor V2 could not load its diagnosis data.";
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    dataQualityIssueCount.set(duplicates.length + outliers.length);
  });

  $effect(() => {
    issuePage = clampPage(issuePage, issuePageCount);
    duplicatePage = clampPage(duplicatePage, duplicatePageCount);
    outlierPage = clampPage(outlierPage, outlierPageCount);
  });

  async function suppress(pair: DuplicatePair) {
    const key = pairKey(pair);
    suppressLoading[key] = true;

    try {
      await ajax("/api/diagnosis/duplicates/suppress", {
        method: "POST",
        body: JSON.stringify({
          posting_id_1: Number(pair.posting1.id),
          posting_id_2: Number(pair.posting2.id)
        })
      });

      duplicates = duplicates.filter(
        (candidate) =>
          !(
            candidate.posting1.id === pair.posting1.id && candidate.posting2.id === pair.posting2.id
          )
      );
    } finally {
      suppressLoading[key] = false;
    }
  }

  function jumpTo(section: DoctorSection) {
    activeSection = section;

    if (section === "overview") {
      document
        .getElementById("doctor-v2-overview")
        ?.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }

    document
      .getElementById(`doctor-v2-${section}`)
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function pairKey(pair: DuplicatePair) {
    return `${pair.posting1.id}-${pair.posting2.id}`;
  }

  function confidenceColor(confidence: number) {
    if (confidence >= 0.8) return COLORS.lossText;
    if (confidence >= 0.5) return "#d98e04";
    return COLORS.gainText;
  }

  function pct(confidence: number) {
    return Math.round(confidence * 100);
  }

  function issueClass(level: string) {
    const tone = issueTone(level);
    if (tone === "danger") return "is-danger";
    if (tone === "warning") return "is-warning";
    if (tone === "success") return "is-success";
    return "is-info";
  }

  function sectionCount(section: DoctorSection) {
    if (section === "diagnosis") return issues.length;
    if (section === "duplicates") return duplicates.length;
    if (section === "outliers") return outliers.length;
    return totalSignals;
  }

  function updateIssueSearchText(event: Event) {
    issueSearchText = (event.currentTarget as HTMLInputElement).value;
    issuePage = 1;
  }

  function updateIssuePageSize(event: Event) {
    issuePageSize = Number((event.currentTarget as HTMLSelectElement).value) || 8;
    issuePage = 1;
  }

  function updateSignalSearchText(event: Event) {
    signalSearchText = (event.currentTarget as HTMLInputElement).value;
    duplicatePage = 1;
    outlierPage = 1;
  }

  function updateMinConfidence(event: Event) {
    minConfidence = Number((event.currentTarget as HTMLSelectElement).value) || 0;
    duplicatePage = 1;
    outlierPage = 1;
  }

  function updatePageSize(event: Event) {
    pageSize = Number((event.currentTarget as HTMLSelectElement).value) || 6;
    duplicatePage = 1;
    outlierPage = 1;
  }

  function ledgerHref(posting: Posting) {
    return `/ledger/editor/${encodeURIComponent(posting.file_name)}#${posting.transaction_begin_line}`;
  }

  function showSection(section: DoctorSection) {
    return activeSection === section;
  }

  function sectionLabel(section: DoctorSection) {
    if (section === "overview") return "Overview";
    if (section === "diagnosis") return "Diagnosis";
    if (section === "duplicates") return "Duplicates";
    return "Outliers";
  }

  function suggestedSection(): Exclude<DoctorSection, "overview"> {
    if ((issueCountsByLevel.danger || 0) > 0) return "diagnosis";
    if (duplicates.length > 0) return "duplicates";
    return "outliers";
  }

  function focusSuggestedSection() {
    jumpTo(suggestedSection());
  }

  function clampPage(page: number, pageCount: number) {
    return Math.min(Math.max(page, 1), Math.max(pageCount, 1));
  }
</script>

<section class="section doctor-v2-page">
  <div class="container is-fluid">
    <div class="doctor-v2-hero box">
      <div>
        <p class="doctor-v2-kicker">Doctor V2</p>
        <h1 class="title is-3 mb-2">Triage the journal in clear passes</h1>
        <p class="subtitle is-6 mb-0 doctor-v2-subtitle">
          Start with the priority queue, then focus on diagnosis rules, duplicate candidates, or
          outliers one section at a time.
        </p>
      </div>
      <div class="doctor-v2-hero-actions">
        <a class="button is-light" href="/more/doctor-v3">Open Doctor V3</a>
        <a class="button is-light" href="/more/doctor">Open legacy Doctor</a>
        <button class="button is-dark" onclick={() => jumpTo("overview")}>Back to overview</button>
      </div>
    </div>

    {#if loading}
      <div class="box has-text-centered py-6">
        <progress class="progress is-small is-dark" max="100">Loading</progress>
        <p class="has-text-grey mb-0">Pulling diagnosis and data-quality results…</p>
      </div>
    {:else if loadError}
      <div class="notification is-danger is-light">{loadError}</div>
    {:else}
      <div class="doctor-v2-layout">
        <aside class="doctor-v2-sidebar">
          <div class="box doctor-v2-sidebar-card">
            <p class="heading mb-2">Workspace</p>
            <div class="doctor-v2-nav">
              {#each ["overview", "diagnosis", "duplicates", "outliers"] as section}
                <button
                  class="doctor-v2-nav-button"
                  class:is-active={activeSection === section}
                  onclick={() => jumpTo(section as DoctorSection)}
                >
                  <span>{section === "overview" ? "Overview" : section}</span>
                  <span class="tag is-rounded">{sectionCount(section as DoctorSection)}</span>
                </button>
              {/each}
            </div>
          </div>

          <div class="box doctor-v2-sidebar-card doctor-v2-focus-card">
            <p class="heading mb-2">Current focus</p>
            <p class="doctor-v2-metric-value mb-1">{sectionLabel(activeSection)}</p>
            <p class="doctor-v2-metric-label mb-3">
              {sectionCount(activeSection)} items in section
            </p>
            <button class="button is-small is-dark is-fullwidth" onclick={focusSuggestedSection}>
              Open next suggested section
            </button>
          </div>
        </aside>

        <div class="doctor-v2-main">
          {#if activeSection === "overview"}
            <section id="doctor-v2-overview" class="box doctor-v2-panel">
              <div
                class="is-flex is-justify-content-space-between is-align-items-flex-start is-flex-wrap-wrap gap-3 mb-4"
              >
                <div>
                  <p class="heading mb-2">Overview</p>
                  <h2 class="title is-4 mb-2">What deserves attention first</h2>
                  <p class="doctor-v2-muted mb-0">
                    Focus on one section at a time. Start with the suggested section below.
                  </p>
                </div>
                <div class="doctor-v2-pill-group doctor-v2-pill-group--compact">
                  <button
                    class="doctor-v2-pill doctor-v2-pill-button"
                    onclick={() => jumpTo("diagnosis")}
                  >
                    <strong>{issueCountsByLevel.danger || 0}</strong>&nbsp;critical rules
                  </button>
                  <button
                    class="doctor-v2-pill doctor-v2-pill-button"
                    onclick={() => jumpTo("duplicates")}
                  >
                    <strong>{duplicates.length}</strong>&nbsp;duplicate pairs
                  </button>
                  <button
                    class="doctor-v2-pill doctor-v2-pill-button"
                    onclick={() => jumpTo("outliers")}
                  >
                    <strong>{outliers.length}</strong>&nbsp;outliers
                  </button>
                </div>
              </div>

              <h3 class="title is-5 mb-3">Priority queue</h3>
              {#if priorityQueue.length === 0}
                <div class="notification is-success is-light mb-0">
                  Doctor V2 has nothing urgent to review.
                </div>
              {:else}
                <div class="doctor-v2-queue">
                  {#each priorityQueue as item}
                    <button class="doctor-v2-queue-item" onclick={() => jumpTo(item.section)}>
                      <div class="doctor-v2-queue-main">
                        <span
                          class="tag {item.tone === 'danger'
                            ? 'is-danger'
                            : item.tone === 'warning'
                              ? 'is-warning'
                              : item.tone === 'success'
                                ? 'is-success'
                                : 'is-info'} is-light">{item.section}</span
                        >
                        <p class="doctor-v2-queue-title">{item.title}</p>
                        <p class="doctor-v2-queue-subtitle">{item.subtitle || item.meta}</p>
                      </div>
                    </button>
                  {/each}
                </div>
              {/if}
            </section>
          {/if}

          {#if showSection("diagnosis")}
            <section id="doctor-v2-diagnosis" class="box doctor-v2-panel">
              <div class="doctor-v2-section-header">
                <div>
                  <p class="heading mb-2">Section 1</p>
                  <h2 class="title is-4 mb-2">Diagnosis rules</h2>
                  <p class="doctor-v2-muted mb-0">
                    These are config-driven rule findings from the server-side Doctor checks.
                  </p>
                </div>
                <span class="tag is-rounded is-dark">{filteredIssues.length} shown</span>
              </div>

              <div class="doctor-v2-toolbar mb-4">
                <div>
                  <label class="label is-small mb-1" for="doctor-v2-issue-search"
                    >Search rules</label
                  >
                  <input
                    id="doctor-v2-issue-search"
                    class="input"
                    type="text"
                    placeholder="Summary, description, details"
                    value={issueSearchText}
                    oninput={updateIssueSearchText}
                  />
                </div>
                <div>
                  <label class="label is-small mb-1" for="doctor-v2-issue-page-size"
                    >Page size</label
                  >
                  <div class="select is-fullwidth">
                    <select
                      id="doctor-v2-issue-page-size"
                      value={issuePageSize}
                      onchange={updateIssuePageSize}
                    >
                      <option value="8">8</option>
                      <option value="12">12</option>
                      <option value="24">24</option>
                    </select>
                  </div>
                </div>
              </div>

              {#if issues.length === 0}
                <p class="has-text-grey mb-0">No diagnosis findings.</p>
              {:else if filteredIssues.length === 0}
                <p class="has-text-grey mb-0">No findings match this search.</p>
              {:else}
                <div class="doctor-v2-pager mb-4">
                  <button
                    class="button is-small"
                    disabled={issuePage === 1}
                    onclick={() => (issuePage = Math.max(1, issuePage - 1))}
                  >
                    Previous
                  </button>
                  <span>Page {issuePage} of {issuePageCount}</span>
                  <button
                    class="button is-small"
                    disabled={issuePage === issuePageCount}
                    onclick={() => (issuePage = Math.min(issuePageCount, issuePage + 1))}
                  >
                    Next
                  </button>
                </div>

                <div class="doctor-v2-issue-grid">
                  {#each pagedIssues as issue}
                    <article class="doctor-v2-issue-card {issueClass(issue.level)}">
                      <div class="doctor-v2-issue-card-header">
                        <span class="tag {issueClass(issue.level)} is-light">
                          {levelLabels[issueTone(issue.level)] || issue.level}
                        </span>
                        <p class="doctor-v2-issue-title">{issue.summary}</p>
                      </div>
                      <div class="doctor-v2-issue-body">
                        {@html `${issue.description}<br/><br/>${issue.details}`}
                      </div>
                    </article>
                  {/each}
                </div>
              {/if}
            </section>
          {/if}

          {#if showSection("duplicates")}
            <section id="doctor-v2-duplicates" class="box doctor-v2-panel">
              <div class="doctor-v2-section-header">
                <div>
                  <p class="heading mb-2">Section 2</p>
                  <h2 class="title is-4 mb-2">Duplicate candidates</h2>
                  <p class="doctor-v2-muted mb-0">
                    Work this queue when you want to reduce false repeats in the journal.
                  </p>
                </div>
                <span class="tag is-rounded is-warning">{filteredDuplicates.length} shown</span>
              </div>

              <div class="doctor-v2-toolbar mb-4">
                <div>
                  <label class="label is-small mb-1" for="doctor-v2-signal-search"
                    >Search signals</label
                  >
                  <input
                    id="doctor-v2-signal-search"
                    class="input"
                    type="text"
                    placeholder="Payee, account, reason, date, amount"
                    value={signalSearchText}
                    oninput={updateSignalSearchText}
                  />
                </div>
                <div>
                  <label class="label is-small mb-1" for="doctor-v2-confidence"
                    >Minimum confidence</label
                  >
                  <div class="select is-fullwidth">
                    <select
                      id="doctor-v2-confidence"
                      value={minConfidence}
                      onchange={updateMinConfidence}
                    >
                      <option value="0">All results</option>
                      <option value="0.5">50%+</option>
                      <option value="0.8">80%+</option>
                    </select>
                  </div>
                </div>
                <div>
                  <label class="label is-small mb-1" for="doctor-v2-page-size">Page size</label>
                  <div class="select is-fullwidth">
                    <select id="doctor-v2-page-size" value={pageSize} onchange={updatePageSize}>
                      <option value="6">6</option>
                      <option value="12">12</option>
                      <option value="24">24</option>
                    </select>
                  </div>
                </div>
              </div>

              {#if duplicates.length === 0}
                <p class="has-text-grey mb-0">No duplicate transactions detected.</p>
              {:else if filteredDuplicates.length === 0}
                <p class="has-text-grey mb-0">No duplicate candidates match these filters.</p>
              {:else}
                <div class="doctor-v2-pager mb-4">
                  <button
                    class="button is-small"
                    disabled={duplicatePage === 1}
                    onclick={() => (duplicatePage = Math.max(1, duplicatePage - 1))}
                  >
                    Previous
                  </button>
                  <span>Page {duplicatePage} of {duplicatePageCount}</span>
                  <button
                    class="button is-small"
                    disabled={duplicatePage === duplicatePageCount}
                    onclick={() =>
                      (duplicatePage = Math.min(duplicatePageCount, duplicatePage + 1))}
                  >
                    Next
                  </button>
                </div>

                <div class="doctor-v2-signal-stack">
                  {#each pagedDuplicates as pair}
                    {@const key = pairKey(pair)}
                    <article class="doctor-v2-signal-card">
                      <div class="doctor-v2-signal-header">
                        <div>
                          <span class="tag is-warning is-light">Duplicate</span>
                          <p class="doctor-v2-signal-title">
                            {pair.posting1.payee || pair.posting2.payee}
                          </p>
                          <p class="doctor-v2-signal-meta">{@html pair.reason}</p>
                        </div>
                        <div class="doctor-v2-signal-actions">
                          <span
                            class="tag is-rounded"
                            style={`background-color: ${confidenceColor(pair.confidence)}; color: white`}
                          >
                            {pct(pair.confidence)}% confidence
                          </span>
                          <button
                            class="button is-small is-light"
                            class:is-loading={suppressLoading[key]}
                            onclick={() => suppress(pair)}
                          >
                            Dismiss pair
                          </button>
                        </div>
                      </div>

                      <div class="doctor-v2-posting-grid">
                        <a class="doctor-v2-posting-card" href={ledgerHref(pair.posting1)}>
                          <p class="doctor-v2-posting-label">Posting 1</p>
                          <p class="doctor-v2-posting-title">{pair.posting1.payee}</p>
                          <p class="doctor-v2-posting-meta">
                            {pair.posting1.date} · {pair.posting1.account}
                          </p>
                          <p class="doctor-v2-posting-amount">
                            {formatCurrency(pair.posting1.amount)}
                          </p>
                        </a>
                        <a class="doctor-v2-posting-card" href={ledgerHref(pair.posting2)}>
                          <p class="doctor-v2-posting-label">Posting 2</p>
                          <p class="doctor-v2-posting-title">{pair.posting2.payee}</p>
                          <p class="doctor-v2-posting-meta">
                            {pair.posting2.date} · {pair.posting2.account}
                          </p>
                          <p class="doctor-v2-posting-amount">
                            {formatCurrency(pair.posting2.amount)}
                          </p>
                        </a>
                      </div>
                    </article>
                  {/each}
                </div>
              {/if}
            </section>
          {/if}

          {#if showSection("outliers")}
            <section id="doctor-v2-outliers" class="box doctor-v2-panel">
              <div class="doctor-v2-section-header">
                <div>
                  <p class="heading mb-2">Section 3</p>
                  <h2 class="title is-4 mb-2">Outlier transactions</h2>
                  <p class="doctor-v2-muted mb-0">
                    Use this section to spot unusually large or unusual entries relative to the
                    account's normal behavior.
                  </p>
                </div>
                <span class="tag is-rounded is-danger">{filteredOutliers.length} shown</span>
              </div>

              {#if outliers.length === 0}
                <p class="has-text-grey mb-0">No statistical outliers detected.</p>
              {:else if filteredOutliers.length === 0}
                <p class="has-text-grey mb-0">No outliers match these filters.</p>
              {:else}
                <div class="doctor-v2-pager mb-4">
                  <button
                    class="button is-small"
                    disabled={outlierPage === 1}
                    onclick={() => (outlierPage = Math.max(1, outlierPage - 1))}
                  >
                    Previous
                  </button>
                  <span>Page {outlierPage} of {outlierPageCount}</span>
                  <button
                    class="button is-small"
                    disabled={outlierPage === outlierPageCount}
                    onclick={() => (outlierPage = Math.min(outlierPageCount, outlierPage + 1))}
                  >
                    Next
                  </button>
                </div>

                <div class="doctor-v2-outlier-grid">
                  {#each pagedOutliers as outlier}
                    <a class="doctor-v2-outlier-card" href={ledgerHref(outlier.posting)}>
                      <div class="doctor-v2-signal-header">
                        <div>
                          <span class="tag is-danger is-light">Outlier</span>
                          <p class="doctor-v2-signal-title">{outlier.posting.payee}</p>
                          <p class="doctor-v2-signal-meta">
                            {outlier.posting.date} · {outlier.posting.account}
                          </p>
                        </div>
                        <span
                          class="tag is-rounded"
                          style={`background-color: ${confidenceColor(outlier.confidence)}; color: white`}
                        >
                          {pct(outlier.confidence)}% confidence
                        </span>
                      </div>

                      <p class="doctor-v2-outlier-amount">
                        {formatCurrency(outlier.posting.amount)}
                      </p>
                      <div class="doctor-v2-stat-grid">
                        <div>
                          <p class="doctor-v2-stat-label">Sigma</p>
                          <p class="doctor-v2-stat-value">{outlier.sigma.toFixed(1)}σ</p>
                        </div>
                        <div>
                          <p class="doctor-v2-stat-label">Mean</p>
                          <p class="doctor-v2-stat-value">{formatCurrency(outlier.mean)}</p>
                        </div>
                        <div>
                          <p class="doctor-v2-stat-label">Std dev</p>
                          <p class="doctor-v2-stat-value">{formatCurrency(outlier.std_dev)}</p>
                        </div>
                      </div>
                    </a>
                  {/each}
                </div>
              {/if}
            </section>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</section>

<style>
  /* ── Light-mode tokens ─────────────────────────────────────────────────── */
  .doctor-v2-page {
    --dv2-panel-bg: rgba(255, 255, 255, 0.98);
    --dv2-border: rgba(17, 24, 39, 0.08);
    --dv2-shadow: 0 20px 45px rgba(15, 23, 42, 0.08);
    --dv2-kicker-color: #8c5f0a;
    --dv2-text-muted: #5b6472;
    --dv2-text-strong: #172033;
    --dv2-text-body: #273244;
    --dv2-nav-bg: rgba(255, 255, 255, 0.7);
    --dv2-active-border: rgba(32, 99, 155, 0.35);
    --dv2-active-bg: linear-gradient(135deg, rgba(206, 231, 255, 0.95), rgba(255, 255, 255, 0.92));
    --dv2-pill-bg: rgba(17, 24, 39, 0.05);
    --dv2-card-bg: rgba(255, 255, 255, 0.88);
    --dv2-critical-card-bg: linear-gradient(180deg, rgba(255, 243, 244, 0.96), white);
    --dv2-warning-card-bg: linear-gradient(180deg, rgba(255, 248, 234, 0.96), white);
    --dv2-info-card-bg: linear-gradient(180deg, rgba(239, 248, 255, 0.96), white);
    --dv2-success-card-bg: linear-gradient(180deg, rgba(240, 251, 244, 0.96), white);
  }

  /* ── Dark-mode token overrides ─────────────────────────────────────────── */
  :global(html[data-theme="dark"]) .doctor-v2-page {
    --dv2-panel-bg: hsl(215, 18%, 16%);
    --dv2-border: rgba(255, 255, 255, 0.08);
    --dv2-shadow: 0 20px 45px rgba(0, 0, 0, 0.4);
    --dv2-kicker-color: #e0a83a;
    --dv2-text-muted: hsl(215, 10%, 55%);
    --dv2-text-strong: hsl(0, 0%, 85%);
    --dv2-text-body: hsl(0, 0%, 72%);
    --dv2-nav-bg: hsl(215, 18%, 19%);
    --dv2-active-border: rgba(100, 160, 220, 0.4);
    --dv2-active-bg: linear-gradient(135deg, hsl(215, 35%, 23%), hsl(215, 18%, 20%));
    --dv2-pill-bg: rgba(255, 255, 255, 0.07);
    --dv2-card-bg: hsl(215, 18%, 19%);
    --dv2-critical-card-bg: hsl(355, 18%, 18%);
    --dv2-warning-card-bg: hsl(40, 15%, 18%);
    --dv2-info-card-bg: hsl(210, 22%, 18%);
    --dv2-success-card-bg: hsl(150, 15%, 18%);
  }

  .doctor-v2-hero,
  .doctor-v2-panel,
  .doctor-v2-sidebar-card {
    border: 1px solid var(--dv2-border);
    box-shadow: var(--dv2-shadow);
    background: var(--dv2-panel-bg);
  }

  .doctor-v2-hero {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    gap: 1.5rem;
    margin-bottom: 1.25rem;
    padding: 1.5rem;
  }

  .doctor-v2-kicker {
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--dv2-kicker-color);
    margin-bottom: 0.5rem;
  }

  .doctor-v2-subtitle,
  .doctor-v2-muted,
  .doctor-v2-queue-subtitle,
  .doctor-v2-queue-meta,
  .doctor-v2-signal-meta,
  .doctor-v2-posting-meta,
  .doctor-v2-guide p,
  .doctor-v2-level-row,
  .doctor-v2-pill,
  .doctor-v2-summary-card p:last-child {
    color: var(--dv2-text-muted);
  }

  .doctor-v2-hero-actions {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .doctor-v2-layout {
    display: grid;
    grid-template-columns: 280px minmax(0, 1fr);
    gap: 1.25rem;
    align-items: start;
  }

  .doctor-v2-sidebar {
    position: sticky;
    top: 0.75rem;
    display: grid;
    gap: 1rem;
  }

  .doctor-v2-sidebar-card {
    padding: 1rem;
  }

  .doctor-v2-nav {
    display: grid;
    gap: 0.6rem;
  }

  .doctor-v2-nav-button {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    border: 1px solid var(--dv2-border);
    border-radius: 0.9rem;
    padding: 0.8rem 0.9rem;
    background: var(--dv2-nav-bg);
    font-weight: 600;
    text-transform: capitalize;
  }

  .doctor-v2-nav-button.is-active {
    border-color: var(--dv2-active-border);
    background: var(--dv2-active-bg);
  }

  .doctor-v2-main {
    display: grid;
    gap: 1.25rem;
  }

  .doctor-v2-panel {
    padding: 1.35rem;
  }

  .doctor-v2-pill-group {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .doctor-v2-pill-group--compact {
    justify-content: flex-end;
  }

  .doctor-v2-pill {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 0.8rem;
    border-radius: 999px;
    background: var(--dv2-pill-bg);
  }

  .doctor-v2-pill-button {
    border: 1px solid var(--dv2-border);
    color: inherit;
    font: inherit;
    cursor: pointer;
  }

  .doctor-v2-pill-button:hover {
    border-color: var(--dv2-active-border);
    background: var(--dv2-active-bg);
  }

  .doctor-v2-summary-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.9rem;
  }

  .doctor-v2-summary-card {
    border-radius: 1rem;
    padding: 1rem;
    min-height: 164px;
    border: 1px solid transparent;
    background: var(--dv2-card-bg);
  }

  .doctor-v2-summary-card.is-critical {
    border-color: rgba(220, 53, 69, 0.18);
    background: var(--dv2-critical-card-bg);
  }

  .doctor-v2-summary-card.is-warning-tone {
    border-color: rgba(217, 142, 4, 0.2);
    background: var(--dv2-warning-card-bg);
  }

  .doctor-v2-summary-card.is-alert-tone {
    border-color: rgba(32, 99, 155, 0.18);
    background: var(--dv2-info-card-bg);
  }

  .doctor-v2-summary-card.is-calm-tone {
    border-color: rgba(33, 150, 83, 0.18);
    background: var(--dv2-success-card-bg);
  }

  .doctor-v2-queue,
  .doctor-v2-signal-stack,
  .doctor-v2-level-list {
    display: grid;
    gap: 0.85rem;
  }

  .doctor-v2-queue-item,
  .doctor-v2-outlier-card,
  .doctor-v2-posting-card {
    width: 100%;
    text-align: left;
    border: 1px solid var(--dv2-border);
    border-radius: 1rem;
    padding: 1rem;
    background: var(--dv2-card-bg);
    transition:
      transform 0.16s ease,
      box-shadow 0.16s ease,
      border-color 0.16s ease;
  }

  .doctor-v2-queue-item:hover,
  .doctor-v2-outlier-card:hover,
  .doctor-v2-posting-card:hover {
    transform: translateY(-1px);
    border-color: rgba(32, 99, 155, 0.22);
    box-shadow: 0 10px 25px rgba(15, 23, 42, 0.08);
  }

  .doctor-v2-queue-title,
  .doctor-v2-signal-title,
  .doctor-v2-posting-title,
  .doctor-v2-issue-title {
    font-size: 1rem;
    font-weight: 700;
    color: var(--dv2-text-strong);
    margin: 0.45rem 0 0.35rem;
  }

  .doctor-v2-queue-meta {
    margin-top: 0.65rem;
    font-size: 0.85rem;
  }

  .doctor-v2-level-row,
  .doctor-v2-pager,
  .doctor-v2-signal-header,
  .doctor-v2-section-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-start;
  }

  .doctor-v2-toolbar {
    display: grid;
    grid-template-columns: minmax(0, 1.5fr) minmax(180px, 0.75fr) minmax(140px, 0.6fr);
    gap: 0.9rem;
  }

  .doctor-v2-issue-grid,
  .doctor-v2-outlier-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .doctor-v2-issue-card,
  .doctor-v2-signal-card {
    border-radius: 1rem;
    border: 1px solid var(--dv2-border);
    background: var(--dv2-card-bg);
    overflow: hidden;
  }

  .doctor-v2-issue-card.is-danger {
    border-color: rgba(220, 53, 69, 0.22);
  }

  .doctor-v2-issue-card.is-warning {
    border-color: rgba(217, 142, 4, 0.22);
  }

  .doctor-v2-issue-card.is-info {
    border-color: rgba(32, 99, 155, 0.18);
  }

  .doctor-v2-issue-card.is-success {
    border-color: rgba(33, 150, 83, 0.18);
  }

  .doctor-v2-issue-card-header,
  .doctor-v2-issue-body,
  .doctor-v2-signal-card {
    padding: 1rem;
  }

  .doctor-v2-issue-body {
    overflow-wrap: anywhere;
    color: var(--dv2-text-body);
    padding-top: 0;
  }

  .doctor-v2-signal-card {
    display: grid;
    gap: 1rem;
  }

  .doctor-v2-signal-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .doctor-v2-posting-grid,
  .doctor-v2-stat-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.85rem;
  }

  .doctor-v2-posting-label,
  .doctor-v2-stat-label,
  .doctor-v2-metric-label {
    font-size: 0.74rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--dv2-text-muted);
    margin-bottom: 0.4rem;
  }

  .doctor-v2-posting-amount,
  .doctor-v2-outlier-amount,
  .doctor-v2-stat-value,
  .doctor-v2-metric-value {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--dv2-text-strong);
  }

  .doctor-v2-outlier-card {
    text-decoration: none;
    color: inherit;
  }

  .doctor-v2-outlier-amount {
    margin: 1rem 0;
    font-size: 1.35rem;
  }

  .doctor-v2-metric-stack {
    display: grid;
    gap: 1rem;
  }

  .doctor-v2-focus-card {
    display: grid;
    gap: 0.45rem;
  }

  .doctor-v2-guide {
    display: grid;
    gap: 0.75rem;
  }

  .doctor-v2-pager {
    align-items: center;
  }

  @media (max-width: 1100px) {
    .doctor-v2-layout {
      grid-template-columns: 1fr;
    }

    .doctor-v2-sidebar {
      position: static;
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .doctor-v2-summary-grid,
    .doctor-v2-issue-grid,
    .doctor-v2-outlier-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 768px) {
    .doctor-v2-hero,
    .doctor-v2-section-header,
    .doctor-v2-signal-header,
    .doctor-v2-level-row,
    .doctor-v2-pager {
      flex-direction: column;
    }

    .doctor-v2-sidebar {
      grid-template-columns: 1fr;
    }

    .doctor-v2-summary-grid,
    .doctor-v2-issue-grid,
    .doctor-v2-outlier-grid,
    .doctor-v2-posting-grid,
    .doctor-v2-stat-grid,
    .doctor-v2-toolbar {
      grid-template-columns: 1fr;
    }

    .doctor-v2-hero-actions,
    .doctor-v2-signal-actions {
      width: 100%;
      justify-content: flex-start;
    }

    .doctor-v2-panel,
    .doctor-v2-hero {
      padding: 1rem;
    }
  }
</style>
