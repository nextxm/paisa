<script lang="ts">
  import { Code2, Download, RotateCcw, Save } from "lucide-svelte";
  import { authHeaders, runtime, triggerSync, updateConfig } from "$lib/api";
  import { ALL_SECTIONS, CONFIG_GROUPS, DEFAULT_SECTION_ID } from "$lib/config_sections";
  import ConfigSchemaField, { type JsonSchemaNode } from "./ConfigSchemaField.svelte";

  let {
    initialConfig,
    schema,
    accounts = [],
    commodities = [],
    deprecatedKeys = []
  }: {
    initialConfig: Record<string, unknown>;
    schema: JsonSchemaNode;
    accounts?: string[];
    commodities?: string[];
    deprecatedKeys?: string[];
  } = $props();

  let activeSectionId = $state(DEFAULT_SECTION_ID);
  let draft = $state<Record<string, unknown>>({});
  let savedSnapshot = $state<Record<string, unknown>>({});
  let rawMode = $state(false);
  let rawJsonText = $state("");
  let saving = $state(false);
  let message = $state<string | null>(null);
  let error = $state<string | null>(null);

  let togglingDebug = $state(false);
  let includeSecrets = $state(false);
  let downloadingBackup = $state(false);

  function deepClone<T>(v: T): T {
    return JSON.parse(JSON.stringify(v ?? {})) as T;
  }

  $effect(() => {
    if (initialConfig && Object.keys(savedSnapshot).length === 0) {
      draft = deepClone(initialConfig);
      savedSnapshot = deepClone(initialConfig);
      rawJsonText = JSON.stringify(initialConfig, null, 2);
    }
  });

  const activeSection = $derived(
    ALL_SECTIONS.find((s) => s.id === activeSectionId) ?? ALL_SECTIONS[0]
  );

  const hasChanges = $derived(JSON.stringify(draft) !== JSON.stringify(savedSnapshot));

  async function saveChanges() {
    saving = true;
    message = null;
    error = null;
    try {
      const payload = rawMode
        ? (JSON.parse(rawJsonText) as Record<string, unknown>)
        : deepClone(draft);
      const res = await updateConfig(payload);
      if (res.success) {
        draft = deepClone(payload);
        savedSnapshot = deepClone(payload);
        rawJsonText = JSON.stringify(payload, null, 2);
        message = "Configuration saved to paisa.yaml.";
        void triggerSync({ journal: true });
      } else {
        error = res.error || "Failed to save configuration.";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Invalid configuration JSON";
    } finally {
      saving = false;
    }
  }

  function discardChanges() {
    draft = deepClone(savedSnapshot);
    rawJsonText = JSON.stringify(savedSnapshot, null, 2);
    message = null;
    error = null;
  }

  async function resetToDefaults() {
    if (!confirm("Reset configuration to defaults? Only journal_path and db_path will be kept.")) {
      return;
    }
    const minimal: Record<string, unknown> = {
      journal_path: savedSnapshot.journal_path,
      db_path: savedSnapshot.db_path
    };
    saving = true;
    error = null;
    message = null;
    try {
      const res = await updateConfig(minimal);
      if (res.success) {
        draft = deepClone(minimal);
        savedSnapshot = deepClone(minimal);
        rawJsonText = JSON.stringify(minimal, null, 2);
        message = "Configuration reset to defaults.";
      } else {
        error = res.error || "Failed to reset configuration.";
      }
    } finally {
      saving = false;
    }
  }

  async function applyProviderDebug(enabled: boolean) {
    togglingDebug = true;
    error = null;
    try {
      const res = await fetch("/api/config/provider-debug-http", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ enabled })
      });
      if (res.ok) {
        draft = { ...draft, provider_debug_http: enabled };
        savedSnapshot = { ...savedSnapshot, provider_debug_http: enabled };
        message = `Provider HTTP debug logging ${enabled ? "enabled" : "disabled"}.`;
      } else {
        error = "Failed to toggle provider debug logging.";
      }
    } finally {
      togglingDebug = false;
    }
  }

  async function downloadBackup() {
    if (downloadingBackup) return;
    downloadingBackup = true;
    try {
      const url = `/api/backup/export${includeSecrets ? "?include_secrets=true" : ""}`;
      const res = await fetch(url, { headers: authHeaders() });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const disp = res.headers.get("Content-Disposition");
      const m = disp?.match(/filename="?([^"]+)"?/);
      const filename = m?.[1] || "paisa-backup.zip";
      const blob = await res.blob();
      const blobUrl = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = blobUrl;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(blobUrl);
    } catch (e) {
      error = e instanceof Error ? e.message : "Backup download failed";
    } finally {
      downloadingBackup = false;
    }
  }
</script>

