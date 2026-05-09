import { mock, describe, expect, test, beforeEach } from "bun:test";
import type { Job } from "./utils";

// ---------------------------------------------------------------------------
// Module mocks — must be declared before any dynamic imports that load
// the modules being mocked.
// ---------------------------------------------------------------------------

// Mock ./utils (and its $lib/utils alias) so that loading sync.ts does NOT
// trigger the utils.ts → ../store → $lib/stores/jobs resolution chain. That
// chain can fail when other test files have already cached store.ts in a
// broken state (no $lib/stores/jobs alias registered). The only runtime
// export from ./utils that sync.ts uses is `ajax`; all tests inject
// `fetchJob`, so the real ajax is never called.
mock.module("./utils", async () => {
  return {
    ajax: async () => {
      throw new Error("ajax not mocked");
    }
  };
});
mock.module("$lib/utils", async () => {
  return {
    ajax: async () => {
      throw new Error("ajax not mocked");
    }
  };
});

mock.module("$app/navigation", () => ({
  goto: async () => {}
}));

const toastMock = mock((_arg: { message: string; type: string; duration?: number }) => {});
mock.module("bulma-toast", () => ({ toast: toastMock }));

// Import the module under test after mocks are in place.
const { startPolling, clearToastedFailures, POLL_INTERVAL_MS, MAX_CONSECUTIVE_ERRORS, MAX_POLLS } =
  await import("./sync");

// Import the real jobs store — same module instance that sync.ts uses.
const { jobs } = await import("./stores/jobs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeJob(overrides: Partial<Job> = {}): Job {
  return {
    id: "test-job-1",
    status: "running",
    created_at: "2024-01-01T00:00:00Z",
    ...overrides
  };
}

/** Poll options shared across most tests: no real HTTP, instant scheduling. */
const fastOptions = { intervalMs: 1, maxPolls: 50, maxErrors: 3 };

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("startPolling — constants", () => {
  test("POLL_INTERVAL_MS is 2000", () => {
    expect(POLL_INTERVAL_MS).toBe(2000);
  });

  test("MAX_CONSECUTIVE_ERRORS is 5", () => {
    expect(MAX_CONSECUTIVE_ERRORS).toBe(5);
  });

  test("MAX_POLLS is 150", () => {
    expect(MAX_POLLS).toBe(150);
  });
});

describe("startPolling — store updates and terminal detection", () => {
  beforeEach(() => {
    jobs.reset();
    toastMock.mockClear();
    clearToastedFailures();
  });

  test("calls onTerminal with completed job when status is completed", async () => {
    let terminalJob: Job | null = null;
    const fetchJob = mock(async (_id: string): Promise<Job> => makeJob({ status: "completed" }));

    startPolling("test-job-1", (j) => (terminalJob = j), {
      ...fastOptions,
      fetchJob
    });

    await Bun.sleep(50);

    expect(terminalJob).not.toBeNull();
    expect(terminalJob!.status).toBe("completed");
  });

  test("calls onTerminal with failed job when status is failed", async () => {
    let terminalJob: Job | null = null;
    const fetchJob = mock(
      async (_id: string): Promise<Job> => makeJob({ status: "failed", error: "parse error" })
    );

    startPolling("test-job-1", (j) => (terminalJob = j), {
      ...fastOptions,
      fetchJob
    });

    await Bun.sleep(50);

    expect(terminalJob).not.toBeNull();
    expect(terminalJob!.status).toBe("failed");
  });

  test("updates the jobs store on each successful poll", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      return makeJob({ status: callCount < 3 ? "running" : "completed" });
    };

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    const snap = jobs.snapshot();
    expect(snap["test-job-1"]).toBeDefined();
    expect(snap["test-job-1"].status).toBe("completed");
  });

  test("stops polling once job reaches completed state", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      return makeJob({ status: callCount < 2 ? "running" : "completed" });
    };

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    // callCount should stabilise at 2 (one "running", one "completed")
    const finalCount = callCount;
    await Bun.sleep(20);
    expect(callCount).toBe(finalCount); // no further polls
  });

  test("stops polling once job reaches failed state", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      return makeJob({ status: callCount < 2 ? "running" : "failed" });
    };

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    const finalCount = callCount;
    await Bun.sleep(20);
    expect(callCount).toBe(finalCount);
  });

  test("shows failure toast when job fails", async () => {
    const fetchJob = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ status: "failed", error: "ledger parse error" })
    );

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    expect(toastMock).toHaveBeenCalledTimes(1);
    const callArg = toastMock.mock.calls[0][0];
    expect(callArg.type).toBe("is-danger");
    expect(callArg.message).toContain("Sync failed");
  });

  test("does not show failure toast when job completes", async () => {
    const fetchJob = mock(async (_id: string): Promise<Job> => makeJob({ status: "completed" }));

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    expect(toastMock).not.toHaveBeenCalled();
  });
});

