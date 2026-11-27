<script lang="ts">
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const tabs: SectionTab[] = [
    {
      label: "Balances",
      href: "/liabilities/balance",
      icon: "fa-scale-balanced",
      match: (path) => path.startsWith("/liabilities/balance")
    },
    {
      label: "Credit Cards",
      href: "/liabilities/credit_cards",
      icon: "fa-credit-card",
      match: (path) => path.startsWith("/liabilities/credit_cards")
    },
    {
      label: "Repayment",
      href: "/liabilities/repayment",
      icon: "fa-calendar-check",
      match: (path) => path.startsWith("/liabilities/repayment")
    },
    {
      label: "Interest",
      href: "/liabilities/interest",
      icon: "fa-percent",
      match: (path) => path.startsWith("/liabilities/interest")
    }
  ];
</script>

<SectionTabs {tabs} />

{@render children()}
