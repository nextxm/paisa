<script lang="ts">
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";

  const tabs: SectionTab[] = [
    {
      label: "Monthly",
      href: "/cash_flow/monthly",
      icon: "fa-calendar-days",
      match: (path) => path.startsWith("/cash_flow/monthly")
    },
    {
      label: "Yearly",
      href: "/cash_flow/yearly",
      icon: "fa-calendar",
      match: (path) => path.startsWith("/cash_flow/yearly")
    },
    {
      label: "Income Statement",
      href: "/cash_flow/income_statement",
      icon: "fa-file-invoice-dollar",
      match: (path) => path.startsWith("/cash_flow/income_statement")
    },
    {
      label: "Money Flow (Sankey)",
      href: "/cash_flow/sankey",
      icon: "fa-diagram-project",
      match: (path) => path.startsWith("/cash_flow/sankey")
    },
    {
      label: "Income Timeline",
      href: "/income",
      icon: "fa-arrow-trend-up",
      match: (path) => path === "/income"
    }
  ];
</script>

<SectionTabs {tabs} />
