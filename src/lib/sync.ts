import * as toast from "bulma-toast";
import { ajax } from "./utils";
import { jobs } from "./stores/jobs";
import type { Job } from "./utils";

/**
 * Set of job IDs for which a failure toast has already been shown.
 * Guards against duplicate toasts when startPolling is called more than
 * once for the same job (e.g. after a page navigation or component re-mount).
 */
const toastedFailureIds = new Set<string>();

/**
 * Clear the deduplication set.
 * Exported exclusively for use in tests — do not call in production code.
 */
export function clearToastedFailures(): void {
  toastedFailureIds.clear();
}

/** Milliseconds between each job-status poll. */
export const POLL_INTERVAL_MS = 2000;

/** Maximum number of consecutive network errors before giving up. */
export const MAX_CONSECUTIVE_ERRORS = 5;

/** Maximum total poll attempts (~5 minutes at 2 s intervals) before giving up. */
export const MAX_POLLS = 150;

/** Escape HTML special characters so error strings are safe to embed in toast HTML. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
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

  return job_id;
}

/**
 * Start polling `GET /api/jobs/:id` until the job reaches a terminal state
 * (completed or failed), or until the polling limits are exhausted.
 *
 * On each successful response the jobs store is updated. On a terminal state
 * the optional `onTerminal` callback is invoked (useful for triggering a
 * data refresh). On failure a toast is shown.
 *
 * @param jobId - The ID of the job to track.
 * @param onTerminal - Called with the final Job when it reaches a terminal state.
 * @param options - Injectable dependencies for testability.
 */
export function startPolling(
  jobId: string,
  onTerminal?: (job: Job) => void,
  options?: {
    /** Override the job-fetch function (for tests, avoids real HTTP). */
    fetchJob?: (id: string) => Promise<Job>;
    /** Override the poll interval in ms (for tests). */
    intervalMs?: number;
    /** Override the max consecutive errors before giving up. */
    maxErrors?: number;
    /** Override the max total polls before giving up. */
    maxPolls?: number;
  }
): void {
  const fetchJob =
    options?.fetchJob ?? ((id: string) => ajax("/api/jobs/:id", { background: true }, { id }));
  const intervalMs = options?.intervalMs ?? POLL_INTERVAL_MS;
  const maxErrors = options?.maxErrors ?? MAX_CONSECUTIVE_ERRORS;
  const maxPolls = options?.maxPolls ?? MAX_POLLS;

  let consecutiveErrors = 0;
  let pollCount = 0;

  async function poll() {
    if (pollCount >= maxPolls) {
      return;
    }
    pollCount++;

    try {
      const job = await fetchJob(jobId);
      consecutiveErrors = 0;
      jobs.upsert(job);

      if (job.status === "completed" || job.status === "failed") {
        if (job.status === "failed" && !toastedFailureIds.has(jobId)) {
          toastedFailureIds.add(jobId);
          toast.toast({
            message: `<b>Sync failed</b>${job.error ? `\n${escapeHtml(job.error)}` : ""}`,
            type: "is-danger",
            duration: 10000
          });
        }
        onTerminal?.(job);
        return;
      }

      setTimeout(poll, intervalMs);
    } catch (err: any) {
      if (err.status === 404) {
        jobs.updateById(jobId, {
          status: "failed",
          error: "Job not found on server (it may have expired or the server restarted)"
        });
        return;
      }
      consecutiveErrors++;
      if (consecutiveErrors >= maxErrors) {
        return;
      }
      setTimeout(poll, intervalMs);
    }
  }

  setTimeout(poll, intervalMs);
}
