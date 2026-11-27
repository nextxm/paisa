<script lang="ts">
  import { onMount } from "svelte";
  import { apiGet, formatMoney, formatShortDate, runtime, triggerSync } from "$lib/api";

  let {
    mode
  }: {
    mode: "import" | "fx" | "sheets";
  } = $props();

  // ---- Import state ----
  let importBaseAccount = $state("Assets:Checking");
  let importDelimiter = $state(",");
  let importFileName = $state("");
  let importContent = $state("");
  let importLoading = $state(false);
  let importCommitting = $state(false);
  let importMessage = $state<string | null>(null);
  let stagedTransactions = $state<
    Array<{
      date: string;
      payee: string;
      memo?: string;
      amount: number;
      is_debit?: boolean;
      selected_account?: string;
      suggested_account?: string;
      is_duplicate?: boolean;
      [key: string]: unknown;
    }>
  >([]);
  let includedStaged = $state<boolean[]>([]);
  let importPresets = $state<Array<{ id?: string; name?: string; delimiter?: string }>>([]);

  async function handleFileSelect(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    importFileName = file.name;
    importContent = await file.text();
    importMessage = null;
  }

  async function parseStatement() {
    if (!importContent.trim()) return;
    importLoading = true;
    importMessage = null;
    try {
      const res = await fetch("/api/import/statement", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          format: "csv",
          content: importContent,
          delimiter: importDelimiter,
          base_account: importBaseAccount
        })
      });
      if (!res.ok) {
        importMessage = `Parse failed: ${res.statusText}`;
        return;
      }
      const data = (await res.json()) as { transactions?: typeof stagedTransactions };
      stagedTransactions = (data.transactions || []).map((tx) => ({
        ...tx,
        selected_account: tx.selected_account || tx.suggested_account || "Expenses:Misc"
      }));
      includedStaged = stagedTransactions.map((tx) => !tx.is_duplicate);
      importMessage = `Parsed ${stagedTransactions.length} rows.`;
    } finally {
      importLoading = false;
    }
  }

  async function commitImport() {
    const selected = stagedTransactions.filter((_, i) => includedStaged[i]);
    if (selected.length === 0) return;
    importCommitting = true;
    importMessage = null;
    try {
      const res = await fetch("/api/import/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          base_account: importBaseAccount,
          commodity: runtime.defaultCurrency,
          group_by_date: false,
          raw_content: "",
          transactions: selected
        })
      });
      const data = (await res.json()) as { success?: boolean; committed_count?: number };
      if (res.ok && data.success) {
        importMessage = `Committed ${data.committed_count ?? selected.length} transactions to ledger.`;
        stagedTransactions = stagedTransactions.filter((_, i) => !includedStaged[i]);
        includedStaged = stagedTransactions.map(() => true);
        void triggerSync({ journal: true });
      } else {
        importMessage = "Commit failed.";
      }
    } finally {
      importCommitting = false;
    }
  }

  // ---- FX state ----
  let fxBase = $state("USD");
  let fxQuote = $state("");
  let fxYear = $state(new Date().getFullYear());
  let fxMonth = $state(new Date().getMonth() + 1);
  let fxBases = $state<string[]>([]);
  let fxQuotes = $state<string[]>([]);
  let fxRates = $state<
    Array<{ date?: string; base?: string; quote?: string; rate?: number; price?: number; source?: string }>
  >([]);
  let fxLoading = $state(false);

  async function loadFxFiltersAndRates() {
    fxLoading = true;
    try {
      const filters = await apiGet<{ bases?: string[]; quotes?: string[] }>("/api/price/filters").catch(
        () => ({ bases: [] as string[], quotes: [] as string[] })
      );
      fxBases = filters.bases || [];
      fxQuotes = filters.quotes || [];
      await fetchFxRates();
    } finally {
      fxLoading = false;
    }
  }

  async function fetchFxRates() {
    fxLoading = true;
    try {
      const params = new URLSearchParams();
      if (fxBase) params.set("base", fxBase);
      if (fxQuote) params.set("quote", fxQuote);
      params.set("year", String(fxYear));
      params.set("month", String(fxMonth).padStart(2, "0"));
      const res = await apiGet<{
        rates?: typeof fxRates;
        base?: string;
        quote?: string;
      }>(`/api/fx-rates?${params.toString()}`);
      fxRates = res.rates || [];
      if (res.base) fxBase = res.base;
      if (res.quote) fxQuote = res.quote;
    } finally {
      fxLoading = false;
    }
  }

  // ---- Sheets state ----
  let sheets = $state<Array<{ name: string; content: string }>>([]);
  let selectedSheetName = $state("");
  let sheetDraft = $state("");
  let newSheetName = $state("");
  let sheetSaving = $state(false);
  let sheetMessage = $state<string | null>(null);

  async function loadSheets() {
    const res = await apiGet<{ files?: Array<{ name: string; content: string }> }>("/api/sheets/files").catch(
      () => ({ files: [] })
    );
    sheets = res.files || [];
    if (sheets.length > 0 && !selectedSheetName) {
      selectedSheetName = sheets[0].name;
      sheetDraft = sheets[0].content || "";
    }
  }

  function selectSheet(name: string) {
    selectedSheetName = name;
    const found = sheets.find((s) => s.name === name);
    sheetDraft = found?.content || "";
    sheetMessage = null;
  }

  async function saveSheet() {
    if (!selectedSheetName) return;
    sheetSaving = true;
    sheetMessage = null;
    try {
      const res = await fetch("/api/sheets/save", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: selectedSheetName, content: sheetDraft })
      });
      const data = (await res.json()) as { saved?: boolean; message?: string };
      sheetMessage = data.saved ? `Saved ${selectedSheetName}` : data.message || "Save failed";
      if (data.saved) {
        sheets = sheets.map((s) => (s.name === selectedSheetName ? { ...s, content: sheetDraft } : s));
      }
    } finally {
      sheetSaving = false;
    }
  }

  async function createSheet() {
    const clean = newSheetName.trim();
    if (!clean) return;
    const name = clean.endsWith(".paisa") ? clean : `${clean}.paisa`;
    const res = await fetch("/api/sheets/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, content: "", operation: "create" })
    });
    const data = (await res.json()) as { saved?: boolean; message?: string };
    if (data.saved) {
      newSheetName = "";
      await loadSheets();
      selectSheet(name);
    } else {
      sheetMessage = data.message || "Failed to create sheet";
    }
  }

  onMount(() => {
    if (mode === "import") {
      void apiGet<{ presets?: typeof importPresets }>("/api/import/presets")
        .then((r) => (importPresets = r.presets || []))
        .catch(() => {});
    }
    if (mode === "fx") void loadFxFiltersAndRates();
    if (mode === "sheets") void loadSheets();
  });

  $effect(() => {
    if (mode === "fx" && fxRates.length === 0 && !fxLoading) void loadFxFiltersAndRates();
    if (mode === "sheets" && sheets.length === 0) void loadSheets();
  });
