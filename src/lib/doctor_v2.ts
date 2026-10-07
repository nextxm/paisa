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

// ---------------------------------------------------------------------------
// Phase B Unified Doctor Findings Types & Helpers
// ---------------------------------------------------------------------------

export type DoctorSeverity = "fix" | "review" | "info";
export type DoctorKind = "rule" | "duplicate" | "outlier";

export interface FindingEvidence {
  posting_id?: number;
  posting_id_2?: number;
  file_name?: string;
  line_number?: number;
  account?: string;
  date?: string;
  amount?: number;
  commodity?: string;
  payee?: string;
  raw_message?: string;
  target_url?: string;
  sigma?: number;
  mean?: number;
  std_dev?: number;
}

export interface FindingAction {
  type: string;
  label: string;
  url?: string;
  params?: Record<string, string>;
}

export interface DoctorFinding {
  id: string;
  rule_id: string;
  kind: DoctorKind;
  title: string;
  summary: string;
  description: string;
  why_it_matters: string;
  how_to_fix: string;
  severity: DoctorSeverity;
  details: string;
  evidence: FindingEvidence[];
  actions: FindingAction[];
  confidence?: number;
  dismissed?: boolean;
  dismiss_note?: string;
}

export interface DoctorSummary {
  total: number;
  fix_count: number;
  review_count: number;
  info_count: number;
  dismissed_count: number;
}

export interface DoctorFindingsResponse {
  findings: DoctorFinding[];
  summary: DoctorSummary;
}

export function groupDoctorFindings(findings: DoctorFinding[]) {
  const groups: Record<string, { rule_id: string; title: string; findings: DoctorFinding[] }> = {};

  for (const f of findings) {
    if (!groups[f.rule_id]) {
      groups[f.rule_id] = {
        rule_id: f.rule_id,
        title: f.title,
        findings: []
      };
    }
    groups[f.rule_id].findings.push(f);
  }

  return Object.values(groups);
}

export function filterDoctorFindings(findings: DoctorFinding[], query: string) {
  const q = normalizeDoctorQuery(query);
  if (!q) return findings;

  return findings.filter((f) => {
    const haystack = [
      f.title,
      f.why_it_matters,
      f.how_to_fix,
      f.details,
      ...f.evidence.map((e) => [e.account, e.payee, e.file_name, e.raw_message].join(" "))
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(q);
  });
}
