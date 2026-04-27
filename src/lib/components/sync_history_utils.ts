import dayjs from "dayjs";
import type { Job } from "$lib/utils";

/** CSS tag class for a given job status. */
export function statusTagClass(status: Job["status"]): string {
  switch (status) {
    case "completed":
      return "is-success";
    case "failed":
      return "is-danger";
    case "running":
      return "is-info";
    case "pending":
      return "is-warning";
    default:
      return "";
  }
}

/** Font-Awesome icon class(es) for a given job status. */
export function statusIconClass(status: Job["status"]): string {
  switch (status) {
    case "completed":
      return "fa-solid fa-check";
    case "failed":
      return "fa-solid fa-xmark";
    case "running":
      return "fa-solid fa-spinner fa-spin";
    case "pending":
      return "fa-regular fa-clock";
    default:
      return "fa-solid fa-circle-question";
  }
}

/** Format an ISO timestamp for display; returns "—" when absent. */
export function formatTs(iso: string | undefined): string {
  if (!iso) return "—";
  return dayjs(iso).format("MMM D, YYYY HH:mm:ss");
}

/** Human-readable wall-clock duration between started_at and finished_at (or now). */
export function formatDuration(job: Job): string {
  if (!job.started_at) return "—";
  const start = dayjs(job.started_at);
  const end = job.finished_at ? dayjs(job.finished_at) : dayjs();
  const secs = end.diff(start, "second");
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}
