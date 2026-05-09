import { writable, derived, get } from "svelte/store";
import type { Job } from "$lib/utils";

/** Internal map from job ID to Job snapshot. */
type JobsMap = Record<string, Job>;

export function createJobsStore() {
  const { subscribe, update, set } = writable<JobsMap>({});

  return {
    subscribe,

    /**
     * Insert a new job or fully replace an existing one identified by job.id.
     * Use this when the API returns a complete Job object.
     */
    upsert(job: Job): void {
      update((current) => ({ ...current, [job.id]: job }));
    },

    /**
     * Shallow-merge partial fields into the job identified by id.
     * Returns true when the job was found and updated, false when unknown.
     */
    updateById(id: string, partial: Partial<Job>): boolean {
      let found = false;
      update((current) => {
        const existing = current[id];
        if (!existing) return current;
        found = true;
        return { ...current, [id]: { ...existing, ...partial } };
      });
      return found;
    },

    /** Remove all tracked jobs from the store. */
    reset(): void {
      set({});
    },

    /** Return a synchronous snapshot of the current jobs map. */
    snapshot(): JobsMap {
      return get({ subscribe });
    }
  };
}

/** Global jobs store – tracks every known background job by ID. */
export const jobs = createJobsStore();

/**
 * Sorted array of all known jobs, oldest first (by created_at).
 * Reacts to every store update.
 */
export const jobsList = derived(jobs, ($jobs) =>
  Object.values($jobs).sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )
);

/**
 * True when at least one tracked job is in a non-terminal state
 * (pending or running).
 */
export const isJobRunning = derived(jobs, ($jobs) =>
  Object.values($jobs).some((j) => j.status === "pending" || j.status === "running")
);

/**
 * The first tracked job that is currently in a non-terminal state
 * (pending or running), or null when no such job exists.
 * Useful for displaying per-job progress in the navbar.
 */
export const runningJob = derived(
  jobs,
  ($jobs) =>
    Object.values($jobs).find((j) => j.status === "pending" || j.status === "running") ?? null
);