</script>

{#if mode === "import"}
  <div class="space-y-4">
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
      <div class="flex flex-wrap items-end gap-3 text-xs">
        <div class="flex-1 min-w-[200px]">
          <label class="block text-[var(--muted)] mb-1" for="imp-account">Source Account</label>
          <input
            id="imp-account"
            type="text"
            list="imp-accounts-list"
            bind:value={importBaseAccount}
            class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          />
          <datalist id="imp-accounts-list">
            {#each runtime.accounts as acc}
              <option value={acc}></option>
            {/each}
          </datalist>
        </div>
        <div class="w-24">
          <label class="block text-[var(--muted)] mb-1" for="imp-delim">Delimiter</label>
          <input
            id="imp-delim"
            type="text"
            bind:value={importDelimiter}
            class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none font-mono"
          />
        </div>
        <div>
          <label class="block text-[var(--muted)] mb-1" for="imp-file">Statement CSV</label>
          <input
            id="imp-file"
            type="file"
            accept=".csv,.txt"
            onchange={handleFileSelect}
            class="text-xs"
          />
        </div>
        <button
          type="button"
          disabled={importLoading || !importContent}
          onclick={parseStatement}
          class="px-4 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
        >
          {importLoading ? "Parsing…" : "Parse Statement"}
        </button>
      </div>

      {#if importPresets.length > 0}
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <span class="text-[var(--muted)]">Presets:</span>
          {#each importPresets as preset}
            <button
              type="button"
              onclick={() => {
                if (preset.delimiter) importDelimiter = preset.delimiter;
              }}
              class="px-2 py-0.5 rounded-md border border-[var(--border)] bg-[var(--surface-elevated)] hover:border-[var(--accent)] cursor-pointer"
            >
              {preset.name || preset.id || "Preset"}
            </button>
          {/each}
        </div>
      {/if}

      {#if importFileName || importMessage}
        <p class="text-xs text-[var(--muted)]">
          {importFileName ? `File: ${importFileName}` : ""} {importMessage ? `· ${importMessage}` : ""}
        </p>
      {/if}
    </div>

    {#if stagedTransactions.length > 0}
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        <div class="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Staged Transactions ({stagedTransactions.length})
          </span>
          {#if !runtime.readonly}
            <button
              type="button"
              disabled={importCommitting}
              onclick={commitImport}
              class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
            >
              {importCommitting ? "Committing…" : "Commit Selected"}
            </button>
          {/if}
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
              <tr>
                <th class="py-2 px-3">Include</th>
                <th class="py-2 px-3">Date</th>
                <th class="py-2 px-3">Payee / Memo</th>
                <th class="py-2 px-3">Category Account</th>
                <th class="py-2 px-3 text-right">Amount</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each stagedTransactions as tx, idx}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-3">
                    <input type="checkbox" bind:checked={includedStaged[idx]} />
                  </td>
                  <td class="py-2 px-3 text-xs text-[var(--muted)]">{tx.date}</td>
                  <td class="py-2 px-3">
                    <div class="font-medium">{tx.payee}</div>
                    {#if tx.memo}
                      <div class="text-xs text-[var(--muted)]">{tx.memo}</div>
                    {/if}
                  </td>
                  <td class="py-2 px-3">
                    <input
                      type="text"
                      list="imp-accounts-list"
                      bind:value={tx.selected_account}
                      class="w-full px-2 py-1 rounded bg-[var(--surface-elevated)] border border-[var(--border)] text-xs outline-none"
                    />
                  </td>
                  <td class="py-2 px-3 text-right font-semibold">
                    {formatMoney(tx.amount || 0)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </div>
{:else if mode === "fx"}
  <div class="space-y-4">
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-end gap-3 text-xs">
      <div>
        <label class="block text-[var(--muted)] mb-1" for="fx-base">Base</label>
        <select
          id="fx-base"
          bind:value={fxBase}
          onchange={fetchFxRates}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          {#each fxBases.length ? fxBases : ["USD", "EUR", "GBP", "INR"] as b}
            <option value={b}>{b}</option>
          {/each}
        </select>
      </div>
      <div>
        <label class="block text-[var(--muted)] mb-1" for="fx-quote">Quote</label>
        <select
          id="fx-quote"
          bind:value={fxQuote}
          onchange={fetchFxRates}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="">Default</option>
          {#each fxQuotes as q}
            <option value={q}>{q}</option>
          {/each}
        </select>
      </div>
      <div>
        <label class="block text-[var(--muted)] mb-1" for="fx-year">Year</label>
        <input
          id="fx-year"
          type="number"
          bind:value={fxYear}
          onchange={fetchFxRates}
          class="w-24 px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
        />
      </div>
      <div>
        <label class="block text-[var(--muted)] mb-1" for="fx-month">Month</label>
        <input
          id="fx-month"
          type="number"
          min="1"
          max="12"
          bind:value={fxMonth}
          onchange={fetchFxRates}
          class="w-20 px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
        />
      </div>
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if fxRates.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">
          {fxLoading ? "Loading FX rates…" : "No FX rates found for this period."}
        </div>
      {:else}
        <table class="w-full text-sm text-left">
          <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
            <tr>
              <th class="py-2.5 px-4 font-medium">Date</th>
              <th class="py-2.5 px-4 font-medium">Pair</th>
              <th class="py-2.5 px-4 text-right font-medium">Rate</th>
              <th class="py-2.5 px-4 font-medium">Source</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each fxRates as r}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2 px-4 text-xs text-[var(--muted)]">
                  {r.date ? formatShortDate(r.date) : "—"}
                </td>
                <td class="py-2 px-4 font-medium">{r.base || fxBase} / {r.quote || fxQuote || runtime.defaultCurrency}</td>
                <td class="py-2 px-4 text-right font-semibold">{(r.rate ?? r.price ?? 0).toFixed(4)}</td>
                <td class="py-2 px-4 text-xs text-[var(--muted)]">{r.source || "—"}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
{:else}
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
    <div class="lg:col-span-3 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Ledger Sheets ({sheets.length})
      </h3>
      {#if !runtime.readonly}
        <div class="flex gap-1.5">
          <input
            type="text"
            bind:value={newSheetName}
            placeholder="new-sheet.paisa"
            class="flex-1 min-w-0 px-2.5 py-1 rounded-md text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          />
          <button
            type="button"
            onclick={createSheet}
            class="px-2.5 py-1 rounded-md bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer"
          >
            New
          </button>
        </div>
      {/if}
      <div class="space-y-1 text-xs">
        {#each sheets as s}
          <button
            type="button"
            onclick={() => selectSheet(s.name)}
            class="w-full text-left px-2.5 py-1.5 rounded-md cursor-pointer truncate {selectedSheetName === s.name
              ? 'bg-[var(--accent)]/15 text-[var(--accent)] font-semibold'
              : 'hover:bg-[var(--surface-elevated)]'}"
          >
            {s.name}
          </button>
        {/each}
      </div>
    </div>

    <div class="lg:col-span-9 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
      <div class="flex items-center justify-between gap-2">
        <span class="text-sm font-semibold">{selectedSheetName || "Select a sheet"}</span>
        {#if !runtime.readonly && selectedSheetName}
          <button
            type="button"
            disabled={sheetSaving}
            onclick={saveSheet}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
          >
            {sheetSaving ? "Saving…" : "Save Sheet"}
          </button>
        {/if}
      </div>
      {#if sheetMessage}
        <p class="text-xs text-[var(--muted)]">{sheetMessage}</p>
      {/if}
      <textarea
        bind:value={sheetDraft}
        readonly={runtime.readonly}
        rows="18"
        class="w-full p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-xs outline-none focus:border-[var(--accent)]"
      ></textarea>
    </div>
  </div>
{/if}
