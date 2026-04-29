<script lang="ts">
  import { iconify } from "$lib/icon";
  import _ from "lodash";
  import Select from "svelte-select";

  let {
    allAccounts,
    accounts = $bindable([] as string[])
  }: { allAccounts: string[]; accounts?: string[] } = $props();

  let allAccountItems: { value: string; label: string; created?: boolean }[] = $state([]);
  let accountItems: { value: string; label: string; created?: boolean }[] = $state([]);

  let filterText = $state("");

  $effect(() => {
    allAccountItems = _.map(allAccounts, (account) => ({
      value: account,
      label: account
    }));
  });

  $effect(() => {
    accountItems = _.map(accounts, (account) => ({
      value: account,
      label: account
    }));
  });

  function handleFilter(e: any) {
    if (accountItems?.find((i) => i.label === filterText)) return;
    if (e.detail.length === 0 && filterText.length > 0) {
      const prev = allAccountItems.filter((i) => !i.created);
      allAccountItems = [...prev, { value: filterText, label: filterText, created: true }];
    }
  }

  function handleChange(e: any) {
    if (e.type === "clear") {
      accountItems = _.without(accountItems, e.detail);
    } else {
      accountItems = _.cloneDeep(e.detail);
    }

    accounts = accountItems.map((i) => i.value);
  }
</script>

<Select
  --list-z-index="5"
  multiple
  class="is-small is-expandable custom-icon"
  items={allAccountItems}
  value={accountItems}
  showChevron={true}
  searchable={true}
  clearable={false}
  onchange={handleChange}
  on:clear={handleChange}
  on:filter={handleFilter}
  bind:filterText
>
  <div slot="selection" let:selection>
    <span>{iconify(selection.label)}</span>
  </div>
  <div slot="item" let:item>
    {item.created ? "Add: " : ""}
    {iconify(item.label)}
  </div>
</Select>
