<script lang="ts">
  import type { SankeyPeriod } from "../../persisted_store";
  import dayjs from "dayjs";
  import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
  import isSameOrBefore from "dayjs/plugin/isSameOrBefore";

  dayjs.extend(isSameOrAfter);
  dayjs.extend(isSameOrBefore);

  let {
    value = $bindable<SankeyPeriod>("month"),
    refDate = $bindable(""),
    minDate = dayjs(),
    maxDate = dayjs()
  }: {
    value: SankeyPeriod;
    refDate?: string;
    minDate?: dayjs.Dayjs;
    maxDate?: dayjs.Dayjs;
  } = $props();

  const options: { label: string; value: SankeyPeriod }[] = [
    { label: "Month", value: "month" },
    { label: "Quarter", value: "quarter" },
    { label: "Year", value: "year" }
  ];

  let prevValue = value;
  $effect(() => {
    if (value !== prevValue) {
      refDate = "";
      prevValue = value;
    }
  });

  const current = $derived(refDate ? dayjs(refDate) : dayjs());

  const label = $derived(getLabel(value, current));

  function getLabel(period: SankeyPeriod, date: dayjs.Dayjs) {
    if (period === "month") return date.format("MMM YYYY");
    if (period === "quarter") {
      const q = Math.floor(date.month() / 3) + 1;
      return `Q${q} ${date.year()}`;
    }
    if (period === "year") return date.format("YYYY");
    return "";
  }

  function prev() {
    refDate = current.subtract(1, value).startOf(value).format("YYYY-MM-DD");
  }

  function next() {
    // If next pushes us past current date max, cap it or just rely on disability
    refDate = current.add(1, value).startOf(value).format("YYYY-MM-DD");
  }

  const canPrev = $derived(
    current.subtract(1, value).endOf(value).isSameOrAfter(minDate.startOf("month"))
  );
  const canNext = $derived(
    current.add(1, value).startOf(value).isSameOrBefore(maxDate.endOf("month"))
  );

  function reset() {
    refDate = "";
  }
</script>

<div class="is-flex is-align-items-center" style="gap: 0.5rem">
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

  <div class="is-flex is-align-items-center">
    <button
      class="button is-small"
      style="border: none; background: transparent; box-shadow: none;"
      aria-label="Previous period"
      disabled={!canPrev}
      onclick={prev}
    >
      <span class="icon is-small"><i class="fas fa-chevron-left"></i></span>
    </button>
    <button
      type="button"
      class="has-text-weight-bold has-text-centered has-text-grey-darker is-size-7"
      style="min-width: 60px;"
      onclick={reset}
      aria-label="Reset to current"
    >
      {label}
    </button>
    <button
      class="button is-small"
      style="border: none; background: transparent; box-shadow: none;"
      aria-label="Next period"
      disabled={!canNext || !refDate}
      onclick={next}
    >
      <span class="icon is-small"><i class="fas fa-chevron-right"></i></span>
    </button>
  </div>
</div>
