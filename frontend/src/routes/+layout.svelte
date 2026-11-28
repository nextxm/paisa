<script lang="ts">
  import "../app.css";
  import { page } from "$app/stores";
  import { goto, invalidateAll } from "$app/navigation";
  import {
    Activity,
    ArrowLeftRight,
    BookOpen,
    Coins,
    Compass,
    Eye,
    EyeOff,
    Home,
    Landmark,
    LogOut,
    Moon,
    PanelLeftClose,
    PanelLeftOpen,
    Plus,
    RefreshCw,
    Search,
    Settings,
    Sun,
    TrendingUp
  } from "lucide-svelte";
  import Logo from "$lib/components/Logo.svelte";
  import QuickAddModal from "$lib/components/QuickAddModal.svelte";
  import { base } from "$app/paths";
  import {
    appPath,
    authHeaders,
    getTheme,
    loadRuntimeConfig,
    logout,
    runtime,
    toggleObscure,
    toggleTheme,
    triggerSync
  } from "$lib/api";
  import { onMount, tick, type Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  let sidebarCollapsed = $state(false);
  let cmdOpen = $state(false);
  let cmdQuery = $state("");
  let cmdInput = $state<HTMLInputElement | null>(null);
  let quickAddOpen = $state(false);
  let theme = $state<"light" | "dark">("light");
  let obscure = $state(false);
  let syncing = $state(false);
  let syncMenuOpen = $state(false);
  let jobsMenuOpen = $state(false);
  interface BackgroundJob {
    id: string;
    name?: string;
    status?: string;
    message?: string;
    error?: string;
    items_completed?: number;
    total_items?: number;
    created_at?: string;
    metadata?: {
      journal?: boolean;
      prices?: boolean;
      force_prices?: boolean;
      portfolios?: boolean;
    };
  }

  let jobs = $state<BackgroundJob[]>([]);
  let hasToken = $state(false);
  let renderKey = $state(0);

  interface PaletteCommand {
    label: string;
    href: string;
    icon: typeof Home;
    parent?: string;
    keywords?: string;
  }

  const navItems = [
    { label: "Home", href: "/", icon: Home },
    { label: "Transactions", href: "/transactions", icon: ArrowLeftRight },
    { label: "Cash Flow", href: "/cash-flow", icon: TrendingUp },
    { label: "Wealth", href: "/wealth", icon: Landmark },
    { label: "Planning", href: "/planning", icon: Compass },
    { label: "Ledger", href: "/ledger", icon: BookOpen },
    { label: "Market Data", href: "/market-data", icon: Coins },
    { label: "Settings", href: "/settings", icon: Settings }
  ];

  const tabItems: PaletteCommand[] = [
    { label: "Cash Flow", parent: "Cash Flow", href: "/cash-flow?tab=cashflow", icon: TrendingUp },
    {
      label: "Income Statement",
      parent: "Cash Flow",
      href: "/cash-flow?tab=statement",
      icon: TrendingUp
    },
    {
      label: "Money Flow",
      parent: "Cash Flow",
      href: "/cash-flow?tab=sankey",
      icon: TrendingUp,
      keywords: "sankey"
    },
    {
      label: "Income Timeline",
      parent: "Cash Flow",
      href: "/cash-flow?tab=income",
      icon: TrendingUp
    },
    {
      label: "Expenses",
      parent: "Cash Flow",
      href: "/cash-flow?tab=expenses",
      icon: TrendingUp,
      keywords: "spending heatmap trends"
    },
    { label: "Budget", parent: "Cash Flow", href: "/cash-flow?tab=budget", icon: TrendingUp },
    { label: "Recurring", parent: "Cash Flow", href: "/cash-flow?tab=recurring", icon: TrendingUp },
    {
      label: "Overview",
      parent: "Wealth",
      href: "/wealth?tab=overview",
      icon: Landmark,
      keywords: "net worth timeline"
    },
    { label: "Assets", parent: "Wealth", href: "/wealth?tab=assets", icon: Landmark },
    {
      label: "Investments",
      parent: "Wealth",
      href: "/wealth?tab=investments",
      icon: Landmark,
      keywords: "xirr holdings"
    },
    {
      label: "Allocation",
      parent: "Wealth",
      href: "/wealth?tab=allocation",
      icon: Landmark,
      keywords: "rebalance target"
    },
    {
      label: "Portfolio",
      parent: "Wealth",
      href: "/wealth?tab=portfolio",
      icon: Landmark,
      keywords: "securities fund industry"
    },
    {
      label: "Inv. Income",
      parent: "Wealth",
      href: "/wealth?tab=income",
      icon: Landmark,
      keywords: "investment income dividends interest"
    },
    {
      label: "Debt & Cards",
      parent: "Wealth",
      href: "/wealth?tab=liabilities",
      icon: Landmark,
      keywords: "liabilities credit cards loans"
    },
    {
      label: "Projection",
      parent: "Planning",
      href: "/planning?tab=projection",
      icon: Compass,
      keywords: "fire monte carlo"
    },
    { label: "Goals", parent: "Planning", href: "/planning?tab=goals", icon: Compass },
    {
      label: "What-If",
      parent: "Planning",
      href: "/planning?tab=whatif",
      icon: Compass,
      keywords: "scenarios"
    },
    {
      label: "Drawdown",
      parent: "Planning",
      href: "/planning?tab=drawdown",
      icon: Compass,
      keywords: "retirement withdrawal"
    },
    { label: "Tax", parent: "Planning", href: "/planning?tab=tax", icon: Compass },
    {
      label: "Import",
      parent: "Ledger",
      href: "/ledger?tab=import",
      icon: BookOpen,
      keywords: "csv"
    },
    {
      label: "Journal Editor",
      parent: "Ledger",
      href: "/ledger?tab=editor",
      icon: BookOpen,
      keywords: "edit"
    },
    {
      label: "Sheets",
      parent: "Ledger",
      href: "/ledger?tab=sheets",
      icon: BookOpen,
      keywords: "google"
    },
    {
      label: "Data Health",
      parent: "Ledger",
      href: "/ledger?tab=health",
      icon: BookOpen,
      keywords: "doctor findings duplicates"
    },
    {
      label: "Commodity Prices",
      parent: "Market Data",
      href: "/market-data?tab=prices",
      icon: Coins,
      keywords: "nav stock"
    },
    {
      label: "FX Rates",
      parent: "Market Data",
      href: "/market-data?tab=fx",
      icon: Coins,
      keywords: "currency exchange"
    },
    {
      label: "Configuration",
      parent: "Settings",
      href: "/settings?tab=config",
      icon: Settings,
      keywords: "config preferences"
    },
    { label: "Logs", parent: "Settings", href: "/settings?tab=logs", icon: Settings },
    {
      label: "About",
      parent: "Settings",
      href: "/settings?tab=about",
      icon: Settings,
      keywords: "version build"
    }
  ];

  const pathname = $derived($page.url.pathname);
  const buildInfo = __BUILD_INFO__;
  const hasRunningJob = $derived(
    jobs.some((j) => j.status === "running" || j.status === "pending")
  );
  const isSyncActive = $derived(syncing || hasRunningJob);
  const filteredCommands = $derived(
    navItems.filter((item) => item.label.toLowerCase().includes(cmdQuery.trim().toLowerCase()))
  );
  const filteredTabs = $derived.by(() => {
    const q = cmdQuery.trim().toLowerCase();
    if (!q) return tabItems;
    return tabItems.filter((item) =>
      `${item.parent ?? ""} ${item.label} ${item.keywords ?? ""}`.toLowerCase().includes(q)
    );
  });
  const filteredAccounts = $derived(
    cmdQuery.trim().length > 1
      ? runtime.accounts
          .filter((a) => a.toLowerCase().includes(cmdQuery.trim().toLowerCase()))
          .slice(0, 8)
      : []
  );

  onMount(() => {
    theme = getTheme();
    try {
      hasToken = Boolean(localStorage.getItem("token"));
    } catch {}
    void loadRuntimeConfig().then((state) => {
      obscure = state.obscure;
      renderKey += 1;
    });

    let es: EventSource | null = null;
    const handleJobMessage = (ev: MessageEvent) => {
      try {
        const parsed = JSON.parse(ev.data) as { jobs?: BackgroundJob[] } | BackgroundJob;
        if (Array.isArray((parsed as { jobs?: BackgroundJob[] }).jobs)) {
          jobs = (parsed as { jobs: BackgroundJob[] }).jobs;
        } else if ((parsed as BackgroundJob).id) {
          const job = parsed as BackgroundJob;
          const idx = jobs.findIndex((j) => j.id === job.id);
          const next = idx >= 0 ? jobs.map((j, i) => (i === idx ? job : j)) : [job, ...jobs];
          next.sort((a, b) => (b.created_at || "").localeCompare(a.created_at || ""));
          jobs = next.slice(0, 20);
        }
      } catch {}
    };
    try {
      es = new EventSource("/api/jobs/stream");
      es.addEventListener("job", handleJobMessage);
      es.onmessage = handleJobMessage;
    } catch {}
    return () => {
      es?.close();
    };
  });

  function openCommandPalette() {
    cmdOpen = true;
    cmdQuery = "";
    void tick().then(() => cmdInput?.focus());
  }

  function handleKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      if (cmdOpen) {
        cmdOpen = false;
      } else {
        openCommandPalette();
      }
    } else if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === "a") {
      e.preventDefault();
      quickAddOpen = true;
    } else if (e.key === "Escape" && cmdOpen) {
      cmdOpen = false;
    }
  }

  async function handleSync(opts: { journal?: boolean; prices?: boolean; portfolios?: boolean }) {
    syncMenuOpen = false;
    syncing = true;
    try {
      await triggerSync(opts);
      await invalidateAll();
      renderKey += 1;
    } finally {
      syncing = false;
    }
  }

  function handleObscure() {
    obscure = toggleObscure();
    renderKey += 1;
  }

  function jobLabel(job: BackgroundJob): string {
    if (job.name) return job.name;
    const parts: string[] = [];
    if (job.metadata?.journal) parts.push("Journal");
    if (job.metadata?.prices) parts.push("Prices");
    if (job.metadata?.portfolios) parts.push("Portfolios");
    if (parts.length > 0) return `Sync: ${parts.join(", ")}`;
    return `Job ${job.id.slice(0, 8)}`;
  }

  function isActive(href: string, current: string): boolean {
    const clean = base && current.startsWith(base) ? current.slice(base.length) || "/" : current;
    if (href === "/") return clean === "/";
    return clean.startsWith(href);
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="min-h-screen flex bg-[var(--bg)] text-[var(--text)]">
  <!-- Desktop Left Rail -->
  <aside
    class="hidden md:flex flex-col shrink-0 border-r border-[var(--border)] bg-[var(--surface)] transition-all duration-150 {sidebarCollapsed
      ? 'w-16'
      : 'w-56'}"
  >
    <div class="h-13 px-3 flex items-center justify-between gap-1 border-b border-[var(--border)]">
      {#if !sidebarCollapsed}
        <div class="flex items-center gap-1.5 min-w-0">
          <a
            href={appPath("/")}
            class="flex items-center gap-2 font-semibold text-base tracking-tight shrink-0"
          >
            <Logo size={22} animation={isSyncActive} />
            <span>Paisa</span>
          </a>
          <a
            href={appPath("/settings?tab=about")}
            title="Paisa {buildInfo.version || 'v2'} ({isSyncActive
              ? 'Syncing…'
              : 'Synced'}) — View About"
            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-mono bg-[var(--surface-elevated)] border border-[var(--border)] hover:border-[var(--accent)] text-[var(--muted)] hover:text-[var(--text)] transition-colors shrink-0"
          >
            <span
              class="w-1.5 h-1.5 rounded-full {isSyncActive
                ? 'bg-[var(--warning)] animate-pulse'
                : 'bg-[var(--gain)]'}"
            ></span>
            <span>v2 · {isSyncActive ? "Syncing" : "Synced"}</span>
          </a>
        </div>
      {/if}
      <button
        type="button"
        class="p-1.5 rounded-md text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer"
        onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
        aria-label="Toggle sidebar"
      >
        {#if sidebarCollapsed}
          <PanelLeftOpen class="w-4 h-4" />
        {:else}
          <PanelLeftClose class="w-4 h-4" />
        {/if}
      </button>
    </div>

    <nav class="flex-1 p-2.5 space-y-1" aria-label="Primary">
      {#each navItems as item}
        {@const Icon = item.icon}
        {@const active = isActive(item.href, pathname)}
        <a
          href={appPath(item.href)}
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors {active
            ? 'bg-[var(--surface-elevated)] text-[var(--text)]'
            : 'text-[var(--muted)] hover:bg-[var(--surface-elevated)]/60 hover:text-[var(--text)]'}"
          title={sidebarCollapsed ? item.label : undefined}
        >
          <Icon class="w-4 h-4 shrink-0" />
          {#if !sidebarCollapsed}
            <span>{item.label}</span>
          {/if}
        </a>
      {/each}
    </nav>
  </aside>

  <!-- Main Column -->
  <div class="flex-1 min-w-0 flex flex-col pb-16 md:pb-0">
    <!-- Top Context Bar -->
    <header
      class="sticky top-0 z-20 h-13 px-4 sm:px-6 bg-[var(--surface)] border-b border-[var(--border)] flex items-center justify-between gap-3"
    >
      <div class="flex items-center gap-3">
        <a href={appPath("/")} class="md:hidden flex items-center gap-2 font-semibold text-base">
          <Logo size={22} />
          <span>Paisa</span>
        </a>

        <button
          type="button"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs text-[var(--muted)] bg-[var(--surface-elevated)] border border-[var(--border)] hover:text-[var(--text)] cursor-pointer"
          onclick={openCommandPalette}
        >
          <Search class="w-3.5 h-3.5" />
          <span class="hidden sm:inline">Search or jump to…</span>
          <kbd
            class="hidden sm:inline px-1.5 py-0.5 text-[10px] rounded bg-[var(--surface)] border border-[var(--border)]"
          >
            Ctrl K
          </kbd>
        </button>
      </div>

      <div class="flex items-center gap-1.5">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 cursor-pointer"
          onclick={() => (quickAddOpen = true)}
          title="Quick Add Transaction (Ctrl+Shift+A)"
        >
          <Plus class="w-3.5 h-3.5" />
          <span class="hidden sm:inline">Quick Add</span>
        </button>

        <div class="relative">
          <button
            type="button"
            class="p-2 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer"
            onclick={() => (syncMenuOpen = !syncMenuOpen)}
            aria-label="Sync options"
            title="Sync"
          >
            <RefreshCw class="w-4 h-4 {syncing ? 'animate-spin' : ''}" />
          </button>
          {#if syncMenuOpen}
            <div
              class="absolute right-0 mt-1 w-44 rounded-lg bg-[var(--surface)] border border-[var(--border)] shadow-lg p-1 text-xs z-30"
            >
              <button
                type="button"
                class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--surface-elevated)] cursor-pointer"
                onclick={() => handleSync({ journal: true })}
              >
                Update Journal
              </button>
              <button
                type="button"
                class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--surface-elevated)] cursor-pointer"
                onclick={() => handleSync({ prices: true })}
              >
                Refresh Prices
              </button>
              <button
                type="button"
                class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--surface-elevated)] cursor-pointer"
                onclick={() => handleSync({ portfolios: true })}
              >
                Refresh Portfolios
              </button>
            </div>
          {/if}
        </div>

        <button
          type="button"
          class="p-2 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer"
          onclick={handleObscure}
          aria-label="Toggle privacy mode"
          title={obscure ? "Show numbers" : "Hide numbers"}
        >
          {#if obscure}
            <EyeOff class="w-4 h-4" />
          {:else}
            <Eye class="w-4 h-4" />
          {/if}
        </button>

        <button
          type="button"
          class="p-2 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer"
          onclick={() => (theme = toggleTheme())}
          aria-label="Toggle theme"
        >
          {#if theme === "dark"}
            <Sun class="w-4 h-4" />
          {:else}
            <Moon class="w-4 h-4" />
          {/if}
        </button>

        <!-- Jobs Monitor -->
        <div class="relative">
          <button
            type="button"
            class="p-2 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer relative"
            onclick={() => (jobsMenuOpen = !jobsMenuOpen)}
            aria-label="Background jobs"
            title="Background jobs"
          >
            <Activity class="w-4 h-4" />
            {#if jobs.some((j) => j.status === "running" || j.status === "pending")}
              <span
                class="w-2 h-2 rounded-full bg-[var(--accent)] absolute top-1.5 right-1.5 animate-pulse"
              ></span>
            {/if}
          </button>
          {#if jobsMenuOpen}
            <div
              class="absolute right-0 mt-1 w-72 rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-lg p-3 z-40 text-xs space-y-2"
            >
              <div class="flex items-center justify-between">
                <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">
                  Background Jobs ({jobs.length})
                </span>
                {#if jobs.length > 0}
                  <button
                    type="button"
                    class="text-[var(--accent)] hover:underline cursor-pointer"
                    onclick={async () => {
                      await fetch("/api/jobs/clear", {
                        method: "POST",
                        headers: authHeaders()
                      }).catch(() => {});
                      jobs = [];
                    }}
                  >
                    Clear
                  </button>
                {/if}
              </div>
              {#if jobs.length === 0}
                <p class="text-[var(--muted)]">No active or recent jobs.</p>
              {:else}
                <div class="max-h-48 overflow-y-auto divide-y divide-[var(--border)]">
                  {#each jobs.slice(0, 8) as j}
                    <div class="py-1.5 space-y-0.5">
                      <div class="flex items-center justify-between gap-2">
                        <span class="truncate font-medium">{jobLabel(j)}</span>
                        <span
                          class="capitalize shrink-0 {j.status === 'failed'
                            ? 'text-[var(--loss)]'
                            : j.status === 'completed'
                              ? 'text-[var(--gain)]'
                              : j.status === 'running'
                                ? 'text-[var(--warning)]'
                                : 'text-[var(--muted)]'}"
                        >
                          {j.status || "done"}
                          {#if j.status === "running" && j.total_items && j.total_items > 0}
                            ({j.items_completed ?? 0}/{j.total_items})
                          {/if}
                        </span>
                      </div>
                      {#if j.error}
                        <p class="text-[11px] text-[var(--loss)] truncate" title={j.error}>
                          {j.error}
                        </p>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>

        {#if hasToken}
          <button
            type="button"
            class="p-2 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--loss)] cursor-pointer"
            onclick={() => void logout()}
            aria-label="Sign out"
            title="Sign out"
          >
            <LogOut class="w-4 h-4" />
          </button>
        {/if}
      </div>
    </header>

    <main class="flex-1 w-full px-4 sm:px-6 lg:px-8 py-6">
      {#key renderKey}
        {@render children()}
      {/key}
    </main>
  </div>
</div>

<!-- Mobile Bottom Navigation -->
<nav
  class="md:hidden fixed bottom-0 inset-x-0 z-30 bg-[var(--surface)] border-t border-[var(--border)] grid grid-cols-5 h-14"
  aria-label="Mobile navigation"
>
  {#each navItems.slice(0, 5) as item}
    {@const Icon = item.icon}
    {@const active = isActive(item.href, pathname)}
    <a
      href={appPath(item.href)}
      class="flex flex-col items-center justify-center gap-0.5 text-[11px] {active
        ? 'text-[var(--accent)] font-semibold'
        : 'text-[var(--muted)]'}"
    >
      <Icon class="w-4 h-4" />
      <span>{item.label}</span>
    </a>
  {/each}
</nav>

<!-- Command Palette -->
{#if cmdOpen}
  <div class="fixed inset-0 z-50 flex items-start justify-center pt-20 px-4">
    <button
      type="button"
      class="fixed inset-0 bg-black/40"
      aria-label="Close command palette"
      onclick={() => (cmdOpen = false)}
    ></button>
    <div
      class="relative z-10 w-full max-w-md rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden"
    >
      <div class="flex items-center gap-2 px-3.5 py-2.5 border-b border-[var(--border)]">
        <Search class="w-4 h-4 text-[var(--muted)]" />
        <input
          bind:this={cmdInput}
          type="text"
          bind:value={cmdQuery}
          placeholder="Jump to page, tab, or account…"
          class="w-full bg-transparent text-sm outline-none placeholder:text-[var(--muted)]"
          onkeydown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              const target =
                (cmdQuery.trim() && filteredTabs.length > 0 && filteredCommands.length === 0
                  ? filteredTabs[0].href
                  : (filteredCommands[0]?.href ?? filteredTabs[0]?.href)) ??
                (filteredAccounts[0]
                  ? `/accounts/${encodeURIComponent(filteredAccounts[0])}`
                  : null);
              if (target) {
                cmdOpen = false;
                goto(appPath(target));
              }
            }
          }}
        />
      </div>
      <div class="p-1.5 max-h-80 overflow-y-auto">
        {#if filteredCommands.length > 0}
          <div
            class="px-3 pt-1 pb-1 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]"
          >
            Pages
          </div>
          {#each filteredCommands as item}
            {@const Icon = item.icon}
            <button
              type="button"
              class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-left hover:bg-[var(--surface-elevated)] cursor-pointer"
              onclick={() => {
                cmdOpen = false;
                goto(appPath(item.href));
              }}
            >
              <Icon class="w-4 h-4 text-[var(--muted)]" />
              <span>{item.label}</span>
            </button>
          {/each}
        {/if}
        {#if filteredTabs.length > 0}
          <div
            class="px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]"
          >
            Tabs
          </div>
          {#each filteredTabs as item}
            {@const Icon = item.icon}
            <button
              type="button"
              class="w-full flex items-center justify-between gap-3 px-3 py-1.5 rounded-lg text-sm text-left hover:bg-[var(--surface-elevated)] cursor-pointer"
              onclick={() => {
                cmdOpen = false;
                goto(appPath(item.href));
              }}
            >
              <span class="flex items-center gap-3 min-w-0">
                <Icon class="w-4 h-4 text-[var(--muted)] shrink-0" />
                <span class="truncate">{item.label}</span>
              </span>
              {#if item.parent}
                <span class="text-xs text-[var(--muted)] shrink-0">{item.parent}</span>
              {/if}
            </button>
          {/each}
        {/if}
        {#if filteredAccounts.length > 0}
          <div
            class="px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]"
          >
            Accounts
          </div>
          {#each filteredAccounts as acc}
            <button
              type="button"
              class="w-full flex items-center gap-3 px-3 py-1.5 rounded-lg text-xs text-left hover:bg-[var(--surface-elevated)] cursor-pointer truncate"
              onclick={() => {
                cmdOpen = false;
                goto(appPath(`/accounts/${encodeURIComponent(acc)}`));
              }}
            >
              <Landmark class="w-3.5 h-3.5 text-[var(--muted)] shrink-0" />
              <span class="truncate">{acc}</span>
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}

<QuickAddModal bind:open={quickAddOpen} onSaved={() => (renderKey += 1)} />