<div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
  <!-- Left Section Nav -->
  <aside
    class="lg:col-span-3 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3 space-y-4 h-fit"
  >
    {#each CONFIG_GROUPS as group}
      <div class="space-y-1">
        <div class="px-2.5 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]">
          {group.label}
        </div>
        {#each group.sections as sec}
          <button
            type="button"
            onclick={() => {
              activeSectionId = sec.id;
              rawMode = false;
            }}
            class="w-full text-left px-2.5 py-1.5 rounded-lg text-xs cursor-pointer transition-colors {!rawMode &&
            activeSectionId === sec.id
              ? 'bg-[var(--accent)]/15 text-[var(--accent)] font-semibold'
              : 'text-[var(--text)] hover:bg-[var(--surface-elevated)]'}"
          >
            {sec.label}
          </button>
        {/each}
      </div>
    {/each}
  </aside>

  <!-- Right Form / Raw Editor -->
  <div class="lg:col-span-9 space-y-4">
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div
        class="px-5 py-3.5 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-3"
      >
        <div>
          <h2 class="text-sm font-semibold">
            {rawMode ? "Raw Configuration (JSON / paisa.yaml)" : activeSection.label}
          </h2>
          <p class="text-xs text-[var(--muted)]">
            {rawMode
              ? "Advanced view of the full paisa.yaml configuration object"
              : activeSection.description}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2 text-xs">
          <button
            type="button"
            onclick={() => {
              if (!rawMode) {
                rawJsonText = JSON.stringify(draft, null, 2);
              } else {
                try {
                  draft = JSON.parse(rawJsonText) as Record<string, unknown>;
                } catch {}
              }
              rawMode = !rawMode;
            }}
            class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
          >
            <Code2 class="w-3.5 h-3.5" />
            <span>{rawMode ? "Form View" : "Raw JSON"}</span>
          </button>

          {#if !runtime.readonly}
            {#if hasChanges}
              <button
                type="button"
                onclick={discardChanges}
                class="px-2.5 py-1.5 rounded-lg border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
              >
                Discard
              </button>
            {/if}
            <button
              type="button"
              onclick={resetToDefaults}
              title="Reset to defaults"
              class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-[var(--border)] text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
            >
              <RotateCcw class="w-3.5 h-3.5" />
              <span>Reset</span>
            </button>
            <button
              type="button"
              disabled={saving || (!hasChanges && !rawMode)}
              onclick={saveChanges}
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium disabled:opacity-45 cursor-pointer"
            >
              <Save class="w-3.5 h-3.5" />
              <span>{saving ? "Saving…" : "Save Config"}</span>
            </button>
          {/if}
        </div>
      </div>

      {#if deprecatedKeys.length > 0 && !message}
        <div
          class="px-5 py-2.5 bg-[var(--warning)]/15 text-xs text-[var(--warning)] border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-2"
        >
          <span>
            Removed/deprecated settings in <code>paisa.yaml</code> were ignored on startup:
            <strong class="font-mono">{deprecatedKeys.join(", ")}</strong>. Save the configuration
            to clean up your file.
          </span>
        </div>
      {/if}
      {#if message}
        <div
          class="px-5 py-2 bg-[var(--gain)]/10 text-xs text-[var(--gain)] border-b border-[var(--border)]"
        >
          {message}
        </div>
      {/if}
      {#if error}
        <div
          class="px-5 py-2 bg-[var(--loss)]/10 text-xs text-[var(--loss)] border-b border-[var(--border)]"
        >
          {error}
        </div>
      {/if}

      <div class="p-5 space-y-4">
        {#if rawMode}
          <textarea
            bind:value={rawJsonText}
            readonly={runtime.readonly}
            rows="22"
            spellcheck="false"
            class="w-full p-4 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-xs outline-none focus:border-[var(--accent)]"
          ></textarea>
        {:else if activeSectionId === "backup"}
          <div
            class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/30 p-4 space-y-3 text-xs"
          >
            <div class="font-semibold text-sm">Full System Backup (.zip)</div>
            <p class="text-[var(--muted)]">
              Includes all ledger journal files, <code>paisa.yaml</code>, and sheet templates/data.
              Excludes the SQLite cache DB.
            </p>
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" bind:checked={includeSecrets} />
              <span>Include password hashes in <code>paisa.yaml</code> (default is sanitized)</span>
            </label>
            <button
              type="button"
              disabled={downloadingBackup}
              onclick={downloadBackup}
              class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
            >
              <Download class="w-3.5 h-3.5" />
              <span>{downloadingBackup ? "Preparing…" : "Download Backup (.zip)"}</span>
            </button>
          </div>
        {:else}
          {#if activeSectionId === "advanced" && !runtime.readonly}
            <div
              class="rounded-lg border border-[var(--warning)]/30 bg-[var(--warning)]/10 p-3.5 flex flex-wrap items-center justify-between gap-3 text-xs"
            >
              <div>
                <div class="font-semibold">Provider HTTP debug logging</div>
                <div class="text-[var(--muted)]">
                  Toggle request/response logging immediately without saving the full form.
                </div>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  disabled={togglingDebug || !draft.provider_debug_http}
                  onclick={() => applyProviderDebug(false)}
                  class="px-2.5 py-1 rounded border border-[var(--border)] bg-[var(--surface)] disabled:opacity-45 cursor-pointer"
                >
                  Disable
                </button>
                <button
                  type="button"
                  disabled={togglingDebug || Boolean(draft.provider_debug_http)}
                  onclick={() => applyProviderDebug(true)}
                  class="px-2.5 py-1 rounded bg-[var(--warning)] text-black font-medium disabled:opacity-45 cursor-pointer"
                >
                  Enable
                </button>
              </div>
            </div>
          {/if}

          {#key activeSectionId}
            {#each activeSection.schemaKeys as key (key)}
              {#if schema?.properties?.[key]}
                <ConfigSchemaField
                  fieldKey={key}
                  bind:value={draft[key]}
                  schema={schema.properties[key]}
                  {accounts}
                  {commodities}
                  required={(schema.required || []).includes(key)}
                  readonly={runtime.readonly}
                />
              {/if}
            {/each}
          {/key}
        {/if}
      </div>
    </div>
  </div>
</div>
