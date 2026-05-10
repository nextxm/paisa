<script lang="ts">
  import { ajax } from "$lib/utils";
  import * as toast from "bulma-toast";
  import dayjs from "dayjs";
  import { refresh } from "../../store";

  let { open = $bindable(false), accounts = [] }: { open: boolean; accounts: string[] } = $props();

  let date = $state(dayjs().format("YYYY-MM-DD"));
  let payee = $state("");
  let narration = $state("");
  let fromAccount = $state("");
  let toAccount = $state("");
  let amount = $state("");
  let commodity = $state(USER_CONFIG.default_currency || "INR");
  let isLoading = $state(false);

  async function submit() {
    if (!date || !fromAccount || !toAccount || !amount || !commodity) {
      toast.toast({ message: "Please fill all required fields", type: "is-danger" });
      return;
    }

    isLoading = true;
    try {
      const response = await ajax("/api/transaction/add", {
        method: "POST",
        body: JSON.stringify({
          date,
          payee,
          narration,
          from_account: fromAccount,
          to_account: toAccount,
          amount,
          commodity
        })
      });

      if (response.success) {
        toast.toast({ message: "Transaction added successfully", type: "is-success" });
        open = false;
        // Reset form
        payee = "";
        narration = "";
        amount = "";
        refresh();
      } else {
        toast.toast({
          message: response.error?.message || "Failed to add transaction",
          type: "is-danger"
        });
      }
    } catch (e: any) {
      toast.toast({ message: e.message || "An error occurred", type: "is-danger" });
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="modal" class:is-active={open}>
  <div class="modal-background" onclick={() => (open = false)}></div>
  <div class="modal-card">
    <header class="modal-card-head">
      <p class="modal-card-title">Quick Add Transaction</p>
      <button class="delete" aria-label="close" onclick={() => (open = false)}></button>
    </header>
    <section class="modal-card-body">
      {#if !USER_CONFIG.add_journal_path}
        <article class="message is-warning is-small">
          <div class="message-body">
            <code>add_journal_path</code> is not configured. Please set it in Settings first.
          </div>
        </article>
      {/if}

      <div class="field">
        <label class="label is-small">Date</label>
        <div class="control">
          <input class="input is-small" type="date" bind:value={date} required />
        </div>
      </div>

      <div class="field">
        <label class="label is-small">Payee</label>
        <div class="control">
          <input class="input is-small" type="text" bind:value={payee} placeholder="e.g. Amazon" />
        </div>
      </div>

      <div class="field">
        <label class="label is-small">Narration</label>
        <div class="control">
          <input
            class="input is-small"
            type="text"
            bind:value={narration}
            placeholder="e.g. New keyboard"
          />
        </div>
      </div>

      <div class="columns is-mobile mb-0">
        <div class="column">
          <div class="field">
            <label class="label is-small">From Account</label>
            <div class="control">
              <input
                class="input is-small"
                list="accounts-list"
                bind:value={fromAccount}
                placeholder="Search account..."
                required
              />
            </div>
          </div>
        </div>
        <div class="column">
          <div class="field">
            <label class="label is-small">To Account</label>
            <div class="control">
              <input
                class="input is-small"
                list="accounts-list"
                bind:value={toAccount}
                placeholder="Search account..."
                required
              />
            </div>
          </div>
        </div>
      </div>

      <div class="columns is-mobile">
        <div class="column">
          <div class="field">
            <label class="label is-small">Amount</label>
            <div class="control">
              <input
                class="input is-small"
                type="text"
                bind:value={amount}
                placeholder="0.00"
                required
              />
            </div>
          </div>
        </div>
        <div class="column">
          <div class="field">
            <label class="label is-small">Currency</label>
            <div class="control">
              <input class="input is-small" type="text" bind:value={commodity} required />
            </div>
          </div>
        </div>
      </div>

      <datalist id="accounts-list">
        {#each accounts as account}
          <option value={account}></option>
        {/each}
      </datalist>
    </section>
    <footer class="modal-card-foot is-justify-content-flex-end">
      <button class="button is-small" onclick={() => (open = false)}>Cancel</button>
      <button
        class="button is-primary is-small {isLoading ? 'is-loading' : ''}"
        onclick={submit}
        disabled={!USER_CONFIG.add_journal_path}
      >
        Add Transaction
      </button>
    </footer>
  </div>
</div>

<style lang="scss">
  .modal-card {
    max-width: 450px;
  }
</style>
