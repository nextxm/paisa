<script lang="ts">
  import type dayjs from "dayjs";
  import { isMobile } from "$lib/utils";

  let {
    value = $bindable(),
    dateMin,
    dateMax
  }: {
    value: number;
    dateMin: dayjs.Dayjs;
    dateMax: dayjs.Dayjs;
  } = $props();

  const options = $derived.by(() => {
    const opts: { label: string; value: number }[] = [{ label: "All", value: -1 }];
    const diff = dateMax.diff(dateMin, "year");
    if (diff >= 10 && !isMobile()) {
      opts.push({ label: "10 years", value: 10 });
    }

    if (diff >= 5 && !isMobile()) {
      opts.push({ label: "5 years", value: 5 });
    }

    if (diff >= 3) {
      opts.push({ label: "3 years", value: 3 });
    }

    if (diff >= 1) {
      opts.push({ label: "1 year", value: 1 });
    }

    return opts;
  });
</script>

{#if options.length > 1}
  <div class="du-tabs du-tabs-boxed du-tabs-sm">
    {#each options as option}
      <button
        type="button"
        class="du-tab {option.value === value ? 'du-tab-active' : ''}"
        onclick={() => (value = option.value)}
      >
        {option.label}
      </button>
    {/each}
  </div>
{/if}
