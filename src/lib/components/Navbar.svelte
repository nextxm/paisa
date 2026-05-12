<script lang="ts">
  import { page } from "$app/stores";
  import { afterNavigate } from "$app/navigation";
  import Actions from "$lib/components/Actions.svelte";
  import { month, year, dateMax, dateMin, dateRangeOption } from "../../store";
  import {
    cashflowExpenseDepth,
    cashflowExpenseDepthAllowed,
    cashflowIncomeDepth,
    cashflowIncomeDepthAllowed,
    cashflowShowTransfers,
    obscure,
    sankeyPeriod,
    sankeyRefDate
  } from "../../persisted_store";
  import _ from "lodash";
  import { financialYear, forEachFinancialYear, helpUrl, isMobile, now } from "$lib/utils";
  import { resolveNavbarSelectionTyped } from "$lib/navbar_selection";
  import { tick } from "svelte";
  import { get } from "svelte/store";
  import DateRange from "./DateRange.svelte";
  import ThemeSwitcher from "./ThemeSwitcher.svelte";
  import MonthPicker from "./MonthPicker.svelte";
  import Logo from "./Logo.svelte";
  import InputRange from "./InputRange.svelte";
  import PeriodSelector from "./PeriodSelector.svelte";
  import SyncingIndicator from "./SyncingIndicator.svelte";
  let { isBurger = $bindable(null) }: { isBurger: boolean | null } = $props();
  const readonly = USER_CONFIG.readonly;

  afterNavigate(() => {
    closeBurger(false);
  });

  $effect(() => {
    if (get(year) == "") {
      year.set(financialYear(now()));
    }
  });

  const RecurringIcons = [
    { icon: "fa-circle-check", color: "success", label: "Cleared" },
    { icon: "fa-circle-check", color: "warning-dark", label: "Cleared late" },
    { icon: "fa-exclamation-triangle", color: "danger", label: "Past due" },
    { icon: "fa-circle-check", color: "grey", label: "Upcoming" }
  ];

  interface Link {
    label: string;
    href: string;
    tag?: string;
    help?: string;
    hide?: boolean;
    dateRangeSelector?: boolean;
    monthPicker?: boolean;
    financialYearPicker?: boolean;
    maxDepthSelector?: boolean;
    recurringIcons?: boolean;
    sankeyPeriodSelector?: boolean;
    children?: Link[];
    disablePreload?: boolean;
  }

  const links: Link[] = [
    { label: "Dashboard", href: "/", hide: true },
    {
      label: "Cash Flow",
      href: "/cash_flow",
      children: [
        { label: "Income Statement", href: "/income_statement", financialYearPicker: true },
        { label: "Monthly", href: "/monthly", dateRangeSelector: true },
        {
          label: "Yearly",
          href: "/yearly",
          financialYearPicker: true,
          maxDepthSelector: true
        },
        {
          label: "Recurring",
          href: "/recurring",
          help: "recurring",
          monthPicker: true,
          recurringIcons: true
        },
        { label: "Sankey", href: "/sankey", sankeyPeriodSelector: true }
      ]
    },
    {
      label: "Expenses",
      href: "/expense",
      children: [
        { label: "Monthly", href: "/monthly", monthPicker: true, dateRangeSelector: true },
        { label: "Yearly", href: "/yearly", financialYearPicker: true },
        { label: "Budget", href: "/budget", help: "budget", monthPicker: true },
        { label: "Flow", href: "/sankey", dateRangeSelector: true }
      ]
    },
    {
      label: "Assets",
      href: "/assets",
      children: [
        { label: "Balance", href: "/balance" },
        { label: "Networth", href: "/networth", dateRangeSelector: true },
        { label: "Investment", href: "/investment" },
        { label: "Gain", href: "/gain" },
        { label: "Allocation", href: "/allocation", help: "allocation-targets" },
        { label: "Analysis", href: "/analysis", tag: "alpha", help: "analysis" }
      ]
    },
    {
      label: "Liabilities",
      href: "/liabilities",
      children: [
        { label: "Balance", href: "/balance" },
        { label: "Credit Cards", href: "/credit_cards", help: "credit-cards" },
        { label: "Repayment", href: "/repayment" },
        { label: "Interest", href: "/interest" }
      ]
    },
    { label: "Income", href: "/income" },
    {
      label: "Analysis",
      href: "/analysis",
      tag: "alpha",
      children: [
        { label: "YoY", href: "/yoy", tag: "alpha" },
        { label: "MoM", href: "/mom", tag: "alpha" }
      ]
    },
    {
      label: "Ledger",
      href: "/ledger",
      children: [
        { label: "Import", href: "/import", help: "import" },
        { label: "Editor", href: "/editor", help: "editor", disablePreload: true },
        { label: "Transactions", href: "/transaction", help: "bulk-edit" },
        { label: "Postings", href: "/posting" },
        { label: "Price", href: "/price" },
        { label: "FX Rates", href: "/fx-rates" }
      ]
    },
    {
      label: "More",
      href: "/more",
      children: [
        { label: "Configuration", href: "/config", help: "config" },
        { label: "Sheets", href: "/sheets", help: "sheets", disablePreload: true },
        { label: "Goals", href: "/goals", help: "goals" },
        { label: "Doctor", href: "/doctor" },
        ...(USER_CONFIG.labs?.firefly_reconcile
          ? [{ label: "Reconciliation", href: "/reconciliation" }]
          : []),
        { label: "Logs", href: "/logs" }
      ]
    }
  ];

  const tax = {
    label: "Tax",
    href: "/tax",
    help: "tax",
    children: [
      { label: "Harvest", href: "/harvest", help: "tax-harvesting" },
      { label: "Capital Gains", href: "/capital_gains", help: "capital-gains" },
      {
        label: "Schedule AL",
        href: "/schedule_al",
        help: "schedule-al",
        financialYearPicker: true
      }
    ]
  };

  if (USER_CONFIG.default_currency == "INR") {
    _.last(links)?.children?.push(tax);
  }

  const about = { label: "About", href: "/about" };
  _.last(links)?.children?.push(about);

  let selectedLink: Link | null = $state(null);
  let selectedSubLink: Link | null = $state(null);
  let selectedSubSubLink: Link | null = $state(null);
  let navMenuEl: HTMLDivElement;
  let burgerButtonEl: HTMLButtonElement;
  let previousFocusEl: HTMLElement | null = null;

  const focusableSelector =
    'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"]), input:not([disabled]), select:not([disabled]), textarea:not([disabled])';

  function getFocusableMenuElements() {
    if (!navMenuEl) return [];

    return Array.from(navMenuEl.querySelectorAll<HTMLElement>(focusableSelector)).filter(
      (el) => el.offsetParent !== null
    );
  }

  async function focusFirstMenuItem() {
    await tick();
    const focusableEls = getFocusableMenuElements();

    if (focusableEls.length > 0) {
      focusableEls[0].focus();
    } else {
      navMenuEl?.focus();
    }
  }

  function restoreBurgerFocus() {
    if (burgerButtonEl) {
      burgerButtonEl.focus();
      return;
    }

    previousFocusEl?.focus?.();
  }

  async function toggleBurger() {
    if (isBurger === true) {
      closeBurger();
      return;
    }

    previousFocusEl = document.activeElement as HTMLElement;
    isBurger = true;
    await focusFirstMenuItem();
  }

  function closeBurger(shouldRestoreFocus = true) {
    if (typeof document !== "undefined" && navMenuEl?.contains(document.activeElement)) {
      (document.activeElement as HTMLElement)?.blur?.();
    }

    isBurger = null;

    if (shouldRestoreFocus && typeof document !== "undefined") {
      tick().then(() => restoreBurgerFocus());
    }
  }

  function closeBurgerOnItemClick(event: MouseEvent) {
    if (!isMobile() || isBurger !== true) return;

    const target = event.target as HTMLElement;
    const link = target.closest("a") as HTMLAnchorElement | null;
    if (link && link.getAttribute("href") && link.getAttribute("href") !== "#") {
      closeBurger(false);
    }
  }

  function handleMenuKeydown(event: KeyboardEvent) {
    if (!isMobile() || isBurger !== true) return;

    if (event.key === "Escape") {
      event.preventDefault();
      closeBurger();
      return;
    }

    if (event.key !== "Tab") return;

    const focusableEls = getFocusableMenuElements();
    if (focusableEls.length === 0) return;

    const firstEl = focusableEls[0];
    const lastEl = focusableEls[focusableEls.length - 1];
    const activeEl = document.activeElement as HTMLElement;

    if (event.shiftKey && activeEl === firstEl) {
      event.preventDefault();
      lastEl.focus();
      return;
    }

    if (!event.shiftKey && activeEl === lastEl) {
      event.preventDefault();
      firstEl.focus();
    }
  }

  const normalizedPath = $derived($page.url.pathname?.replace(/(.+)\/$/, ""));

  // isNavInert: only make the mobile drawer inert when it's closed
  const isNavInert = $derived(
    isBurger !== true && typeof window !== "undefined" && window.innerWidth < 769 ? true : undefined
  );

  $effect(() => {
    if (typeof document !== "undefined") {
      document.body.classList.toggle("mobile-menu-open", isBurger === true && isMobile());
    }
  });

  $effect(() => {
    const selection = resolveNavbarSelectionTyped(links, normalizedPath);
    selectedLink = selection.selectedLink;
    selectedSubLink = selection.selectedSubLink;
    selectedSubSubLink = selection.selectedSubSubLink;
  });

  $effect(() => {
    navMenuEl?.addEventListener("click", closeBurgerOnItemClick);

    return () => {
      navMenuEl?.removeEventListener("click", closeBurgerOnItemClick);
      if (typeof document !== "undefined") {
        document.body.classList.remove("mobile-menu-open");
      }
    };
  });
