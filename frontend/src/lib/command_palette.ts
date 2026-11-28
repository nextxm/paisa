export function buildCommandPaletteTargets(
  commands: ReadonlyArray<{ href: string }>,
  tabs: ReadonlyArray<{ href: string }>,
  accounts: ReadonlyArray<string>
): string[] {
  return [
    ...commands.map((c) => c.href),
    ...tabs.map((t) => t.href),
    ...accounts.map((acc) => `/accounts/${encodeURIComponent(acc)}`)
  ];
}

export function nextCommandIndex(current: number, total: number, delta: 1 | -1): number {
  if (total <= 0) return 0;
  const clamped = Math.max(0, Math.min(current, total - 1));
  return (clamped + delta + total) % total;
}
