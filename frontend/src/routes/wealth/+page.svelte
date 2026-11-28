<script lang="ts">
  import dayjs from "dayjs";
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import LineAreaChart from "$lib/components/LineAreaChart.svelte";
  import type { NetworthPoint } from "$lib/networth_chart";
  import StackedBarChart, { type StackedBarColumn } from "$lib/components/StackedBarChart.svelte";
  import {
    apiGet,
    appPath,
    authHeaders,
    formatMoney,
    formatShortDate,
    type AssetBreakdown,
    type Posting
  } from "$lib/api";

  type WealthTab =
    | "overview"
    | "assets"
    | "investments"
    | "allocation"
    | "portfolio"
    | "income"
    | "liabilities";

  interface GainRow {
    account: string;
    investmentAmount?: number;
    marketAmount?: number;
    gainAmount?: number;
    xirr?: number;
    price_appreciation?: number;
    income_received?: number;
    total_return?: number;
    ttm_yield?: number;
  }

  interface LiabilityRow {
    group: string;
    drawn_amount?: number;
    repaid_amount?: number;
    interest_amount?: number;
    balance_amount?: number;
    originalBalances?: Array<{ currency: string; amount: number }>;
  }

  interface PortfolioAggregate {
    id: string;
    group: string;
    sub_group: string;
    amount: number;
    percentage: number;
    breakdowns?: Array<{
      commodity_name?: string;
      commodity?: string;
      amount?: number;
      percentage?: number;
    }>;
  }

  interface InterestAccount {
    account: string;
    apr: number;
    overview_timeline?: Array<{
      date: string;
      drawn_amount: number;
      repaid_amount: number;
      interest_amount: number;
    }>;
  }

  interface RepaymentRow {
    date: string;
    account?: string;
    commodity?: string;
    original_amount?: number;
    amount?: number;
    repaid_amount?: number;
  }

  interface AllocationAggregate {
    name?: string;
    account?: string;
    market_amount: number;
  }

  interface CurrencyExposureItem {
    currency: string;
    amount: number;
    percent?: number;
    percentage?: number;
  }

  interface InvestmentIncomeHolding {
    holding?: string;
    name?: string;
    account?: string;
    type?: string;
    total_income?: number;
    ttm_income?: number;
    current_balance?: number;
    ttm_yield?: number;
    yield?: number;
    yearly_income?: Record<string, number>;
  }

  interface InvestmentIncomeTimelinePoint {
    date: string;
    dividend?: number;
    interest?: number;
    distribution?: number;
    total?: number;
    amount?: number;
  }

  interface AllocationTargetSummary {
    name: string;
    target: number;
    current: number;
    drift?: number;
    status?: string;
  }

  interface CreditCardSummary {
    name?: string;
    account: string;
    currency?: string;
    balance?: number;
    creditLimit?: number;
    limit?: number;
    unbilled?: number;
    due_date?: string;
    originalBalances?: Array<{ currency: string; amount: number }>;
  }

  let activeTab = $state<WealthTab>("overview");
  $effect(() => {
    const tab = $page.url.searchParams.get("tab");
    if (
      tab === "overview" ||
      tab === "assets" ||
      tab === "investments" ||
      tab === "allocation" ||
      tab === "portfolio" ||
      tab === "income" ||
      tab === "liabilities"
    ) {
      activeTab = tab;
    }
  });
  let yearsWindow = $state<number>(0);
  let showFxImpact = $state(true);
  let search = $state("");
  let asOfDate = $state(new Date().toISOString().slice(0, 10));
  let reportCurrency = $state("");
  let availableCurrencies = $state<string[]>([]);
  let assetsRefreshing = $state(false);
  let hideZeroAssets = $state(true);
  let filterInactive = $state(true);
  let flatAssets = $state(true);
  type AssetSortField =
    | "group"
    | "investmentAmount"
    | "withdrawalAmount"
    | "balanceUnits"
    | "marketAmount"
    | "gainAmount"
    | "xirr"
    | "absoluteReturn";
  let assetSortField = $state<AssetSortField>("marketAmount");
  let assetSortAsc = $state(false);
  let targetsEditing = $state(false);
  let targetDraft = $state<Record<string, number>>({});
  let targetsSaving = $state(false);
  let targetsMessage = $state<string | null>(null);

  let rebalanceCash = $state(50000);
  let rebalanceResult = $state<{
    suggestions?: Array<{
      name?: string;
      account?: string;
      amount: number;
      target?: number;
      current?: number;
    }>;
    allocations?: Array<{ name?: string; amount: number }>;
  } | null>(null);
  let rebalanceLoading = $state(false);

  let selectedHolding = $state<string | null>(null);
  let holdingDetail = $state<Record<string, unknown> | null>(null);
  let selectedCard = $state<string | null>(null);
  let cardDetail = $state<Record<string, unknown> | null>(null);
  let investmentPostings = $state<Posting[]>([]);
  let investmentYearly = $state<
    Array<{
      year?: string;
      start_date?: string;
      end_date?: string;
      gross_salary_income?: number;
      gross_other_income?: number;
      net_tax?: number;
      net_investment?: number;
      net_income?: number;
      net_expense?: number;
      savings_rate?: number;
      invested?: number;
      market_value?: number;
      gain?: number;
      postings?: Posting[];
    }>
  >([]);

  let loading = $state(true);
  let error = $state<string | null>(null);

  let timeline = $state<NetworthPoint[]>([]);
  let xirr = $state(0);
  let assetBreakdowns = $state<AssetBreakdown[]>([]);
  let gains = $state<GainRow[]>([]);
  let liabilities = $state<LiabilityRow[]>([]);
  let allocationItems = $state<AllocationAggregate[]>([]);
  let allocationTargets = $state<AllocationTargetSummary[]>([]);
  let allocationTimeline = $state<
    Array<{ date?: string; aggregates?: Record<string, AllocationAggregate> }>
  >([]);
  let currencyExposure = $state<CurrencyExposureItem[]>([]);
  let investmentIncomeTotal = $state(0);
  let dividendTTM = $state(0);
  let interestTTM = $state(0);
  let distributionTTM = $state(0);
  let investmentIncomeHoldings = $state<InvestmentIncomeHolding[]>([]);
  let investmentIncomeTimeline = $state<InvestmentIncomeTimelinePoint[]>([]);
  let creditCards = $state<CreditCardSummary[]>([]);
  let portfolioCommodities = $state<string[]>([]);
  let selectedPortfolioCommodities = $state<string[]>([]);
  let portfolioNameAndType = $state<PortfolioAggregate[]>([]);
  let portfolioSecurityType = $state<PortfolioAggregate[]>([]);
  let portfolioRating = $state<PortfolioAggregate[]>([]);
  let portfolioIndustry = $state<PortfolioAggregate[]>([]);
  let interestAccounts = $state<InterestAccount[]>([]);
  let repayments = $state<RepaymentRow[]>([]);

  async function refreshAssets() {
    assetsRefreshing = true;
    try {
      const params = new URLSearchParams();
      if (asOfDate) params.set("as_of_date", asOfDate);
      if (reportCurrency) params.set("report_currency", reportCurrency);
      const q = params.toString();
      const balRes = await apiGet<{ asset_breakdowns?: Record<string, AssetBreakdown> }>(
        q ? `/api/assets/balance?${q}` : "/api/assets/balance"
      );
      assetBreakdowns = Object.values(balRes.asset_breakdowns || {}).sort(
        (a, b) => (b.marketAmount || 0) - (a.marketAmount || 0)
      );
    } finally {
      assetsRefreshing = false;
    }
  }

  onMount(async () => {
    const loadOverview = async () => {
      const [nwRes, currRes] = await Promise.all([
        apiGet<{ xirr?: number; networthTimeline?: NetworthPoint[] }>("/api/networth"),
        apiGet<{ currencies?: string[] }>("/api/price/currencies").catch(() => ({
          currencies: [] as string[]
        }))
      ]);
      timeline = nwRes.networthTimeline || [];
      xirr = nwRes.xirr || 0;
      availableCurrencies = currRes.currencies || [];
    };

    const loadAssets = async () => {
      const [balRes, ccRes] = await Promise.all([
        apiGet<{ asset_breakdowns?: Record<string, AssetBreakdown> }>("/api/assets/balance").catch(
          () => ({ asset_breakdowns: {} as Record<string, AssetBreakdown> })
        ),
        apiGet<{ creditCards?: CreditCardSummary[] }>("/api/credit_cards").catch(() => ({
          creditCards: [] as CreditCardSummary[]
        }))
      ]);
      assetBreakdowns = Object.values(balRes.asset_breakdowns || {}).sort(
        (a, b) => (b.marketAmount || 0) - (a.marketAmount || 0)
      );
      creditCards = ccRes.creditCards || [];
    };

    const loadInvestments = async () => {
      const [gainRes, invRes] = await Promise.all([
        apiGet<{ gain_breakdown?: GainRow[] }>("/api/gain").catch(() => ({
          gain_breakdown: [] as GainRow[]
        })),
        apiGet<{ assets?: Posting[]; yearly_cards?: typeof investmentYearly }>(
          "/api/investment"
        ).catch(() => ({
          assets: [] as Posting[],
          yearly_cards: [] as typeof investmentYearly
        }))
      ]);
      gains = (gainRes.gain_breakdown || []).sort(
        (a, b) => (b.marketAmount || 0) - (a.marketAmount || 0)
      );
      investmentPostings = invRes.assets || [];
      investmentYearly = invRes.yearly_cards || [];
    };

    const loadAllocation = async () => {
      const [allocRes, fxExpRes] = await Promise.all([
        apiGet<{
          aggregates?: Record<string, AllocationAggregate>;
          aggregates_timeline?: Array<{
            date?: string;
            aggregates?: Record<string, AllocationAggregate>;
          }>;
          allocation_targets?: AllocationTargetSummary[];
        }>("/api/allocation").catch(() => ({
          aggregates: {} as Record<string, AllocationAggregate>,
          aggregates_timeline: [] as Array<{
            date?: string;
            aggregates?: Record<string, AllocationAggregate>;
          }>,
          allocation_targets: [] as AllocationTargetSummary[]
        })),
        apiGet<{ currency_exposure?: CurrencyExposureItem[] }>("/api/currency-exposure").catch(
          () => ({ currency_exposure: [] as CurrencyExposureItem[] })
        )
      ]);
      allocationItems = Object.values(allocRes.aggregates || {})
        .filter((a) => (a.market_amount || 0) > 0)
        .sort((a, b) => (b.market_amount || 0) - (a.market_amount || 0));
      const rawTimeline = allocRes.aggregates_timeline as unknown;
      allocationTimeline = Array.isArray(rawTimeline)
        ? rawTimeline
        : rawTimeline && typeof rawTimeline === "object"
          ? Object.entries(rawTimeline as Record<string, Record<string, AllocationAggregate>>).map(
              ([date, aggregates]) => ({ date, aggregates })
            )
          : [];
      allocationTargets = Array.isArray(allocRes.allocation_targets)
        ? allocRes.allocation_targets
        : [];
      currencyExposure = Array.isArray(fxExpRes.currency_exposure)
        ? fxExpRes.currency_exposure
        : [];
    };

    const loadPortfolio = async () => {
      const portRes = await apiGet<{
        commodities?: string[];
        name_and_security_type?: PortfolioAggregate[];
        security_type?: PortfolioAggregate[];
        rating?: PortfolioAggregate[];
        industry?: PortfolioAggregate[];
      }>("/api/portfolio_allocation").catch(() => ({
        commodities: [] as string[],
        name_and_security_type: [] as PortfolioAggregate[],
        security_type: [] as PortfolioAggregate[],
        rating: [] as PortfolioAggregate[],
        industry: [] as PortfolioAggregate[]
      }));
      portfolioCommodities = portRes.commodities || [];
      selectedPortfolioCommodities = [...portfolioCommodities];
      portfolioNameAndType = portRes.name_and_security_type || [];
      portfolioSecurityType = portRes.security_type || [];
      portfolioRating = portRes.rating || [];
      portfolioIndustry = portRes.industry || [];
    };

    const loadIncome = async () => {
      const incRes = await apiGet<{
        ttm_total?: number;
        holdings?: InvestmentIncomeHolding[];
        timeline?: InvestmentIncomeTimelinePoint[];
        income_by_type?: Record<string, InvestmentIncomeHolding[]>;
      }>("/api/income/investment").catch(() => ({
        ttm_total: 0,
        holdings: [] as InvestmentIncomeHolding[],
        timeline: [] as InvestmentIncomeTimelinePoint[],
        income_by_type: {} as Record<string, InvestmentIncomeHolding[]>
      }));
      investmentIncomeTotal = incRes.ttm_total || 0;
      investmentIncomeHoldings = incRes.holdings || [];
      investmentIncomeTimeline = incRes.timeline || [];
      const byType = incRes.income_by_type || {};
      dividendTTM = (byType.Dividend || []).reduce((s, h) => s + (h.ttm_income || 0), 0);
      interestTTM = (byType.Interest || []).reduce((s, h) => s + (h.ttm_income || 0), 0);
      distributionTTM = (byType.Distribution || []).reduce((s, h) => s + (h.ttm_income || 0), 0);
    };

    const loadLiabilities = async () => {
      const [liabRes, intRes, repRes] = await Promise.all([
        apiGet<{ liability_breakdowns?: LiabilityRow[] | Record<string, LiabilityRow> }>(
          "/api/liabilities/balance"
        ).catch(() => ({ liability_breakdowns: [] as LiabilityRow[] })),
        apiGet<{
          interest_timeline_breakdown?: Record<string, InterestAccount> | InterestAccount[];
        }>("/api/liabilities/interest").catch(() => ({
          interest_timeline_breakdown: [] as InterestAccount[]
        })),
        apiGet<{ repayments?: RepaymentRow[] }>("/api/liabilities/repayment").catch(() => ({
          repayments: [] as RepaymentRow[]
        }))
      ]);
      liabilities = Array.isArray(liabRes.liability_breakdowns)
        ? liabRes.liability_breakdowns
        : Object.values(liabRes.liability_breakdowns || {});
      interestAccounts = Array.isArray(intRes.interest_timeline_breakdown)
        ? intRes.interest_timeline_breakdown
        : Object.values(intRes.interest_timeline_breakdown || {});
      repayments = repRes.repayments || [];
    };

    const loaders: Record<WealthTab, () => Promise<void>> = {
      overview: loadOverview,
      assets: loadAssets,
      investments: loadInvestments,
      allocation: loadAllocation,
      portfolio: loadPortfolio,
      income: loadIncome,
      liabilities: loadLiabilities
    };

    const initialTab = activeTab;
    try {
      await loaders[initialTab]();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load wealth data";
    } finally {
      loading = false;
    }

    // Prefetch remaining tabs in the background after initial render.
    const allTabs: WealthTab[] = [
      "overview",
      "assets",
      "investments",
      "allocation",
      "portfolio",
      "income",
      "liabilities"
    ];
    void Promise.allSettled(allTabs.filter((t) => t !== initialTab).map((t) => loaders[t]()));
  });

  const latestPoint = $derived(timeline.length > 0 ? timeline[timeline.length - 1] : null);
  const windowedTimeline = $derived.by(() => {
    if (yearsWindow <= 0 || !latestPoint?.date) return timeline;
    const cutoff = dayjs(latestPoint.date).subtract(yearsWindow, "year");
    return timeline.filter((pt) => !dayjs(pt.date).isBefore(cutoff, "day"));
  });
  const totalAllocation = $derived(
    allocationItems.reduce((sum, a) => sum + (a.market_amount || 0), 0)
  );

  const invPalette = ["#3b82f6", "#10b981", "#f59e0b", "#8b5cf6", "#ec4899", "#14b8a6", "#6366f1"];

  let invIncomeGranularity = $state<"monthly" | "yearly">("monthly");

  const aggregatedInvestmentIncome = $derived.by(() => {
    if (investmentIncomeTimeline.length === 0) return [];
    const byKey = new Map<
      string,
      {
        key: string;
        label: string;
        dividend: number;
        interest: number;
        distribution: number;
        total: number;
      }
    >();
    for (const pt of investmentIncomeTimeline) {
      const d = dayjs(pt.date);
      const isMonthly = invIncomeGranularity === "monthly";
      const key = d.isValid()
        ? isMonthly
          ? d.format("YYYY-MM")
          : d.format("YYYY")
        : String(pt.date || "").slice(0, isMonthly ? 7 : 4);
      if (!key) continue;
      const label = d.isValid() ? (isMonthly ? d.format("MMM 'YY") : d.format("YYYY")) : key;
      const div = pt.dividend || 0;
      const intr = pt.interest || 0;
      const dist = pt.distribution || 0;
      const tot = div + intr + dist > 0 ? div + intr + dist : (pt.total ?? pt.amount ?? 0);

      const cur = byKey.get(key) || {
        key,
        label,
        dividend: 0,
        interest: 0,
        distribution: 0,
        total: 0
      };
      cur.dividend += div;
      cur.interest += intr;
      cur.distribution += dist;
      cur.total += tot;
      byKey.set(key, cur);
    }
    return Array.from(byKey.values()).sort((a, b) => a.key.localeCompare(b.key));
  });

  const investmentIncomeColumns = $derived.by((): StackedBarColumn[] => {
    const slice =
      invIncomeGranularity === "monthly"
        ? aggregatedInvestmentIncome.slice(-24)
        : aggregatedInvestmentIncome;
    return slice.map((pt) => {
      const segments =
        pt.dividend + pt.interest + pt.distribution > 0
          ? [
              { name: "Dividend", amount: pt.dividend, color: "#3b82f6" },
              { name: "Interest", amount: pt.interest, color: "#10b981" },
              { name: "Distribution", amount: pt.distribution, color: "#f59e0b" }
            ].filter((s) => s.amount > 0)
          : pt.total > 0
            ? [{ name: "Income", amount: pt.total, color: "#10b981" }]
            : [];
      return {
        label: pt.label,
        total: pt.total,
        segments
      };
    });
  });

  function holdingSparklinePoints(yearlyIncome?: Record<string, number>): string {
    const entries = Object.entries(yearlyIncome || {}).sort(([a], [b]) => a.localeCompare(b));
    if (entries.length < 2) return "";
    const vals = entries.map(([, v]) => v || 0);
    const maxV = Math.max(1, ...vals);
    const minV = Math.min(0, ...vals);
    const span = Math.max(1, maxV - minV);
    return vals
      .map((v, i) => {
        const x = (i / (vals.length - 1)) * 96 + 2;
        const y = 22 - ((v - minV) / span) * 18;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(" ");
  }

  let portfolioSearch = $state("");
  let portfolioGroupTab = $state<"all" | "security_type" | "rating" | "industry" | "security">(
    "all"
  );

  function filterPortfolioItems(items: PortfolioAggregate[]) {
    const q = portfolioSearch.trim().toLowerCase();
    const mapped = items
      .map((item) => {
        const rawBreakdowns = item.breakdowns || [];
        const filteredBreakdowns =
          rawBreakdowns.length > 0
            ? rawBreakdowns.filter((b) => {
                const name = b.commodity_name || b.commodity;
                return !name || selectedPortfolioCommodities.includes(name);
              })
            : [];
        const amount =
          rawBreakdowns.length > 0
            ? filteredBreakdowns.reduce((s, b) => s + (b.amount || 0), 0)
            : item.amount || 0;
        return {
          ...item,
          amount,
          filteredBreakdowns
        };
      })
      .filter((item) => item.amount > 0);

    const total = mapped.reduce((s, item) => s + item.amount, 0);
    return mapped
      .map((item) => ({
        ...item,
        percentage: total > 0 ? (item.amount / total) * 100 : 0,
        fundShares: item.filteredBreakdowns
          .map((b) => {
            const fund = b.commodity_name || b.commodity || "Other";
            const amt = b.amount || 0;
            return {
              fund,
              amount: amt,
              sharePct: item.amount > 0 ? (amt / item.amount) * 100 : 0
            };
          })
          .filter((f) => f.amount > 0)
          .sort((a, b) => b.amount - a.amount)
      }))
      .filter((item) => {
        if (!q) return true;
        return (
          (item.group || "").toLowerCase().includes(q) ||
          (item.sub_group || "").toLowerCase().includes(q) ||
          item.fundShares.some((f) => f.fund.toLowerCase().includes(q))
        );
      })
      .sort((a, b) => b.amount - a.amount);
  }

  function fundColor(fund: string): string {
    const idx = portfolioCommodities.indexOf(fund);
    return invPalette[(idx >= 0 ? idx : 0) % invPalette.length];
  }
  const monthlyInvestmentColumns = $derived.by((): StackedBarColumn[] => {
    if (investmentPostings.length === 0) return [];
    const byMonth = new Map<string, Map<string, number>>();
    const catTotals = new Map<string, number>();
    for (const p of investmentPostings) {
      const m = (p.date || "").slice(0, 7);
      if (!m) continue;
      const cat = (p.account || "").split(":")[1] || p.account || "Investment";
      const amt = Math.abs(p.amount || 0);
      if (amt <= 0) continue;
      if (!byMonth.has(m)) byMonth.set(m, new Map());
      byMonth.get(m)!.set(cat, (byMonth.get(m)!.get(cat) || 0) + amt);
      catTotals.set(cat, (catTotals.get(cat) || 0) + amt);
    }
    const cats = Array.from(catTotals.keys()).sort();
    const months = Array.from(byMonth.keys()).sort().slice(-24);
    return months.map((m) => {
      const bucket = byMonth.get(m)!;
      const segments = cats
        .map((c, idx) => ({
          name: c,
          amount: bucket.get(c) || 0,
          color: invPalette[idx % invPalette.length]
        }))
        .filter((s) => s.amount > 0);
      const total = segments.reduce((s, x) => s + x.amount, 0);
      return { label: m, total, segments };
    });
  });

  const allocationTimelineColumns = $derived.by((): StackedBarColumn[] => {
    if (!Array.isArray(allocationTimeline) || allocationTimeline.length === 0) return [];
    const topNames = allocationItems.slice(0, 8).map((a) => a.name || a.account || "Asset");
    const topSet = new Set(topNames);
    return allocationTimeline.slice(-24).map((pt) => {
      const source = pt.aggregates || (pt as unknown as Record<string, AllocationAggregate>);
      const vals = Object.values(source || {});
      const byName = new Map<string, number>();
      let other = 0;
      let total = 0;
      for (const a of vals) {
        const amt = Math.max(0, a.market_amount || 0);
        if (amt <= 0) continue;
        total += amt;
        const n = a.name || a.account || "Asset";
        if (topSet.has(n)) byName.set(n, (byName.get(n) || 0) + amt);
        else other += amt;
      }
      const segments = topNames
        .map((n, idx) => ({
          name: n,
          amount: byName.get(n) || 0,
          color: invPalette[idx % invPalette.length]
        }))
        .filter((s) => s.amount > 0);
      if (other > 0) segments.push({ name: "Other", amount: other, color: "#94a3b8" });
      return { label: (pt.date || "").slice(0, 7), total, segments };
    });
  });

  function toggleAssetSort(field: AssetSortField) {
    if (assetSortField === field) {
      assetSortAsc = !assetSortAsc;
    } else {
      assetSortField = field;
      assetSortAsc = field === "group";
    }
  }

  const filteredAssets = $derived.by(() => {
    const q = search.trim().toLowerCase();
    const list = assetBreakdowns.filter(
      (a) =>
        (!filterInactive || !a.inactive) &&
        (!hideZeroAssets || Math.abs(a.marketAmount || 0) > 0.01) &&
        (!q || a.group.toLowerCase().includes(q))
    );
    if (!flatAssets) {
      return [...list].sort((a, b) => a.group.localeCompare(b.group));
    }
    return [...list].sort((a, b) => {
      const av = assetSortField === "group" ? a.group : Number(a[assetSortField] || 0);
      const bv = assetSortField === "group" ? b.group : Number(b[assetSortField] || 0);
      const cmp =
        typeof av === "string" && typeof bv === "string"
          ? av.localeCompare(bv)
          : (av as number) - (bv as number);
      return assetSortAsc ? cmp : -cmp;
    });
  });

  function formatOriginalText(a: AssetBreakdown): string {
    if (a.originalBalances && a.originalBalances.length > 0) {
      return a.originalBalances.map((ob) => formatMoney(ob.amount, ob.currency)).join(" · ");
    }
    return "";
  }

  function exportAssetsCsv() {
    const headers = [
      "Account",
      "Investment Amount",
      "Withdrawal Amount",
      "Balance Units",
      "Original Value",
      "Market Value",
      "Change",
      "XIRR",
      "Absolute Return"
    ];
    const lines = [
      headers.join(","),
      ...filteredAssets.map((a) =>
        [
          `"${a.group.replace(/"/g, '""')}"`,
          (a.investmentAmount || 0).toFixed(2),
          (a.withdrawalAmount || 0).toFixed(2),
          (a.balanceUnits || 0).toFixed(4),
          `"${formatOriginalText(a).replace(/"/g, '""')}"`,
          (a.marketAmount || 0).toFixed(2),
          (a.gainAmount || 0).toFixed(2),
          (a.xirr || 0).toFixed(2),
          ((a.absoluteReturn || 0) * 100).toFixed(2)
        ].join(",")
      )
    ];
    const blob = new Blob([lines.join("\n")], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `paisa-assets-${asOfDate}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function exportAssetsExcel() {
    const rowsHtml = filteredAssets
      .map(
        (a) =>
          `<tr><td>${a.group}</td><td>${(a.investmentAmount || 0).toFixed(2)}</td><td>${(a.withdrawalAmount || 0).toFixed(2)}</td><td>${(a.balanceUnits || 0).toFixed(4)}</td><td>${formatOriginalText(a)}</td><td>${(a.marketAmount || 0).toFixed(2)}</td><td>${(a.gainAmount || 0).toFixed(2)}</td><td>${(a.xirr || 0).toFixed(2)}</td><td>${((a.absoluteReturn || 0) * 100).toFixed(2)}%</td></tr>`
      )
      .join("");
    const html = `<table><thead><tr><th>Account</th><th>Investment Amount</th><th>Withdrawal Amount</th><th>Balance Units</th><th>Original Value</th><th>Market Value</th><th>Change</th><th>XIRR</th><th>Absolute Return</th></tr></thead><tbody>${rowsHtml}</tbody></table>`;
    const blob = new Blob([html], { type: "application/vnd.ms-excel" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `paisa-assets-${asOfDate}.xls`;
    a.click();
    URL.revokeObjectURL(url);
  }

  async function runRebalance() {
    rebalanceLoading = true;
    try {
      const res = await fetch("/api/allocation/rebalance", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ cash: Number(rebalanceCash) || 0 })
      });
      if (res.ok) rebalanceResult = await res.json();
    } finally {
      rebalanceLoading = false;
    }
  }
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Wealth</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Net worth, asset balances, investments, allocation, income, and debt
      </p>
    </div>

    {#if activeTab === "overview"}
      <div
        class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
      >
        {#each [1, 3, 5, 0] as y}
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {yearsWindow === y
              ? 'bg-[var(--surface-elevated)] font-semibold text-[var(--text)]'
              : 'text-[var(--muted)] hover:text-[var(--text)]'}"
            onclick={() => (yearsWindow = y)}
          >
            {y === 0 ? "All" : `${y}Y`}
          </button>
        {/each}
      </div>
    {:else if activeTab === "assets"}
      <div class="flex flex-wrap items-center gap-2">
        <input
          type="date"
          bind:value={asOfDate}
          onchange={refreshAssets}
          class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)] tabular-nums"
        />
        {#if availableCurrencies.length > 1}
          <select
            bind:value={reportCurrency}
            onchange={refreshAssets}
            class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
          >
            <option value="">Default Currency</option>
            {#each availableCurrencies as cur}
              <option value={cur}>{cur}</option>
            {/each}
          </select>
        {/if}
        <input
          type="search"
          bind:value={search}
          placeholder="Filter asset accounts…"
          class="px-3 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
        />
        <label class="inline-flex items-center gap-1.5 text-xs text-[var(--muted)] cursor-pointer">
          <input type="checkbox" bind:checked={flatAssets} />
          Flat Accounts
        </label>
        <label class="inline-flex items-center gap-1.5 text-xs text-[var(--muted)] cursor-pointer">
          <input type="checkbox" bind:checked={filterInactive} />
          Filter Inactive
        </label>
        <label class="inline-flex items-center gap-1.5 text-xs text-[var(--muted)] cursor-pointer">
          <input type="checkbox" bind:checked={hideZeroAssets} />
          Filter 0
        </label>
        <button
          type="button"
          onclick={exportAssetsCsv}
          class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
        >
          CSV
        </button>
        <button
          type="button"
          onclick={exportAssetsExcel}
          class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
        >
          Excel
        </button>
      </div>
    {/if}
  </div>

  <!-- Underline Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [{ id: "overview", label: "Overview" }, { id: "assets", label: "Assets" }, { id: "investments", label: "Investments" }, { id: "allocation", label: "Allocation" }, { id: "portfolio", label: "Portfolio" }, { id: "income", label: "Inv. Income" }, { id: "liabilities", label: "Debt & Cards" }] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 transition-colors whitespace-nowrap cursor-pointer {activeTab ===
        tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)] hover:text-[var(--text)]'}"
        onclick={() => (activeTab = tab.id as WealthTab)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Failed to load wealth data</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "overview"}
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Net Worth
        </div>
        <div class="text-xl font-semibold tabular-nums mt-1">
          {formatMoney(latestPoint?.balanceAmount || 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Total Gain
        </div>
        <div
          class="text-xl font-semibold tabular-nums mt-1 {(latestPoint?.gainAmount || 0) >= 0
            ? 'text-[var(--gain)]'
            : 'text-[var(--loss)]'}"
        >
          {formatMoney(latestPoint?.gainAmount || 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Inv. Return
        </div>
        <div
          class="text-xl font-semibold tabular-nums mt-1 {(latestPoint?.investment_return || 0) >= 0
            ? 'text-[var(--gain)]'
            : 'text-[var(--loss)]'}"
        >
          {formatMoney(latestPoint?.investment_return ?? latestPoint?.gainAmount ?? 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          FX Impact
        </div>
        <div
          class="text-xl font-semibold tabular-nums mt-1 {(latestPoint?.fx_impact || 0) >= 0
            ? 'text-[var(--warning)]'
            : 'text-[var(--loss)]'}"
        >
          {formatMoney(latestPoint?.fx_impact || 0)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Portfolio XIRR
        </div>
        <div class="text-xl font-semibold tabular-nums mt-1">{xirr.toFixed(1)}%</div>
      </div>
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--muted)]">
        <span class="font-semibold uppercase tracking-wider">Net Worth Timeline</span>
        <div class="flex flex-wrap items-center gap-4">
          <label class="inline-flex items-center gap-1.5 cursor-pointer text-[var(--text)]">
            <input type="checkbox" bind:checked={showFxImpact} />
            Show FX Impact
          </label>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-3 h-0.5 bg-[var(--accent)] inline-block"></span>
            Net Worth
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-3 h-0.5 bg-[var(--muted)] inline-block"></span>
            Net Investment
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--gain)]/40 inline-block"></span>
            Gain
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--loss)]/40 inline-block"></span>
            Loss
          </span>
          {#if showFxImpact}
            <span class="inline-flex items-center gap-1.5">
              <span class="w-2.5 h-2.5 rounded-xs bg-[var(--accent)]/30 inline-block"></span>
              Investment Return
            </span>
            <span class="inline-flex items-center gap-1.5">
              <span class="w-2.5 h-2.5 rounded-xs bg-[var(--warning)]/40 inline-block"></span>
              FX Impact
            </span>
          {/if}
        </div>
      </div>
      <LineAreaChart data={windowedTimeline} {showFxImpact} />
    </div>
  {:else if activeTab === "assets"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("group")}
                >
                  Account {assetSortField === "group" ? (assetSortAsc ? "▲" : "▼") : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("investmentAmount")}
                >
                  Investment {assetSortField === "investmentAmount"
                    ? assetSortAsc
                      ? "▲"
                      : "▼"
                    : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("withdrawalAmount")}
                >
                  Withdrawal {assetSortField === "withdrawalAmount"
                    ? assetSortAsc
                      ? "▲"
                      : "▼"
                    : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("balanceUnits")}
                >
                  Balance Units {assetSortField === "balanceUnits"
                    ? assetSortAsc
                      ? "▲"
                      : "▼"
                    : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">Original Value</th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("marketAmount")}
                >
                  Market Value {assetSortField === "marketAmount" ? (assetSortAsc ? "▲" : "▼") : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("gainAmount")}
                >
                  Change {assetSortField === "gainAmount" ? (assetSortAsc ? "▲" : "▼") : ""}
                </button>
              </th>
              <th class="py-2.5 px-3 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("xirr")}
                >
                  XIRR {assetSortField === "xirr" ? (assetSortAsc ? "▲" : "▼") : ""}
                </button>
              </th>
              <th class="py-2.5 px-4 text-right font-medium">
                <button
                  type="button"
                  class="cursor-pointer hover:text-[var(--text)]"
                  onclick={() => toggleAssetSort("absoluteReturn")}
                >
                  Abs. Return {assetSortField === "absoluteReturn"
                    ? assetSortAsc
                      ? "▲"
                      : "▼"
                    : ""}
                </button>
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each filteredAssets as asset}
              {@const depth = Math.max(0, asset.group.split(":").length - 2)}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2.5 px-4 font-medium whitespace-nowrap">
                  <a
                    href={appPath(`/accounts/${encodeURIComponent(asset.group)}`)}
                    class="hover:text-[var(--accent)] hover:underline"
                    style={!flatAssets ? `padding-left: ${depth * 16}px` : ""}
                  >
                    {!flatAssets ? asset.group.split(":").pop() : asset.group}
                  </a>
                </td>
                <td class="py-2.5 px-3 text-right text-xs text-[var(--muted)]">
                  {asset.investmentAmount
                    ? formatMoney(asset.investmentAmount, reportCurrency || undefined)
                    : ""}
                </td>
                <td class="py-2.5 px-3 text-right text-xs text-[var(--muted)]">
                  {asset.withdrawalAmount
                    ? formatMoney(asset.withdrawalAmount, reportCurrency || undefined)
                    : ""}
                </td>
                <td class="py-2.5 px-3 text-right text-xs">
                  {asset.balanceUnits
                    ? asset.balanceUnits.toLocaleString(undefined, { maximumFractionDigits: 4 })
                    : ""}
                </td>
                <td class="py-2.5 px-3 text-right text-xs font-medium whitespace-nowrap">
                  {#if asset.originalBalances && asset.originalBalances.length > 0}
                    {#each asset.originalBalances as ob}
                      <div>{formatMoney(ob.amount, ob.currency)}</div>
                    {/each}
                  {/if}
                </td>
                <td class="py-2.5 px-3 text-right font-semibold">
                  {asset.marketAmount
                    ? formatMoney(asset.marketAmount, reportCurrency || undefined)
                    : ""}
                </td>
                <td
                  class="py-2.5 px-3 text-right text-xs {(asset.gainAmount || 0) >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {asset.gainAmount
                    ? formatMoney(asset.gainAmount, reportCurrency || undefined)
                    : ""}
                </td>
                <td
                  class="py-2.5 px-3 text-right text-xs {(asset.xirr || 0) >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {asset.xirr ? `${asset.xirr.toFixed(2)}%` : ""}
                </td>
                <td
                  class="py-2.5 px-4 text-right text-xs {(asset.absoluteReturn || 0) >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {asset.absoluteReturn ? `${(asset.absoluteReturn * 100).toFixed(2)}%` : ""}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else if activeTab === "investments"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if gains.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">No investment holdings found.</div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Holding</th>
                <th class="py-2.5 px-3 text-right font-medium">Invested</th>
                <th class="py-2.5 px-3 text-right font-medium">Market Value</th>
                <th class="py-2.5 px-3 text-right font-medium">Price Gain</th>
                <th class="py-2.5 px-3 text-right font-medium">Income</th>
                <th class="py-2.5 px-3 text-right font-medium">Total Return</th>
                <th class="py-2.5 px-3 text-right font-medium">TTM Yield</th>
                <th class="py-2.5 px-4 text-right font-medium">XIRR</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each gains as g}
                <tr
                  class="hover:bg-[var(--surface-elevated)]/50 cursor-pointer"
                  onclick={async () => {
                    selectedHolding = g.account;
                    holdingDetail = await apiGet<Record<string, unknown>>(
                      `/api/gain/${encodeURIComponent(g.account)}`
                    ).catch(() => null);
                  }}
                >
                  <td class="py-2.5 px-4 font-medium">{g.account}</td>
                  <td class="py-2.5 px-3 text-right">{formatMoney(g.investmentAmount || 0)}</td>
                  <td class="py-2.5 px-3 text-right font-semibold">
                    {formatMoney(g.marketAmount || 0)}
                  </td>
                  <td
                    class="py-2.5 px-3 text-right {(g.price_appreciation ?? g.gainAmount ?? 0) >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {formatMoney(g.price_appreciation ?? g.gainAmount ?? 0)}
                  </td>
                  <td class="py-2.5 px-3 text-right text-[var(--gain)]">
                    {formatMoney(g.income_received || 0)}
                  </td>
                  <td
                    class="py-2.5 px-3 text-right font-medium {(g.total_return ??
                      g.gainAmount ??
                      0) >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {formatMoney(g.total_return ?? g.gainAmount ?? 0)}
                  </td>
                  <td class="py-2.5 px-3 text-right text-xs text-[var(--muted)]">
                    {g.ttm_yield ? `${(g.ttm_yield * 100).toFixed(2)}%` : "—"}
                  </td>
                  <td class="py-2.5 px-4 text-right">{(g.xirr || 0).toFixed(1)}%</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>

    {#if selectedHolding && holdingDetail}
      <div class="mt-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-2">
        <div class="flex items-center justify-between">
          <h4 class="text-sm font-semibold">Holding Detail: {selectedHolding}</h4>
          <button
            type="button"
            onclick={() => (selectedHolding = null)}
            class="text-xs text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
          >
            Close
          </button>
        </div>
        <pre
          class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto max-h-60">{JSON.stringify(
            holdingDetail,
            null,
            2
          )}</pre>
      </div>
    {/if}

    {#if monthlyInvestmentColumns.length > 0}
      <div class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Monthly Investment Timeline (Last 24 Months)
        </div>
        <StackedBarChart columns={monthlyInvestmentColumns} />
      </div>
    {/if}

    {#if investmentYearly.length > 0}
      {@const maxFyInv = Math.max(
        1,
        ...investmentYearly.map((y) => Math.abs(y.net_investment ?? y.invested ?? 0))
      )}
      <div class="mt-5 grid grid-cols-1 lg:grid-cols-12 gap-5">
        <!-- Chart 2: Financial Year Investment Timeline -->
        <div
          class="lg:col-span-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
        >
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Financial Year Investment Timeline
          </div>
          <div class="h-44 flex items-end gap-2 pt-4">
            {#each investmentYearly as y}
              {@const val = y.net_investment ?? y.invested ?? 0}
              {@const h = Math.max(2, (Math.abs(val) / maxFyInv) * 130)}
              {@const fyLabel =
                y.start_date && y.end_date
                  ? `${y.start_date.slice(0, 4)}-${y.end_date.slice(2, 4)}`
                  : y.year || "—"}
              <div
                class="flex-1 flex flex-col items-center gap-1 min-w-0"
                title="{fyLabel}: {formatMoney(val)}"
              >
                <div class="w-full flex items-end justify-center h-[130px]">
                  <div
                    class="w-6 rounded-t {val >= 0
                      ? 'bg-[var(--accent)]/85'
                      : 'bg-[var(--loss)]/85'}"
                    style="height: {h}px"
                  ></div>
                </div>
                <span class="text-[10px] text-[var(--muted)] truncate">{fyLabel}</span>
              </div>
            {/each}
          </div>
        </div>

        <!-- Yearly Summary Cards / Tables -->
        <div class="lg:col-span-7 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each [...investmentYearly].reverse() as y}
            {@const fyLabel =
              y.start_date && y.end_date
                ? `${y.start_date.slice(0, 4)} - ${y.end_date.slice(2, 4)}`
                : y.year || "—"}
            <div
              class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden text-xs"
            >
              <div
                class="px-4 py-2.5 border-b border-[var(--border)] bg-[var(--surface-elevated)]/40 font-semibold text-sm"
              >
                {fyLabel}
              </div>
              <table class="w-full text-left">
                <tbody class="divide-y divide-[var(--border)]/60 tabular-nums">
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Gross Salary Income</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.gross_salary_income || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Gross Other Income</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.gross_other_income || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Tax</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.net_tax || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Net Income</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.net_income || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Net Expense</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.net_expense || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Investment</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{formatMoney(y.net_investment || 0)}</td
                    >
                  </tr>
                  <tr class="hover:bg-[var(--surface-elevated)]/30">
                    <td class="py-1.5 px-4 text-[var(--muted)]">Savings Rate</td>
                    <td class="py-1.5 px-4 text-right font-semibold"
                      >{(y.savings_rate || 0).toFixed(2)}%</td
                    >
                  </tr>
                </tbody>
              </table>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {:else if activeTab === "allocation"}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <div
        class="lg:col-span-8 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
      >
        <div class="flex items-center justify-between">
          <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Asset Allocation
          </h3>
          <button
            type="button"
            onclick={() => {
              if (!targetsEditing) {
                const next: Record<string, number> = {};
                for (const a of allocationItems) {
                  const key = a.name || a.account || "Asset";
                  next[key] =
                    totalAllocation > 0 ? Math.round((a.market_amount / totalAllocation) * 100) : 0;
                }
                targetDraft = next;
              }
              targetsEditing = !targetsEditing;
              targetsMessage = null;
            }}
            class="text-xs text-[var(--accent)] hover:underline cursor-pointer"
          >
            {targetsEditing ? "Done" : "Edit Targets"}
          </button>
        </div>
        {#if targetsEditing}
          <div
            class="p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] space-y-2 text-xs"
          >
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {#each Object.keys(targetDraft) as key}
                <div class="flex items-center justify-between gap-2">
                  <span class="truncate">{key}</span>
                  <input
                    type="number"
                    min="0"
                    max="100"
                    bind:value={targetDraft[key]}
                    class="w-20 px-2 py-1 rounded bg-[var(--surface)] border border-[var(--border)] text-right tabular-nums"
                  />
                </div>
              {/each}
            </div>
            <div class="flex items-center justify-between pt-1">
              <span class="text-[var(--muted)]">{targetsMessage || ""}</span>
              <button
                type="button"
                disabled={targetsSaving}
                onclick={async () => {
                  targetsSaving = true;
                  const payload = Object.entries(targetDraft).map(([name, target]) => ({
                    name,
                    target: Number(target) || 0,
                    drift: 5,
                    accounts: [],
                    commodities: []
                  }));
                  const res = await fetch("/api/allocation/targets", {
                    method: "POST",
                    headers: { "Content-Type": "application/json", ...authHeaders() },
                    body: JSON.stringify(payload)
                  });
                  targetsSaving = false;
                  targetsMessage = res.ok ? "Targets saved." : "Failed to save targets.";
                }}
                class="px-3 py-1 rounded bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
              >
                {targetsSaving ? "Saving…" : "Save Targets"}
              </button>
            </div>
          </div>
        {/if}
        {#if allocationItems.length === 0}
          <p class="text-sm text-[var(--muted)]">No allocation data available.</p>
        {:else}
          <!-- Allocation by Category by Value (Proportional Stacked Bar) -->
          <div class="h-5 rounded-lg overflow-hidden flex w-full bg-[var(--surface-elevated)]">
            {#each allocationItems.slice(0, 8) as item, idx}
              {@const pct = totalAllocation > 0 ? (item.market_amount / totalAllocation) * 100 : 0}
              {#if pct > 0.5}
                <div
                  style="width: {pct}%; background: {invPalette[idx % invPalette.length]}"
                  title="{item.name || item.account || 'Asset'}: {formatMoney(
                    item.market_amount
                  )} ({pct.toFixed(1)}%)"
                ></div>
              {/if}
            {/each}
          </div>
          <div class="space-y-3 pt-1">
            {#each allocationItems.slice(0, 15) as item, idx}
              {@const pct = totalAllocation > 0 ? (item.market_amount / totalAllocation) * 100 : 0}
              <div class="space-y-1 text-xs">
                <div class="flex justify-between">
                  <span class="font-medium truncate inline-flex items-center gap-1.5">
                    <span
                      class="w-2 h-2 rounded-xs shrink-0"
                      style="background: {invPalette[idx % invPalette.length]}"
                    ></span>
                    {item.name || item.account || "Asset"}
                  </span>
                  <span class="tabular-nums">
                    <strong>{formatMoney(item.market_amount)}</strong>
                    <span class="text-[var(--muted)] ml-1">({pct.toFixed(1)}%)</span>
                  </span>
                </div>
                <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                  <div
                    class="h-full rounded-full"
                    style="width: {pct}%; background: {invPalette[idx % invPalette.length]}"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div
        class="lg:col-span-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
      >
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Currency Exposure
        </h3>
        {#if currencyExposure.length === 0}
          <p class="text-sm text-[var(--muted)]">Single currency portfolio.</p>
        {:else}
          {@const fxTotal = currencyExposure.reduce((s, e) => s + (e.amount || 0), 0)}
          <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
            {#each currencyExposure as c}
              {@const fxPct =
                fxTotal > 0 ? ((c.amount || 0) / fxTotal) * 100 : (c.percent ?? c.percentage ?? 0)}
              <div class="py-2 flex justify-between gap-2">
                <span class="font-medium">{c.currency}</span>
                <span>
                  <strong class="mr-1.5">{formatMoney(c.amount || 0)}</strong>
                  <span class="text-[var(--muted)]">({fxPct.toFixed(1)}%)</span>
                </span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    {#if allocationTargets.length > 0}
      <div class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Target Allocation vs Current ({allocationTargets.length})
        </h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {#each allocationTargets as t}
            <div
              class="rounded-lg bg-[var(--surface-elevated)]/60 border border-[var(--border)] p-3 space-y-1.5 text-xs"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-semibold truncate">{t.name}</span>
                {#if t.status}
                  <span
                    class="px-1.5 py-0.5 rounded bg-[var(--surface)] border border-[var(--border)] capitalize text-[10px]"
                  >
                    {t.status}
                  </span>
                {/if}
              </div>
              <div class="flex justify-between tabular-nums text-[var(--muted)]">
                <span
                  >Current: <strong class="text-[var(--text)]"
                    >{(t.current || 0).toFixed(1)}%</strong
                  ></span
                >
                <span>Target: {(t.target || 0).toFixed(1)}%</span>
              </div>
              <div class="h-1.5 rounded-full bg-[var(--surface)] overflow-hidden">
                <div
                  class="h-full rounded-full bg-[var(--accent)]"
                  style="width: {Math.min(100, t.current || 0)}%"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <div class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Cash Rebalance Calculator
        </h3>
        <div class="flex items-center gap-2 text-xs">
          <input
            type="number"
            bind:value={rebalanceCash}
            class="w-32 px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
          />
          <button
            type="button"
            disabled={rebalanceLoading}
            onclick={runRebalance}
            class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
          >
            {rebalanceLoading ? "Calculating…" : "Calculate Rebalance"}
          </button>
        </div>
      </div>
      {#if rebalanceResult}
        {@const items = rebalanceResult.suggestions || rebalanceResult.allocations || []}
        {#if items.length > 0}
          <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
            {#each items as item}
              <div class="py-2 flex justify-between gap-2">
                <span class="font-medium">{item.name || "Asset"}</span>
                <span class="font-semibold text-[var(--accent)]"
                  >{formatMoney(item.amount || 0)}</span
                >
              </div>
            {/each}
          </div>
        {:else}
          <pre
            class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto">{JSON.stringify(
              rebalanceResult,
              null,
              2
            )}</pre>
        {/if}
      {/if}
    </div>

    {#if allocationTimelineColumns.length > 0}
      <div class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Allocation Timeline
        </div>
        <StackedBarChart columns={allocationTimelineColumns} />
      </div>
    {/if}

    {#if allocationItems.length > 0}
      <div
        class="mt-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden"
      >
        <div
          class="px-4 py-3 border-b border-[var(--border)] text-xs font-semibold uppercase tracking-wider text-[var(--muted)]"
        >
          Portfolio Allocation Table
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Account</th>
                <th class="py-2.5 px-4 text-right font-medium">Market Value</th>
                <th class="py-2.5 px-4 text-right font-medium">% of Portfolio</th>
                <th class="py-2.5 px-4 font-medium w-48">Share</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each allocationItems as item, idx}
                {@const pct =
                  totalAllocation > 0 ? (item.market_amount / totalAllocation) * 100 : 0}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-4 font-medium">{item.account || item.name || "Asset"}</td>
                  <td class="py-2 px-4 text-right font-semibold"
                    >{formatMoney(item.market_amount)}</td
                  >
                  <td class="py-2 px-4 text-right">{pct.toFixed(2)}%</td>
                  <td class="py-2 px-4">
                    <div class="h-2 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                      <div
                        class="h-full rounded-full"
                        style="width: {pct}%; background: {invPalette[idx % invPalette.length]}"
                      ></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {:else if activeTab === "portfolio"}
    {#if portfolioSecurityType.length === 0 && portfolioRating.length === 0 && portfolioIndustry.length === 0 && portfolioNameAndType.length === 0}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
      >
        No underlying portfolio breakdown metadata available.
      </div>
    {:else}
      {@const portfolioGroups = [
        {
          id: "security_type",
          title: "Security Type",
          items: filterPortfolioItems(portfolioSecurityType)
        },
        { id: "rating", title: "Credit Rating", items: filterPortfolioItems(portfolioRating) },
        { id: "industry", title: "Industry", items: filterPortfolioItems(portfolioIndustry) },
        {
          id: "security",
          title: "Security / Holding",
          items: filterPortfolioItems(portfolioNameAndType)
        }
      ].filter((g) => portfolioGroupTab === "all" || portfolioGroupTab === g.id)}
      <div class="space-y-5">
        <div
          class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-center justify-between gap-3 text-xs"
        >
          {#if portfolioCommodities.length > 0}
            <div class="flex flex-wrap items-center gap-3">
              <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">Funds:</span>
              {#each portfolioCommodities as comm}
                <label class="inline-flex items-center gap-1.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedPortfolioCommodities.includes(comm)}
                    onchange={() => {
                      selectedPortfolioCommodities = selectedPortfolioCommodities.includes(comm)
                        ? selectedPortfolioCommodities.filter((c) => c !== comm)
                        : [...selectedPortfolioCommodities, comm];
                    }}
                  />
                  <span
                    class="w-2.5 h-2.5 rounded-full inline-block"
                    style="background: {fundColor(comm)}"
                  ></span>
                  <span>{comm}</span>
                </label>
              {/each}
            </div>
          {/if}

          <div class="flex flex-wrap items-center gap-2 ml-auto">
            <div
              class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5"
            >
              {#each [{ id: "all", label: "All" }, { id: "security_type", label: "Type" }, { id: "rating", label: "Rating" }, { id: "industry", label: "Industry" }, { id: "security", label: "Security" }] as pt}
                <button
                  type="button"
                  onclick={() => (portfolioGroupTab = pt.id as typeof portfolioGroupTab)}
                  class="px-2.5 py-1 rounded-md cursor-pointer {portfolioGroupTab === pt.id
                    ? 'bg-[var(--surface)] font-semibold'
                    : 'text-[var(--muted)]'}"
                >
                  {pt.label}
                </button>
              {/each}
            </div>
            <input
              type="search"
              bind:value={portfolioSearch}
              placeholder="Filter security, industry, or fund…"
              class="px-2.5 py-1 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none w-52"
            />
          </div>
        </div>

        {#each portfolioGroups as group (group.id)}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
            <div
              class="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between"
            >
              <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                {group.title} ({group.items.length})
              </h3>
            </div>
            {#if group.items.length === 0}
              <div class="p-6 text-center text-xs text-[var(--muted)]">No matching data.</div>
            {:else}
              <div class="overflow-x-auto max-h-[520px] overflow-y-auto">
                <table class="w-full text-xs text-left">
                  <thead
                    class="uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/60 sticky top-0"
                  >
                    <tr>
                      <th class="py-2 px-4 font-medium">{group.title}</th>
                      {#if group.id === "security"}
                        <th class="py-2 px-3 font-medium">Type</th>
                      {/if}
                      <th class="py-2 px-3 text-right font-medium">Amount</th>
                      <th class="py-2 px-3 text-right font-medium">Portfolio %</th>
                      <th class="py-2 px-3 font-medium w-44">Stacked Share</th>
                      <th class="py-2 px-4 font-medium">Fund Share Breakdown</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-[var(--border)] tabular-nums">
                    {#each group.items as item}
                      <tr class="hover:bg-[var(--surface-elevated)]/40">
                        <td class="py-2 px-4 font-medium">
                          {item.group || item.sub_group || "Other"}
                        </td>
                        {#if group.id === "security"}
                          <td class="py-2 px-3 text-[var(--muted)]">{item.sub_group || "—"}</td>
                        {/if}
                        <td class="py-2 px-3 text-right font-semibold whitespace-nowrap">
                          {formatMoney(item.amount)}
                        </td>
                        <td class="py-2 px-3 text-right whitespace-nowrap">
                          {item.percentage.toFixed(2)}%
                        </td>
                        <td class="py-2 px-3">
                          <div
                            class="h-2.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden flex"
                          >
                            {#each item.fundShares as fs}
                              <div
                                class="h-full"
                                style="width: {(fs.sharePct * item.percentage) /
                                  100}%; background: {fundColor(fs.fund)}"
                                title="{fs.fund}: {formatMoney(fs.amount)} ({fs.sharePct.toFixed(
                                  1
                                )}% of this row)"
                              ></div>
                            {/each}
                          </div>
                        </td>
                        <td class="py-2 px-4">
                          <div class="flex flex-wrap gap-1.5">
                            {#each item.fundShares as fs}
                              <span
                                class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)] text-[11px]"
                                title="{fs.fund}: {formatMoney(fs.amount)}"
                              >
                                <span
                                  class="w-2 h-2 rounded-full shrink-0"
                                  style="background: {fundColor(fs.fund)}"
                                ></span>
                                <span class="font-medium">{fs.fund}</span>
                                <span class="text-[var(--muted)]">{fs.sharePct.toFixed(1)}%</span>
                              </span>
                            {/each}
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {:else if activeTab === "income"}
    <div class="space-y-5">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            TTM Investment Income
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
            {formatMoney(investmentIncomeTotal)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Dividend (TTM)
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--accent)] mt-1">
            {formatMoney(dividendTTM)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Interest (TTM)
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(interestTTM)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Distribution (TTM)
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(distributionTTM)}
          </div>
        </div>
      </div>

      {#if investmentIncomeColumns.length > 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              {invIncomeGranularity === "monthly" ? "Monthly" : "Yearly"} Investment Income Timeline
            </span>
            <div class="flex flex-wrap items-center gap-3">
              <div class="flex items-center gap-3 text-[11px] text-[var(--muted)]">
                <span class="inline-flex items-center gap-1"
                  ><span class="w-2.5 h-2.5 rounded-sm bg-[#3b82f6]"></span>Dividend</span
                >
                <span class="inline-flex items-center gap-1"
                  ><span class="w-2.5 h-2.5 rounded-sm bg-[#10b981]"></span>Interest</span
                >
                <span class="inline-flex items-center gap-1"
                  ><span class="w-2.5 h-2.5 rounded-sm bg-[#f59e0b]"></span>Distribution</span
                >
              </div>
              <div
                class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
              >
                <button
                  type="button"
                  onclick={() => (invIncomeGranularity = "monthly")}
                  class="px-2.5 py-1 rounded-md cursor-pointer {invIncomeGranularity === 'monthly'
                    ? 'bg-[var(--surface)] font-semibold'
                    : 'text-[var(--muted)]'}"
                >
                  Monthly
                </button>
                <button
                  type="button"
                  onclick={() => (invIncomeGranularity = "yearly")}
                  class="px-2.5 py-1 rounded-md cursor-pointer {invIncomeGranularity === 'yearly'
                    ? 'bg-[var(--surface)] font-semibold'
                    : 'text-[var(--muted)]'}"
                >
                  Yearly
                </button>
              </div>
            </div>
          </div>
          <StackedBarChart columns={investmentIncomeColumns} />
        </div>
      {/if}

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if investmentIncomeHoldings.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">
            No investment income recorded.
          </div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
              >
                <tr>
                  <th class="py-2.5 px-4 font-medium">Holding</th>
                  <th class="py-2.5 px-3 font-medium">Type</th>
                  <th class="py-2.5 px-3 text-right font-medium">Total Income</th>
                  <th class="py-2.5 px-3 text-right font-medium">TTM Income</th>
                  <th class="py-2.5 px-3 text-right font-medium">Balance</th>
                  <th class="py-2.5 px-3 text-right font-medium">TTM Yield</th>
                  <th class="py-2.5 px-4 text-right font-medium">Yearly Trend</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each investmentIncomeHoldings as h}
                  {@const spark = holdingSparklinePoints(h.yearly_income)}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2.5 px-4 font-medium"
                      >{h.holding || h.account || h.name || "—"}</td
                    >
                    <td class="py-2.5 px-3 text-xs text-[var(--muted)]">{h.type || "—"}</td>
                    <td class="py-2.5 px-3 text-right"
                      >{formatMoney(h.total_income ?? h.ttm_income ?? 0)}</td
                    >
                    <td class="py-2.5 px-3 text-right font-semibold text-[var(--gain)]">
                      {formatMoney(h.ttm_income || 0)}
                    </td>
                    <td class="py-2.5 px-3 text-right">{formatMoney(h.current_balance || 0)}</td>
                    <td class="py-2.5 px-3 text-right text-xs text-[var(--muted)]">
                      {(h.ttm_yield ?? h.yield)
                        ? `${((h.ttm_yield ?? h.yield ?? 0) * 100).toFixed(2)}%`
                        : "—"}
                    </td>
                    <td class="py-2.5 px-4 text-right">
                      {#if spark}
                        <svg viewBox="0 0 100 26" class="w-24 h-6 inline-block text-[var(--gain)]">
                          <polyline
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.75"
                            points={spark}
                          />
                        </svg>
                      {:else}
                        <span class="text-xs text-[var(--muted)]">—</span>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

      {#if aggregatedInvestmentIncome.length > 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div
            class="px-4 py-3 border-b border-[var(--border)] text-xs font-semibold uppercase tracking-wider text-[var(--muted)]"
          >
            Investment Income Timeline ({aggregatedInvestmentIncome.length}
            {invIncomeGranularity === "monthly" ? "months" : "years"})
          </div>
          <div class="overflow-x-auto max-h-80 overflow-y-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50 sticky top-0"
              >
                <tr>
                  <th class="py-2 px-4 font-medium"
                    >{invIncomeGranularity === "monthly" ? "Month" : "Year"}</th
                  >
                  <th class="py-2 px-3 text-right font-medium">Dividend</th>
                  <th class="py-2 px-3 text-right font-medium">Interest</th>
                  <th class="py-2 px-3 text-right font-medium">Distribution</th>
                  <th class="py-2 px-4 text-right font-medium">Total</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each [...aggregatedInvestmentIncome].reverse() as pt}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2 px-4 font-medium">{pt.label}</td>
                    <td class="py-2 px-3 text-right">{formatMoney(pt.dividend)}</td>
                    <td class="py-2 px-3 text-right">{formatMoney(pt.interest)}</td>
                    <td class="py-2 px-3 text-right">{formatMoney(pt.distribution)}</td>
                    <td class="py-2 px-4 text-right font-semibold text-[var(--gain)]">
                      {formatMoney(pt.total)}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <div class="space-y-5">
      {#if creditCards.length > 0}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each creditCards as cc}
            <button
              type="button"
              onclick={async () => {
                selectedCard = cc.account;
                cardDetail = await apiGet<Record<string, unknown>>(
                  `/api/credit_cards/${encodeURIComponent(cc.account)}`
                ).catch(() => null);
              }}
              class="text-left rounded-xl bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)] p-4 space-y-2 cursor-pointer"
            >
              <div class="flex justify-between items-center">
                <span class="font-semibold text-sm">{cc.name || cc.account}</span>
                {#if cc.due_date}
                  <span class="text-xs text-[var(--muted)]">Due {formatShortDate(cc.due_date)}</span
                  >
                {/if}
              </div>
              <div class="flex justify-between text-xs tabular-nums">
                <span class="text-[var(--muted)]">Balance</span>
                <span class="font-semibold text-[var(--loss)]">
                  {#if cc.originalBalances && cc.originalBalances.length > 0}
                    {cc.originalBalances
                      .map((ob) => formatMoney(ob.amount, ob.currency))
                      .join(" · ")}
                  {:else}
                    {formatMoney(cc.balance || 0, cc.currency || undefined)}
                  {/if}
                </span>
              </div>
            </button>
          {/each}
        </div>
        {#if selectedCard && cardDetail}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Card Statement Detail: {selectedCard}
              </span>
              <button
                type="button"
                onclick={() => (selectedCard = null)}
                class="text-xs text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
              >
                Close
              </button>
            </div>
            <pre
              class="text-xs font-mono bg-[var(--surface-elevated)] p-3 rounded-lg overflow-x-auto max-h-56">{JSON.stringify(
                cardDetail,
                null,
                2
              )}</pre>
          </div>
        {/if}
      {/if}

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if liabilities.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">No liabilities recorded.</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
              >
                <tr>
                  <th class="py-2.5 px-4 font-medium">Liability</th>
                  <th class="py-2.5 px-4 text-right font-medium">Drawn</th>
                  <th class="py-2.5 px-4 text-right font-medium">Repaid</th>
                  <th class="py-2.5 px-4 text-right font-medium">Interest</th>
                  <th class="py-2.5 px-4 text-right font-medium">Original Balance</th>
                  <th class="py-2.5 px-4 text-right font-medium">Balance</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each liabilities as l}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2.5 px-4 font-medium">{l.group}</td>
                    <td class="py-2.5 px-4 text-right">{formatMoney(l.drawn_amount || 0)}</td>
                    <td class="py-2.5 px-4 text-right text-[var(--gain)]">
                      {formatMoney(l.repaid_amount || 0)}
                    </td>
                    <td class="py-2.5 px-4 text-right text-[var(--muted)]">
                      {formatMoney(l.interest_amount || 0)}
                    </td>
                    <td class="py-2.5 px-4 text-right text-xs text-[var(--muted)]">
                      {#if l.originalBalances && l.originalBalances.length > 0}
                        {#each l.originalBalances as ob}
                          <div>{formatMoney(ob.amount, ob.currency)}</div>
                        {/each}
                      {:else}
                        —
                      {/if}
                    </td>
                    <td class="py-2.5 px-4 text-right font-semibold text-[var(--loss)]">
                      {formatMoney(l.balance_amount || 0)}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

      {#if interestAccounts.length > 0 || repayments.length > 0}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
          {#if interestAccounts.length > 0}
            <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
              <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Effective APR by Liability
              </h3>
              <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
                {#each interestAccounts as acc}
                  <div class="py-2 flex justify-between gap-2">
                    <span class="truncate font-medium">{acc.account}</span>
                    <span class="font-semibold">{(acc.apr || 0).toFixed(2)}% APR</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if repayments.length > 0}
            <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
              <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Recent Repayments ({repayments.length})
              </h3>
              <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
                {#each repayments.slice(-8).reverse() as rep}
                  <div class="py-2 flex justify-between gap-2">
                    <span class="truncate text-xs text-[var(--muted)]">
                      {formatShortDate(rep.date)}
                      {rep.account ? `· ${rep.account}` : ""}
                    </span>
                    <span class="font-medium text-[var(--gain)]">
                      {formatMoney(
                        rep.repaid_amount ?? rep.original_amount ?? rep.amount ?? 0,
                        rep.commodity || undefined
                      )}
                    </span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>
