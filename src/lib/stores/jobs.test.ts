import { describe, expect, test, beforeEach } from "bun:test";
import { get } from "svelte/store";
import { jobs, jobsList, isJobRunning } from "./jobs";
import type { Job } from "$lib/utils";

function makeJob(overrides: Partial<Job> = {}): Job {
  return {
    id: "test-id-1",
    status: "pending",
    created_at: "2024-01-01T00:00:00Z",
    ...overrides
  };
}

describe("jobs store", () => {
  beforeEach(() => {
    jobs.reset();
  });

  test("starts empty", () => {
    expect(get(jobs)).toEqual({});
  });

  test("upsert inserts a new job", () => {
    const job = makeJob();
    jobs.upsert(job);
    expect(get(jobs)["test-id-1"]).toEqual(job);
  });

  test("upsert replaces an existing job", () => {
    jobs.upsert(makeJob({ status: "pending" }));
    jobs.upsert(makeJob({ status: "running" }));
    expect(get(jobs)["test-id-1"].status).toBe("running");
  });

  test("upsert stores multiple distinct jobs", () => {
    jobs.upsert(makeJob({ id: "a" }));
    jobs.upsert(makeJob({ id: "b" }));
    const state = get(jobs);
    expect(Object.keys(state)).toHaveLength(2);
    expect(state["a"]).toBeDefined();
    expect(state["b"]).toBeDefined();
  });

  test("updateById merges partial fields", () => {
    jobs.upsert(makeJob({ status: "pending" }));
    const updated = jobs.updateById("test-id-1", { status: "running" });
    expect(updated).toBe(true);
    expect(get(jobs)["test-id-1"].status).toBe("running");
  });

  test("updateById preserves fields not in partial", () => {
    const job = makeJob({ details: ["step1 ok"] });
    jobs.upsert(job);
    jobs.updateById("test-id-1", { status: "completed" });
    const result = get(jobs)["test-id-1"];
    expect(result.status).toBe("completed");
    expect(result.details).toEqual(["step1 ok"]);
  });

  test("updateById returns false and is a no-op for unknown id", () => {
    jobs.upsert(makeJob());
    const updated = jobs.updateById("unknown-id", { status: "completed" });
    expect(updated).toBe(false);
    expect(get(jobs)["test-id-1"].status).toBe("pending");
  });

  test("reset removes all jobs", () => {
    jobs.upsert(makeJob({ id: "a" }));
    jobs.upsert(makeJob({ id: "b" }));
    jobs.reset();
    expect(get(jobs)).toEqual({});
  });

  test("snapshot returns current state synchronously", () => {
    const job = makeJob();
    jobs.upsert(job);
    expect(jobs.snapshot()).toEqual({ "test-id-1": job });
  });
});

describe("jobsList derived store", () => {
  beforeEach(() => {
    jobs.reset();
  });

  test("returns empty array when store is empty", () => {
    expect(get(jobsList)).toEqual([]);
  });

  test("returns jobs sorted by created_at ascending", () => {
    jobs.upsert(makeJob({ id: "newer", created_at: "2024-06-01T00:00:00Z" }));
    jobs.upsert(makeJob({ id: "older", created_at: "2024-01-01T00:00:00Z" }));
    jobs.upsert(makeJob({ id: "middle", created_at: "2024-03-01T00:00:00Z" }));
    const list = get(jobsList);
    expect(list.map((j) => j.id)).toEqual(["older", "middle", "newer"]);
  });

  test("updates when a job is upserted", () => {
    expect(get(jobsList)).toHaveLength(0);
    jobs.upsert(makeJob());
    expect(get(jobsList)).toHaveLength(1);
  });
});

describe("isJobRunning derived store", () => {
  beforeEach(() => {
    jobs.reset();
  });

  test("false when store is empty", () => {
    expect(get(isJobRunning)).toBe(false);
  });

  test("true when a job is pending", () => {
    jobs.upsert(makeJob({ status: "pending" }));
    expect(get(isJobRunning)).toBe(true);
  });

  test("true when a job is running", () => {
    jobs.upsert(makeJob({ status: "running" }));
    expect(get(isJobRunning)).toBe(true);
  });

  test("false when all jobs are in terminal states", () => {
    jobs.upsert(makeJob({ id: "a", status: "completed" }));
    jobs.upsert(makeJob({ id: "b", status: "failed" }));
    expect(get(isJobRunning)).toBe(false);
  });

  test("true when at least one non-terminal job exists alongside terminal ones", () => {
    jobs.upsert(makeJob({ id: "a", status: "completed" }));
    jobs.upsert(makeJob({ id: "b", status: "running" }));
    expect(get(isJobRunning)).toBe(true);
  });

  test("updates to false after job moves to completed", () => {
    jobs.upsert(makeJob({ status: "running" }));
    expect(get(isJobRunning)).toBe(true);
    jobs.updateById("test-id-1", { status: "completed" });
    expect(get(isJobRunning)).toBe(false);
  });
});
