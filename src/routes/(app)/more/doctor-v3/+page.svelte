<script lang="ts">
  import { onMount } from "svelte";
  import COLORS from "$lib/colors";
  import { issueTone } from "$lib/doctor_v2";
  import { ajax, formatCurrency } from "$lib/utils";
  import type { DuplicatePair, Issue, OutlierTransaction, Posting } from "$lib/utils";
  import { dataQualityIssueCount } from "../../../../store";

  type TriageKind = "diagnosis" | "duplicates" | "outliers";
  type FocusKind = "all" | TriageKind;
  type ReviewMode = "cards" | "focus";
  type GroupBy = "none" | "type" | "account" | "year" | "month";

  type TriageItem = {
    id: string;
    kind: TriageKind;
    score: number;
    title: string;
    subtitle: string;
    searchable: string;
    confidence: number;
    issue?: Issue;
    pair?: DuplicatePair;
    outlier?: OutlierTransaction;
  };

  let loading = $state(true);
  let loadError = $state("");

  let issues: Issue[] = $state([]);
  let duplicates: DuplicatePair[] = $state([]);
  let outliers: OutlierTransaction[] = $state([]);

  let suppressLoading: Record<string, boolean> = $state({});

  let reviewMode: ReviewMode = $state("cards");
  let focusKind: FocusKind = $state("all");
  let groupBy: GroupBy = $state("none");
  let groupFilterValue = $state("");
  let queryText = $state("");
  let minConfidence = $state(0.5);
  let accountFilter = $state("");
  let dateFrom = $state("");
  let dateTo = $state("");
  let amountMinText = $state("");
  let amountMaxText = $state("");
  let currentIndex = $state(0);

  let totalSignals = $derived(issues.length + duplicates.length + outliers.length);

  let triageItems = $derived.by(() => {
    const rows: TriageItem[] = [];

    issues.forEach((issue, index) => {
      const tone = issueTone(issue.level);
      const toneWeight =
        tone === "danger" ? 320 : tone === "warning" ? 220 : tone === "success" ? 100 : 140;

      rows.push({
        id: `issue-${index}`,
        kind: "diagnosis",
        score: toneWeight - index,
        title: issue.summary,
        subtitle: issue.level,
        searchable: [issue.summary, issue.level, issue.description, issue.details]
          .join(" ")
          .toLowerCase(),
        confidence:
          tone === "danger" ? 0.95 : tone === "warning" ? 0.75 : tone === "success" ? 0.4 : 0.55,
        issue
      });
    });

    duplicates.forEach((pair, index) => {
      rows.push({
        id: `duplicate-${pair.posting1.id}-${pair.posting2.id}`,
        kind: "duplicates",
        score: 150 + Math.round(pair.confidence * 100) - index,
        title: pair.posting1.payee || pair.posting2.payee || "Potential duplicate",
        subtitle: `${pair.posting1.account} and ${pair.posting2.account}`,
        searchable: [
          pair.reason,
          pair.posting1.payee,
          pair.posting1.account,
          pair.posting1.date,
          String(pair.posting1.amount),
          pair.posting2.payee,
          pair.posting2.account,
          pair.posting2.date,
          String(pair.posting2.amount)
        ]
          .join(" ")
          .toLowerCase(),
        confidence: pair.confidence,
        pair
      });
    });

    outliers.forEach((outlier, index) => {
      rows.push({
        id: `outlier-${outlier.posting.id}`,
        kind: "outliers",
        score: 130 + Math.round(outlier.confidence * 100) - index,
        title: outlier.posting.payee || "Outlier transaction",
        subtitle: outlier.posting.account,
        searchable: [
          outlier.posting.payee,
          outlier.posting.account,
          outlier.posting.date,
          String(outlier.posting.amount)
        ]
          .join(" ")
          .toLowerCase(),
        confidence: outlier.confidence,
        outlier
      });
    });

    return rows.sort((left, right) => right.score - left.score);
  });

  let accountSuggestions = $derived.by(() => {
    const counts = new Map<string, number>();

    const bump = (account: string) => {
      if (!account) return;
      counts.set(account, (counts.get(account) || 0) + 1);
    };

    duplicates.forEach((pair) => {
      bump(pair.posting1.account);
      bump(pair.posting2.account);
    });

    outliers.forEach((outlier) => bump(outlier.posting.account));

    return [...counts.entries()]
      .sort((left, right) => right[1] - left[1])
      .slice(0, 8)
      .map(([account]) => account);
  });

  let visibleItems = $derived.by(() => {
    const q = queryText.trim().toLowerCase();
    const accountQuery = accountFilter.trim().toLowerCase();
    const minAmount = parseNumberFilter(amountMinText);
    const maxAmount = parseNumberFilter(amountMaxText);
    const hasTxnFilters =
      accountQuery.length > 0 ||
      dateFrom.length > 0 ||
      dateTo.length > 0 ||
      minAmount !== null ||
      maxAmount !== null;

    return triageItems.filter((item) => {
      if (focusKind !== "all" && item.kind !== focusKind) return false;
      if (item.kind !== "diagnosis" && item.confidence < minConfidence) return false;
      if (q && !item.searchable.includes(q)) return false;

      if (!hasTxnFilters && !groupFilterValue) return true;

      if (groupBy !== "none" && groupFilterValue) {
        const groupKey = itemGroupKey(item, groupBy);
        if (groupKey !== groupFilterValue) return false;
      }

      const postings = postingsForItem(item);
      if (postings.length === 0) return false;

      return postings.some((posting) =>
        postingMatchesFilters(posting, accountQuery, dateFrom, dateTo, minAmount, maxAmount)
      );
    });
  });

  let groupFilterOptions = $derived.by(() => {
    if (groupBy === "none") return [];

    const groups = new Set<string>();
    visibleItems.forEach((item) => groups.add(itemGroupKey(item, groupBy)));
    return [...groups].sort((a, b) => a.localeCompare(b));
  });

  type GroupedItems = { group: string; items: TriageItem[] }[];

  let visibleGroups: GroupedItems = $derived.by(() => {
    if (groupBy === "none") return [];

    const groups = new Map<string, TriageItem[]>();
    for (const item of visibleItems) {
      const key = itemGroupKey(item, groupBy);
      const group = key || "Unassigned";
      if (!groups.has(group)) groups.set(group, []);
      groups.get(group)!.push(item);
    }

    const entries = [...groups.entries()];

    if (groupBy === "month" || groupBy === "year") {
      entries.sort(([a], [b]) => b.localeCompare(a));
    } else if (groupBy === "account") {
      entries.sort(([a], [b]) => a.localeCompare(b));
    } else if (groupBy === "type") {
      const order: Record<string, number> = { diagnosis: 0, duplicates: 1, outliers: 2 };
      entries.sort(([a], [b]) => (order[a] ?? 3) - (order[b] ?? 3));
    }

    return entries.map(([group, items]) => ({ group, items }));
  });

  let currentItem = $derived(visibleItems[currentIndex] || null);
  let progressCount = $derived(visibleItems.length === 0 ? 0 : currentIndex + 1);
  let progressPercent = $derived(
    visibleItems.length === 0 ? 0 : Math.round((progressCount / visibleItems.length) * 100)
  );

  let counts = $derived({
    all: totalSignals,
    diagnosis: issues.length,
    duplicates: duplicates.length,
    outliers: outliers.length
  });

  onMount(() => {
    const keyHandler = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const tagName = target?.tagName?.toLowerCase() || "";
      const isInput = tagName === "input" || tagName === "select" || tagName === "textarea";
      if (isInput) return;

      if (event.key === "j") {
        event.preventDefault();
        moveNext();
      }
      if (event.key === "k") {
        event.preventDefault();
        movePrevious();
      }
    };

    window.addEventListener("keydown", keyHandler);

    void loadData();

    return () => {
      window.removeEventListener("keydown", keyHandler);
    };
  });

  async function loadData() {
    try {
      const [diagnosis, dataQuality] = await Promise.all([
        ajax("/api/diagnosis"),
        ajax("/api/diagnosis/duplicates")
      ]);

      issues = diagnosis.issues || [];
      duplicates = dataQuality.duplicates || [];
      outliers = dataQuality.outliers || [];
      syncIssueBadge();
    } catch (error) {
      console.error(error);
      loadError = "Doctor V3 could not load diagnosis data.";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    const maxIndex = Math.max(visibleItems.length - 1, 0);
    if (currentIndex > maxIndex) currentIndex = maxIndex;
  });

  function syncIssueBadge() {
    dataQualityIssueCount.set(duplicates.length + outliers.length);
  }

  function setReviewMode(mode: ReviewMode) {
    reviewMode = mode;
  }

  function setFocusKind(next: FocusKind) {
    focusKind = next;
    currentIndex = 0;
  }

  function updateQueryText(event: Event) {
    queryText = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function updateMinConfidence(event: Event) {
    minConfidence = Number((event.currentTarget as HTMLSelectElement).value) || 0;
    currentIndex = 0;
  }

  function updateAccountFilter(event: Event) {
    accountFilter = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function updateDateFrom(event: Event) {
    dateFrom = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function updateDateTo(event: Event) {
    dateTo = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function updateAmountMin(event: Event) {
    amountMinText = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function updateAmountMax(event: Event) {
    amountMaxText = (event.currentTarget as HTMLInputElement).value;
    currentIndex = 0;
  }

  function useAccountChip(account: string) {
    accountFilter = account;
    currentIndex = 0;
  }

  function clearQuickFilters() {
    accountFilter = "";
    dateFrom = "";
    dateTo = "";
    amountMinText = "";
    amountMaxText = "";
    currentIndex = 0;
  }

  function movePrevious() {
    if (currentIndex > 0) currentIndex -= 1;
  }

  function moveNext() {
    if (currentIndex < visibleItems.length - 1) currentIndex += 1;
  }

  function inspectItem(index: number) {
    currentIndex = Math.max(0, Math.min(index, visibleItems.length - 1));
    reviewMode = "focus";
  }

  function jumpToItem(index: number) {
    currentIndex = Math.max(0, Math.min(index, visibleItems.length - 1));
  }

  function openKind(kind: TriageKind) {
    setFocusKind(kind);
  }

  function postingsForItem(item: TriageItem): Posting[] {
    if (item.pair) return [item.pair.posting1, item.pair.posting2];
    if (item.outlier) return [item.outlier.posting];
    return [];
  }

  function postingMatchesFilters(
    posting: Posting,
    accountQuery: string,
    fromDate: string,
    toDate: string,
    minAmount: number | null,
    maxAmount: number | null
  ) {
    if (accountQuery && !posting.account.toLowerCase().includes(accountQuery)) return false;

    const normalizedDate = normalizeDate(posting.date);
    if (fromDate && (!normalizedDate || normalizedDate < fromDate)) return false;
    if (toDate && (!normalizedDate || normalizedDate > toDate)) return false;

    const amountAbs = Math.abs(Number(posting.amount));
    if (minAmount !== null && (!Number.isFinite(amountAbs) || amountAbs < minAmount)) return false;
    if (maxAmount !== null && (!Number.isFinite(amountAbs) || amountAbs > maxAmount)) return false;

    return true;
  }

  function normalizeDate(value: unknown) {
    if (!value) return "";

    if (typeof value === "string") {
      return value.slice(0, 10);
    }

    if (
      typeof value === "object" &&
      value !== null &&
      "format" in value &&
      typeof (value as { format?: unknown }).format === "function"
    ) {
      return (value as { format: (pattern: string) => string }).format("YYYY-MM-DD");
    }

    return String(value).slice(0, 10);
  }

  function parseNumberFilter(value: string) {
    if (!value.trim()) return null;
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }

  function itemGroupKey(item: TriageItem, by: GroupBy) {
    if (by === "type") {
      if (item.kind === "diagnosis") return "Diagnosis";
      if (item.kind === "duplicates") return "Duplicates";
      return "Outliers";
    }

    const posting = primaryPosting(item);
    if (!posting) return "Unassigned";

    if (by === "account") return posting.account;
    const normalizedDate = normalizeDate(posting.date);
    if (by === "year") return normalizedDate.slice(0, 4);
    if (by === "month") return normalizedDate.slice(0, 7);

    return "Unassigned";
  }

  function updateGroupBy(event: Event) {
    groupBy = (event.currentTarget as HTMLSelectElement).value as GroupBy;
    groupFilterValue = "";
    currentIndex = 0;
  }

  function updateGroupFilter(event: Event) {
    groupFilterValue = (event.currentTarget as HTMLSelectElement).value;
    currentIndex = 0;
  }

  function groupFilterLabel() {
    if (groupBy === "type") return "Type";
    if (groupBy === "account") return "Account";
    if (groupBy === "year") return "Year";
    if (groupBy === "month") return "Month";
    return "Group";
  }

  function pairKey(pair: DuplicatePair) {
    return `${pair.posting1.id}-${pair.posting2.id}`;
  }

  function ledgerHref(posting: Posting) {
    return `/ledger/editor/${encodeURIComponent(posting.file_name)}#${posting.transaction_begin_line}`;
  }

  function primaryPosting(item: TriageItem) {
    if (item.pair) return item.pair.posting1;
    if (item.outlier) return item.outlier.posting;
    return null;
  }

  function pct(confidence: number) {
    return Math.round(confidence * 100);
  }

  function confidenceColor(confidence: number) {
    if (confidence >= 0.8) return COLORS.lossText;
    if (confidence >= 0.5) return "#d98e04";
    return COLORS.gainText;
  }

  function severityLabel(level: string) {
    const tone = issueTone(level);
    if (tone === "danger") return "critical";
    if (tone === "warning") return "warning";
    if (tone === "success") return "ok";
    return "info";
  }

  async function suppressDuplicate(pair: DuplicatePair) {
    const key = pairKey(pair);
    suppressLoading = { ...suppressLoading, [key]: true };

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
      syncIssueBadge();
    } finally {
      suppressLoading = { ...suppressLoading, [key]: false };
    }
  }
</script>

<section class="section doctor-v3-page">
  <div class="container is-fluid">
    <header class="doctor-v3-hero box">
      <div>
        <p class="doctor-v3-kicker">Doctor V3 Triage Studio</p>
        <h1 class="title is-2 mb-2">Fast triage with quick slicing</h1>
        <p class="subtitle is-6 mb-0 doctor-v3-subtitle">
          Filter by account, date, and amount, then review as cards or deep focus mode.
        </p>
      </div>
      <div class="doctor-v3-actions">
        <a class="button is-light" href="/more/doctor-v2">Open Doctor V2</a>
        <a class="button is-light" href="/more/doctor">Open legacy Doctor</a>
      </div>
    </header>

    {#if loading}
      <div class="box has-text-centered py-6">
        <progress class="progress is-small is-dark" max="100">Loading</progress>
        <p class="has-text-grey mb-0">Building your triage queue...</p>
      </div>
    {:else if loadError}
      <div class="notification is-danger is-light">{loadError}</div>
    {:else}
      <section class="box doctor-v3-controls mb-4">
        <div class="doctor-v3-top-row">
          <div class="doctor-v3-segment">
            <button
              class="button"
              class:is-dark={focusKind === "all"}
              onclick={() => setFocusKind("all")}>All ({counts.all})</button
            >
            <button
              class="button"
              class:is-dark={focusKind === "diagnosis"}
              onclick={() => setFocusKind("diagnosis")}>Diagnosis ({counts.diagnosis})</button
            >
            <button
              class="button"
              class:is-dark={focusKind === "duplicates"}
              onclick={() => setFocusKind("duplicates")}>Duplicates ({counts.duplicates})</button
            >
            <button
              class="button"
              class:is-dark={focusKind === "outliers"}
              onclick={() => setFocusKind("outliers")}>Outliers ({counts.outliers})</button
            >
          </div>

          <div class="doctor-v3-segment">
            <button
              class="button"
              class:is-dark={reviewMode === "cards"}
              onclick={() => setReviewMode("cards")}>Cards View</button
            >
            <button
              class="button"
              class:is-dark={reviewMode === "focus"}
              onclick={() => setReviewMode("focus")}>Focus View</button
            >
          </div>
        </div>

        <div class="doctor-v3-input-grid">
          <label class="doctor-v3-field">
            <span>Search</span>
            <input
              class="input"
              type="text"
              placeholder="Payee, reason, account, diagnosis text"
              value={queryText}
              oninput={updateQueryText}
            />
          </label>

          <label class="doctor-v3-field">
            <span>Min confidence</span>
            <div class="select is-fullwidth">
              <select value={minConfidence} onchange={updateMinConfidence}>
                <option value="0">All confidence</option>
                <option value="0.5">50% and above</option>
                <option value="0.8">80% and above</option>
              </select>
            </div>
          </label>

          <label class="doctor-v3-field">
            <span>Account contains</span>
            <input
              class="input"
              type="text"
              list="doctor-v3-account-options"
              placeholder="Expenses:Food or Assets:Bank"
              value={accountFilter}
              oninput={updateAccountFilter}
            />
          </label>

          <label class="doctor-v3-field">
            <span>Date from</span>
            <input class="input" type="date" value={dateFrom} oninput={updateDateFrom} />
          </label>

          <label class="doctor-v3-field">
            <span>Date to</span>
            <input class="input" type="date" value={dateTo} oninput={updateDateTo} />
          </label>

          <label class="doctor-v3-field">
            <span>Amount min (abs)</span>
            <input
              class="input"
              type="number"
              step="0.01"
              placeholder="0"
              value={amountMinText}
              oninput={updateAmountMin}
            />
          </label>

          <label class="doctor-v3-field">
            <span>Amount max (abs)</span>
            <input
              class="input"
              type="number"
              step="0.01"
              placeholder="100000"
              value={amountMaxText}
              oninput={updateAmountMax}
            />
          </label>

          <label class="doctor-v3-field">
            <span>Group by</span>
            <div class="select is-fullwidth">
              <select bind:value={groupBy} onchange={updateGroupBy}>
                <option value="none">None</option>
                <option value="type">Type</option>
                <option value="account">Account</option>
                <option value="year">Year</option>
                <option value="month">Month</option>
              </select>
            </div>
          </label>

          {#if groupBy !== "none"}
            <label class="doctor-v3-field">
              <span>Filter {groupFilterLabel()}</span>
              <div class="select is-fullwidth">
                <select value={groupFilterValue} onchange={updateGroupFilter}>
                  <option value="">All {groupFilterLabel()}</option>
                  {#each groupFilterOptions as option}
                    <option value={option}>{option}</option>
                  {/each}
                </select>
              </div>
            </label>
          {/if}

          <div class="doctor-v3-progress">
            <p class="mb-1">In queue: <strong>{visibleItems.length}</strong></p>
            <p class="mb-2">
              Current: <strong>{progressCount}</strong> / {visibleItems.length || 0}
            </p>
            <progress class="progress is-info mb-0" max="100" value={progressPercent}
              >{progressPercent}%</progress
            >
            <button class="button is-small is-light mt-2" onclick={clearQuickFilters}
              >Clear quick filters</button
            >
          </div>
        </div>

        <div class="doctor-v3-account-chips">
          {#each accountSuggestions as account}
            <button class="tag is-light doctor-v3-chip" onclick={() => useAccountChip(account)}
              >{account}</button
            >
          {/each}
        </div>

        <datalist id="doctor-v3-account-options">
          {#each accountSuggestions as account}
            <option value={account}></option>
          {/each}
        </datalist>
      </section>

      {#if visibleItems.length === 0}
        <div class="notification is-success is-light">
          <strong>Queue clear.</strong> No items match your current filters.
        </div>
      {:else if reviewMode === "cards"}
        {#if groupBy === "none"}
          <section class="doctor-v3-cardwall">
            {#each visibleItems.slice(0, 48) as item, index}
              {@const posting = primaryPosting(item)}
              <article class="box doctor-v3-tile">
                <div class="doctor-v3-tile-head">
                  <span class="tag is-light">{item.kind}</span>
                  <span
                    class="tag is-rounded doctor-v3-confidence"
                    style={`background-color: ${confidenceColor(item.confidence)}`}
                    >{pct(item.confidence)}%</span
                  >
                </div>

                <h3 class="title is-6 mb-1">{item.title}</h3>
                <p class="doctor-v3-subtitle mb-3">{item.subtitle}</p>

                {#if posting}
                  <p class="doctor-v3-subtitle mb-3">
                    {posting.date} · {posting.account} · {formatCurrency(posting.amount)}
                  </p>
                {/if}

                <div class="doctor-v3-actions-row">
                  <button class="button is-small is-light" onclick={() => inspectItem(index)}
                    >Inspect in focus</button
                  >
                  {#if posting}
                    <a class="button is-small is-light" href={ledgerHref(posting)}>Open in ledger</a
                    >
                  {/if}
                </div>
              </article>
            {/each}
          </section>
        {:else}
          {#each visibleGroups as group}
            <div class="doctor-v3-group">
              <div class="doctor-v3-group-header">
                <div>{group.group}</div>
                <span class="tag is-light">{group.items.length}</span>
              </div>
              <section class="doctor-v3-cardwall doctor-v3-group-cards">
                {#each group.items.slice(0, 48) as item, index}
                  {@const posting = primaryPosting(item)}
                  <article class="box doctor-v3-tile">
                    <div class="doctor-v3-tile-head">
                      <span class="tag is-light">{item.kind}</span>
                      <span
                        class="tag is-rounded doctor-v3-confidence"
                        style={`background-color: ${confidenceColor(item.confidence)}`}
                        >{pct(item.confidence)}%</span
                      >
                    </div>

                    <h3 class="title is-6 mb-1">{item.title}</h3>
                    <p class="doctor-v3-subtitle mb-3">{item.subtitle}</p>

                    {#if posting}
                      <p class="doctor-v3-subtitle mb-3">
                        {posting.date} · {posting.account} · {formatCurrency(posting.amount)}
                      </p>
                    {/if}

                    <div class="doctor-v3-actions-row">
                      <button class="button is-small is-light" onclick={() => inspectItem(index)}
                        >Inspect in focus</button
                      >
                      {#if posting}
                        <a class="button is-small is-light" href={ledgerHref(posting)}
                          >Open in ledger</a
                        >
                      {/if}
                    </div>
                  </article>
                {/each}
              </section>
            </div>
          {/each}
        {/if}
      {:else}
        <div class="doctor-v3-workspace">
          <aside class="box doctor-v3-rail">
            <h2 class="title is-6 mb-3">Queue</h2>
            <div class="doctor-v3-rail-list">
              {#each visibleItems.slice(0, 24) as item, index}
                <button
                  class="doctor-v3-rail-item"
                  class:is-active={index === currentIndex}
                  onclick={() => jumpToItem(index)}
                >
                  <span class="tag is-light">{item.kind}</span>
                  <strong>{item.title}</strong>
                  <span class="doctor-v3-rail-subtitle">{item.subtitle}</span>
                </button>
              {/each}
            </div>
            {#if visibleItems.length > 24}
              <p class="has-text-grey is-size-7 mt-2 mb-0">
                Showing top 24 ranked items. Use filters to narrow further.
              </p>
            {/if}
          </aside>

          <section class="box doctor-v3-focus">
            <div class="doctor-v3-focus-head">
              <div>
                <p class="doctor-v3-focus-kind mb-1">{currentItem?.kind}</p>
                <h2 class="title is-4 mb-1">{currentItem?.title}</h2>
                <p class="doctor-v3-subtitle mb-0">{currentItem?.subtitle}</p>
              </div>
              {#if currentItem}
                <span
                  class="tag is-rounded doctor-v3-confidence"
                  style={`background-color: ${confidenceColor(currentItem.confidence)}`}
                  >{pct(currentItem.confidence)}%</span
                >
              {/if}
            </div>

            {#if currentItem?.kind === "diagnosis" && currentItem.issue}
              <article class="doctor-v3-detail">
                <div class="doctor-v3-inline-tags mb-3">
                  <span class="tag is-dark">{severityLabel(currentItem.issue.level)}</span>
                  <button class="button is-small is-light" onclick={() => openKind("diagnosis")}
                    >Show diagnosis only</button
                  >
                </div>
                <div class="doctor-v3-rich-copy">
                  {@html `${currentItem.issue.description}<br/><br/>${currentItem.issue.details}`}
                </div>
              </article>
            {/if}

            {#if currentItem?.kind === "duplicates" && currentItem.pair}
              {@const key = pairKey(currentItem.pair)}
              <article class="doctor-v3-detail">
                <div class="doctor-v3-inline-tags mb-3">
                  <span class="tag is-warning is-light">Potential duplicate</span>
                  <button class="button is-small is-light" onclick={() => openKind("duplicates")}
                    >Show duplicates only</button
                  >
                </div>

                <p class="doctor-v3-reason mb-3">{@html currentItem.pair.reason}</p>

                <div class="doctor-v3-compare">
                  <a class="doctor-v3-posting" href={ledgerHref(currentItem.pair.posting1)}>
                    <p>{currentItem.pair.posting1.date}</p>
                    <strong>{currentItem.pair.posting1.payee}</strong>
                    <span>{currentItem.pair.posting1.account}</span>
                    <strong>{formatCurrency(currentItem.pair.posting1.amount)}</strong>
                  </a>
                  <a class="doctor-v3-posting" href={ledgerHref(currentItem.pair.posting2)}>
                    <p>{currentItem.pair.posting2.date}</p>
                    <strong>{currentItem.pair.posting2.payee}</strong>
                    <span>{currentItem.pair.posting2.account}</span>
                    <strong>{formatCurrency(currentItem.pair.posting2.amount)}</strong>
                  </a>
                </div>

                <div class="doctor-v3-actions-row mt-3">
                  <button
                    class="button is-small is-light"
                    class:is-loading={suppressLoading[key]}
                    onclick={() => suppressDuplicate(currentItem.pair!)}>Dismiss pair</button
                  >
                </div>
              </article>
            {/if}

            {#if currentItem?.kind === "outliers" && currentItem.outlier}
              <article class="doctor-v3-detail">
                <div class="doctor-v3-inline-tags mb-3">
                  <span class="tag is-danger is-light">Outlier</span>
                  <button class="button is-small is-light" onclick={() => openKind("outliers")}
                    >Show outliers only</button
                  >
                </div>

                <a class="doctor-v3-posting" href={ledgerHref(currentItem.outlier.posting)}>
                  <p>{currentItem.outlier.posting.date}</p>
                  <strong>{currentItem.outlier.posting.payee}</strong>
                  <span>{currentItem.outlier.posting.account}</span>
                  <strong>{formatCurrency(currentItem.outlier.posting.amount)}</strong>
                </a>

                <p class="doctor-v3-math mt-3 mb-0">
                  {currentItem.outlier.sigma.toFixed(1)}sigma from mean · mean {formatCurrency(
                    currentItem.outlier.mean
                  )} · std dev {formatCurrency(currentItem.outlier.std_dev)}
                </p>
              </article>
            {/if}

            <footer class="doctor-v3-nav-row">
              <button class="button" disabled={currentIndex === 0} onclick={movePrevious}
                >Previous (k)</button
              >
              <button
                class="button is-dark"
                disabled={currentIndex >= visibleItems.length - 1}
                onclick={moveNext}>Next (j)</button
              >
            </footer>
          </section>
        </div>
      {/if}
    {/if}
  </div>
</section>

<style>
  .doctor-v3-page {
    --dv3-bg: hsl(41, 73%, 95%);
    --dv3-surface: hsla(37, 63%, 99%, 0.98);
    --dv3-border: hsl(35, 36%, 78%);
    --dv3-accent: hsl(18, 74%, 42%);
    --dv3-accent-soft: hsl(18, 82%, 93%);
    --dv3-text: hsl(214, 23%, 18%);
    --dv3-muted: hsl(216, 12%, 40%);
    --dv3-shadow: 0 14px 30px hsla(18, 58%, 45%, 0.12);
    font-family: "Space Grotesk", "Avenir Next", "Segoe UI", sans-serif;
    background:
      radial-gradient(circle at 92% -10%, hsl(15, 88%, 86%), transparent 34%),
      radial-gradient(circle at 0% 112%, hsl(44, 100%, 84%), transparent 38%), var(--dv3-bg);
    padding: 2rem 0 3rem;
  }

  :global(.doctor-v3-page .title),
  :global(.doctor-v3-page .subtitle),
  :global(.doctor-v3-page p),
  :global(.doctor-v3-page strong),
  :global(.doctor-v3-page span),
  :global(.doctor-v3-page label) {
    color: var(--dv3-text);
  }

  .doctor-v3-page .container.is-fluid {
    max-width: 1280px;
    padding-left: 1.5rem;
    padding-right: 1.5rem;
  }

  .doctor-v3-page .box,
  .doctor-v3-page .notification,
  .doctor-v3-page .doctor-v3-progress {
    background: var(--dv3-surface);
    border-color: var(--dv3-border);
    color: var(--dv3-text);
  }

  .doctor-v3-page .box {
    box-shadow: var(--dv3-shadow);
  }

  .doctor-v3-page .button.is-light {
    background-color: hsl(0, 0%, 100%);
    color: var(--dv3-text);
    border-color: var(--dv3-border);
  }

  .doctor-v3-page .button.is-dark {
    background-color: var(--dv3-text);
    color: white;
    border-color: transparent;
  }

  .doctor-v3-page input,
  .doctor-v3-page select {
    background: hsl(0, 0%, 100%);
    color: var(--dv3-text);
    border: 1px solid var(--dv3-border);
  }

  .doctor-v3-page input::placeholder {
    color: var(--dv3-muted);
  }

  .doctor-v3-hero,
  .doctor-v3-controls,
  .doctor-v3-rail,
  .doctor-v3-focus,
  .doctor-v3-tile {
    border: 1px solid var(--dv3-border);
    box-shadow: var(--dv3-shadow);
    backdrop-filter: blur(8px);
  }

  .doctor-v3-hero {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 1.25rem;
    padding: 1.2rem 1.3rem;
    margin-bottom: 1rem;
  }

  .doctor-v3-kicker {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    margin-bottom: 0.65rem;
    padding: 0.32rem 0.7rem;
    border-radius: 999px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-weight: 700;
    font-size: 0.74rem;
    color: var(--dv3-accent);
    background: var(--dv3-accent-soft);
  }

  .doctor-v3-actions {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .doctor-v3-controls {
    display: grid;
    gap: 1rem;
    padding: 1.2rem 1.25rem;
  }

  .doctor-v3-top-row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .doctor-v3-segment {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .doctor-v3-input-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1rem;
    align-items: end;
  }

  .doctor-v3-field {
    display: grid;
    gap: 0.35rem;
  }

  .doctor-v3-field span {
    font-size: 0.76rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--dv3-muted);
    font-weight: 700;
  }

  .doctor-v3-progress {
    padding: 1rem;
    border-radius: 1rem;
    background: hsla(0, 0%, 100%, 0.88);
    min-width: 220px;
  }

  .doctor-v3-account-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .doctor-v3-chip {
    cursor: pointer;
    border: 1px solid var(--dv3-border);
    background: hsl(0, 0%, 100%);
    color: var(--dv3-text);
  }

  .doctor-v3-cardwall {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1rem;
  }

  .doctor-v3-group {
    margin-bottom: 1.6rem;
  }

  .doctor-v3-group-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    padding: 0.9rem 1.1rem;
    border-radius: 1rem;
    background: hsl(0, 0%, 100%);
    border: 1px solid var(--dv3-border);
    box-shadow: 0 6px 16px hsla(214, 23%, 18%, 0.06);
    margin-bottom: 0.9rem;
  }

  .doctor-v3-group-cards {
    gap: 0.95rem;
  }

  .doctor-v3-tile {
    display: grid;
    gap: 0.85rem;
    padding: 1rem;
    border-radius: 1rem;
    background: hsl(0, 0%, 100%);
    transition:
      transform 180ms ease,
      box-shadow 180ms ease;
  }

  .doctor-v3-tile:hover {
    transform: translateY(-3px);
    box-shadow: 0 18px 34px hsla(18, 58%, 45%, 0.14);
  }

  .doctor-v3-tile-head {
    display: flex;
    justify-content: space-between;
    gap: 0.85rem;
    align-items: center;
  }

  .doctor-v3-workspace {
    display: grid;
    grid-template-columns: minmax(250px, 300px) minmax(0, 1fr);
    gap: 1rem;
  }

  .doctor-v3-rail {
    display: grid;
    gap: 1rem;
    padding: 1rem;
  }

  .doctor-v3-rail-list {
    display: grid;
    gap: 0.7rem;
    max-height: 70vh;
    overflow: auto;
    padding-right: 0.2rem;
  }

  .doctor-v3-rail-item {
    display: grid;
    gap: 0.35rem;
    border-radius: 1rem;
    padding: 1rem;
    border: 1px solid var(--dv3-border);
    background: hsl(0, 0%, 100%);
    cursor: pointer;
    transition:
      border-color 120ms ease,
      transform 120ms ease,
      box-shadow 120ms ease;
  }

  .doctor-v3-rail-item:hover {
    transform: translateX(1px);
    box-shadow: 0 8px 20px hsla(214, 23%, 18%, 0.08);
  }

  .doctor-v3-rail-item.is-active {
    border-color: var(--dv3-accent);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--dv3-accent) 20%, transparent);
  }

  .doctor-v3-focus {
    display: grid;
    gap: 1rem;
    padding: 1rem;
  }

  .doctor-v3-focus-head {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-start;
  }

  .doctor-v3-focus-kind {
    font-size: 0.74rem;
    letter-spacing: 0.09em;
    text-transform: uppercase;
    font-weight: 700;
    color: var(--dv3-accent);
  }

  .doctor-v3-confidence {
    color: white;
    font-size: 0.8rem;
    font-weight: 700;
    min-width: 3.8rem;
    justify-content: center;
  }

  .doctor-v3-detail {
    border: 1px solid var(--dv3-border);
    border-radius: 1rem;
    background: hsl(0, 0%, 100%);
    padding: 1rem;
  }

  .doctor-v3-inline-tags {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .doctor-v3-rich-copy {
    color: var(--dv3-text);
    line-height: 1.55;
    overflow-wrap: anywhere;
  }

  .doctor-v3-compare {
    display: grid;
    gap: 0.85rem;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }

  .doctor-v3-posting {
    display: grid;
    gap: 0.35rem;
    padding: 0.95rem;
    text-decoration: none;
    border-radius: 0.95rem;
    border: 1px solid var(--dv3-border);
    background: hsl(34, 68%, 98%);
  }

  .doctor-v3-actions-row {
    display: flex;
    gap: 0.7rem;
    flex-wrap: wrap;
  }

  .doctor-v3-nav-row {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    align-items: center;
    margin-top: 0.25rem;
  }

  @media (max-width: 1080px) {
    .doctor-v3-workspace {
      grid-template-columns: 1fr;
    }

    .doctor-v3-rail-list {
      max-height: 38vh;
    }

    .doctor-v3-hero,
    .doctor-v3-focus-head,
    .doctor-v3-nav-row {
      flex-direction: column;
      align-items: flex-start;
    }
  }

  @media (max-width: 760px) {
    .doctor-v3-input-grid,
    .doctor-v3-cardwall,
    .doctor-v3-compare {
      grid-template-columns: 1fr;
    }

    .doctor-v3-hero {
      grid-template-columns: 1fr;
    }
  }
</style>
