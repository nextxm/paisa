<script lang="ts">
  import { onMount } from "svelte";
  import { Save, ShieldCheck, Trash2 } from "lucide-svelte";
  import { apiGet, authHeaders, runtime, triggerSync } from "$lib/api";

  interface LedgerFile {
    file_name: string;
    content: string;
  }

  let files = $state<LedgerFile[]>([]);
  let selectedFileName = $state("");
  let content = $state("");
  let loading = $state(true);
  let saving = $state(false);
  let status = $state<string | null>(null);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      const res = await apiGet<{ files?: LedgerFile[] }>("/api/editor/files");
      files = res.files || [];
      if (files.length > 0) {
        selectedFileName = files[0].file_name;
        content = files[0].content || "";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load ledger files";
    } finally {
      loading = false;
    }
  });

  function selectFile(name: string) {
    selectedFileName = name;
    const found = files.find((f) => f.file_name === name);
    content = found?.content || "";
    status = null;
  }

  async function saveFile() {
    if (!selectedFileName) return;
    saving = true;
    status = null;
    error = null;
    try {
      const res = await fetch("/api/editor/save", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ file_name: selectedFileName, content })
      });
      const data = (await res.json()) as { saved?: boolean; message?: string };
      if (!res.ok || data.saved === false) {
        error = data.message || "Failed to save ledger file";
        return;
      }
      files = files.map((f) => (f.file_name === selectedFileName ? { ...f, content } : f));
      await triggerSync({ journal: true });
      status = "Saved and synced.";
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to save ledger file";
    } finally {
      saving = false;
    }
  }

  async function validateFile() {
    if (!selectedFileName) return;
    status = null;
    error = null;
    const res = await fetch("/api/editor/validate", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: selectedFileName, content })
    });
    const data = (await res.json()) as { errors?: Array<{ line?: number; message?: string }>; output?: string };
    if (data.errors && data.errors.length > 0) {
      error = data.errors.map((e) => `Line ${e.line ?? "?"}: ${e.message ?? "Syntax error"}`).join(" · ");
    } else {
      status = "Ledger syntax valid.";
    }
  }

  async function deleteBackups() {
    if (!selectedFileName) return;
    const res = await fetch("/api/editor/file/delete_backups", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: selectedFileName })
    });
    if (res.ok) status = "Backups deleted.";
  }
</script>

{#if loading}
  <div class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"></div>
{:else if files.length === 0}
  <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]">
    No ledger files found.
  </div>
{:else}
  <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
    <div class="px-4 py-3 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <select
          value={selectedFileName}
          onchange={(e) => selectFile((e.currentTarget as HTMLSelectElement).value)}
          aria-label="Select ledger file"
          class="px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
        >
          {#each files as f}
            <option value={f.file_name}>{f.file_name}</option>
          {/each}
        </select>
        {#if status}
          <span class="text-xs text-[var(--gain)]">{status}</span>
        {/if}
      </div>

      {#if !runtime.readonly}
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
            onclick={validateFile}
          >
            <ShieldCheck class="w-3.5 h-3.5" />
            <span>Validate</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:border-[var(--loss)] text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
            onclick={deleteBackups}
          >
            <Trash2 class="w-3.5 h-3.5" />
            <span>Clear Backups</span>
          </button>
          <button
            type="button"
            disabled={saving}
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
            onclick={saveFile}
          >
            <Save class="w-3.5 h-3.5" />
            <span>{saving ? "Saving…" : "Save & Sync"}</span>
          </button>
        </div>
      {/if}
    </div>

    {#if error}
      <div class="px-4 py-2 bg-[var(--loss)]/10 text-xs text-[var(--loss)]">{error}</div>
    {/if}

    <textarea
      bind:value={content}
      readonly={runtime.readonly}
      rows="20"
      spellcheck="false"
      class="w-full p-4 bg-transparent font-mono text-xs leading-relaxed outline-none resize-y"
    ></textarea>
  </div>
{/if}
