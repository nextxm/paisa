<script lang="ts">
  import { goto } from "$app/navigation";
  import { ajax } from "$lib/utils";
  import { tick } from "svelte";
  import _ from "lodash";
  import QuickAddModal from "./QuickAddModal.svelte";
  import { commandPaletteOpen } from "../../store";

  interface Command {
    id: string;
    label: string;
    description?: string;
    icon: string;
    category: string;
    action: () => void | Promise<void>;
  }

  let open = $state(false);
  let query = $state("");
  let selectedIndex = $state(0);
  let inputEl: HTMLInputElement | null = $state(null);
  let listEl: HTMLUListElement | null = $state(null);
  let showQuickAdd = $state(false);
  let accounts: string[] = $state([]);

  const navPages: Command[] = [
    {
      id: "dashboard",
      label: "Dashboard",
      icon: "fa-house",
      category: "Pages",
      action: () => navigate("/")
    },
    // Cash Flow
    {
      id: "income_statement",
      label: "Income Statement",
      description: "Cash Flow",
      icon: "fa-file-invoice-dollar",
      category: "Pages",
      action: () => navigate("/cash_flow/income_statement")
    },
    {
      id: "cashflow_monthly",
      label: "Monthly Cash Flow",
      description: "Cash Flow",
      icon: "fa-calendar-days",
      category: "Pages",
      action: () => navigate("/cash_flow/monthly")
    },
    {
      id: "cashflow_yearly",
      label: "Yearly Cash Flow",
      description: "Cash Flow",
      icon: "fa-calendar",
      category: "Pages",
      action: () => navigate("/cash_flow/yearly")
    },
    {
      id: "recurring",
      label: "Recurring Transactions",
      description: "Cash Flow",
      icon: "fa-rotate",
      category: "Pages",
      action: () => navigate("/cash_flow/recurring")
    },
    {
      id: "sankey_cashflow",
      label: "Sankey (Cash Flow)",
      description: "Cash Flow",
      icon: "fa-chart-diagram",
      category: "Pages",
      action: () => navigate("/cash_flow/sankey")
    },
    // Expenses
    {
      id: "expense_monthly",
      label: "Monthly Expenses",
      description: "Expenses",
      icon: "fa-receipt",
      category: "Pages",
      action: () => navigate("/expense/monthly")
    },
    {
      id: "expense_yearly",
      label: "Yearly Expenses",
      description: "Expenses",
      icon: "fa-receipt",
      category: "Pages",
      action: () => navigate("/expense/yearly")
    },
    {
      id: "budget",
      label: "Budget",
      description: "Expenses",
      icon: "fa-piggy-bank",
      category: "Pages",
      action: () => navigate("/expense/budget")
    },
    {
      id: "expense_flow",
      label: "Expense Flow",
      description: "Expenses",
      icon: "fa-chart-diagram",
      category: "Pages",
      action: () => navigate("/expense/sankey")
    },
    // Assets
    {
      id: "assets_balance",
      label: "Assets Balance",
      description: "Assets",
      icon: "fa-scale-balanced",
      category: "Pages",
      action: () => navigate("/assets/balance")
    },
    {
      id: "networth",
      label: "Networth",
      description: "Assets",
      icon: "fa-chart-line",
      category: "Pages",
      action: () => navigate("/assets/networth")
    },
    {
      id: "investment",
      label: "Investment",
      description: "Assets",
      icon: "fa-chart-pie",
      category: "Pages",
      action: () => navigate("/assets/investment")
    },
    {
      id: "gain",
      label: "Gain",
      description: "Assets",
      icon: "fa-arrow-trend-up",
      category: "Pages",
      action: () => navigate("/assets/gain")
    },
    {
      id: "allocation",
      label: "Allocation",
      description: "Assets",
      icon: "fa-chart-pie",
      category: "Pages",
      action: () => navigate("/assets/allocation")
    },
    // Liabilities
    {
      id: "liabilities_balance",
      label: "Liabilities Balance",
      description: "Liabilities",
      icon: "fa-scale-unbalanced-flip",
      category: "Pages",
      action: () => navigate("/liabilities/balance")
    },
    {
      id: "credit_cards",
      label: "Credit Cards",
      description: "Liabilities",
      icon: "fa-credit-card",
      category: "Pages",
      action: () => navigate("/liabilities/credit_cards")
    },
    {
      id: "repayment",
      label: "Loan Repayment",
      description: "Liabilities",
      icon: "fa-hand-holding-dollar",
      category: "Pages",
      action: () => navigate("/liabilities/repayment")
    },
    {
      id: "interest",
      label: "Interest",
      description: "Liabilities",
      icon: "fa-percent",
      category: "Pages",
      action: () => navigate("/liabilities/interest")
    },
    // Income
    {
      id: "income",
      label: "Income",
      icon: "fa-money-bill-trend-up",
      category: "Pages",
      action: () => navigate("/income")
    },
    // Ledger
    {
      id: "import",
      label: "Import",
      description: "Ledger",
      icon: "fa-file-import",
      category: "Pages",
      action: () => navigate("/ledger/import")
    },
    {
      id: "editor",
      label: "Editor",
      description: "Ledger",
      icon: "fa-file-pen",
      category: "Pages",
      action: () => navigate("/ledger/editor")
    },
    {
      id: "transactions",
      label: "Transactions",
      description: "Ledger",
      icon: "fa-list-ul",
      category: "Pages",
      action: () => navigate("/ledger/transaction")
    },
    {
      id: "postings",
      label: "Postings",
      description: "Ledger",
      icon: "fa-table-list",
      category: "Pages",
      action: () => navigate("/ledger/posting")
    },
    {
      id: "price",
      label: "Price",
      description: "Ledger",
      icon: "fa-tag",
      category: "Pages",
      action: () => navigate("/ledger/price")
    },
    {
      id: "fx_rates",
      label: "FX Rates",
      description: "Ledger",
      icon: "fa-money-bill-transfer",
      category: "Pages",
      action: () => navigate("/ledger/fx-rates")
    },
    // More
    {
      id: "configuration",
      label: "Configuration",
      description: "More",
      icon: "fa-gear",
      category: "Pages",
      action: () => navigate("/more/config")
    },
    {
      id: "sheets",
      label: "Sheets",
      description: "More",
      icon: "fa-table-cells",
      category: "Pages",
      action: () => navigate("/more/sheets")
    },
    {
      id: "goals",
      label: "Goals",
      description: "More",
      icon: "fa-bullseye",
      category: "Pages",
      action: () => navigate("/more/goals")
    },
    {
      id: "doctor",
      label: "Doctor",
      description: "More",
      icon: "fa-stethoscope",
      category: "Pages",
      action: () => navigate("/more/doctor")
    },
    {
      id: "logs",
      label: "Logs",
      description: "More",
      icon: "fa-scroll",
      category: "Pages",
      action: () => navigate("/more/logs")
    }
  ];

  const currencyCommands: Command[] = $derived.by(() => {
    // USER_CONFIG is a TypeScript global declared in src/app.d.ts, injected at runtime.
    const currencies: string[] = USER_CONFIG.currencies || [];
    if (currencies.length === 0) return [];
    return currencies.map((c) => ({
      id: `currency_${c}`,
      label: `FX Rates for ${c}`,
      description: "Currency",
      icon: "fa-money-bill-transfer",
      category: "Currency",
      action: () => navigate("/ledger/fx-rates")
    }));
  });

  const allCommands: Command[] = $derived([
    {
      id: "quick_add",
      label: "Quick Add Transaction",
      description: "Add a new transaction",
      icon: "fa-circle-plus",
      category: "Actions",
      action: openQuickAdd
    },
    ...navPages,
    ...(USER_CONFIG.default_currency === "INR"
      ? [
          {
            id: "tax_harvest",
            label: "Tax Harvest",
            description: "Tax",
            icon: "fa-seedling",
            category: "Pages",
            action: () => navigate("/more/tax/harvest")
          },
          {
            id: "capital_gains",
            label: "Capital Gains",
            description: "Tax",
            icon: "fa-coins",
            category: "Pages",
            action: () => navigate("/more/tax/capital_gains")
          },
          {
            id: "schedule_al",
            label: "Schedule AL",
            description: "Tax",
            icon: "fa-file-lines",
            category: "Pages",
            action: () => navigate("/more/tax/schedule_al")
          }
        ]
      : []),
    ...currencyCommands
  ]);

  const filteredCommands: Command[] = $derived.by(() => {
    if (!query.trim()) return allCommands;
    const q = query.toLowerCase();
    return allCommands.filter(
      (cmd) =>
        cmd.label.toLowerCase().includes(q) ||
        (cmd.description || "").toLowerCase().includes(q) ||
        cmd.category.toLowerCase().includes(q)
    );
  });

  $effect(() => {
    // Reset selection to top whenever the filtered result set changes
    const _ = filteredCommands.length;
    selectedIndex = 0;
  });

  $effect(() => {
    // Sync with the shared store so external callers can open the palette
    if ($commandPaletteOpen && !open) {
      openPalette();
      commandPaletteOpen.set(false);
    }
  });

  async function openPalette() {
    query = "";
    selectedIndex = 0;
    open = true;
    await tick();
    inputEl?.focus();
  }

  function closePalette() {
    open = false;
    query = "";
  }

  function navigate(href: string) {
    closePalette();
    goto(href);
  }

  async function openQuickAdd() {
    closePalette();
    showQuickAdd = true;

    if (accounts.length === 0) {
      const response = await ajax("/api/config", { background: true });
      accounts = response.accounts || [];
    }
  }

  async function runCommand(cmd: Command) {
    await cmd.action();
  }

  function handleKeydown(e: KeyboardEvent) {
    const isCtrlK = (e.ctrlKey || e.metaKey) && e.key === "k";
    if (isCtrlK) {
      e.preventDefault();
      if (open) {
        closePalette();
      } else {
        openPalette();
      }
      return;
    }

    if (!open) return;

    if (e.key === "Escape") {
      e.preventDefault();
      closePalette();
      return;
    }

    if (e.key === "ArrowDown") {
      e.preventDefault();
      selectedIndex = Math.min(selectedIndex + 1, filteredCommands.length - 1);
      scrollToSelected();
      return;
    }

    if (e.key === "ArrowUp") {
      e.preventDefault();
      selectedIndex = Math.max(selectedIndex - 1, 0);
      scrollToSelected();
      return;
    }

    if (e.key === "Enter") {
      e.preventDefault();
      if (filteredCommands[selectedIndex]) {
        runCommand(filteredCommands[selectedIndex]);
      }
      return;
    }
  }

  function scrollToSelected() {
    const itemEl = listEl?.children[selectedIndex] as HTMLElement | undefined;
    itemEl?.scrollIntoView({ block: "nearest" });
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<QuickAddModal bind:open={showQuickAdd} {accounts} />

{#if open}
  <!-- Backdrop -->
  <button class="command-palette-backdrop" aria-label="Close command palette" onclick={closePalette}
  ></button>

  <!-- Palette -->
  <div class="command-palette" role="dialog" aria-label="Command palette" aria-modal="true">
    <div class="command-palette-search">
      <span class="icon command-palette-search-icon">
        <i class="fas fa-magnifying-glass"></i>
      </span>
      <input
        bind:this={inputEl}
        bind:value={query}
        class="command-palette-input"
        type="text"
        placeholder="Search pages and actions..."
        autocomplete="off"
        autocorrect="off"
        spellcheck="false"
        aria-label="Search commands"
        aria-controls="command-palette-list"
        aria-activedescendant={filteredCommands[selectedIndex]
          ? `cp-item-${filteredCommands[selectedIndex].id}`
          : undefined}
      />
      <kbd class="command-palette-esc-hint">Esc</kbd>
    </div>

    {#if filteredCommands.length > 0}
      <ul
        bind:this={listEl}
        id="command-palette-list"
        class="command-palette-list"
        role="listbox"
        aria-label="Commands"
      >
        {#each filteredCommands as cmd, i}
          <li id="cp-item-{cmd.id}" role="option" aria-selected={i === selectedIndex}>
            <button
              type="button"
              class="command-palette-item"
              class:is-selected={i === selectedIndex}
              onclick={() => runCommand(cmd)}
              onmouseenter={() => (selectedIndex = i)}
            >
              <span class="command-palette-item-icon">
                <i class="fas {cmd.icon}"></i>
              </span>
              <span class="command-palette-item-body">
                <span class="command-palette-item-label">{cmd.label}</span>
                {#if cmd.description}
                  <span class="command-palette-item-desc">{cmd.description}</span>
                {/if}
              </span>
              <span class="command-palette-item-category">{cmd.category}</span>
            </button>
          </li>
        {/each}
      </ul>
    {:else}
      <div class="command-palette-empty">No results for &ldquo;{query}&rdquo;</div>
    {/if}

    <div class="command-palette-footer">
      <span><kbd>↑</kbd><kbd>↓</kbd> navigate</span>
      <span><kbd>↵</kbd> select</span>
      <span><kbd>Esc</kbd> close</span>
      <span class="command-palette-footer-shortcut"><kbd>Ctrl</kbd><kbd>K</kbd> toggle</span>
    </div>
  </div>
{/if}

<style lang="scss">
  .command-palette {
    --cp-surface: var(--bulma-scheme-main, #ffffff);
    --cp-surface-alt: var(--bulma-background, #f5f5f5);
    --cp-border: var(--bulma-border, #dbdbdb);
    --cp-text: var(--bulma-text, #363636);
    --cp-text-muted: var(--bulma-text-weak, #7a7a7a);
    --cp-backdrop: rgba(0, 0, 0, 0.45);
    --cp-primary-soft: var(--bulma-primary-10, rgba(72, 95, 199, 0.1));
  }

  :global(html[data-theme="dark"]) .command-palette {
    --cp-surface: hsl(215, 18%, 14%);
    --cp-surface-alt: hsl(215, 18%, 20%);
    --cp-border: hsl(215, 18%, 26%);
    --cp-text: hsl(0, 0%, 90%);
    --cp-text-muted: hsl(0, 0%, 70%);
    --cp-backdrop: rgba(0, 0, 0, 0.62);
    --cp-primary-soft: rgba(114, 141, 255, 0.22);
  }

  .command-palette-backdrop {
    position: fixed;
    inset: 0;
    background: var(--cp-backdrop);
    z-index: 9000;
    cursor: default;
    border: none;
    padding: 0;
  }

  .command-palette {
    position: fixed;
    top: 12vh;
    left: 50%;
    transform: translateX(-50%);
    width: min(640px, calc(100vw - 2rem));
    max-height: 70vh;
    background: var(--cp-surface);
    border-radius: 0.75rem;
    box-shadow:
      0 24px 60px rgba(0, 0, 0, 0.25),
      0 4px 16px rgba(0, 0, 0, 0.12);
    z-index: 9001;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--cp-border);
  }

  .command-palette-search {
    display: flex;
    align-items: center;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--cp-border);
    gap: 0.5rem;
  }

  .command-palette-search-icon {
    color: var(--cp-text-muted);
    flex-shrink: 0;
    font-size: 0.9rem;
  }

  .command-palette-input {
    flex: 1;
    border: none;
    outline: none;
    background: transparent;
    font-size: 1rem;
    color: var(--cp-text);
    min-width: 0;

    &::placeholder {
      color: var(--cp-text-muted);
    }
  }

  .command-palette-esc-hint {
    flex-shrink: 0;
    padding: 0.15rem 0.35rem;
    border-radius: 0.3rem;
    background: var(--cp-surface-alt);
    border: 1px solid var(--cp-border);
    font-size: 0.7rem;
    color: var(--cp-text-muted);
    font-family: inherit;
  }

  .command-palette-list {
    flex: 1;
    overflow-y: auto;
    list-style: none;
    margin: 0;
    padding: 0.375rem;

    li {
      display: block;
    }
  }

  .command-palette-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0.75rem;
    border-radius: 0.45rem;
    cursor: pointer;
    transition: background 80ms ease;
    user-select: none;
    width: 100%;
    border: none;
    background: transparent;
    text-align: left;
    font-family: inherit;
    font-size: inherit;

    &.is-selected {
      background: var(--cp-primary-soft);
    }

    &:hover {
      background: var(--cp-surface-alt);
    }

    &.is-selected:hover {
      background: var(--cp-primary-soft);
    }
  }

  .command-palette-item-icon {
    width: 1.5rem;
    height: 1.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.35rem;
    background: var(--cp-surface-alt);
    flex-shrink: 0;
    font-size: 0.75rem;
    color: var(--bulma-primary, #485fc7);
  }

  .command-palette-item-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .command-palette-item-label {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--cp-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .command-palette-item-desc {
    font-size: 0.75rem;
    color: var(--cp-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .command-palette-item-category {
    font-size: 0.7rem;
    color: var(--cp-text-muted);
    flex-shrink: 0;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .command-palette-empty {
    padding: 2rem 1rem;
    text-align: center;
    color: var(--cp-text-muted);
    font-size: 0.9rem;
  }

  .command-palette-footer {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem 1rem;
    border-top: 1px solid var(--cp-border);
    font-size: 0.7rem;
    color: var(--cp-text-muted);

    kbd {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0.1rem 0.3rem;
      border-radius: 0.25rem;
      background: var(--cp-surface-alt);
      border: 1px solid var(--cp-border);
      font-size: 0.65rem;
      font-family: inherit;
      margin-right: 0.15rem;
    }
  }

  .command-palette-footer-shortcut {
    margin-left: auto;
  }
</style>
