<script lang="ts">
  import {
    type AccountReconciliationStatus,
    type AssetBreakdown,
    buildTree,
    lastName
  } from "$lib/utils";
  import {
    reconciliationLabel,
    reconciliationTagClass,
    reconciliationIcon,
    reconciliationTextClass
  } from "$lib/reconciliation";
  import { iconText } from "$lib/icon";
  import _ from "lodash";
  import Table from "./Table.svelte";
  import type { ColumnDefinition } from "tabulator-tables";
  import {
    formatCurrencyChange,
    formatOriginalBalances,
    nonZeroCurrency,
    nonZeroCurrencyLink,
    nonZeroFloatChange,
    nonZeroPercentageChange
  } from "$lib/table_formatters";

  let {
    breakdowns,
    reconciliationStatuses = {},
    indent = true,
    filterInactive = true,
    filterZero = true
  }: {
    breakdowns: Record<string, AssetBreakdown>;
    reconciliationStatuses?: Record<string, AccountReconciliationStatus>;
    indent?: boolean;
    filterInactive?: boolean;
    filterZero?: boolean;
  } = $props();

  function accountNameWithReconciliation(account: string, cell: any, compact = false) {
    const status = reconciliationStatuses[account];
    const label = status ? reconciliationLabel(status) : "Last reconciled: never";
    const klass = status ? reconciliationTagClass(status) : "is-danger";
    const icon = status
      ? reconciliationIcon(status)
      : reconciliationIcon({ days_since: null } as any);
    const accountText = compact ? lastName(account) : account;
    let children = "";
    const data = cell.getData();
    const childCount = data._children?.length || 0;
    if (childCount > 0) {
      children = `(${childCount})`;
    }
    return `
<span class="whitespace-nowrap" style="max-width: max(15rem, 33.33vw); overflow: hidden;">
  <span class="has-text-grey custom-icon">${iconText(account)}</span>
  <a href="/assets/gain/${account}">${accountText}</a>
  <span class="has-text-grey-light is-size-7">${children}</span>
  ${
    USER_CONFIG.enable_reconciliation
      ? `
  <button type="button" onclick="openReconciliationModal('${account.replace(/'/g, "\\'")}')" class="button is-ghost p-0 h-auto ml-2 is-small ${reconciliationTextClass(
    status
  )}" title="${label}" style="vertical-align: baseline; height: 1.2em; width: 1.2em; line-height: 1;">
    <span class="custom-icon" style="font-size: 0.9em;">${icon}</span>
  </button>`
      : ""
  }
</span>
`;
  }

  const columns: ColumnDefinition[] = $derived([
    {
      title: "Account",
      field: "group",
      formatter: (cell) => accountNameWithReconciliation(cell.getValue(), cell, indent),
      frozen: true
    },
    {
      title: "Investment Amount",
      field: "investmentAmount",
      hozAlign: "right",
      vertAlign: "middle",
      formatter: nonZeroCurrency
    },
    {
      title: "Withdrawal Amount",
      field: "withdrawalAmount",
      hozAlign: "right",
      formatter: nonZeroCurrency
    },
    {
      title: "Balance Units",
      field: "balanceUnits",
      hozAlign: "right",
      formatter: nonZeroCurrencyLink
    },
    {
      title: "Original Value",
      field: "originalBalances",
      hozAlign: "right",
      formatter: formatOriginalBalances
    },
    {
      title: "Market Value",
      field: "marketAmount",
      hozAlign: "right",
      formatter: nonZeroCurrencyLink
    },
    { title: "Change", field: "gainAmount", hozAlign: "right", formatter: formatCurrencyChange },
    { title: "XIRR", field: "xirr", hozAlign: "right", formatter: nonZeroFloatChange },
    {
      title: "Absolute Return",
      field: "absoluteReturn",
      hozAlign: "right",
      formatter: nonZeroPercentageChange
    }
  ]);

  let tree = $derived.by(() => {
    if (!breakdowns) return [];
    let filteredBreakdowns = Object.values(breakdowns);
    if (filterInactive) {
      filteredBreakdowns = filteredBreakdowns.filter((i) => !i.inactive);
    }
    if (filterZero) {
      filteredBreakdowns = filteredBreakdowns.filter((i) => i.marketAmount !== 0);
    }
    return buildTree(filteredBreakdowns, (i) => i.group);
  });

  let displayBreakdowns = $derived.by(() => {
    if (!breakdowns) return [];
    let values = Object.values(breakdowns);
    if (filterInactive) {
      values = values.filter((i) => !i.inactive);
    }
    if (filterZero) {
      values = values.filter((i) => i.marketAmount !== 0);
    }
    return values;
  });
</script>

{#if indent}
  <Table data={tree} tree {columns} />
{:else}
  <Table data={displayBreakdowns} {columns} />
{/if}
