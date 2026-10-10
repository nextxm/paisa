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
      { label: "Heatmap", href: "/heatmap" },
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

  test("selects expenses Heatmap child for /expense/heatmap route", () => {
    const selection = resolveNavbarSelectionTyped(links, "/expense/heatmap");

    expect(selection.selectedLink?.label).toBe("Expenses");
    expect(selection.selectedSubLink?.label).toBe("Heatmap");
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

describe("grouped navbar selection", () => {
  const groupedLinks: Link[] = [
    { label: "Dashboard", href: "/" },
    {
      label: "Cash Flow",
      href: "/cash_flow",
      children: [
        {
          label: "Cash Flow",
          href: "",
          children: [
            { label: "Monthly Cash Flow", href: "/monthly" },
            { label: "Yearly Cash Flow", href: "/yearly" },
            { label: "Income Statement", href: "/income_statement" }
          ]
        },
        {
          label: "Expenses & Budget",
          href: "/../expense",
          children: [
            { label: "Budget Tracker", href: "/budget" },
            { label: "Monthly Expenses", href: "/monthly" },
            { label: "Yearly Expenses", href: "/yearly" },
            { label: "Expense Breakdown", href: "/sankey" },
            { label: "Heatmap", href: "/heatmap" },
            { label: "YoY", href: "/yoy" },
            { label: "MoM", href: "/mom" }
          ]
        },
        {
          label: "Income & Returns",
          href: "/../income",
          children: [
            { label: "Income Timeline", href: "" },
            { label: "Investment Income", href: "/investment" }
          ]
        },
        {
          label: "Flows & Recurring",
          href: "",
          children: [
            { label: "Money Flow", href: "/sankey" },
            { label: "Recurring Bills", href: "/recurring" }
          ]
        }
      ]
    },
    {
      label: "Net Worth",
      href: "/assets",
      children: [
        {
          label: "Net Worth",
          href: "",
          children: [
            { label: "Net Worth History", href: "/networth" },
            { label: "Asset Balances", href: "/balance" }
          ]
        },
        {
          label: "Investments & Portfolio",
          href: "",
          children: [
            { label: "Holdings & Valuation", href: "/investment" },
            { label: "Performance & Gain", href: "/gain" },
            { label: "Target Asset Allocation", href: "/allocation" },
            { label: "Underlying Portfolio", href: "/portfolio" }
          ]
        },
        {
          label: "Liabilities & Debt",
          href: "/../liabilities",
          children: [
            { label: "Liabilities Balances", href: "/balance" },
            { label: "Credit Cards", href: "/credit_cards" },
            { label: "Loan Repayments", href: "/repayment" },
            { label: "Interest Analysis", href: "/interest" }
          ]
        }
      ]
    },
    {
      label: "Planning",
      href: "/planning",
      children: [
        {
          label: "Projections & Life Plan",
          href: "",
          children: [
            { label: "Life Plan & DNA", href: "/life" },
            { label: "Net Worth Projections", href: "/projection" },
            { label: "What-If Scenarios", href: "/life/whatif" },
            { label: "Drawdown Strategy", href: "/life/drawdown" }
          ]
        },
        {
          label: "Financial Goals",
          href: "",
          children: [
            { label: "Retirement & Savings Goals", href: "/goals" },
            { label: "Milestone Life Goals", href: "/life/goals" }
          ]
        }
      ]
    },
    {
      label: "Tools",
      href: "/more",
      children: [
        { label: "Data Health", href: "/doctor" },
        { label: "Configuration", href: "/config" },
        { label: "Sheets", href: "/sheets" },
        { label: "System Logs", href: "/logs" },
        { label: "About", href: "/about" }
      ]
    }
  ];

  test("resolves 3-level grouped cash flow monthly", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/cash_flow/monthly");
    expect(selection.selectedLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubSubLink?.label).toBe("Monthly Cash Flow");
  });

  test("resolves cross-rooted expenses budget under Cash Flow", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/expense/budget");
    expect(selection.selectedLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubLink?.label).toBe("Expenses & Budget");
    expect(selection.selectedSubSubLink?.label).toBe("Budget Tracker");
  });

  test("resolves cross-rooted income timeline under Cash Flow", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/income");
    expect(selection.selectedLink?.label).toBe("Cash Flow");
    expect(selection.selectedSubLink?.label).toBe("Income & Returns");
    expect(selection.selectedSubSubLink?.label).toBe("Income Timeline");
  });

  test("resolves cross-rooted liabilities under Net Worth", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/liabilities/credit_cards");
    expect(selection.selectedLink?.label).toBe("Net Worth");
    expect(selection.selectedSubLink?.label).toBe("Liabilities & Debt");
    expect(selection.selectedSubSubLink?.label).toBe("Credit Cards");
  });

  test("resolves investments allocation under Net Worth", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/assets/allocation");
    expect(selection.selectedLink?.label).toBe("Net Worth");
    expect(selection.selectedSubLink?.label).toBe("Investments & Portfolio");
    expect(selection.selectedSubSubLink?.label).toBe("Target Asset Allocation");
  });

  test("resolves planning life drawdown", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/planning/life/drawdown");
    expect(selection.selectedLink?.label).toBe("Planning");
    expect(selection.selectedSubLink?.label).toBe("Projections & Life Plan");
    expect(selection.selectedSubSubLink?.label).toBe("Drawdown Strategy");
  });

  test("resolves tools /more/doctor", () => {
    const selection = resolveNavbarSelectionTyped(groupedLinks, "/more/doctor");
    expect(selection.selectedLink?.label).toBe("Tools");
    expect(selection.selectedSubLink?.label).toBe("Data Health");
    expect(selection.selectedSubSubLink).toBeNull();
  });
});
