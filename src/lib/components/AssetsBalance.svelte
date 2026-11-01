<script lang="ts">
  import { type AssetBreakdown, buildTree, isMobile } from "$lib/utils";
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

  function accountName(account: string, indent: number, mobile: boolean) {
    const parts = account.split(":");
    const name = parts[parts.length - 1];
    if (mobile) {
      return `<span class="truncate is-inline-block" style="max-width: 100%; vertical-align: bottom;" title="${account}"><a href="/accounts/${encodeURIComponent(account)}/transactions">${name}</a></span>`;
    }
    const padding = indent * 20;
    return `<span style="padding-left: ${padding}px"><a href="/accounts/${encodeURIComponent(account)}/transactions">${name}</a></span>`;
  }

  const columns: ColumnDefinition[] = $derived.by(() => {
    const indent = props.indent;
    const mobile = isMobile();

    const accountCol: ColumnDefinition = {
      title: "Account",
      field: "group",
      headerSort: false,
      width: mobile ? 140 : 300,
      maxWidth: mobile ? 155 : undefined,
      formatter: (cell) => {
        const account = cell.getValue();
        if (indent) {
          return accountName(account, cell.getData().indent, mobile);
        }
        if (mobile) {
          return `<span class="truncate is-inline-block" style="max-width: 100%; vertical-align: bottom;" title="${account}"><a href="/accounts/${encodeURIComponent(account)}/transactions">${account}</a></span>`;
        }
        return `<a href="/accounts/${encodeURIComponent(account)}/transactions">${account}</a>`;
      },
      frozen: true
    };

    const investmentCol: ColumnDefinition = {
      title: "Investment Amount",
      field: "investmentAmount",
      hozAlign: "right",
      vertAlign: "middle",
      formatter: nonZeroCurrency
    };

    const withdrawalCol: ColumnDefinition = {
      title: "Withdrawal Amount",
      field: "withdrawalAmount",
      hozAlign: "right",
      formatter: nonZeroCurrency
    };

    const unitsCol: ColumnDefinition = {
      title: "Balance Units",
      field: "balanceUnits",
      hozAlign: "right",
      formatter: nonZeroCurrencyLink
    };

    const originalCol: ColumnDefinition = {
      title: "Original Value",
      field: "originalBalances",
      hozAlign: "right",
      formatter: formatOriginalBalances
    };

    const marketCol: ColumnDefinition = {
      title: "Market Value",
      field: "marketAmount",
      hozAlign: "right",
      formatter: nonZeroCurrencyLink
    };

    const changeCol: ColumnDefinition = {
      title: "Change",
      field: "gainAmount",
      hozAlign: "right",
      formatter: formatCurrencyChange
    };

    const xirrCol: ColumnDefinition = {
      title: "XIRR",
      field: "xirr",
      hozAlign: "right",
      formatter: nonZeroFloatChange
    };

    const returnCol: ColumnDefinition = {
      title: "Absolute Return",
      field: "absoluteReturn",
      hozAlign: "right",
      formatter: nonZeroPercentageChange
    };

    if (mobile) {
      return [
        accountCol,
        marketCol,
        changeCol,
        xirrCol,
        returnCol,
        investmentCol,
        withdrawalCol,
        unitsCol,
        originalCol
      ];
    }

    return [
      accountCol,
      investmentCol,
      withdrawalCol,
      unitsCol,
      originalCol,
      marketCol,
      changeCol,
      xirrCol,
      returnCol
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
