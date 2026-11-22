<script lang="ts">
  import _ from "lodash";
  import Select from "svelte-select";

  let {
    accounts,
    onpreview
  }: {
    accounts: string[];
    onpreview?: (data: { operation: string; args: Record<string, any> }) => void;
  } = $props();

  const selectItems = $derived(
    _.map(accounts, (account) => {
      return { id: account, name: account };
    })
  );

  let selectedOldAccount: { id: string; name: string } | undefined = $state(undefined);
  let selectedNewAccount: { id: string; name: string } | undefined = $state(undefined);

  const OPERATIONS = [
    { id: "rename_account", label: "Recategorize / Rename Account" },
    { id: "add_tag", label: "Add Tag" },
    { id: "remove_tag", label: "Remove Tag" },
    { id: "toggle_status", label: "Toggle Cleared / Pending" },
    { id: "rename_payee", label: "Rename Payee" },
    { id: "delete", label: "Delete Transactions" }
  ];
  let selectedOperation = $state(OPERATIONS[0].id);

  let oldAccountName = $state("");
  let newAccountName = $state("");
  let newAccountFilterText = $state("");
  let tag = $state("");
  let status = $state<"toggle" | "*" | "!" | "">("toggle");
  let oldPayee = $state("");
  let newPayee = $state("");

  function handlePreview() {
    let args: Record<string, any> = {};
    switch (selectedOperation) {
      case "rename_account":
        args = {
          oldAccountName,
          newAccountName: newAccountName || newAccountFilterText
        };
        break;
      case "add_tag":
      case "remove_tag":
        args = { tag };
        break;
      case "toggle_status":
        args = { status };
        break;
      case "rename_payee":
        args = { oldPayee, newPayee };
        break;
      case "delete":
        args = {};
        break;
    }
    onpreview?.({ operation: selectedOperation, args });
  }
</script>

<div class="field is-grouped is-grouped-multiline">
  <div class="control">
    <div class="select">
      <select bind:value={selectedOperation}>
        {#each OPERATIONS as operation}
          <option value={operation.id}>{operation.label}</option>
        {/each}
      </select>
    </div>
  </div>

  {#if selectedOperation === "rename_account"}
    <div class="control is-expanded">
      <Select
        bind:value={selectedOldAccount}
        showChevron={true}
        items={selectItems}
        label="name"
        itemId="id"
        placeholder="Old Account name"
        searchable={true}
        clearable={false}
        on:change={() => {
          oldAccountName = selectedOldAccount?.name ?? "";
        }}
      />
    </div>
    <div class="control is-expanded">
      <Select
        bind:value={selectedNewAccount}
        bind:filterText={newAccountFilterText}
        showChevron={true}
        items={selectItems}
        label="name"
        itemId="id"
        placeholder="New Account name (select or type)"
        searchable={true}
        clearable={true}
        on:change={() => {
          newAccountName = selectedNewAccount?.name ?? "";
        }}
        on:clear={() => {
          newAccountName = "";
        }}
      />
    </div>
  {:else if selectedOperation === "add_tag" || selectedOperation === "remove_tag"}
    <div class="control is-expanded">
      <input
        bind:value={tag}
        class="input"
        type="text"
        placeholder="Tag name (e.g. reimbursable)"
      />
    </div>
  {:else if selectedOperation === "toggle_status"}
    <div class="control">
      <div class="select">
        <select bind:value={status}>
          <option value="toggle">Flip Cleared (*) ↔ Pending (!)</option>
          <option value="*">Mark Cleared (*)</option>
          <option value="!">Mark Pending (!)</option>
          <option value="">Clear Status Marker</option>
        </select>
      </div>
    </div>
  {:else if selectedOperation === "rename_payee"}
    <div class="control is-expanded">
      <input
        bind:value={oldPayee}
        class="input"
        type="text"
        placeholder="Current Payee (optional, blank = all filtered)"
      />
    </div>
    <div class="control is-expanded">
      <input bind:value={newPayee} class="input" type="text" placeholder="New Payee" />
    </div>
  {:else if selectedOperation === "delete"}
    <div class="control is-flex is-align-items-center">
      <span class="tag is-warning is-light">
        Deletes all currently filtered transactions (previewed before saving)
      </span>
    </div>
  {/if}

  <p class="control">
    <button
      type="button"
      class="button {selectedOperation === 'delete' ? 'is-danger' : 'is-link'}"
      onclick={handlePreview}>Preview</button
    >
  </p>
</div>
