<script lang="ts">
  import { onMount } from "svelte";
  import { apiGet, authHeaders, formatShortDate, runtime } from "$lib/api";
  import LedgerImportPanel from "./LedgerImportPanel.svelte";

  let {
    mode
  }: {
    mode: "import" | "fx" | "sheets";
  } = $props();

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
  let sheets = $state<Array<{ name: string; content: string; versions?: string[] }>>([]);
  let selectedSheetName = $state("");
  let selectedSheetVersion = $state("");
  const activeSheetObj = $derived(sheets.find((s) => s.name === selectedSheetName));
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
    if (mode === "fx") void loadFxFiltersAndRates();
    if (mode === "sheets") void loadSheets();
  });

  $effect(() => {
    if (mode === "fx" && fxRates.length === 0 && !fxLoading) void loadFxFiltersAndRates();
    if (mode === "sheets" && sheets.length === 0) void loadSheets();
  });
</script>

{#if mode === "import"}
  <LedgerImportPanel />
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
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="text-sm font-semibold">{selectedSheetName || "Select a sheet"}</span>
        {#if !runtime.readonly && selectedSheetName}
          <div class="flex flex-wrap items-center gap-2">
            {#if activeSheetObj?.versions?.length}
              <select
                bind:value={selectedSheetVersion}
                class="px-2 py-1 rounded-md text-xs bg-[var(--surface-elevated)] border border-[var(--border)]"
              >
                <option value="">Version…</option>
                {#each activeSheetObj.versions as v}
                  <option value={v}>{v}</option>
                {/each}
              </select>
              <button
                type="button"
                disabled={!selectedSheetVersion}
                onclick={async () => {
                  if (!selectedSheetVersion) return;
                  const res = await fetch("/api/sheets/file", {
                    method: "POST",
                    headers: { "Content-Type": "application/json", ...authHeaders() },
                    body: JSON.stringify({ name: selectedSheetName, version: selectedSheetVersion })
                  });
                  const d = (await res.json()) as { file?: { content?: string } };
                  if (d.file?.content !== undefined) {
                    sheetDraft = d.file.content;
                    sheetMessage = `Loaded version ${selectedSheetVersion}`;
                  }
                }}
                class="px-2.5 py-1 rounded-lg border border-[var(--border)] text-xs cursor-pointer disabled:opacity-50"
              >
                Revert
              </button>
            {/if}
            <button
              type="button"
              onclick={async () => {
                await fetch("/api/sheets/file/delete_backups", {
                  method: "POST",
                  headers: { "Content-Type": "application/json", ...authHeaders() },
                  body: JSON.stringify({ name: selectedSheetName })
                });
                sheetMessage = "Sheet backups cleared.";
              }}
              class="px-2.5 py-1 rounded-lg border border-[var(--border)] text-xs text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
            >
              Clear Backups
            </button>
            <button
              type="button"
              disabled={sheetSaving}
              onclick={saveSheet}
              class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
            >
              {sheetSaving ? "Saving…" : "Save Sheet"}
            </button>
          </div>
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
