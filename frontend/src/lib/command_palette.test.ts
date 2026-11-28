import { describe, expect, test } from "bun:test";
import { buildCommandPaletteTargets, nextCommandIndex } from "./command_palette";

describe("command_palette helpers", () => {
  test("buildCommandPaletteTargets flattens pages, tabs, and accounts in display order", () => {
    const targets = buildCommandPaletteTargets(
      [{ href: "/" }, { href: "/transactions" }],
      [{ href: "/wealth?tab=allocation" }],
      ["Assets:Bank:HDFC"]
    );
    expect(targets).toEqual([
      "/",
      "/transactions",
      "/wealth?tab=allocation",
      "/accounts/Assets%3ABank%3AHDFC"
    ]);
  });

  test("nextCommandIndex wraps around forward and backward", () => {
    expect(nextCommandIndex(0, 0, 1)).toBe(0);
    expect(nextCommandIndex(0, 0, -1)).toBe(0);
    expect(nextCommandIndex(0, 4, 1)).toBe(1);
    expect(nextCommandIndex(3, 4, 1)).toBe(0);
    expect(nextCommandIndex(0, 4, -1)).toBe(3);
    expect(nextCommandIndex(2, 4, -1)).toBe(1);
    expect(nextCommandIndex(10, 3, -1)).toBe(1);
  });
});