</script>

<nav class="navbar px-2 is-transparent" aria-label="main navigation">
  <div class="navbar-brand">
    <button
      type="button"
      bind:this={burgerButtonEl}
      class="navbar-burger mobile-drawer-toggle"
      class:is-active={isBurger === true}
      onclick={toggleBurger}
      aria-label="menu"
      aria-expanded={isBurger === true}
      aria-controls="primary-nav-menu"
    >
      <span aria-hidden="true"></span>
      <span aria-hidden="true"></span>
      <span aria-hidden="true"></span>
    </button>

    <a
      href="/"
      class:is-active={normalizedPath == "/"}
      class="navbar-item is-size-4 has-text-weight-medium"
    >
      {#if $obscure}
        <span class="icon is-small is-size-5">
          <i class="fas fa-user-secret"></i>
        </span><span class="ml-2 is-primary-color">Paisa</span>
      {:else}
        <Logo size={22} /><span class="ml-1 is-primary-color">Paisa</span>
      {/if}
    </a>

    <div class="navbar-item navbar-actions-row mobile-top-actions">
      <SyncingIndicator />
      <ThemeSwitcher />
      <Actions />
    </div>
  </div>

  <div
    id="primary-nav-menu"
    bind:this={navMenuEl}
    class="navbar-menu"
    class:is-active={isBurger === true}
    tabindex="-1"
    onkeydown={handleMenuKeydown}
    aria-hidden={isNavInert ? "true" : undefined}
    inert={isNavInert}
  >
    <div class="navbar-start">
      {#each links as link}
        {#if _.isEmpty(link.children)}
          {#if !link.hide}
            <a
              class="navbar-item"
              href={link.href}
              data-sveltekit-preload-data={link.disablePreload ? "tap" : "hover"}
              class:is-active={normalizedPath == link.href}>{link.label}</a
            >
          {/if}
        {:else}
          <div class="navbar-item has-dropdown is-hoverable">
            <a
              href={"#"}
              role="button"
              class="navbar-link"
              class:is-active={normalizedPath.startsWith(link.href)}
              onclick={(e) => {
                e.preventDefault();
                isMobile() && e.currentTarget.parentElement?.classList.toggle("is-active");
              }}
              onkeydown={(e) =>
                e.key === "Enter" &&
                isMobile() &&
                e.currentTarget.parentElement?.classList.toggle("is-active")}>{link.label}</a
            >
            <div class="navbar-dropdown {!isMobile() && 'is-boxed'}">
              {#each link.children as sublink}
                {@const href = link.href + sublink.href}
                {#if _.isEmpty(sublink.children)}
                  <a
                    class="navbar-item"
                    {href}
                    data-sveltekit-preload-data={sublink.disablePreload ? "tap" : "hover"}
                    class:is-active={normalizedPath.startsWith(href)}>{sublink.label}</a
                  >
                {:else}
                  <div class="nested has-dropdown navbar-item">
                    <a
                      href={"#"}
                      role="button"
                      class="navbar-link is-arrowless is-flex is-justify-content-space-between is-active"
                      class:is-active={normalizedPath.startsWith(href)}
                      onclick={(e) => {
                        e.preventDefault();
                        isMobile() && e.currentTarget.parentElement?.classList.toggle("is-active");
                      }}
                      onkeydown={(e) =>
                        e.key === "Enter" &&
                        isMobile() &&
                        e.currentTarget.parentElement?.classList.toggle("is-active")}
                    >
                      <span>{sublink.label}</span>
                      <span class="icon is-small">
                        <i
                          class="fas {isMobile() ? 'fa-angle-down' : 'fa-angle-right'}"
                          aria-hidden="true"
                        ></i>
                      </span>
                    </a>

                    <div class="dropdown-menu">
                      <div class="dropdown-content">
                        {#each sublink.children as subsublink}
                          <a
                            href={href + subsublink.href}
                            class="navbar-item"
                            data-sveltekit-preload-data={subsublink.disablePreload
                              ? "tap"
                              : "hover"}
                            class:is-active={normalizedPath == href + subsublink.href}
                            >{subsublink.label}</a
                          >
                        {/each}
                      </div>
                    </div>
                  </div>
                {/if}
              {/each}
            </div>
          </div>
        {/if}
      {/each}
    </div>
    <div class="navbar-end" style="margin-right: 0.3em">
      <div class="navbar-item navbar-actions-row menu-actions-row">
        {#if readonly}
          <span
            class="mt-1 tag is-rounded is-danger is-light invertable"
            data-tippy-content="<p>Paisa is in readonly mode</p>">readonly</span
          >
        {/if}

        <SyncingIndicator />
        <ThemeSwitcher />
        <Actions />
      </div>
    </div>
  </div>
</nav>

{#if isBurger === true && isMobile()}
  <button
    type="button"
    class="mobile-nav-backdrop"
    aria-label="Close navigation menu"
    onclick={() => closeBurger()}
  ></button>
{/if}

<div class="mt-3 px-3 is-flex is-justify-content-space-between">
  {#if selectedLink}
    <nav
      style="margin-left: 0.73rem;"
      class="breadcrumb has-chevron-separator mb-0 is-small app-breadcrumb"
      aria-label="breadcrumbs"
    >
      <ul>
        <li class="breadcrumb-node">
          <span class="is-inactive breadcrumb-node-label">{selectedLink.label}</span>
          {#if selectedLink.help}
            <a
              class="is-clear ml-1 breadcrumb-help"
              href={helpUrl(selectedLink.help)}
              aria-label={`Help for ${selectedLink.label}`}
              ><span class="icon is-small">
                <i class="fas fa-question"></i>
              </span></a
            >
          {/if}

          {#if selectedLink.tag}
            <span class="tag is-rounded is-warning breadcrumb-alpha-tag">{selectedLink.tag}</span>
          {/if}
        </li>
        {#if selectedSubLink}
          <li class="breadcrumb-node">
            <span class="is-inactive breadcrumb-node-label">{selectedSubLink.label}</span>

            {#if selectedSubLink.help}
              <a
                class="is-clear ml-1 breadcrumb-help"
                href={helpUrl(selectedSubLink.help)}
                aria-label={`Help for ${selectedSubLink.label}`}
                ><span class="icon is-small">
                  <i class="fas fa-question"></i>
                </span></a
              >
            {/if}

            {#if selectedSubLink.tag}
              <span class="tag is-rounded is-warning breadcrumb-alpha-tag"
                >{selectedSubLink.tag}</span
              >
            {/if}
          </li>
        {/if}

        {#if selectedSubLink}
          {#if selectedSubSubLink}
            <li>
              <span class="is-inactive">{selectedSubSubLink.label}</span>
            </li>
          {:else if selectedLink.href + selectedSubLink.href != normalizedPath}
            <li>
              <span class="is-inactive"
                >{decodeURIComponent(_.last(normalizedPath.split("/")) ?? "")}</span
              >
            </li>
          {/if}
        {/if}
      </ul>
    </nav>
  {/if}

  <div class="mr-3 is-flex" style="gap: 12px">
    {#if selectedSubLink?.recurringIcons}
      <div class="flex gap-5 items-center has-text-grey">
        {#each RecurringIcons as icon}
          <div data-tippy-content="<p>{icon.label}</p>">
            <span class="icon is-small has-text-{icon.color}">
              <i class={"fas " + icon.icon}></i>
            </span>
            <span class="is-hidden-mobile">{icon.label}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if selectedSubLink?.maxDepthSelector}
      <div class="dropdown is-right is-hoverable">
        <div class="dropdown-trigger">
          <button class="button is-small" aria-haspopup="true" aria-label="Flow settings">
            <span class="icon is-small">
              <i class="fas fa-sliders"></i>
            </span>
          </button>
        </div>
        <div class="dropdown-menu" role="menu">
          <div class="dropdown-content px-2 py-2">
            {#if $cashflowExpenseDepthAllowed.max > 1 || $cashflowIncomeDepthAllowed.max > 1}
              <InputRange
                label="Expenses"
                bind:value={$cashflowExpenseDepth}
                allowed={$cashflowExpenseDepthAllowed}
              />
              <InputRange
                label="Income"
                bind:value={$cashflowIncomeDepth}
                allowed={$cashflowIncomeDepthAllowed}
              />
            {/if}
            <label class="checkbox is-size-7 mt-1 ml-1">
              <input type="checkbox" bind:checked={$cashflowShowTransfers} />
              Show Transfers
            </label>
          </div>
        </div>
      </div>
    {/if}

    {#if selectedSubLink?.dateRangeSelector || selectedLink?.dateRangeSelector}
      <div>
        <DateRange bind:value={$dateRangeOption} dateMin={$dateMin} dateMax={$dateMax} />
      </div>
    {/if}

    {#if selectedSubLink?.monthPicker || selectedLink?.monthPicker}
      <MonthPicker bind:value={$month} max={$dateMax} min={$dateMin} />
    {/if}

    {#if selectedSubLink?.sankeyPeriodSelector || selectedLink?.sankeyPeriodSelector}
      <PeriodSelector
        bind:value={$sankeyPeriod}
        bind:refDate={$sankeyRefDate}
        minDate={$dateMin}
        maxDate={$dateMax}
      />
    {/if}

    {#if selectedSubSubLink?.financialYearPicker || selectedSubLink?.financialYearPicker || selectedLink?.financialYearPicker}
      <div class="has-text-centered">
        <div class="select is-small">
          <select bind:value={$year}>
            {#each forEachFinancialYear($dateMin, $dateMax).reverse() as fy}
              <option>{financialYear(fy)}</option>
            {/each}
          </select>
        </div>
      </div>
    {/if}
  </div>
</div>

<style lang="scss">
  :global(body.mobile-menu-open) {
    overflow: hidden;
  }

  .navbar-actions-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }

  .navbar-actions-row :global(.theme-toggle) {
    --size: 1.9rem;
    border-radius: 0.45rem;
  }

  .mobile-top-actions {
    display: none !important;
  }

  .menu-actions-row {
    display: flex;
  }

  @media screen and (max-width: 1023px) {
    .navbar-menu {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      width: min(86vw, 22rem);
      display: flex !important;
      flex-direction: column;
      overflow-y: auto;
      z-index: 30;
      padding-top: 0.75rem;
      padding-bottom: 0.75rem;
      border-right: 1px solid rgba(127, 127, 127, 0.2);
      transform: translateX(calc(-100% - 0.75rem));
      opacity: 0;
      visibility: hidden;
      pointer-events: none;
      transition:
        transform 220ms ease,
        opacity 180ms ease,
        visibility 0s linear 220ms;
      will-change: transform;
    }

    .navbar-brand {
      align-items: center;
      flex-wrap: nowrap;
      justify-content: flex-start;
      width: 100%;
    }

    .navbar-brand .navbar-burger.mobile-drawer-toggle {
      order: -1;
      margin-left: 0 !important;
      margin-right: 0.2rem;
      flex-shrink: 0;
    }

    .mobile-top-actions {
      display: inline-flex !important;
      flex: 1 1 auto;
      min-width: 0;
      overflow-x: auto;
      overflow-y: hidden;
      scrollbar-width: none;
      -ms-overflow-style: none;
      margin-left: auto;
      padding-right: 0.1rem;
      align-items: center;
      gap: 0.2rem;
      flex-wrap: nowrap;
    }

    .mobile-top-actions::-webkit-scrollbar {
      display: none;
    }

    .mobile-top-actions :global(.navbar-actions-strip) {
      flex-wrap: nowrap;
      min-width: max-content;
    }

    .menu-actions-row {
      display: none !important;
    }

    .mobile-nav-backdrop {
      position: fixed;
      inset: 0;
      border: 0;
      background: rgba(10, 10, 10, 0.4);
      z-index: 29;
      cursor: pointer;
    }

    .navbar-menu.is-active {
      transform: translateX(0);
      opacity: 1;
      visibility: visible;
      pointer-events: auto;
      transition:
        transform 220ms ease,
        opacity 180ms ease,
        visibility 0s linear 0s;
    }

    .navbar-menu.is-active .navbar-end {
      margin-top: auto;
      margin-right: 0 !important;
      padding-bottom: 0.25rem;
    }

    .navbar-actions-row {
      gap: 0.25rem;
      padding-right: 0;
    }

    .navbar-actions-row :global(.theme-toggle) {
      --size: 2rem;
      border-radius: 0.5rem;
    }
  }

  @media screen and (max-width: 640px) {
    .navbar-brand {
      flex-wrap: wrap;
    }

    .navbar-actions-row {
      gap: 0.2rem;
      width: 100%;
      justify-content: flex-start;
    }

    .mobile-top-actions {
      flex: 1 1 100%;
      width: 100%;
      margin-left: 0;
      overflow: visible;
      padding: 0.15rem 0 0.1rem 0.15rem;
      justify-content: flex-start;
    }

    .navbar-actions-row :global(.theme-toggle) {
      --size: 2.2rem;
      border-radius: 0.5rem;
    }
  }
</style>
