export type SuggestionKind = "account" | "commodity" | null;

export function getConfigSuggestionKind(
  fieldKey: string,
  parentKey = "",
  widget?: string
): SuggestionKind {
  if (widget === "account") return "account";
  if (widget === "commodity") return "commodity";

  const keyLower = fieldKey.toLowerCase();
  const parentLower = parentKey.toLowerCase();

  if (keyLower.includes("account")) return "account";
  if (keyLower.includes("commodity") || keyLower.includes("commodities")) return "commodity";
  if (keyLower === "name" && parentLower.startsWith("commodities")) return "commodity";
  if (keyLower === "name" && parentLower.startsWith("accounts")) return "account";

  return null;
}

export function filterSuggestions(
  options: ReadonlyArray<string>,
  query: string,
  limit = 12
): string[] {
  const q = query.trim().toLowerCase();
  const unique = Array.from(new Set(options.filter(Boolean)));
  if (!q) {
    return unique.slice(0, limit);
  }

  const prefix: string[] = [];
  const contains: string[] = [];
  for (const opt of unique) {
    const lower = opt.toLowerCase();
    if (lower === q) {
      prefix.unshift(opt);
    } else if (lower.startsWith(q)) {
      prefix.push(opt);
    } else if (lower.includes(q)) {
      contains.push(opt);
    }
  }
  return [...prefix, ...contains].slice(0, limit);
}
