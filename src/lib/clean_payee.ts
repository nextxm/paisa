const NOISE_WORDS = new Set([
  "UPI",
  "NEFT",
  "IMPS",
  "RTGS",
  "POS",
  "ATM",
  "ACH",
  "BIL",
  "MMT",
  "BBPS",
  "ECOM",
  "VPS",
  "IPS",
  "DR",
  "CR",
  "IN",
  "IND",
  "PVT",
  "LTD",
  "BANK"
]);

function isNoiseToken(w: string): boolean {
  if (w.includes("@")) return true;
  const upper = w.toUpperCase();
  if (NOISE_WORDS.has(upper)) return true;
  if (/^\d{4,}$/.test(w)) return true;
  if (/^(REF|TXN|TRN|UTR|RRN|CHQ|AUTH|CRN|ID|NUM|NO)[A-Z0-9]*$/i.test(w)) return true;
  if (/\d{4,}/.test(w)) return true;
  if (w.length >= 10 && /\d/.test(w) && /[a-zA-Z]/.test(w)) return true;
  return false;
}

export function cleanPayee(raw?: string): string {
  if (!raw) return "";
  const trimmed = raw.trim();
  if (!trimmed) return "";

  const parts = trimmed.split("_");
  let bestCandidate = "";

  for (let i = parts.length - 1; i >= 0; i--) {
    const part = parts[i];
    const s = part.replace(
      /\b(UPI|NEFT|IMPS|RTGS|POS|ATM|ACH|BIL|MMT|BBPS|ECOM|VPS|IPS)[-/\s:*]+/gi,
      " "
    );
    const tokens = s.split(/[-/|*:_]+|\s+/);
    const meaningful: string[] = [];

    for (const t of tokens) {
      const cleanT = t.trim();
      if (!cleanT || isNoiseToken(cleanT)) continue;
      meaningful.push(cleanT);
    }

    if (meaningful.length > 0) {
      const cand = meaningful.join(" ").trim();
      if (cand.length > bestCandidate.length) {
        bestCandidate = cand;
      }
    }
  }

  if (bestCandidate) {
    return bestCandidate.length > 36 ? bestCandidate.slice(0, 36).trim() : bestCandidate;
  }

  let fallback = trimmed.replace(
    /\b(UPI|NEFT|IMPS|RTGS|POS|ATM|ACH|BIL|MMT|BBPS|ECOM|VPS|IPS)[-/\s:*]+/gi,
    ""
  );
  fallback = fallback.replace(/\s+/g, " ").trim();
  return fallback.length > 36 ? fallback.slice(0, 36).trim() : fallback;
}
