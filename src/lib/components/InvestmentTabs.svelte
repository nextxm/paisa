<script lang="ts">
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";

  const tabs: SectionTab[] = [
    {
      label: "Holdings & Valuation",
      href: "/assets/investment",
      icon: "fa-briefcase",
      match: (path) => path.startsWith("/assets/investment")
    },
    {
      label: "Gain / XIRR",
      href: "/assets/gain",
      icon: "fa-chart-line",
      match: (path) => path.startsWith("/assets/gain")
    },
    {
      label: "Allocation",
      href: "/assets/allocation",
      icon: "fa-chart-pie",
      match: (path) => path.startsWith("/assets/allocation")
    },
    {
      label: "Underlying Portfolio",
      href: "/assets/portfolio",
      icon: "fa-layer-group",
      match: (path) => path.startsWith("/assets/portfolio")
    },
    {
      label: "Investment Income",
      href: "/income/investment",
      icon: "fa-coins",
      match: (path) => path.startsWith("/income/investment")
    }
  ];
</script>

<SectionTabs {tabs} />
