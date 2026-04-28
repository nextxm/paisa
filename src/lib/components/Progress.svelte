<script lang="ts">
  import { formatPercentage } from "$lib/utils";
  import { dropRight, floor, range } from "lodash";

  let { small = false, progressPercent, showPercent = true }: {
    small?: boolean;
    progressPercent: number;
    showPercent?: boolean;
  } = $props();

  function computeProgress(p: number) {
    let times = range(0, floor(p / 100));
    let remainder = p % 100;
    if (remainder === 0) {
      times = dropRight(times, 1);
      remainder = p === 0 ? 0 : 100;
    }
    return { times, remainder };
  }

  const progress = $derived(computeProgress(progressPercent));
</script>

<div>
  {#each progress.times as _t}
    <div style="position: relative;" class="mb-1">
      <progress
        class="progress is-success {small ? 'is-extra-small' : 'is-small'}"
        value={100}
        max="100">{100}%</progress
      >
    </div>
  {/each}

  <div style="position: relative;">
    <progress
      class="progress is-success mb-1 {small ? 'is-small' : 'is-large'}"
      value={progress.remainder}
      max="100">{progress.remainder}%</progress
    >
    {#if small && showPercent}
      <span class="has-text-weight-bold">{formatPercentage(progressPercent / 100, 2)}</span>
    {/if}

    {#if !small && showPercent}
      <span
        class="has-text-weight-bold progress-percent {progress.remainder < 10 && 'less-than-10'}"
        style={progress.remainder > 10 ? `right: ${100 - progress.remainder}%;` : `left: ${progress.remainder}%;`}
        >{formatPercentage(progressPercent / 100, 2)}</span
      >
    {/if}
  </div>
</div>
