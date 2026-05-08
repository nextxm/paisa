<script lang="ts">
  import { type AssetBreakdown, buildTree } from "$lib/utils";
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

  let props: {
    breakdowns: Record<string, AssetBreakdown>;
    indent?: boolean;
    filterInactive?: boolean;
    filterZero?: boolean;
  } = $props();

  function accountName(account: string, indent: number) {
    const parts = account.split(":");
    const name = parts[parts.length - 1];
    const padding = indent * 20;
    return `<span style="padding-left: ${padding}px">${name}</span>`;
  }

  const columns: ColumnDefinition[] = $derived.by(() => {
    const indent = props.indent;

    return [
      {
        title: "Account",
        field: "group",
        headerSort: false,
        width: 300,
        formatter: (cell) => {
          const account = cell.getValue();
          if (indent) {
            return accountName(account, cell.getData().indent);
          }
          return account;
        },
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
    ];
  });

  let filteredBreakdowns = $derived.by(() => {
    if (!props.breakdowns) return [];
    let values = Object.values(props.breakdowns);
    if (props.filterInactive) {
      values = values.filter((i) => !i.inactive);
    }
    if (props.filterZero) {
      values = values.filter((i) => i.marketAmount !== 0);
    }
    return values;
  });

  let tree = $derived(buildTree(filteredBreakdowns, (i) => i.group));

  let displayBreakdowns = $derived.by(() => {
    if (props.indent) return tree;
    return filteredBreakdowns;
  });
</script>

{#if props.indent}
  <Table data={tree} tree={true} {columns} />
{:else}
  <Table data={displayBreakdowns} {columns} />
{/if}
