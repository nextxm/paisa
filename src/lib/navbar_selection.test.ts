import { describe, expect, test } from "bun:test";
import { resolveNavbarSelectionTyped, type NavLink } from "./navbar_selection";

interface Link extends NavLink {
  label: string;
  children?: Link[];
}

const links: Link[] = [
  { label: "Dashboard", href: "/" },
  {
    label: "Cash Flow",
    href: "/cash_flow",
    children: [
      { label: "Monthly", href: "/monthly" },
      {
        label: "Recurring",
        href: "/recurring",
        children: [{ label: "Upcoming", href: "/upcoming" }]
      }
    ]
  },
  {
    label: "Expenses",
    href: "/expense",
    children: [
      { label: "Monthly", href: "/monthly" },
      { label: "Yearly", href: "/yearly" },
      { label: "Budget", href: "/budget" },
      { label: "Flow", href: "/sankey" },
      { label: "YoY", href: "/yoy" },
      { label: "MoM", href: "/mom" }
    ]
  },
  {
    label: "Income",
    href: "/income",
    children: [
      { label: "Timeline", href: "" },
      { label: "Investment", href: "/investment" }
    ]
  },
  {
    label: "Planning",
    href: "/planning",
    children: [
      { label: "Goals", href: "/goals" },
      {
        label: "Tax",
        href: "/tax",
        children: [{ label: "Harvest", href: "/harvest" }]
      }
    ]
  }
];

describe("navbar selection", () => {
  test("selects dashboard for root path", () => {
    const selection = resolveNavbarSelectionTyped(links, "/");

    expect(selection.selectedLink?.label).toBe("Dashboard");
    expect(selection.selectedSubLink).toBeNull();
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects parent and child for nested path", () => {
    const selection = resolveNavbarSelectionTyped(links, "/cash_flow/monthly");

    expect(selection.selectedLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubLink?.label).toBe("Monthly");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects parent, child, and grandchild when available", () => {
    const selection = resolveNavbarSelectionTyped(links, "/cash_flow/recurring/upcoming");

    expect(selection.selectedLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubLink?.label).toBe("Recurring");
    expect(selection.selectedSubSubLink?.label).toBe("Upcoming");
  });

  test("returns empty selection for empty path", () => {
    const selection = resolveNavbarSelectionTyped(links, "");

    expect(selection.selectedLink).toBeNull();
    expect(selection.selectedSubLink).toBeNull();
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects income timeline child for /income route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/income");

    expect(selection.selectedLink?.label).toBe("Income");
    expect(selection.selectedSubLink?.label).toBe("Timeline");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects income investment child for /income/investment route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/income/investment");

    expect(selection.selectedLink?.label).toBe("Income");
    expect(selection.selectedSubLink?.label).toBe("Investment");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects expenses YoY child for /expense/yoy route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/expense/yoy");

    expect(selection.selectedLink?.label).toBe("Expenses");
    expect(selection.selectedSubLink?.label).toBe("YoY");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects expenses MoM child for /expense/mom route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/expense/mom");

    expect(selection.selectedLink?.label).toBe("Expenses");
    expect(selection.selectedSubLink?.label).toBe("MoM");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects planning goals for /planning/goals route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/planning/goals");

    expect(selection.selectedLink?.label).toBe("Planning");
    expect(selection.selectedSubLink?.label).toBe("Goals");
    expect(selection.selectedSubSubLink).toBeNull();
  });

  test("selects planning tax harvest hierarchy for /planning/tax/harvest route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/planning/tax/harvest");

    expect(selection.selectedLink?.label).toBe("Planning");
    expect(selection.selectedSubLink?.label).toBe("Tax");
    expect(selection.selectedSubSubLink?.label).toBe("Harvest");
  });
});
