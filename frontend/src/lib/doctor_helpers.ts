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
}

export interface FindingAction {
  type: string;
  label: string;
  url?: string;
}

export interface DoctorFinding {
  id: string;
  rule_id: string;
  kind?: string;
  severity: "fix" | "review" | "info";
  title: string;
  description?: string;
  why_it_matters?: string;
  how_to_fix?: string;
  details?: string;
  account?: string;
  evidence?: FindingEvidence[];
  actions?: FindingAction[];
  dismissed?: boolean;
  dismiss_note?: string;
}

export interface EditorLocation {
  fileName: string;
  lineNumber: number;
}

export function parseEditorTargetUrl(url?: string): EditorLocation | null {
  if (!url || !url.startsWith("/ledger/editor/")) return null;
  const rest = url.slice("/ledger/editor/".length);
  const [rawFile, rawLine] = rest.split("#");
  if (!rawFile) return null;
  const fileName = decodeURIComponent(rawFile);
  const parsedLine = rawLine ? Number.parseInt(rawLine, 10) : 0;
  return {
    fileName,
    lineNumber: Number.isFinite(parsedLine) && parsedLine > 0 ? parsedLine : 0
  };
}

export function getEvidenceEditorLocation(ev?: FindingEvidence): EditorLocation | null {
  if (!ev) return null;
  if (ev.file_name) {
    return {
      fileName: ev.file_name,
      lineNumber: ev.line_number && ev.line_number > 0 ? ev.line_number : 0
    };
  }
  return parseEditorTargetUrl(ev.target_url);
}

export function getPrimaryEditorLocation(finding: DoctorFinding): EditorLocation | null {
  for (const ev of finding.evidence || []) {
    const loc = getEvidenceEditorLocation(ev);
    if (loc) return loc;
  }
  for (const action of finding.actions || []) {
    const loc = parseEditorTargetUrl(action.url);
    if (loc) return loc;
  }
  return null;
}

export function formatEvidenceLabel(ev: FindingEvidence): string {
  const parts: string[] = [];
  if (ev.account) parts.push(ev.account);
  if (ev.date) parts.push(ev.date);
  if (ev.payee) parts.push(ev.payee);
  if (ev.amount !== undefined && ev.amount !== 0) {
    parts.push(ev.commodity ? `${ev.amount} ${ev.commodity}` : `${ev.amount}`);
  }
  if (ev.file_name) {
    parts.push(ev.line_number ? `${ev.file_name}:${ev.line_number}` : ev.file_name);
  }
  if (parts.length === 0 && ev.raw_message) {
    return ev.raw_message;
  }
  return parts.join(" • ");
}

export function matchesFindingSearch(finding: DoctorFinding, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  const haystack = [
    finding.title,
    finding.details,
    finding.description,
    finding.why_it_matters,
    finding.account,
    ...(finding.evidence || []).map((e) =>
      [e.account, e.date, e.payee, e.file_name, e.raw_message, e.commodity].join(" ")
    )
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  return haystack.includes(q);
}
