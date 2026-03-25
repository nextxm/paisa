<script lang="ts">
  import BoxedTabs from "./BoxedTabs.svelte";
  import type { SankeyPeriod } from "../../persisted_store";

  export let value: SankeyPeriod = "month";

  const options: { label: string; value: SankeyPeriod }[] = [
    { label: "Month", value: "month" },
    { label: "Quarter", value: "quarter" },
    { label: "Year", value: "year" }
  ];
</script>

<BoxedTabs bind:value {options} />
