<script lang="ts">
  import _ from "lodash";
  import type { SankeyPeriod } from "../../persisted_store";

  type TabValue = string | number | SankeyPeriod;

  let {
    options = [],
    value = $bindable<TabValue | undefined>(undefined)
  }: { options: { label: string; value: TabValue }[]; value?: TabValue } = $props();

  $effect(() => {
    if (value && !options.find((option) => option.value === value) && !_.isEmpty(options)) {
      value = _.last(options)!.value;
    }
  });
</script>

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
