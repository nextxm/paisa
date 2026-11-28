import { describe, expect, test } from "bun:test";
import { ALL_SECTIONS, CONFIG_GROUPS, DEFAULT_SECTION_ID } from "./config_sections";

describe("config_sections", () => {
  test("DEFAULT_SECTION_ID exists in ALL_SECTIONS", () => {
    expect(ALL_SECTIONS.some((s) => s.id === DEFAULT_SECTION_ID)).toBe(true);
  });

  test("all section ids and schemaKeys are unique", () => {
    const ids = ALL_SECTIONS.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);

    const allKeys = ALL_SECTIONS.flatMap((s) => s.schemaKeys);
    expect(new Set(allKeys).size).toBe(allKeys.length);
  });

  test("CONFIG_GROUPS flatten to ALL_SECTIONS", () => {
    expect(CONFIG_GROUPS.length).toBeGreaterThan(0);
    expect(ALL_SECTIONS.length).toBe(
      CONFIG_GROUPS.reduce((sum, group) => sum + group.sections.length, 0)
    );
  });
});
