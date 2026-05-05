<script lang="ts">
  import { type AssetBreakdown, buildTree } from "$lib/utils";
  import _ from "lodash";
  import Table from "./Table.svelte";
  import type { ColumnDefinition } from "tabulator-tables";
  import {
    accountName,
    formatCurrencyChange,
    formatOriginalBalances,
    indendedAssetAccountName,
    nonZeroCurrency,
    nonZeroCurrencyLink,
    nonZeroFloatChange,
    nonZeroPercentageChange
  } from "$lib/table_formatters";

  let {
    breakdowns,
    indent = true
  }: { breakdowns: Record<string, AssetBreakdown>; indent?: boolean } = $props();

  const columns: ColumnDefinition[] = $derived([
    {
      title: "Account",
      field: "group",
      formatter: indent ? indendedAssetAccountName : accountName,
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

  let tree: AssetBreakdown[] = $state([]);
  $effect(() => {
    if (breakdowns) {
      tree = buildTree(Object.values(breakdowns), (i) => i.group);
    }
  });
</script>

{#if indent}
  <Table data={tree} tree {columns} />
{:else}
  <Table data={Object.values(breakdowns)} {columns} />
{/if}
