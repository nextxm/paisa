<script lang="ts">
  import _ from "lodash";

  let { options, value = $bindable() }: { options: { label: string; value: any }[]; value: any } =
    $props();

  $effect(() => {
    if (value && !options.find((option) => option.value === value) && !_.isEmpty(options)) {
      value = _.last(options)!.value;
    }
  });
</script>

<div class="du-tabs du-tabs-boxed du-tabs-sm">
  {#each options as option}
    <a
      class="du-tab {option.value === value ? 'du-tab-active' : ''}"
      onclick={() => (value = option.value)}
    >
      {option.label}
    </a>
  {/each}
</div>
