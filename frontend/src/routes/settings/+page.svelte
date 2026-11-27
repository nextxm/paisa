<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { apiGet, authHeaders, getFullConfig, runtime, updateConfig } from "$lib/api";

  type SettingsTab = "config" | "logs" | "about" | "firefly";

  interface LogEntry {
    time?: string;
    level?: string;
    message?: string;
  }

  let activeTab = $state<SettingsTab>("config");
  let loading = $state(true);
  let configJson = $state<Record<string, unknown> | null>(null);
  let configDraft = $state("");
  let configError = $state<string | null>(null);
  let configSaving = $state(false);
  let configMessage = $state<string | null>(null);
  let logs = $state<LogEntry[]>([]);
  let fireflyItems = $state<Array<Record<string, unknown>>>([]);

  onMount(async () => {
    try {
      const [cfg, logRes, ffRes] = await Promise.all([
        getFullConfig().catch((e) => {
          configError = e instanceof Error ? e.message : "Failed to load configuration";
          return null;
        }),
        apiGet<{ logs?: LogEntry[] }>("/api/logs").catch(() => ({ logs: [] })),
        apiGet<{ items?: Array<Record<string, unknown>> }>("/api/firefly/reconcile").catch(() => ({
          items: []
        }))
      ]);
      configJson = cfg;
      configDraft = cfg ? JSON.stringify(cfg, null, 2) : "";
      logs = logRes.logs || [];
      fireflyItems = ffRes.items || [];
    } finally {
      loading = false;
    }
  });
</script>

<div class="space-y-5">
  <div>
    <h1 class="text-xl font-semibold tracking-tight">Settings</h1>
    <p class="text-xs text-[var(--muted)] mt-0.5">
      Configuration, backups, system logs, integrations, and application details
    </p>
  </div>

  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "config", label: "Configuration" },
      { id: "logs", label: `System Logs (${logs.length})` },
      { id: "firefly", label: `Firefly (${fireflyItems.length})` },
      { id: "about", label: "About" }
    ] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 whitespace-nowrap cursor-pointer {activeTab === tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)]'}"
        onclick={() => (activeTab = tab.id as SettingsTab)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"></div>
  {:else if activeTab === "config"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Active Configuration (`paisa.yaml`)
        </h3>
        {#if !runtime.readonly && configJson}
          <button
            type="button"
            disabled={configSaving}
            onclick={async () => {
              configSaving = true;
              configMessage = null;
              try {
                const parsed = JSON.parse(configDraft) as Record<string, unknown>;
                const res = await updateConfig(parsed);
                if (res.success) {
                  configJson = parsed;
                  configMessage = "Configuration saved.";
                } else {
                  configMessage = res.error || "Failed to save configuration.";
                }
              } catch (e) {
                configMessage = e instanceof Error ? e.message : "Invalid JSON";
              } finally {
                configSaving = false;
              }
            }}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
          >
            {configSaving ? "Saving…" : "Save Config"}
          </button>
        {/if}
      </div>
      {#if configMessage}
        <p class="text-xs text-[var(--muted)]">{configMessage}</p>
      {/if}
      {#if configJson}
        <textarea
          bind:value={configDraft}
          readonly={runtime.readonly}
          rows="20"
          class="w-full p-4 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-xs outline-none focus:border-[var(--accent)]"
        ></textarea>
      {:else}
        <p class="text-sm text-[var(--muted)]">
          Configuration unavailable{configError ? `: ${configError}` : "."}
        </p>
      {/if}
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Backup &amp; Export
      </h3>
      <p class="text-xs text-[var(--muted)]">
        Download a ZIP archive of your ledger files and configuration.
      </p>
      <button
        type="button"
        onclick={async () => {
          const res = await fetch("/api/backup/export", { headers: authHeaders() });
          if (!res.ok) return;
          const blob = await res.blob();
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = "paisa-backup.zip";
          a.click();
          URL.revokeObjectURL(url);
        }}
        class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer"
      >
        Export Backup (.zip)
      </button>
    </div>
  {:else if activeTab === "logs"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if logs.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">No system logs available.</div>
      {:else}
        <div class="divide-y divide-[var(--border)] font-mono text-xs">
          {#each logs.slice(0, 150) as log}
            <div class="px-4 py-2 flex items-start gap-3">
              <span class="text-[var(--muted)] shrink-0">
                {log.time ? dayjs(log.time).format("HH:mm:ss") : ""}
              </span>
              <span class="uppercase font-semibold shrink-0 w-14 text-[var(--muted)]">
                {log.level || "info"}
              </span>
              <span class="break-all">{log.message || ""}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else if activeTab === "firefly"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
        Firefly III Reconciliation ({fireflyItems.length})
      </h3>
      {#if fireflyItems.length === 0}
        <p class="text-sm text-[var(--muted)]">No Firefly reconciliation items configured or pending.</p>
      {:else}
        <pre class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto max-h-96">{JSON.stringify(fireflyItems, null, 2)}</pre>
      {/if}
    </div>
  {:else}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-6 space-y-4">
      <div class="space-y-1">
        <h3 class="text-base font-semibold">Paisa Personal Finance Manager</h3>
        <p class="text-sm text-[var(--muted)]">
          Double-entry plain-text accounting powered by ledger-cli, hledger, beancount, Go, SQLite, and SvelteKit.
        </p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs pt-2 border-t border-[var(--border)]">
        <div>
          <div class="text-[var(--muted)]">Default Currency</div>
          <div class="font-semibold mt-0.5">{runtime.defaultCurrency}</div>
        </div>
        <div>
          <div class="text-[var(--muted)]">Locale</div>
          <div class="font-semibold mt-0.5">{runtime.locale}</div>
        </div>
        <div>
          <div class="text-[var(--muted)]">Accounts Indexed</div>
          <div class="font-semibold mt-0.5 tabular-nums">{runtime.accounts.length}</div>
        </div>
      </div>
    </div>
  {/if}
</div>
