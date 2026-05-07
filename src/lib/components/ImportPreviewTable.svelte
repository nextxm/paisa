<script lang="ts">
  import type { ImportPreviewRow } from "$lib/utils";
  import _ from "lodash";

  let {
    rows = [],
    included = $bindable([])
  }: {
    rows?: ImportPreviewRow[];
    included?: boolean[];
  } = $props();

  let columns = $derived.by(() => {
    const keys = new Set<string>();
    rows.forEach((r) => {
      Object.keys(r.row || {}).forEach((k) => {
        if (k !== "index") keys.add(k);
      });
    });
    return _.sortBy(Array.from(keys), [(k) => k.length, (k) => k]);
  });

  let allIncluded = $derived(rows.length > 0 && rows.every((_, i) => !!included[i]));
</script>

<div class="table-wrapper">
  <table class="mt-0 table is-bordered is-size-7 is-narrow has-sticky-header has-sticky-column">
    <thead>
      <tr>
        <th class="has-background-light">
          <input
            type="checkbox"
            checked={allIncluded}
            onchange={(e) => {
              const checked = (e.currentTarget as HTMLInputElement).checked;
              included = rows.map(() => checked);
            }}
          />
        </th>
        <th class="has-background-light">Status</th>
        {#each columns as column}
          <th class="has-background-light">{column}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each rows as row, ri}
        <tr>
          <th class="has-background-light">
            <div class="is-flex is-align-items-center">
              <input type="checkbox" bind:checked={included[ri]} />
              <b class="ml-2">{row.index}</b>
            </div>
          </th>
          <td>
            {#if row.valid}
              <span class="tag is-success is-light">valid</span>
            {:else}
              <span class="tag is-danger is-light">invalid</span>
              {#if row.errors?.length}
                <div class="has-text-danger mt-1">{row.errors.join("; ")}</div>
              {/if}
            {/if}
          </td>
          {#each columns as column}
            <td>{row.row?.[column] || ""}</td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style lang="scss">
  .table-wrapper {
    overflow-x: auto;
    overflow-y: auto;
    max-height: calc(100vh - 205px);
  }
</style>
