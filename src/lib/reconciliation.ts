import type { AccountReconciliationStatus } from "./utils";

export function reconciliationTagClass(status: AccountReconciliationStatus): string {
  if (status.days_since === null || status.days_since > status.frequency_days) return "is-danger";
  if (status.days_since >= Math.floor(status.frequency_days * 0.8)) return "is-warning";
  return "is-success";
}

export function reconciliationLabel(status: AccountReconciliationStatus): string {
  if (status.days_since === null || status.last_reconciled === null) {
    return "Last reconciled: never";
  }
  if (status.days_since === 0) {
    return "Last reconciled: today";
  }
  if (status.days_since === 1) {
    return "Last reconciled: 1 day ago";
  }
  return `Last reconciled: ${status.days_since} days ago`;
}
