import { describe, expect, test } from "bun:test";
import type { Job } from "$lib/utils";
import { statusTagClass, statusIconClass, formatTs, formatDuration } from "./sync_history_utils";

// ---------------------------------------------------------------------------
// Tests for the SyncHistoryOverlay utility helpers (sync_history_utils.ts)
// ---------------------------------------------------------------------------

describe("SyncHistoryOverlay – statusTagClass", () => {
  test("returns is-success for completed", () => {
    expect(statusTagClass("completed")).toBe("is-success");
  });
  test("returns is-danger for failed", () => {
    expect(statusTagClass("failed")).toBe("is-danger");
  });
  test("returns is-info for running", () => {
    expect(statusTagClass("running")).toBe("is-info");
  });
  test("returns is-warning for pending", () => {
    expect(statusTagClass("pending")).toBe("is-warning");
  });
});

describe("SyncHistoryOverlay – statusIconClass", () => {
  test("completed has check icon", () => {
    expect(statusIconClass("completed")).toContain("fa-check");
  });
  test("failed has xmark icon", () => {
    expect(statusIconClass("failed")).toContain("fa-xmark");
  });
  test("running has spinner icon", () => {
    expect(statusIconClass("running")).toContain("fa-spinner");
  });
  test("pending has clock icon", () => {
    expect(statusIconClass("pending")).toContain("fa-clock");
  });
});

describe("SyncHistoryOverlay – formatTs", () => {
  test("returns em-dash for undefined", () => {
    expect(formatTs(undefined)).toBe("—");
  });
  test("formats a known UTC ISO timestamp (run with TZ=UTC)", () => {
    // 2024-03-15T08:05:00Z → "Mar 15, 2024 08:05:00" when TZ=UTC
    const result = formatTs("2024-03-15T08:05:00Z");
    expect(result).toBe("Mar 15, 2024 08:05:00");
  });
  test("formats midnight correctly (run with TZ=UTC)", () => {
    const result = formatTs("2024-01-01T00:00:00Z");
    expect(result).toBe("Jan 1, 2024 00:00:00");
  });
});

describe("SyncHistoryOverlay – formatDuration", () => {
  function makeJob(overrides: Partial<Job> = {}): Job {
    return {
      id: "j1",
      status: "completed",
      created_at: "2024-01-01T00:00:00Z",
      ...overrides
    };
  }

  test("returns em-dash when started_at is absent", () => {
    expect(formatDuration(makeJob())).toBe("—");
  });

  test("returns seconds when duration < 60s", () => {
    const job = makeJob({
      started_at: "2024-01-01T00:00:00Z",
      finished_at: "2024-01-01T00:00:30Z"
    });
    expect(formatDuration(job)).toBe("30s");
  });

  test("returns minutes and seconds when duration >= 60s", () => {
    const job = makeJob({
      started_at: "2024-01-01T00:00:00Z",
      finished_at: "2024-01-01T00:02:15Z"
    });
    expect(formatDuration(job)).toBe("2m 15s");
  });

  test("returns 0s for same start and finish", () => {
    const job = makeJob({
      started_at: "2024-01-01T00:00:00Z",
      finished_at: "2024-01-01T00:00:00Z"
    });
    expect(formatDuration(job)).toBe("0s");
  });
});
