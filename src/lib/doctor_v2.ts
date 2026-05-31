import type { DuplicatePair, Issue, OutlierTransaction } from "$lib/utils";

export type DoctorSection = "overview" | "diagnosis" | "duplicates" | "outliers";

export interface DoctorPriorityItem {
  id: string;
  section: Exclude<DoctorSection, "overview">;
  title: string;
  subtitle: string;
  meta: string;
  tone: "danger" | "warning" | "info" | "success";
  score: number;
}

const ISSUE_LEVEL_WEIGHT: Record<string, number> = {
  danger: 300,
  warning: 200,
  info: 100,
  success: 50
};

export function normalizeDoctorQuery(value: string) {
  return value.trim().toLowerCase();
}

export function issueTone(level: string): DoctorPriorityItem["tone"] {
  const normalized = normalizeDoctorQuery(level);
  if (normalized === "danger" || normalized === "warning" || normalized === "success") {
    return normalized;
  }
  return "info";
}

export function countIssuesByLevel(issues: Issue[]) {
  return issues.reduce<Record<string, number>>((counts, issue) => {
    const level = issueTone(issue.level);
    counts[level] = (counts[level] || 0) + 1;
    return counts;
  }, {});
}

export function filterIssues(issues: Issue[], query: string) {
  const q = normalizeDoctorQuery(query);
  if (!q) return issues;

  return issues.filter((issue) => {
    const haystack = [issue.level, issue.summary, issue.description, issue.details]
      .join(" ")
      .toLowerCase();
    return haystack.includes(q);
  });
}

export function filterDuplicatePairs(
  duplicates: DuplicatePair[],
  query: string,
  minConfidence: number
) {
  const q = normalizeDoctorQuery(query);

  return duplicates.filter((pair) => {
    if (pair.confidence < minConfidence) return false;
    if (!q) return true;

    const haystack = [
      pair.reason,
      pair.posting1.date,
      pair.posting1.payee,
      pair.posting1.account,
      String(pair.posting1.amount),
      pair.posting2.date,
      pair.posting2.payee,
      pair.posting2.account,
      String(pair.posting2.amount)
    ]
      .join(" ")
      .toLowerCase();

    return haystack.includes(q);
  });
}

export function filterOutliers(
  outliers: OutlierTransaction[],
  query: string,
  minConfidence: number
) {
  const q = normalizeDoctorQuery(query);

  return outliers.filter((outlier) => {
    if (outlier.confidence < minConfidence) return false;
    if (!q) return true;

    const haystack = [
      outlier.posting.date,
      outlier.posting.payee,
      outlier.posting.account,
      String(outlier.posting.amount)
    ]
      .join(" ")
      .toLowerCase();

    return haystack.includes(q);
  });
}

export function buildPriorityQueue(
  issues: Issue[],
  duplicates: DuplicatePair[],
  outliers: OutlierTransaction[]
) {
  const issueItems: DoctorPriorityItem[] = issues.map((issue, index) => {
    const tone = issueTone(issue.level);

    return {
      id: `issue-${index}`,
      section: "diagnosis",
      title: issue.summary,
      subtitle: stripHtml(issue.description),
      meta: stripHtml(issue.details),
      tone,
      score: (ISSUE_LEVEL_WEIGHT[tone] || ISSUE_LEVEL_WEIGHT.info) - index
    };
  });

  const duplicateItems: DoctorPriorityItem[] = duplicates.map((pair, index) => ({
    id: `duplicate-${pair.posting1.id}-${pair.posting2.id}`,
    section: "duplicates",
    title: pair.posting1.payee || pair.posting2.payee || "Potential duplicate transaction",
    subtitle: `${pair.posting1.account} and ${pair.posting2.account}`,
    meta: `${Math.round(pair.confidence * 100)}% confidence • ${stripHtml(pair.reason)}`,
    tone: pair.confidence >= 0.8 ? "danger" : "warning",
    score: Math.round(pair.confidence * 100) + 150 - index
  }));

  const outlierItems: DoctorPriorityItem[] = outliers.map((outlier, index) => ({
    id: `outlier-${outlier.posting.id}`,
    section: "outliers",
    title: outlier.posting.payee || "Outlier transaction",
    subtitle: outlier.posting.account,
    meta: `${Math.round(outlier.confidence * 100)}% confidence • ${outlier.sigma.toFixed(1)}σ from mean`,
    tone: outlier.confidence >= 0.8 ? "danger" : "warning",
    score: Math.round(outlier.confidence * 100) + 120 - index
  }));

  return [...issueItems, ...duplicateItems, ...outlierItems].sort(
    (left, right) => right.score - left.score
  );
}

function stripHtml(value: string) {
  return value
    .replace(/<[^>]+>/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}
