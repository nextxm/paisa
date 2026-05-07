<script lang="ts">
  import Select from "svelte-select";
  import type { ImportPreset } from "$lib/utils";

  let {
    presets = [],
    selectedPreset = $bindable(null),
    onsavecurrent = () => {}
  }: {
    presets?: ImportPreset[];
    selectedPreset?: ImportPreset | null;
    onsavecurrent?: () => void;
  } = $props();
</script>

<div class="field is-grouped mb-3">
  <p class="control is-expanded">
    <Select
      bind:value={selectedPreset}
      showChevron={true}
      items={presets}
      label="name"
      itemId="id"
      searchable={true}
      clearable={true}
      floatingConfig={{ strategy: "fixed" }}
    >
      <div slot="selection" let:selection>
        {selection.name}
        <span class="tag is-small is-link invertable is-light">{selection.preset_type}</span>
      </div>
      <div slot="item" let:item>
        <span class="name">{item.name}</span>
        <span class="tag is-small is-link invertable is-light">{item.preset_type}</span>
      </div>
    </Select>
  </p>
  <p class="control">
    <button
      class="button"
      type="button"
      onclick={onsavecurrent}
      aria-label="Save current as preset"
    >
      <span class="icon"><i class="fas fa-bookmark"></i></span>
      <span>Save Current</span>
    </button>
  </p>
</div>
