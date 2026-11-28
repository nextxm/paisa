export interface NetworthPoint {
  date: string;
  balanceAmount: number;
  netInvestmentAmount: number;
  gainAmount: number;
  investment_return?: number;
  fx_impact?: number;
}

export function getInvestmentReturn(d: NetworthPoint): number {
  return d.investment_return ?? (d.gainAmount || 0) - (d.fx_impact ?? 0);
}

export function computeNetworthBounds(
  data: ReadonlyArray<NetworthPoint>,
  showFxImpact: boolean
): { min: number; max: number } {
  let min = 0;
  let max = 0;
  for (const d of data) {
    const balance = d.balanceAmount || 0;
    const invested = d.netInvestmentAmount || 0;
    min = Math.min(min, balance, invested);
    max = Math.max(max, balance, invested);
    if (showFxImpact) {
      const exFx = invested + getInvestmentReturn(d);
      min = Math.min(min, exFx);
      max = Math.max(max, exFx);
    }
  }
  if (max === min) {
    max = min + 1000;
  }
  return { min, max: max * 1.06 };
}

export function buildLinePath(xs: ReadonlyArray<number>, ys: ReadonlyArray<number>): string {
  if (xs.length === 0 || xs.length !== ys.length) return "";
  return xs.map((x, i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)},${ys[i].toFixed(1)}`).join(" ");
}

export function buildBandPath(
  xs: ReadonlyArray<number>,
  y0s: ReadonlyArray<number>,
  y1s: ReadonlyArray<number>
): string {
  if (xs.length === 0 || xs.length !== y0s.length || xs.length !== y1s.length) return "";
  const top = xs
    .map((x, i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y1s[i].toFixed(1)}`)
    .join(" ");
  const bottom = [...xs]
    .reverse()
    .map((x, idx) => {
      const origIdx = xs.length - 1 - idx;
      return `L${x.toFixed(1)},${y0s[origIdx].toFixed(1)}`;
    })
    .join(" ");
  return `${top} ${bottom} Z`;
}
