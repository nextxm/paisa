export function normalizeDoctorQuery(value: string) {
  return value.trim().toLowerCase();
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
