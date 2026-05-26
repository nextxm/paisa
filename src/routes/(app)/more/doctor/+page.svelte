<script lang="ts">
  import { onMount } from "svelte";
  import COLORS from "$lib/colors";
  import { ajax, formatCurrency } from "$lib/utils";
  import type { DuplicatePair, OutlierTransaction } from "$lib/utils";
  import { renderIssues } from "$lib/doctor";
  import { dataQualityIssueCount } from "../../../../store";

  let issues = $state([]);
  let duplicates: DuplicatePair[] = $state([]);
  let outliers: OutlierTransaction[] = $state([]);
  let suppressLoading: Record<string, boolean> = $state({});

  onMount(async () => {
    ({ issues } = await ajax("/api/diagnosis"));
    renderIssues(issues);
    const dq = await ajax("/api/diagnosis/duplicates");
    duplicates = dq.duplicates || [];
    outliers = dq.outliers || [];
    dataQualityIssueCount.set(duplicates.length + outliers.length);
  });

  async function suppress(pair: DuplicatePair) {
    const key = `${pair.posting1.id}-${pair.posting2.id}`;
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
        (p) => !(p.posting1.id === pair.posting1.id && p.posting2.id === pair.posting2.id)
      );
    } finally {
      suppressLoading[key] = false;
    }
  }

  function pct(confidence: number) {
    return Math.round(confidence * 100);
  }

  function confidenceColor(confidence: number) {
    if (confidence >= 0.8) return COLORS.lossText;
    if (confidence >= 0.5) return "#e6a817";
    return COLORS.gainText;
  }
</script>

<section class="section tab-doctor">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12 has-text-centered">
        <div>
          <b
            class="p-1 has-text-white"
            style="background-color: {issues.length > 0 ? COLORS.lossText : COLORS.gainText}"
            >{issues.length}</b
          > potential issue(s) found.
        </div>
      </div>
    </div>
    <div class="columns is-flex-wrap-wrap" id="d3-diagnosis"></div>

    <!-- Data Quality Section -->
    <hr />
    <h2 class="title is-5">
      Data Quality
      {#if duplicates.length + outliers.length > 0}
        <span class="tag is-danger is-rounded ml-2">{duplicates.length + outliers.length}</span>
      {/if}
    </h2>

    <!-- Duplicate Pairs -->
    <h3 class="title is-6 mt-4">Potential Duplicates</h3>
    {#if duplicates.length === 0}
      <p class="has-text-grey">No duplicate transactions detected.</p>
    {:else}
      {#each duplicates as pair}
        {@const key = `${pair.posting1.id}-${pair.posting2.id}`}
        <div class="box mb-4">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
            <div>
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <span class="tag is-warning is-light mr-2 invertable">Duplicate</span>
              <span class="has-text-grey is-size-7">{@html pair.reason}</span>
            </div>
            <div class="is-flex is-align-items-center">
              <span
                class="tag is-rounded mr-3"
                style="background-color: {confidenceColor(pair.confidence)}; color: white"
              >
                {pct(pair.confidence)}% confidence
              </span>
              <button
                class="button is-small is-light"
                class:is-loading={suppressLoading[key]}
                onclick={() => suppress(pair)}
                title="Mark as not a duplicate"
              >
                Dismiss
              </button>
            </div>
          </div>
          <div class="columns">
            <div class="column is-6">
              <div class="notification is-light invertable py-2 px-3">
                <p class="is-size-7 has-text-grey mb-1">Posting 1</p>
                <p>
                  <strong>{pair.posting1.date}</strong>
                  &nbsp;
                  <a
                    href="/ledger/editor/{encodeURIComponent(pair.posting1.file_name)}#{pair
                      .posting1.transaction_begin_line}"
                  >
                    {pair.posting1.payee}
                  </a>
                </p>
                <p class="is-size-7">{pair.posting1.account}</p>
                <p class="has-text-weight-semibold">{formatCurrency(pair.posting1.amount)}</p>
              </div>
            </div>
            <div class="column is-6">
              <div class="notification is-light invertable py-2 px-3">
                <p class="is-size-7 has-text-grey mb-1">Posting 2</p>
                <p>
                  <strong>{pair.posting2.date}</strong>
                  &nbsp;
                  <a
                    href="/ledger/editor/{encodeURIComponent(pair.posting2.file_name)}#{pair
                      .posting2.transaction_begin_line}"
                  >
                    {pair.posting2.payee}
                  </a>
                </p>
                <p class="is-size-7">{pair.posting2.account}</p>
                <p class="has-text-weight-semibold">{formatCurrency(pair.posting2.amount)}</p>
              </div>
            </div>
          </div>
        </div>
      {/each}
    {/if}

    <!-- Outlier Transactions -->
    <h3 class="title is-6 mt-5">Outlier Transactions</h3>
    {#if outliers.length === 0}
      <p class="has-text-grey">No statistical outliers detected.</p>
    {:else}
      <div class="columns is-flex-wrap-wrap">
        {#each outliers as outlier}
          <div class="column is-6">
            <div class="box">
              <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                <span class="tag is-danger is-light invertable">Outlier</span>
                <span
                  class="tag is-rounded"
                  style="background-color: {confidenceColor(outlier.confidence)}; color: white"
                >
                  {pct(outlier.confidence)}% confidence
                </span>
              </div>
              <p>
                <strong>{outlier.posting.date}</strong>
                &nbsp;
                <a
                  href="/ledger/editor/{encodeURIComponent(outlier.posting.file_name)}#{outlier
                    .posting.transaction_begin_line}"
                >
                  {outlier.posting.payee}
                </a>
              </p>
              <p class="is-size-7 has-text-grey">{outlier.posting.account}</p>
              <p class="has-text-weight-semibold">{formatCurrency(outlier.posting.amount)}</p>
              <p class="is-size-7 has-text-grey mt-1">
                {outlier.sigma.toFixed(1)}σ above mean &nbsp;(mean:
                {formatCurrency(outlier.mean)}, σ: {formatCurrency(outlier.std_dev)})
              </p>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</section>
