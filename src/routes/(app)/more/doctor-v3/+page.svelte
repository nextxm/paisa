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
  let signalSearchText = $state("");
  let minConfidence = $state(0.5);
  let issuePage = $state(1);
  let duplicatePage = $state(1);
  let outlierPage = $state(1);

  const pageSize = 6;

  let issueCountsByLevel = $derived(countIssuesByLevel(issues));
  let filteredIssues = $derived(filterIssues(issues, issueSearchText));
  let filteredDuplicates = $derived(
    filterDuplicatePairs(duplicates, signalSearchText, minConfidence)
  );
  let filteredOutliers = $derived(filterOutliers(outliers, signalSearchText, minConfidence));
  let priorityQueue = $derived(buildPriorityQueue(issues, duplicates, outliers).slice(0, 8));

  let issuePageCount = $derived(Math.max(1, Math.ceil(filteredIssues.length / pageSize)));
  let duplicatePageCount = $derived(Math.max(1, Math.ceil(filteredDuplicates.length / pageSize)));
  let outlierPageCount = $derived(Math.max(1, Math.ceil(filteredOutliers.length / pageSize)));

  let pagedIssues = $derived.by(() => {
    const start = (issuePage - 1) * pageSize;
    return filteredIssues.slice(start, start + pageSize);
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
      loadError = "Doctor V3 could not load diagnosis data.";
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

  function clampPage(page: number, pageCount: number) {
    return Math.min(Math.max(page, 1), Math.max(pageCount, 1));
  }

  function sectionCount(section: DoctorSection) {
    if (section === "diagnosis") return filteredIssues.length;
    if (section === "duplicates") return filteredDuplicates.length;
    if (section === "outliers") return filteredOutliers.length;
    return totalSignals;
  }

  function setSection(section: DoctorSection) {
    activeSection = section;
  }

  function nextSuggestedSection(): Exclude<DoctorSection, "overview"> {
    if ((issueCountsByLevel.danger || 0) > 0) return "diagnosis";
    if (filteredDuplicates.length > 0) return "duplicates";
    return "outliers";
  }

  function startTriage() {
    setSection(nextSuggestedSection());
  }

  function pct(confidence: number) {
    return Math.round(confidence * 100);
  }

  function confidenceColor(confidence: number) {
    if (confidence >= 0.8) return COLORS.lossText;
    if (confidence >= 0.5) return "#d98e04";
    return COLORS.gainText;
  }

  function issueClass(level: string) {
    const tone = issueTone(level);
    if (tone === "danger") return "is-danger";
    if (tone === "warning") return "is-warning";
    if (tone === "success") return "is-success";
    return "is-info";
  }

  function pairKey(pair: DuplicatePair) {
    return `${pair.posting1.id}-${pair.posting2.id}`;
  }

  function ledgerHref(posting: Posting) {
    return `/ledger/editor/${encodeURIComponent(posting.file_name)}#${posting.transaction_begin_line}`;
  }

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

  function updateIssueSearchText(event: Event) {
    issueSearchText = (event.currentTarget as HTMLInputElement).value;
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
</script>

<section class="section doctor-v3-page">
  <div class="container is-fluid">
    <header class="box doctor-v3-hero">
      <div>
        <p class="doctor-v3-kicker">Doctor V3</p>
        <h1 class="title is-3 mb-2">One section at a time</h1>
        <p class="subtitle is-6 doctor-v3-subtitle mb-0">
          Pick a lane, clear it, then move to the next. No split attention.
        </p>
      </div>
      <div class="doctor-v3-hero-actions">
        <a class="button is-light" href="/more/doctor-v2">Open Doctor V2</a>
        <a class="button is-light" href="/more/doctor">Open legacy Doctor</a>
      </div>
    </header>

    {#if loading}
      <div class="box has-text-centered py-6">
        <progress class="progress is-small is-dark" max="100">Loading</progress>
        <p class="has-text-grey mb-0">Loading diagnosis and data quality results...</p>
      </div>
    {:else if loadError}
      <div class="notification is-danger is-light">{loadError}</div>
    {:else}
      <div class="doctor-v3-nav box mb-4">
        <button class="button is-dark" onclick={startTriage}>Start with suggested section</button>
        <button
          class="button"
          class:is-link={activeSection === "overview"}
          onclick={() => setSection("overview")}>Overview ({sectionCount("overview")})</button
        >
        <button
          class="button"
          class:is-link={activeSection === "diagnosis"}
          onclick={() => setSection("diagnosis")}>Diagnosis ({sectionCount("diagnosis")})</button
        >
        <button
          class="button"
          class:is-link={activeSection === "duplicates"}
          onclick={() => setSection("duplicates")}>Duplicates ({sectionCount("duplicates")})</button
        >
        <button
          class="button"
          class:is-link={activeSection === "outliers"}
          onclick={() => setSection("outliers")}>Outliers ({sectionCount("outliers")})</button
        >
      </div>

      {#if activeSection === "overview"}
        <section class="box doctor-v3-panel">
          <h2 class="title is-4 mb-3">Priority queue</h2>
          {#if priorityQueue.length === 0}
            <div class="notification is-success is-light mb-0">No urgent items found.</div>
          {:else}
            <div class="doctor-v3-list">
              {#each priorityQueue as item}
                <button class="doctor-v3-queue-item" onclick={() => setSection(item.section)}>
                  <span
                    class="tag {item.tone === 'danger'
                      ? 'is-danger'
                      : item.tone === 'warning'
                        ? 'is-warning'
                        : item.tone === 'success'
                          ? 'is-success'
                          : 'is-info'} is-light">{item.section}</span
                  >
                  <strong>{item.title}</strong>
                  <span>{item.meta}</span>
                </button>
              {/each}
            </div>
          {/if}
        </section>
      {/if}

      {#if activeSection === "diagnosis"}
        <section class="box doctor-v3-panel">
          <div class="doctor-v3-headline">
            <h2 class="title is-4 mb-0">Diagnosis</h2>
            <span class="tag is-dark">{filteredIssues.length} items</span>
          </div>
          <div class="doctor-v3-toolbar mb-4">
            <input
              class="input"
              type="text"
              placeholder="Search diagnosis findings"
              value={issueSearchText}
              oninput={updateIssueSearchText}
            />
          </div>

          {#if filteredIssues.length === 0}
            <p class="has-text-grey mb-0">No diagnosis findings for this filter.</p>
          {:else}
            <div class="doctor-v3-pager mb-3">
              <button
                class="button is-small"
                disabled={issuePage === 1}
                onclick={() => (issuePage = Math.max(1, issuePage - 1))}>Previous</button
              >
              <span>Page {issuePage} of {issuePageCount}</span>
              <button
                class="button is-small"
                disabled={issuePage === issuePageCount}
                onclick={() => (issuePage = Math.min(issuePageCount, issuePage + 1))}>Next</button
              >
            </div>
            <div class="doctor-v3-list">
              {#each pagedIssues as issue}
                <article class="doctor-v3-card {issueClass(issue.level)}">
                  <div class="doctor-v3-card-title">
                    <span class="tag {issueClass(issue.level)} is-light"
                      >{issueTone(issue.level)}</span
                    >
                    <strong>{issue.summary}</strong>
                  </div>
                  <div class="doctor-v3-card-body">
                    {@html `${issue.description}<br/><br/>${issue.details}`}
                  </div>
                </article>
              {/each}
            </div>
          {/if}
        </section>
      {/if}

      {#if activeSection === "duplicates"}
        <section class="box doctor-v3-panel">
          <div class="doctor-v3-headline">
            <h2 class="title is-4 mb-0">Duplicates</h2>
            <span class="tag is-warning">{filteredDuplicates.length} items</span>
          </div>
          <div class="doctor-v3-toolbar doctor-v3-toolbar-grid mb-4">
            <input
              class="input"
              type="text"
              placeholder="Search payee/account/reason"
              value={signalSearchText}
              oninput={updateSignalSearchText}
            />
            <div class="select is-fullwidth">
              <select value={minConfidence} onchange={updateMinConfidence}>
                <option value="0">All confidence</option>
                <option value="0.5">50%+</option>
                <option value="0.8">80%+</option>
              </select>
            </div>
          </div>

          {#if filteredDuplicates.length === 0}
            <p class="has-text-grey mb-0">No duplicate candidates for this filter.</p>
          {:else}
            <div class="doctor-v3-pager mb-3">
              <button
                class="button is-small"
                disabled={duplicatePage === 1}
                onclick={() => (duplicatePage = Math.max(1, duplicatePage - 1))}>Previous</button
              >
              <span>Page {duplicatePage} of {duplicatePageCount}</span>
              <button
                class="button is-small"
                disabled={duplicatePage === duplicatePageCount}
                onclick={() => (duplicatePage = Math.min(duplicatePageCount, duplicatePage + 1))}
                >Next</button
              >
            </div>

            <div class="doctor-v3-list">
              {#each pagedDuplicates as pair}
                {@const key = pairKey(pair)}
                <article class="doctor-v3-card">
                  <div class="doctor-v3-card-title">
                    <span class="tag is-warning is-light">Duplicate</span>
                    <strong>{pair.posting1.payee || pair.posting2.payee}</strong>
                    <span
                      class="tag is-rounded"
                      style={`background-color: ${confidenceColor(pair.confidence)}; color: white`}
                      >{pct(pair.confidence)}%</span
                    >
                  </div>

                  <p class="doctor-v3-card-meta">{@html pair.reason}</p>

                  <div class="doctor-v3-dual">
                    <a class="doctor-v3-mini" href={ledgerHref(pair.posting1)}>
                      <p>{pair.posting1.date}</p>
                      <strong>{pair.posting1.payee}</strong>
                      <span>{pair.posting1.account}</span>
                      <strong>{formatCurrency(pair.posting1.amount)}</strong>
                    </a>
                    <a class="doctor-v3-mini" href={ledgerHref(pair.posting2)}>
                      <p>{pair.posting2.date}</p>
                      <strong>{pair.posting2.payee}</strong>
                      <span>{pair.posting2.account}</span>
                      <strong>{formatCurrency(pair.posting2.amount)}</strong>
                    </a>
                  </div>

                  <div>
                    <button
                      class="button is-small is-light"
                      class:is-loading={suppressLoading[key]}
                      onclick={() => suppress(pair)}>Dismiss pair</button
                    >
                  </div>
                </article>
              {/each}
            </div>
          {/if}
        </section>
      {/if}

      {#if activeSection === "outliers"}
        <section class="box doctor-v3-panel">
          <div class="doctor-v3-headline">
            <h2 class="title is-4 mb-0">Outliers</h2>
            <span class="tag is-danger">{filteredOutliers.length} items</span>
          </div>

          {#if filteredOutliers.length === 0}
            <p class="has-text-grey mb-0">No outliers for this filter.</p>
          {:else}
            <div class="doctor-v3-pager mb-3">
              <button
                class="button is-small"
                disabled={outlierPage === 1}
                onclick={() => (outlierPage = Math.max(1, outlierPage - 1))}>Previous</button
              >
              <span>Page {outlierPage} of {outlierPageCount}</span>
              <button
                class="button is-small"
                disabled={outlierPage === outlierPageCount}
                onclick={() => (outlierPage = Math.min(outlierPageCount, outlierPage + 1))}
                >Next</button
              >
            </div>

            <div class="doctor-v3-list">
              {#each pagedOutliers as outlier}
                <a class="doctor-v3-card" href={ledgerHref(outlier.posting)}>
                  <div class="doctor-v3-card-title">
                    <span class="tag is-danger is-light">Outlier</span>
                    <strong>{outlier.posting.payee}</strong>
                    <span
                      class="tag is-rounded"
                      style={`background-color: ${confidenceColor(outlier.confidence)}; color: white`}
                      >{pct(outlier.confidence)}%</span
                    >
                  </div>
                  <p class="doctor-v3-card-meta">
                    {outlier.posting.date} · {outlier.posting.account}
                  </p>
                  <strong class="doctor-v3-amount">{formatCurrency(outlier.posting.amount)}</strong>
                  <p class="doctor-v3-card-meta">
                    {outlier.sigma.toFixed(1)}σ · mean {formatCurrency(outlier.mean)} · std dev
                    {formatCurrency(outlier.std_dev)}
                  </p>
                </a>
              {/each}
            </div>
          {/if}
        </section>
      {/if}
    {/if}
  </div>
</section>

<style>
  .doctor-v3-page {
    --dv3-bg: hsl(215, 18%, 15%);
    --dv3-border: rgba(255, 255, 255, 0.09);
    --dv3-muted: hsl(215, 9%, 62%);
    --dv3-strong: hsl(0, 0%, 92%);
    --dv3-card: hsl(215, 18%, 19%);
    --dv3-soft: hsl(215, 16%, 22%);
  }

  .doctor-v3-hero,
  .doctor-v3-nav,
  .doctor-v3-panel {
    background: var(--dv3-bg);
    border: 1px solid var(--dv3-border);
  }

  .doctor-v3-hero {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-end;
    margin-bottom: 1rem;
  }

  .doctor-v3-kicker {
    font-size: 0.78rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: #e0a83a;
    font-weight: 700;
    margin-bottom: 0.5rem;
  }

  .doctor-v3-subtitle,
  .doctor-v3-card-meta,
  .doctor-v3-mini span,
  .doctor-v3-mini p {
    color: var(--dv3-muted);
  }

  .doctor-v3-hero-actions,
  .doctor-v3-nav {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .doctor-v3-nav {
    padding: 0.75rem;
  }

  .doctor-v3-panel {
    padding: 1rem;
  }

  .doctor-v3-headline {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: center;
    margin-bottom: 0.85rem;
  }

  .doctor-v3-toolbar {
    display: grid;
    gap: 0.75rem;
  }

  .doctor-v3-toolbar-grid {
    grid-template-columns: minmax(0, 1fr) 180px;
  }

  .doctor-v3-list {
    display: grid;
    gap: 0.75rem;
  }

  .doctor-v3-queue-item,
  .doctor-v3-card,
  .doctor-v3-mini {
    width: 100%;
    border: 1px solid var(--dv3-border);
    background: var(--dv3-card);
    border-radius: 0.9rem;
    padding: 0.9rem;
    text-align: left;
    color: inherit;
    text-decoration: none;
  }

  .doctor-v3-card {
    display: grid;
    gap: 0.7rem;
  }

  .doctor-v3-card-title {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .doctor-v3-card-body {
    overflow-wrap: anywhere;
    color: var(--dv3-strong);
  }

  .doctor-v3-dual {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.6rem;
  }

  .doctor-v3-mini {
    display: grid;
    gap: 0.2rem;
    background: var(--dv3-soft);
  }

  .doctor-v3-amount {
    font-size: 1.25rem;
  }

  .doctor-v3-pager {
    display: flex;
    align-items: center;
    gap: 0.7rem;
  }

  @media (max-width: 860px) {
    .doctor-v3-hero,
    .doctor-v3-headline,
    .doctor-v3-pager {
      flex-direction: column;
      align-items: flex-start;
    }

    .doctor-v3-dual,
    .doctor-v3-toolbar-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
