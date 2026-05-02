import { describe, expect, test } from "bun:test";

import {
  MOBILE_CONFIG_SIDEBAR_MAX_WIDTH,
  shouldCollapseConfigSidebarOnNavigate
} from "./config_sidebar";

describe("config sidebar mobile navigation", () => {
  test("collapses sidebar at mobile widths", () => {
    expect(shouldCollapseConfigSidebarOnNavigate(MOBILE_CONFIG_SIDEBAR_MAX_WIDTH)).toBe(true);
    expect(shouldCollapseConfigSidebarOnNavigate(480)).toBe(true);
  });

  test("does not collapse sidebar at desktop widths", () => {
    expect(shouldCollapseConfigSidebarOnNavigate(MOBILE_CONFIG_SIDEBAR_MAX_WIDTH + 1)).toBe(false);
  });

  test("does not collapse sidebar when viewport width is unavailable", () => {
    expect(shouldCollapseConfigSidebarOnNavigate(undefined)).toBe(false);
    expect(shouldCollapseConfigSidebarOnNavigate(null)).toBe(false);
  });
});
