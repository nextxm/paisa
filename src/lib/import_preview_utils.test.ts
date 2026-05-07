import { describe, expect, test } from "bun:test";
import {
  defaultIncludedFromValidation,
  filterSelectedRows,
  toCSVContent
} from "./import_preview_utils";

describe("import_preview_utils", () => {
  test("toCSVContent serializes matrix data", () => {
    expect(
      toCSVContent([
        ["A", "B"],
        ["1", "2"]
      ])
    ).toContain("A,B");
    expect(
      toCSVContent([
        ["A", "B"],
        ["1", "2"]
      ])
    ).toContain("1,2");
  });

  test("filterSelectedRows keeps only checked rows", () => {
    expect(filterSelectedRows(["r1", "r2", "r3"], [true, false, true])).toEqual(["r1", "r3"]);
  });

  test("defaultIncludedFromValidation includes only valid rows by default", () => {
    expect(
      defaultIncludedFromValidation([{ valid: true }, { valid: false }, { valid: true }])
    ).toEqual([true, false, true]);
  });
});