describe("startPolling — error handling and retry", () => {
  beforeEach(() => {
    jobs.reset();
    toastMock.mockClear();
    clearToastedFailures();
  });

  test("retries after a transient network error", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      if (callCount === 1) throw new Error("network error");
      return makeJob({ status: "completed" });
    };

    let terminated = false;
    startPolling("test-job-1", () => (terminated = true), { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    expect(terminated).toBe(true);
    expect(callCount).toBeGreaterThanOrEqual(2);
  });

  test("stops after maxErrors consecutive network errors", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      throw new Error("server unreachable");
    };

    startPolling("test-job-1", undefined, { ...fastOptions, maxErrors: 3, fetchJob });

    await Bun.sleep(100);

    // With maxErrors=3, polling stops after 3 failures.
    expect(callCount).toBe(3);
  });

  test("resets consecutive error count after a successful poll", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      // Error on first two calls, then succeed
      if (callCount <= 2) throw new Error("transient");
      return makeJob({ status: "completed" });
    };

    let terminated = false;
    startPolling("test-job-1", () => (terminated = true), {
      ...fastOptions,
      maxErrors: 3,
      fetchJob
    });

    await Bun.sleep(100);

    // Should have completed despite two errors (consecutive count was below 3).
    expect(terminated).toBe(true);
  });

  test("stops after maxPolls total polls without reaching terminal state", async () => {
    let callCount = 0;
    const fetchJob = async (_id: string): Promise<Job> => {
      callCount++;
      return makeJob({ status: "running" }); // never terminates
    };

    startPolling("test-job-1", undefined, { ...fastOptions, maxPolls: 5, fetchJob });

    await Bun.sleep(100);

    expect(callCount).toBe(5);
  });

  test("stops polling and marks job as failed on 404 error", async () => {
    const fetchJob = mock(async (_id: string): Promise<Job> => {
      const err = new Error("Not found") as any;
      err.status = 404;
      throw err;
    });

    // Seed the store with a running job so we can see it change to failed
    jobs.upsert(makeJob({ id: "test-job-1", status: "running" }));

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });

    await Bun.sleep(50);

    const snap = jobs.snapshot();
    expect(snap["test-job-1"].status).toBe("failed");
    expect(snap["test-job-1"].error).toContain("Job not found on server");

    // Should stop polling immediately (only 1 call)
    expect(fetchJob).toHaveBeenCalledTimes(1);
  });
});

describe("startPolling — failure toast deduplication", () => {
  beforeEach(() => {
    jobs.reset();
    toastMock.mockClear();
    clearToastedFailures();
  });

  test("shows toast only once when startPolling is called twice for the same failed job", async () => {
    const fetchJob = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ status: "failed", error: "ledger parse error" })
    );

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);

    // Second call for the same job (simulates re-mount or navigation)
    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);

    expect(toastMock).toHaveBeenCalledTimes(1);
  });

  test("shows a toast for each distinct failed job", async () => {
    const fetchJobA = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ id: "job-a", status: "failed", error: "error A" })
    );
    const fetchJobB = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ id: "job-b", status: "failed", error: "error B" })
    );

    startPolling("job-a", undefined, { ...fastOptions, fetchJob: fetchJobA });
    startPolling("job-b", undefined, { ...fastOptions, fetchJob: fetchJobB });
    await Bun.sleep(50);

    expect(toastMock).toHaveBeenCalledTimes(2);
  });

  test("clearToastedFailures resets deduplication so a second poll shows a new toast", async () => {
    const fetchJob = mock(
      async (_id: string): Promise<Job> => makeJob({ status: "failed", error: "oops" })
    );

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);
    expect(toastMock).toHaveBeenCalledTimes(1);

    // After clearing, the same job ID can produce a new toast
    clearToastedFailures();
    toastMock.mockClear();

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);
    expect(toastMock).toHaveBeenCalledTimes(1);
  });

  test("toast message includes the failure reason", async () => {
    const fetchJob = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ status: "failed", error: "Ledger parse error at line 42" })
    );

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);

    expect(toastMock).toHaveBeenCalledTimes(1);
    const callArg = toastMock.mock.calls[0][0];
    expect(callArg.message).toContain("Ledger parse error at line 42");
  });

  test("toast message escapes HTML in the failure reason", async () => {
    const fetchJob = mock(
      async (_id: string): Promise<Job> =>
        makeJob({ status: "failed", error: "<script>alert('xss')</script>" })
    );

    startPolling("test-job-1", undefined, { ...fastOptions, fetchJob });
    await Bun.sleep(50);

    expect(toastMock).toHaveBeenCalledTimes(1);
    const callArg = toastMock.mock.calls[0][0];
    expect(callArg.message).not.toContain("<script>");
    expect(callArg.message).toContain("&lt;script&gt;");
  });
});
