<script lang="ts">
  import BoxedTabs from "./BoxedTabs.svelte";
  import type { SankeyPeriod } from "../../persisted_store";
  import dayjs from "dayjs";
  import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
  import isSameOrBefore from "dayjs/plugin/isSameOrBefore";

  dayjs.extend(isSameOrAfter);
  dayjs.extend(isSameOrBefore);

  export let value: SankeyPeriod = "month";
  export let refDate: string = "";
  export let minDate: dayjs.Dayjs = dayjs();
  export let maxDate: dayjs.Dayjs = dayjs();

  const options: { label: string; value: SankeyPeriod }[] = [
    { label: "Month", value: "month" },
    { label: "Quarter", value: "quarter" },
    { label: "Year", value: "year" }
  ];

  let prevValue = value;
  $: if (value !== prevValue) {
    refDate = "";
    prevValue = value;
  }

  $: current = refDate ? dayjs(refDate) : dayjs();
  
  $: label = getLabel(value, current);

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

  $: canPrev = current.subtract(1, value).endOf(value).isSameOrAfter(minDate.startOf("month"));
  $: canNext = current.add(1, value).startOf(value).isSameOrBefore(maxDate.endOf("month"));

  function reset() {
    refDate = "";
  }
</script>

<div class="is-flex is-align-items-center" style="gap: 0.5rem">
  <BoxedTabs bind:value {options} />
  
  <div class="is-flex is-align-items-center">
    <button class="button is-small" style="border: none; background: transparent; box-shadow: none;" disabled={!canPrev} on:click={prev}>
      <span class="icon is-small"><i class="fas fa-chevron-left" /></span>
    </button>
    <a class="has-text-weight-bold has-text-centered has-text-grey-darker is-size-7" style="min-width: 60px;" on:click={reset} aria-label="Reset to current">
      {label}
    </a>
    <button class="button is-small" style="border: none; background: transparent; box-shadow: none;" disabled={!canNext || !refDate} on:click={next}>
      <span class="icon is-small"><i class="fas fa-chevron-right" /></span>
    </button>
  </div>
</div>
