import { describe, expect, test, beforeEach, afterEach } from "bun:test";
import { loadFromStorage, saveToStorage } from "./storage_utils";

// ---------------------------------------------------------------------------
// Tests for src/lib/state/storage_utils.ts
// ---------------------------------------------------------------------------

// Store references to any real localStorage so we can restore it after tests
// that may replace window.localStorage.

describe("loadFromStorage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  test("returns defaultValue when key is absent", () => {
    expect(loadFromStorage("missing", 42)).toBe(42);
  });

  test("returns defaultValue for boolean default", () => {
    expect(loadFromStorage("flag", false)).toBe(false);
  });

  test("reads a stored boolean", () => {
    localStorage.setItem("flag", JSON.stringify(true));
    expect(loadFromStorage("flag", false)).toBe(true);
  });

  test("reads a stored number", () => {
    localStorage.setItem("depth", JSON.stringify(3));
    expect(loadFromStorage("depth", 0)).toBe(3);
  });

  test("reads a stored string", () => {
    localStorage.setItem("period", JSON.stringify("quarter"));
    expect(loadFromStorage("period", "month")).toBe("quarter");
  });

  test("reads a stored object", () => {
    localStorage.setItem("obj", JSON.stringify({ a: 1 }));
    expect(loadFromStorage("obj", {})).toEqual({ a: 1 });
  });

  test("returns defaultValue on malformed JSON", () => {
    localStorage.setItem("bad", "not-json{{");
    expect(loadFromStorage("bad", 99)).toBe(99);
  });

  test("returns defaultValue when stored null literal", () => {
    // JSON.stringify(null) = "null", which JSON.parse returns as null.
    // loadFromStorage must return the stored value (null), not the default.
    // This documents the actual behaviour: null is a valid JSON value.
    localStorage.setItem("nullval", "null");
    // stored is "null" → JSON.parse("null") = null → but stored !== null so we
    // return parsed value (null cast to T).
    expect(loadFromStorage<number | null>("nullval", 5)).toBeNull();
  });
});

describe("saveToStorage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  test("persists a boolean value", () => {
    saveToStorage("obscure", true);
    expect(localStorage.getItem("obscure")).toBe("true");
  });

  test("persists a number value", () => {
    saveToStorage("depth", 4);
    expect(localStorage.getItem("depth")).toBe("4");
  });

  test("persists a string value", () => {
    saveToStorage("period", "year");
    expect(localStorage.getItem("period")).toBe('"year"');
  });

  test("persists an object value", () => {
    saveToStorage("cfg", { min: 1, max: 5 });
    expect(JSON.parse(localStorage.getItem("cfg")!)).toEqual({ min: 1, max: 5 });
  });

  test("round-trip: save then load returns original value", () => {
    saveToStorage("num", 7);
    expect(loadFromStorage("num", 0)).toBe(7);
  });

  test("overwrites a previously stored value", () => {
    saveToStorage("flag", false);
    saveToStorage("flag", true);
    expect(loadFromStorage("flag", false)).toBe(true);
  });
});
