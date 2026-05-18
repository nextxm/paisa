import * as toast from "bulma-toast";
import { ajax } from "./utils";
import { jobs } from "./stores/jobs";
import type { Job } from "./utils";

const AUTH_TOKEN_KEY = "token";

/** Set of job IDs for which a failure toast has already been shown. */
const toastedFailureIds = new Set<string>();

/** Registered terminal-state listeners per job ID. */
const terminalListeners = new Map<string, Set<(job: Job) => void>>();

/** Maximum delay between SSE reconnect attempts. */
export const MAX_RECONNECT_DELAY_MS = 10000;

/** Initial SSE reconnect delay. */
export const INITIAL_RECONNECT_DELAY_MS = 1000;

let streamAbortController: AbortController | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS;

/**
 * Clear test-only singleton state. Exported for tests.
 */
export function clearSyncStateForTests(): void {
  toastedFailureIds.clear();
  terminalListeners.clear();
  if (streamAbortController) {
    streamAbortController.abort();
  }
  streamAbortController = null;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS;
}

/**
 * Escape HTML special characters so error strings are safe to embed in toast HTML.
 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function notifyTerminal(job: Job): void {
  const listeners = terminalListeners.get(job.id);
  if (!listeners) return;
  listeners.forEach((listener) => listener(job));
  terminalListeners.delete(job.id);
}

function maybeShowFailureToast(job: Job): void {
  if (job.status !== "failed" || toastedFailureIds.has(job.id)) {
    return;
  }
  toastedFailureIds.add(job.id);
  toast.toast({
    message: `<b>Sync failed</b>${job.error ? `\n${escapeHtml(job.error)}` : ""}`,
    type: "is-danger",
    duration: 10000
  });
}

function parseAndApplyEventData(rawData: string): void {
  if (!rawData.trim()) return;

  let job: Job;
  try {
    job = JSON.parse(rawData) as Job;
  } catch {
    return;
  }

  if (!job?.id) return;

  jobs.upsert(job);
  if (job.status === "completed" || job.status === "failed") {
    maybeShowFailureToast(job);
    notifyTerminal(job);
  }
}

function processSSEChunk(buffer: string): string {
  let separator = buffer.indexOf("\n\n");
  while (separator !== -1) {
    const eventBlock = buffer.slice(0, separator);
    buffer = buffer.slice(separator + 2);

    const dataLines = eventBlock
      .split("\n")
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice(5).trimStart());
    if (dataLines.length > 0) {
      parseAndApplyEventData(dataLines.join("\n"));
    }

    separator = buffer.indexOf("\n\n");
  }

  return buffer;
}

function scheduleReconnect(): void {
  if (reconnectTimer) return;

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    void ensureJobsStream();
  }, reconnectDelayMs);

  reconnectDelayMs = Math.min(reconnectDelayMs * 2, MAX_RECONNECT_DELAY_MS);
}

export async function ensureJobsStream(): Promise<void> {
  if (streamAbortController) {
    return;
  }

  const token = localStorage.getItem(AUTH_TOKEN_KEY);
  if (!token) {
    return;
  }

  const controller = new AbortController();
  streamAbortController = controller;

  try {
    const response = await fetch("/api/jobs/stream", {
      method: "GET",
      headers: {
        Accept: "text/event-stream",
        "X-Auth": token
      },
      cache: "no-store",
      signal: controller.signal
    });

    if (!response.ok || !response.body) {
      throw new Error(`Failed to connect to job stream (${response.status})`);
    }

    reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS;

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      buffer = processSSEChunk(buffer);
    }
  } catch (err) {
    if (!(err instanceof DOMException && err.name === "AbortError")) {
      scheduleReconnect();
    }
  } finally {
    if (streamAbortController === controller) {
      streamAbortController = null;
    }
  }
}

export async function sync(request: Record<string, any>): Promise<string | null> {
  let job_id: string;
  try {
    ({ job_id } = await ajax("/api/sync", {
      method: "POST",
      body: JSON.stringify(request)
    }));
  } catch (err) {
    toast.toast({
      message: `<b>Failed to submit sync request</b>\n${
        err instanceof Error ? escapeHtml(err.message) : escapeHtml(String(err))
      }`,
      type: "is-danger",
      duration: 10000
    });
    return null;
  }

  jobs.upsert({
    id: job_id,
    status: "pending",
    created_at: new Date().toISOString(),
    metadata: request
  });

  void ensureJobsStream();

  return job_id;
}

/**
 * Legacy name kept for call-site compatibility. This now subscribes to SSE
 * updates instead of polling.
 */
export function startPolling(jobId: string, onTerminal?: (job: Job) => void): void {
  if (onTerminal) {
    const listeners = terminalListeners.get(jobId) ?? new Set<(job: Job) => void>();
    listeners.add(onTerminal);
    terminalListeners.set(jobId, listeners);
  }

  const existing = jobs.snapshot()[jobId];
  if (existing && (existing.status === "completed" || existing.status === "failed")) {
    maybeShowFailureToast(existing);
    notifyTerminal(existing);
    return;
  }

  void ensureJobsStream();
}

/**
 * Clear the failure-toast deduplication set.
 * Exported for use in tests.
 */
export function clearToastedFailures(): void {
  toastedFailureIds.clear();
}
