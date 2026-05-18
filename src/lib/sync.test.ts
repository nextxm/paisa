import { mock, describe, expect, test, beforeEach, afterEach } from "bun:test";
import type { Job } from "./utils";

const ajaxMock = mock(async () => ({ job_id: "job-from-sync" }));
mock.module("./utils", async () => ({ ajax: ajaxMock }));
mock.module("$lib/utils", async () => ({ ajax: ajaxMock }));
mock.module("$app/navigation", () => ({ goto: async () => {} }));

const toastMock = mock((_arg: { message: string; type: string; duration?: number }) => {});
mock.module("bulma-toast", () => ({ toast: toastMock }));

const {
  startPolling,
  sync,
  ensureJobsStream,
  clearToastedFailures,
  clearSyncStateForTests,
  INITIAL_RECONNECT_DELAY_MS,
  MAX_RECONNECT_DELAY_MS
} = await import("./sync");
const { jobs } = await import("./stores/jobs");

function makeJob(overrides: Partial<Job> = {}): Job {
  return {
    id: "test-job-1",
    status: "running",
    created_at: "2024-01-01T00:00:00Z",
    ...overrides
  };
}

function makeSSEEvent(job: Job): string {
  return `event: job\ndata: ${JSON.stringify(job)}\n\n`;
}

function makeStream(events: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      events.forEach((event) => controller.enqueue(encoder.encode(event)));
      controller.close();
    }
  });
}

describe("sync SSE stream", () => {
  beforeEach(() => {
    jobs.reset();
    toastMock.mockClear();
    ajaxMock.mockClear();
    clearToastedFailures();
    clearSyncStateForTests();
    localStorage.setItem("token", "test-token");
  });

  afterEach(() => {
    clearSyncStateForTests();
  });

  test("exports reconnect constants", () => {
    expect(INITIAL_RECONNECT_DELAY_MS).toBe(1000);
    expect(MAX_RECONNECT_DELAY_MS).toBe(10000);
  });

  test("updates store from SSE events and invokes onTerminal", async () => {
    const running = makeJob({ status: "running" });
    const completed = makeJob({ status: "completed", items_completed: 3, total_items: 3 });

    const fetchMock = mock(
      async () =>
        new Response(makeStream([makeSSEEvent(running), makeSSEEvent(completed)]), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
    );
    globalThis.fetch = fetchMock as any;

    let terminal: Job | null = null;
    startPolling("test-job-1", (job) => {
      terminal = job;
    });

    await Bun.sleep(30);

    const snap = jobs.snapshot();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(snap["test-job-1"].status).toBe("completed");
    expect(terminal?.status).toBe("completed");
  });

  test("shows failed toast only once per job id", async () => {
    const failed = makeJob({ status: "failed", error: "ledger parse error" });

    const fetchMock = mock(
      async () =>
        new Response(makeStream([makeSSEEvent(failed)]), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
    );
    globalThis.fetch = fetchMock as any;

    startPolling("test-job-1");
    await Bun.sleep(30);
    startPolling("test-job-1");
    await Bun.sleep(10);

    expect(toastMock).toHaveBeenCalledTimes(1);
  });

  test("sync seeds pending job and starts stream", async () => {
    const fetchMock = mock(
      async () =>
        new Response(makeStream([]), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
    );
    globalThis.fetch = fetchMock as any;

    const jobId = await sync({ journal: true });
    await Bun.sleep(10);

    expect(jobId).toBe("job-from-sync");
    expect(ajaxMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(jobs.snapshot()["job-from-sync"].status).toBe("pending");
  });

  test("ensureJobsStream does nothing without auth token", async () => {
    localStorage.removeItem("token");
    const fetchMock = mock(
      async () =>
        new Response(makeStream([]), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
    );
    globalThis.fetch = fetchMock as any;

    await ensureJobsStream();

    expect(fetchMock).not.toHaveBeenCalled();
  });
});
