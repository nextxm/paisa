import type { AccountReconciliationStatus } from "./utils";
import { iconGlyph } from "./icon";

export function reconciliationTagClass(
  status: AccountReconciliationStatus | undefined | null
): string {
  if (!status || status.days_since === null || status.days_since > status.frequency_days)
    return "is-danger";
  if (status.days_since >= Math.floor(status.frequency_days * 0.8)) return "is-warning";
  return "is-success";
}

export function reconciliationIcon(status: AccountReconciliationStatus | undefined | null): string {
  if (!status || status.days_since === null || status.days_since > status.frequency_days) {
    return iconGlyph("fa6-solid:circle-exclamation");
  }
  if (status.days_since >= Math.floor(status.frequency_days * 0.8)) {
    return iconGlyph("fa6-solid:triangle-exclamation");
  }
  return iconGlyph("fa6-solid:circle-check");
}

export function reconciliationLabel(
  status: AccountReconciliationStatus | undefined | null
): string {
  if (!status || status.days_since === null || status.last_reconciled === null) {
    return "Last reconciled: never";
  }
  let label = "";
  if (status.days_since === 0) {
    label = "today";
  } else if (status.days_since === 1) {
    label = "1 day ago";
  } else {
    label = `${status.days_since} days ago`;
  }
  return `Last reconciled: ${label} (${status.last_reconciled})`;
}
export function reconciliationTextClass(
  status: AccountReconciliationStatus | undefined | null
): string {
  if (!status || status.days_since === null || status.days_since > status.frequency_days)
    return "has-text-danger";
  if (status.days_since >= Math.floor(status.frequency_days * 0.8)) return "has-text-warning";
  return "has-text-success";
}
