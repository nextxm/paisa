<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { ExternalLink } from "lucide-svelte";
  import { apiGet, runtime } from "$lib/api";
  import Logo from "$lib/components/Logo.svelte";
  import ConfigPanel from "$lib/components/ConfigPanel.svelte";
  import type { JsonSchemaNode } from "$lib/components/ConfigSchemaField.svelte";

  type SettingsTab = "config" | "logs" | "about";

  interface LogEntry {
    time?: string;
    level?: string;
    message?: string;
  }

  let activeTab = $state<SettingsTab>("config");
  let loading = $state(true);
  let configJson = $state<Record<string, unknown> | null>(null);
  let configSchema = $state<JsonSchemaNode | null>(null);
  let configAccounts = $state<string[]>([]);
  let configError = $state<string | null>(null);
  let logs = $state<LogEntry[]>([]);

  const buildInfo = __BUILD_INFO__;

  onMount(async () => {
    try {
      const [cfgRes, logRes] = await Promise.all([
        apiGet<{
          config?: Record<string, unknown>;
          schema?: JsonSchemaNode;
          accounts?: string[];
        }>("/api/config").catch((e) => {
          configError = e instanceof Error ? e.message : "Failed to load configuration";
          return null;
        }),
        apiGet<{ logs?: LogEntry[] }>("/api/logs").catch(() => ({ logs: [] }))
      ]);
      configJson = cfgRes?.config || null;
      configSchema = cfgRes?.schema || null;
      configAccounts = cfgRes?.accounts || [];
      logs = logRes.logs || [];
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
    {#if configJson && configSchema}
      <ConfigPanel initialConfig={configJson} schema={configSchema} accounts={configAccounts} />
    {:else}
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5">
        <p class="text-sm text-[var(--muted)]">
          Configuration unavailable{configError ? `: ${configError}` : "."}
        </p>
      </div>
    {/if}
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
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-6 space-y-4">
        <div class="flex items-center gap-3">
          <Logo size={48} />
          <div>
            <h3 class="text-base font-semibold">Paisa Personal Finance Manager</h3>
            <p class="text-xs text-[var(--muted)]">
              Double-entry plain-text accounting powered by ledger-cli, hledger, beancount, Go, SQLite, and SvelteKit.
            </p>
          </div>
        </div>
        <p class="text-xs text-[var(--muted)]">
          Forked from <code>ananthakumaran/paisa</code>, maintained at <code>nextxm/paisa</code> under GNU AGPL-3.0-or-later.
        </p>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs pt-3 border-t border-[var(--border)]">
          <div>
            <div class="text-[var(--muted)]">Version</div>
            <div class="font-semibold mt-0.5 font-mono">{buildInfo.version || "dev"}</div>
          </div>
          <div>
            <div class="text-[var(--muted)]">Build</div>
            <div class="font-semibold mt-0.5 font-mono">{buildInfo.buildDate.substring(0, 10)}</div>
          </div>
          <div>
            <div class="text-[var(--muted)]">Branch</div>
            <div class="font-semibold mt-0.5 font-mono">
              {buildInfo.branch || "—"}{buildInfo.commitHash ? ` (${buildInfo.commitHash})` : ""}
            </div>
          </div>
          {#if buildInfo.tag}
            <div>
              <div class="text-[var(--muted)]">Tag</div>
              <div class="font-semibold mt-0.5 font-mono">{buildInfo.tag}</div>
            </div>
          {/if}
        </div>
        <div class="grid grid-cols-3 gap-4 text-xs pt-3 border-t border-[var(--border)]">
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

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-6 space-y-3">
        <h3 class="text-sm font-semibold">Links</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
          {#each [
            { name: "Issue", href: "https://github.com/nextxm/paisa/issues" },
            { name: "Discussions", href: "https://github.com/nextxm/paisa/discussions" },
            { name: "Source Code", href: "https://github.com/nextxm/paisa" },
            { name: "Upstream Project", href: "https://github.com/ananthakumaran/paisa" },
            { name: "License (AGPL-3.0-or-later)", href: "https://www.gnu.org/licenses/agpl-3.0.html" },
            { name: "Documentation", href: "https://nextxm.github.io/paisa/" },
            { name: "Releases", href: "https://github.com/nextxm/paisa/releases" }
          ] as link}
            <a
              href={link.href}
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center justify-between px-3 py-2 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] hover:text-[var(--accent)] transition-colors"
            >
              <span>{link.name}</span>
              <ExternalLink class="w-3.5 h-3.5 text-[var(--muted)]" />
            </a>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>
