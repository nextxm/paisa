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
    label: "Income",
    href: "/income",
    children: [
      { label: "Timeline", href: "" },
      { label: "Investment", href: "/investment" }
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
});
