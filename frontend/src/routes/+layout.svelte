<script lang="ts">
  import "../app.css";
  import { page } from "$app/stores";
  import { goto, invalidateAll } from "$app/navigation";
  import {
    ArrowLeftRight,
    BookOpen,
    Compass,
    Eye,
    EyeOff,
    Home,
    Landmark,
    Moon,
    PanelLeftClose,
    PanelLeftOpen,
    Plus,
    RefreshCw,
    Search,
    Sun,
    TrendingUp
  } from "lucide-svelte";
  import Logo from "$lib/components/Logo.svelte";
  import QuickAddModal from "$lib/components/QuickAddModal.svelte";
  import {
    getTheme,
    loadRuntimeConfig,
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
  let renderKey = $state(0);

  const navItems = [
    { label: "Home", href: "/", icon: Home },
    { label: "Transactions", href: "/transactions", icon: ArrowLeftRight },
    { label: "Cash Flow", href: "/cash-flow", icon: TrendingUp },
    { label: "Wealth", href: "/wealth", icon: Landmark },
    { label: "Planning", href: "/planning", icon: Compass },
    { label: "Ledger", href: "/ledger", icon: BookOpen }
  ];

  const pathname = $derived($page.url.pathname);
  const filteredCommands = $derived(
    navItems.filter((item) => item.label.toLowerCase().includes(cmdQuery.trim().toLowerCase()))
  );
  const filteredAccounts = $derived(
    cmdQuery.trim().length > 1
      ? runtime.accounts
          .filter((a) => a.toLowerCase().includes(cmdQuery.trim().toLowerCase()))
          .slice(0, 8)
      : []
  );

  onMount(async () => {
    theme = getTheme();
    const state = await loadRuntimeConfig();
    obscure = state.obscure;
    renderKey += 1;
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

  function isActive(href: string, current: string): boolean {
    if (href === "/") return current === "/";
    return current.startsWith(href);
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
    <div class="h-13 px-4 flex items-center justify-between border-b border-[var(--border)]">
      {#if !sidebarCollapsed}
        <a href="/" class="flex items-center gap-2 font-semibold text-base tracking-tight">
          <Logo size={22} />
          <span>Paisa</span>
        </a>
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
          href={item.href}
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
        <a href="/" class="md:hidden flex items-center gap-2 font-semibold text-base">
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
      href={item.href}
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
          placeholder="Jump to page…"
          class="w-full bg-transparent text-sm outline-none placeholder:text-[var(--muted)]"
        />
      </div>
      <div class="p-1.5 max-h-64 overflow-y-auto">
        {#each filteredCommands as item}
          {@const Icon = item.icon}
          <button
            type="button"
            class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-left hover:bg-[var(--surface-elevated)] cursor-pointer"
            onclick={() => {
              cmdOpen = false;
              goto(item.href);
            }}
          >
            <Icon class="w-4 h-4 text-[var(--muted)]" />
            <span>{item.label}</span>
          </button>
        {/each}
        {#if filteredAccounts.length > 0}
          <div class="px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-[var(--muted)]">
            Accounts
          </div>
          {#each filteredAccounts as acc}
            <button
              type="button"
              class="w-full flex items-center gap-3 px-3 py-1.5 rounded-lg text-xs text-left hover:bg-[var(--surface-elevated)] cursor-pointer truncate"
              onclick={() => {
                cmdOpen = false;
                goto(`/transactions?account=${encodeURIComponent(acc)}`);
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
