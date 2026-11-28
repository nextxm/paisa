<script lang="ts">
  import dayjs from "dayjs";
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import BarTrendChart, { type MonthlyBarPoint } from "$lib/components/BarTrendChart.svelte";
  import SankeyChart from "$lib/components/SankeyChart.svelte";
  import StackedBarChart, { type StackedBarColumn } from "$lib/components/StackedBarChart.svelte";
  import { apiGet, formatMoney, runtime, type Posting } from "$lib/api";

  type StudioTab =
    | "cashflow"
    | "statement"
    | "sankey"
    | "income"
    | "expenses"
    | "budget"
    | "recurring";
  type CashFlowGranularity = "monthly" | "yearly";
  type ExpenseView = "breakdown" | "heatmap" | "trends";
  type SankeyPeriod = "month" | "quarter" | "year";

  interface CashFlowRow {
    date: string;
    income: number;
    expenses: number;
    investment: number;
    tax: number;
    liabilities: number;
    balance: number;
  }

  interface IncomeStatement {
    startingBalance: number;
    endingBalance: number;
    income?: Record<string, number>;
    interest?: Record<string, number>;
    pnl?: Record<string, number>;
    equity?: Record<string, number>;
    liabilities?: Record<string, number>;
    expenses?: Record<string, number>;
    tax?: Record<string, number>;
  }

  interface SankeyLink {
    source: string;
    target: string;
    value: number;
  }

  interface BudgetAccount {
    account: string;
    forecast: number;
    actual: number;
    rollover: number;
    available: number;
  }

  interface BudgetMonth {
    date: string;
    accounts: BudgetAccount[];
  }

  interface RecurringItem {
    name?: string;
    account?: string;
    payee?: string;
    narration?: string;
    amount?: number;
    next_date?: string;
    frequency?: string;
  }

  let activeTab = $state<StudioTab>("cashflow");
  $effect(() => {
    const tab = $page.url.searchParams.get("tab");
    if (
      tab === "cashflow" ||
      tab === "statement" ||
      tab === "sankey" ||
      tab === "income" ||
      tab === "expenses" ||
      tab === "budget" ||
      tab === "recurring"
    ) {
      activeTab = tab;
    }
  });
  let cfGranularity = $state<CashFlowGranularity>("monthly");
  let expenseView = $state<ExpenseView>("breakdown");
  let sankeyPeriod = $state<SankeyPeriod>("month");

  let monthsWindow = $state<number>(12);
  let selectedMonth = $state(dayjs().format("YYYY-MM"));
  let selectedYear = $state("");
  let groupDepth = $state<1 | 2>(1);

  let loading = $state(true);
  let error = $state<string | null>(null);

  let cashFlows = $state<CashFlowRow[]>([]);
  let monthWiseExpenses = $state<Record<string, Posting[]>>({});
  let monthWiseIncomes = $state<Record<string, Posting[]>>({});
  let monthWiseInvestments = $state<Record<string, Posting[]>>({});
  let monthWiseTaxes = $state<Record<string, Posting[]>>({});
  let monthWiseLiabilities = $state<Record<string, Posting[]>>({});
  let breakdownLevel = $state<1 | 2 | "all">(2);
  let expenseTrendGranularity = $state<"monthly" | "yearly">("monthly");
  let yearlyStatements = $state<Record<string, IncomeStatement>>({});
  let sankeyLinks = $state<SankeyLink[]>([]);
  let sankeyLoading = $state(false);
  let sankeyFrom = $state(dayjs().startOf("month").format("YYYY-MM-DD"));
  let sankeyTo = $state(dayjs().endOf("month").format("YYYY-MM-DD"));
  let sankeyCustomRange = $state(false);
  let sankeyExpenseOnly = $state(false);

  function setSankeyRangeForPeriod(period: SankeyPeriod) {
    const ref = selectedMonth ? dayjs(`${selectedMonth}-01`) : dayjs();
    if (period === "quarter") {
      const qStartMonth = Math.floor(ref.month() / 3) * 3;
      const start = ref.month(qStartMonth).startOf("month");
      sankeyFrom = start.format("YYYY-MM-DD");
      sankeyTo = start.add(2, "month").endOf("month").format("YYYY-MM-DD");
    } else if (period === "year") {
      sankeyFrom = ref.startOf("year").format("YYYY-MM-DD");
      sankeyTo = ref.endOf("year").format("YYYY-MM-DD");
    } else {
      sankeyFrom = ref.startOf("month").format("YYYY-MM-DD");
      sankeyTo = ref.endOf("month").format("YYYY-MM-DD");
    }
    sankeyCustomRange = false;
  }
  let dailyExpenses = $state<Array<{ date: string; amount: number; category?: string }>>([]);
  let dailyLoading = $state(false);
  let recurringView = $state<"list" | "calendar">("calendar");
  let budgetsByMonth = $state<Record<string, BudgetMonth>>({});
  let availableForBudgeting = $state(0);
  let recurringItems = $state<RecurringItem[]>([]);
  let yearlyIncomeCards = $state<
    Array<{
      year?: string;
      start_date?: string;
      end_date?: string;
      net_income?: number;
      net_tax?: number;
      gross_income?: number;
    }>
  >([]);

  function formatIncomeCardYear(yc: {
    year?: string;
    start_date?: string;
    end_date?: string;
  }): string {
    if (yc.year) return yc.year;
    const startY = yc.start_date ? dayjs(yc.start_date).year() : null;
    const endY = yc.end_date ? dayjs(yc.end_date).year() : null;
    if (startY !== null && endY !== null) {
      return startY === endY ? String(startY) : `FY ${startY}-${String(endY).slice(-2)}`;
    }
    return startY !== null ? String(startY) : "—";
  }
  let incomeTimeline = $state<Array<{ date: string; postings?: Posting[] }>>([]);
  let taxTimeline = $state<Array<{ date: string; postings?: Posting[] }>>([]);
  let ttmInvestmentIncome = $state(0);

  onMount(async () => {
    try {
      const [cfRes, expRes, stmtRes, budgetRes, recRes, incRes, invIncRes] = await Promise.all([
        apiGet<{ cash_flows?: CashFlowRow[] }>("/api/cash_flow"),
        apiGet<{
          month_wise?: {
            incomes?: Record<string, Posting[]>;
            expenses?: Record<string, Posting[]>;
            investments?: Record<string, Posting[]>;
            taxes?: Record<string, Posting[]>;
            liabilities?: Record<string, Posting[]>;
          };
        }>("/api/expense"),
        apiGet<{ yearly?: Record<string, IncomeStatement> }>("/api/income_statement").catch(() => ({
          yearly: {}
        })),
        apiGet<{
          budgetsByMonth?: Record<string, BudgetMonth>;
          availableForBudgeting?: number;
        }>("/api/budget").catch(() => ({ budgetsByMonth: {}, availableForBudgeting: 0 })),
        apiGet<{ transaction_sequences?: RecurringItem[] }>("/api/recurring").catch(() => ({
          transaction_sequences: []
        })),
        apiGet<{
          yearly_cards?: Array<{
            year?: string;
            start_date?: string;
            end_date?: string;
            net_income?: number;
            net_tax?: number;
            gross_income?: number;
          }>;
          income_timeline?: Array<{ date: string; postings?: Posting[] }>;
          tax_timeline?: Array<{ date: string; postings?: Posting[] }>;
        }>("/api/income").catch(() => ({
          yearly_cards: [],
          income_timeline: [],
          tax_timeline: []
        })),
        apiGet<{ ttm_total?: number }>("/api/income/investment").catch(() => ({ ttm_total: 0 }))
      ]);

      cashFlows = (cfRes.cash_flows || []).map((c) => ({
        ...c,
        income: Math.abs(c.income || 0),
        expenses: Math.abs(c.expenses || 0),
        investment: Math.abs(c.investment || 0),
        tax: Math.abs(c.tax || 0),
        liabilities: Math.abs(c.liabilities || 0)
      }));
      monthWiseExpenses = expRes.month_wise?.expenses || {};
      monthWiseIncomes = expRes.month_wise?.incomes || {};
      monthWiseInvestments = expRes.month_wise?.investments || {};
      monthWiseTaxes = expRes.month_wise?.taxes || {};
      monthWiseLiabilities = expRes.month_wise?.liabilities || {};
      yearlyStatements = stmtRes.yearly || {};
      budgetsByMonth = budgetRes.budgetsByMonth || {};
      availableForBudgeting = budgetRes.availableForBudgeting || 0;
      recurringItems = recRes.transaction_sequences || [];
      yearlyIncomeCards = incRes.yearly_cards || [];
      incomeTimeline = incRes.income_timeline || [];
      taxTimeline = incRes.tax_timeline || [];
      ttmInvestmentIncome = invIncRes.ttm_total || 0;

      const availableMonths = Object.keys(monthWiseExpenses).sort();
      if (availableMonths.length > 0 && !monthWiseExpenses[selectedMonth]) {
        selectedMonth = availableMonths[availableMonths.length - 1];
      }
      if (!sankeyCustomRange) {
        setSankeyRangeForPeriod(sankeyPeriod);
      }

      const stmtYears = Object.keys(yearlyStatements).sort().reverse();
      if (stmtYears.length > 0) {
        selectedYear = stmtYears[0];
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load cash flow data";
    } finally {
      loading = false;
    }
  });

  async function loadSankey() {
    sankeyLoading = true;
    try {
      const baseUrl =
        sankeyFrom && sankeyTo
          ? `/api/sankey?from=${encodeURIComponent(sankeyFrom)}&to=${encodeURIComponent(sankeyTo)}`
          : `/api/sankey?period=${encodeURIComponent(sankeyPeriod)}`;
      const url = sankeyExpenseOnly ? `${baseUrl}&expense_only=true` : baseUrl;
      const res = await apiGet<{ links?: SankeyLink[] }>(url);
      sankeyLinks = (res.links || []).slice().sort((a, b) => b.value - a.value);
    } catch {
      sankeyLinks = [];
    } finally {
      sankeyLoading = false;
    }
  }

  async function loadDailyExpenses() {
    if (!selectedMonth) return;
    dailyLoading = true;
    try {
      const from = dayjs(`${selectedMonth}-01`).startOf("month").format("YYYY-MM-DD");
      const to = dayjs(`${selectedMonth}-01`).endOf("month").format("YYYY-MM-DD");
      const res = await apiGet<{
        daily?: Array<{ date: string; amount: number; category?: string }>;
        expenses?: Array<{ date: string; amount: number; category?: string }>;
      }>(
        `/api/expense/daily?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&group_by=category`
      );
      dailyExpenses = res.daily || res.expenses || [];
    } catch {
      dailyExpenses = [];
    } finally {
      dailyLoading = false;
    }
  }

  $effect(() => {
    if (activeTab === "sankey") {
      const _p = sankeyPeriod;
      const _f = sankeyFrom;
      const _t = sankeyTo;
      const _e = sankeyExpenseOnly;
      void loadSankey();
    }
    if (activeTab === "expenses" && selectedMonth) {
      void loadDailyExpenses();
    }
  });

  const windowedFlows = $derived(monthsWindow > 0 ? cashFlows.slice(-monthsWindow) : cashFlows);

  const yearlyFlows = $derived.by(() => {
    const byYear = new Map<string, CashFlowRow>();
    for (const row of cashFlows) {
      const y = dayjs(row.date).format("YYYY");
      const existing = byYear.get(y) || {
        date: `${y}-01-01`,
        income: 0,
        expenses: 0,
        investment: 0,
        tax: 0,
        liabilities: 0,
        balance: 0
      };
      existing.income += row.income;
      existing.expenses += row.expenses;
      existing.investment += row.investment;
      existing.tax += row.tax;
      existing.liabilities += row.liabilities;
      existing.balance = row.balance;
      byYear.set(y, existing);
    }
    return Array.from(byYear.values());
  });

  const activeFlowRows = $derived(cfGranularity === "yearly" ? yearlyFlows : windowedFlows);

  const chartPoints = $derived<MonthlyBarPoint[]>(
    activeFlowRows.map((c) => ({
      date: c.date,
      income: c.income,
      expenses: c.expenses,
      investment: c.investment,
      liabilities: c.liabilities,
      balance: c.balance
    }))
  );

  const categoryPalette = [
    "#3b82f6",
    "#10b981",
    "#f59e0b",
    "#ef4444",
    "#8b5cf6",
    "#ec4899",
    "#14b8a6",
    "#6366f1",
    "#94a3b8"
  ];

  const expenseStackedColumns = $derived.by((): StackedBarColumn[] => {
    const months = Object.keys(monthWiseExpenses).sort();
    if (months.length === 0) return [];

    // Aggregate totals per category to pick top 8 categories consistently
    const catTotals = new Map<string, number>();
    const periodMap = new Map<string, Map<string, number>>();
    for (const m of months) {
      const pKey = expenseTrendGranularity === "yearly" ? m.slice(0, 4) : m;
      if (!periodMap.has(pKey)) periodMap.set(pKey, new Map());
      const bucket = periodMap.get(pKey)!;
      for (const p of monthWiseExpenses[m] || []) {
        const parts = p.account.split(":");
        const cat = (groupDepth === 1 ? parts.slice(0, 2) : parts.slice(0, 3))
          .join(":")
          .replace(/^Expenses:/, "");
        const amt = Math.max(0, p.amount || 0);
        if (amt <= 0) continue;
        catTotals.set(cat, (catTotals.get(cat) || 0) + amt);
        bucket.set(cat, (bucket.get(cat) || 0) + amt);
      }
    }

    const topCats = Array.from(catTotals.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 8)
      .map(([c]) => c);
    const topSet = new Set(topCats);

    const periodKeys = Array.from(periodMap.keys()).sort();
    const slicedKeys =
      expenseTrendGranularity === "monthly" && monthsWindow > 0
        ? periodKeys.slice(-monthsWindow)
        : periodKeys;

    return slicedKeys.map((pk) => {
      const bucket = periodMap.get(pk)!;
      const segments: Array<{ name: string; amount: number; color: string }> = [];
      let other = 0;
      let total = 0;
      for (const [cat, amt] of bucket.entries()) {
        total += amt;
        if (!topSet.has(cat)) other += amt;
      }
      topCats.forEach((cat, idx) => {
        const amt = bucket.get(cat) || 0;
        if (amt > 0) segments.push({ name: cat, amount: amt, color: categoryPalette[idx] });
      });
      if (other > 0) {
        segments.push({ name: "Other", amount: other, color: categoryPalette[8] });
      }
      return {
        label: expenseTrendGranularity === "yearly" ? pk : dayjs(`${pk}-01`).format("MMM 'YY"),
        total,
        segments
      };
    });
  });

  const momMonthlySeries = $derived.by(() => {
    const months = Object.keys(monthWiseExpenses).sort().slice(-13);
    const totals = months.map((m) => ({
      month: m,
      label: dayjs(`${m}-01`).format("MMM 'YY"),
      total: (monthWiseExpenses[m] || []).reduce((s, p) => s + Math.max(0, p.amount || 0), 0)
    }));
    return totals.slice(1).map((cur, idx) => {
      const prev = totals[idx].total;
      const deltaPct = prev > 0 ? ((cur.total - prev) / prev) * 100 : 0;
      return { ...cur, prev, deltaPct };
    });
  });

  const momCategoryVariance = $derived.by(() => {
    const prevMonth = dayjs(`${selectedMonth}-01`).subtract(1, "month").format("YYYY-MM");
    const sumCats = (m: string) => {
      const map = new Map<string, number>();
      for (const p of monthWiseExpenses[m] || []) {
        const cat = p.account
          .split(":")
          .slice(0, 2)
          .join(":")
          .replace(/^Expenses:/, "");
        map.set(cat, (map.get(cat) || 0) + Math.max(0, p.amount || 0));
      }
      return map;
    };
    const curMap = sumCats(selectedMonth);
    const prevMap = sumCats(prevMonth);
    const allCats = new Set([...curMap.keys(), ...prevMap.keys()]);
    return Array.from(allCats)
      .map((name) => {
        const cur = curMap.get(name) || 0;
        const prev = prevMap.get(name) || 0;
        const diff = cur - prev;
        const pct = prev > 0 ? (diff / prev) * 100 : null;
        return { name, cur, prev, diff, pct };
      })
      .sort((a, b) => Math.abs(b.diff) - Math.abs(a.diff))
      .slice(0, 10);
  });

  const yoyYearSeries = $derived.by(() => {
    const byYear = new Map<string, number[]>();
    for (const [m, postings] of Object.entries(monthWiseExpenses)) {
      const yr = m.slice(0, 4);
      const moIdx = Math.max(0, Math.min(11, Number(m.slice(5, 7)) - 1));
      if (!byYear.has(yr)) byYear.set(yr, Array(12).fill(0));
      const total = postings.reduce((s, p) => s + Math.max(0, p.amount || 0), 0);
      byYear.get(yr)![moIdx] += total;
    }
    return Array.from(byYear.entries())
      .sort((a, b) => b[0].localeCompare(a[0]))
      .slice(0, 4)
      .map(([year, months], idx) => ({
        year,
        months,
        total: months.reduce((a, b) => a + b, 0),
        color: categoryPalette[idx % categoryPalette.length]
      }));
  });

  const expenseTrendPoints = $derived<MonthlyBarPoint[]>(
    (expenseTrendGranularity === "yearly" ? yearlyFlows : windowedFlows).map((c) => ({
      date: c.date,
      income: 0,
      expenses: c.expenses,
      investment: 0,
      balance: 0
    }))
  );

  const avgIncome = $derived(
    activeFlowRows.length > 0
      ? activeFlowRows.reduce((s, c) => s + c.income, 0) / activeFlowRows.length
      : 0
  );
  const avgExpenses = $derived(
    activeFlowRows.length > 0
      ? activeFlowRows.reduce((s, c) => s + c.expenses, 0) / activeFlowRows.length
      : 0
  );
  const avgSavingsRate = $derived(
    avgIncome > 0 ? ((avgIncome - avgExpenses) / avgIncome) * 100 : 0
  );
  const totalIncome = $derived(activeFlowRows.reduce((s, c) => s + c.income, 0));
  const totalExpenses = $derived(activeFlowRows.reduce((s, c) => s + c.expenses, 0));
  const totalInvestments = $derived(activeFlowRows.reduce((s, c) => s + c.investment, 0));
  const totalNetSavings = $derived(totalIncome - totalExpenses);

  function aggregateByLevel(postings: Posting[], negate: boolean, level: 1 | 2 | "all") {
    const map = new Map<
      string,
      { account: string; commodity: string; quantity: number; amount: number }
    >();
    for (const p of postings) {
      const parts = p.account.split(":");
      const account =
        level === "all"
          ? p.account
          : level === 1
            ? parts.slice(0, 2).join(":")
            : parts.slice(0, 3).join(":");
      const commodity = p.commodity || runtime.defaultCurrency;
      const mapKey = `${account}::${commodity}`;
      const rawQty = p.quantity ?? p.amount ?? 0;
      const qtyVal = negate ? -rawQty : rawQty;
      const amtVal = negate ? -(p.amount || 0) : p.amount || 0;
      const cur = map.get(mapKey) || { account, commodity, quantity: 0, amount: 0 };
      cur.quantity += qtyVal;
      cur.amount += amtVal;
      map.set(mapKey, cur);
    }
    return Array.from(map.values())
      .filter((r) => Math.abs(r.quantity) > 0.005 || Math.abs(r.amount) > 0.01)
      .sort((a, b) => Math.abs(b.amount) - Math.abs(a.amount));
  }

  function sumByCommodity(rows: Array<{ commodity: string; quantity: number }>) {
    const map = new Map<string, number>();
    for (const r of rows) {
      map.set(r.commodity, (map.get(r.commodity) || 0) + r.quantity);
    }
    return Array.from(map.entries()).map(([commodity, amount]) => ({ commodity, amount }));
  }

  const monthlyBreakdown = $derived({
    incomes: aggregateByLevel(monthWiseIncomes[selectedMonth] || [], true, breakdownLevel),
    expenses: aggregateByLevel(monthWiseExpenses[selectedMonth] || [], false, breakdownLevel),
    investments: aggregateByLevel(monthWiseInvestments[selectedMonth] || [], false, breakdownLevel),
    taxes: aggregateByLevel(monthWiseTaxes[selectedMonth] || [], false, breakdownLevel),
    liabilities: aggregateByLevel(monthWiseLiabilities[selectedMonth] || [], true, breakdownLevel)
  });

  const monthlyNetByCurrency = $derived.by(() => {
    const map = new Map<string, number>();
    for (const r of monthlyBreakdown.incomes) {
      map.set(r.commodity, (map.get(r.commodity) || 0) + r.quantity);
    }
    for (const r of monthlyBreakdown.expenses) {
      map.set(r.commodity, (map.get(r.commodity) || 0) - r.quantity);
    }
    for (const r of monthlyBreakdown.taxes) {
      map.set(r.commodity, (map.get(r.commodity) || 0) - r.quantity);
    }
    return Array.from(map.entries())
      .map(([commodity, amount]) => ({ commodity, amount }))
      .sort((a, b) => a.commodity.localeCompare(b.commodity));
  });

  const incomeSummary = $derived.by(() => {
    const gross = incomeTimeline.reduce(
      (sum, row) => sum + (row.postings || []).reduce((s, p) => s - (p.amount || 0), 0),
      0
    );
    const tax = taxTimeline.reduce(
      (sum, row) => sum + (row.postings || []).reduce((s, p) => s + (p.amount || 0), 0),
      0
    );
    return {
      gross: gross || yearlyIncomeCards.reduce((s, c) => s + (c.gross_income || 0), 0),
      tax: tax || yearlyIncomeCards.reduce((s, c) => s + (c.net_tax || 0), 0)
    };
  });

  const monthlyIncomeBarPoints = $derived<MonthlyBarPoint[]>(
    incomeTimeline.map((row) => {
      const inc = (row.postings || []).reduce((s, p) => s - (p.amount || 0), 0);
      const taxRow = taxTimeline.find((t) => t.date === row.date);
      const tax = taxRow ? (taxRow.postings || []).reduce((s, p) => s + (p.amount || 0), 0) : 0;
      return {
        date: row.date,
        income: Math.max(0, inc),
        expenses: Math.max(0, tax),
        investment: 0,
        balance: 0
      };
    })
  );

  const statementYears = $derived(Object.keys(yearlyStatements).sort().reverse());
  const currentStatement = $derived(selectedYear ? yearlyStatements[selectedYear] : undefined);

  type StatementGroupKey =
    | "income"
    | "tax"
    | "interest"
    | "pnl"
    | "equity"
    | "liabilities"
    | "expenses";
  const statementGroupDefs: Array<{ key: StatementGroupKey; label: string; multiplier: number }> = [
    { key: "income", label: "Income", multiplier: -1 },
    { key: "tax", label: "Tax", multiplier: -1 },
    { key: "interest", label: "Interest", multiplier: -1 },
    { key: "pnl", label: "Gain / Loss", multiplier: 1 },
    { key: "equity", label: "Equity", multiplier: -1 },
    { key: "liabilities", label: "Liabilities", multiplier: -1 },
    { key: "expenses", label: "Expenses", multiplier: -1 }
  ];

  const statementGroups = $derived.by(() => {
    const stmts = Object.values(yearlyStatements);
    if (stmts.length === 0) return [];
    return statementGroupDefs
      .map((def) => {
        const set = new Set<string>();
        for (const s of stmts) {
          const rec = s[def.key] || {};
          for (const acct of Object.keys(rec)) set.add(acct);
        }
        return { ...def, accounts: Array.from(set).sort() };
      })
      .filter((g) => g.accounts.length > 0);
  });

  const statementWaterfall = $derived.by(() => {
    if (!currentStatement) return [];
    const sumRec = (rec?: Record<string, number>) =>
      Object.values(rec || {}).reduce((a, b) => a + (b || 0), 0);
    const steps: Array<{
      label: string;
      start: number;
      end: number;
      delta: number;
      kind: "total" | "pos" | "neg";
    }> = [];
    let running = currentStatement.startingBalance || 0;
    steps.push({ label: "Start", start: 0, end: running, delta: running, kind: "total" });
    for (const def of statementGroupDefs) {
      // Liabilities are a balance-sheet account already included in startingBalance
      // and endingBalance, so they do not drive net worth change.
      if (def.key === "liabilities") continue;
      const raw = sumRec(currentStatement[def.key]);
      const delta = raw * def.multiplier;
      if (Math.abs(delta) < 0.01) continue;
      const next = running + delta;
      steps.push({
        label: def.label,
        start: running,
        end: next,
        delta,
        kind: delta >= 0 ? "pos" : "neg"
      });
      running = next;
    }
    const endBal = currentStatement.endingBalance || running;
    steps.push({ label: "End", start: 0, end: endBal, delta: endBal, kind: "total" });
    return steps;
  });

  const availableMonths = $derived(Object.keys(monthWiseExpenses).sort().reverse());
  const selectedMonthPostings = $derived(monthWiseExpenses[selectedMonth] || []);
  const selectedMonthTotal = $derived(
    selectedMonthPostings.reduce((sum, p) => sum + (p.amount || 0), 0)
  );

  const groupedExpenseCategories = $derived.by(() => {
    const map = new Map<string, number>();
    for (const p of selectedMonthPostings) {
      const parts = p.account.split(":");
      const key = groupDepth === 1 ? parts.slice(0, 2).join(":") : parts.slice(0, 3).join(":");
      map.set(key, (map.get(key) || 0) + (p.amount || 0));
    }
    return Array.from(map.entries())
      .map(([name, amount]) => ({ name, amount }))
      .sort((a, b) => b.amount - a.amount);
  });

  const dailyExpenseMap = $derived.by(() => {
    const map = new Map<number, number>();
    for (const p of selectedMonthPostings) {
      const day = dayjs(p.date).date();
      map.set(day, (map.get(day) || 0) + (p.amount || 0));
    }
    return map;
  });

  const daysInSelectedMonth = $derived(dayjs(`${selectedMonth}-01`).daysInMonth());
  const maxDailySpend = $derived(Math.max(1, ...Array.from(dailyExpenseMap.values())));

  const momComparison = $derived.by(() => {
    const prevMonth = dayjs(`${selectedMonth}-01`).subtract(1, "month").format("YYYY-MM");
    const yoyMonth = dayjs(`${selectedMonth}-01`).subtract(1, "year").format("YYYY-MM");
    const sumMonth = (m: string) =>
      (monthWiseExpenses[m] || []).reduce((s, p) => s + (p.amount || 0), 0);
    const prevTotal = sumMonth(prevMonth);
    const yoyTotal = sumMonth(yoyMonth);
    return {
      prevMonth,
      prevTotal,
      momDeltaPct: prevTotal > 0 ? ((selectedMonthTotal - prevTotal) / prevTotal) * 100 : null,
      yoyMonth,
      yoyTotal,
      yoyDeltaPct: yoyTotal > 0 ? ((selectedMonthTotal - yoyTotal) / yoyTotal) * 100 : null
    };
  });

  const currentBudgetAccounts = $derived(budgetsByMonth[selectedMonth]?.accounts || []);
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Cash Flow &amp; Spending</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Cash flow, income statements, money flow, expenses, budgets, and recurring commitments
      </p>
    </div>

    {#if activeTab === "cashflow"}
      <div class="flex flex-wrap items-center gap-2">
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {cfGranularity === 'monthly'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (cfGranularity = "monthly")}
          >
            Monthly
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {cfGranularity === 'yearly'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (cfGranularity = "yearly")}
          >
            Yearly
          </button>
        </div>

        {#if cfGranularity === "monthly"}
          <div
            class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
          >
            {#each [6, 12, 24, 0] as w}
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {monthsWindow === w
                  ? 'bg-[var(--surface-elevated)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (monthsWindow = w)}
              >
                {w === 0 ? "All" : `${w}M`}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {:else if activeTab === "statement" && statementYears.length > 0}
      <select
        bind:value={selectedYear}
        aria-label="Select financial year"
        class="px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] outline-none"
      >
        {#each statementYears as y}
          <option value={y}>FY {y}</option>
        {/each}
      </select>
    {:else if activeTab === "sankey"}
      <div
        class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
      >
        {#each ["month", "quarter", "year"] as p}
          <button
            type="button"
            class="px-2.5 py-1 rounded-md capitalize cursor-pointer {sankeyPeriod === p
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => {
              sankeyPeriod = p as SankeyPeriod;
              setSankeyRangeForPeriod(sankeyPeriod);
            }}
          >
            {p}
          </button>
        {/each}
      </div>
    {:else if activeTab === "expenses" || activeTab === "budget"}
      <select
        bind:value={selectedMonth}
        aria-label="Select month"
        class="px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] outline-none"
      >
        {#each availableMonths as m}
          <option value={m}>{dayjs(`${m}-01`).format("MMM YYYY")}</option>
        {/each}
      </select>
    {/if}
  </div>

  <!-- Underline Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [{ id: "cashflow", label: "Cash Flow" }, { id: "statement", label: "Income Statement" }, { id: "sankey", label: "Money Flow" }, { id: "income", label: "Income Timeline" }, { id: "expenses", label: "Expenses" }, { id: "budget", label: "Budget" }, { id: "recurring", label: "Recurring" }] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 transition-colors whitespace-nowrap cursor-pointer {activeTab ===
        tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)] hover:text-[var(--text)]'}"
        onclick={() => (activeTab = tab.id as StudioTab)}
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
      <p class="font-medium text-[var(--loss)]">Failed to load cash flow data</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "cashflow"}
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg {cfGranularity === "yearly" ? "Yearly" : "Monthly"} Income
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
          {formatMoney(avgIncome)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg {cfGranularity === "yearly" ? "Yearly" : "Monthly"} Expenses
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-1">
          {formatMoney(avgExpenses)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg Savings Rate
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">{avgSavingsRate.toFixed(1)}%</div>
      </div>
    </div>

    <!-- Period Cash Flow Summary -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Total Income
        </div>
        <div class="text-lg font-semibold tabular-nums text-[var(--gain)] mt-1">
          {formatMoney(totalIncome)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Total Expenses
        </div>
        <div class="text-lg font-semibold tabular-nums text-[var(--loss)] mt-1">
          {formatMoney(totalExpenses)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Total Invested
        </div>
        <div class="text-lg font-semibold tabular-nums mt-1">
          {formatMoney(totalInvestments)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Net Cash Flow
        </div>
        <div
          class="text-lg font-semibold tabular-nums mt-1 {totalNetSavings >= 0
            ? 'text-[var(--gain)]'
            : 'text-[var(--loss)]'}"
        >
          {formatMoney(totalNetSavings)}
        </div>
      </div>
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--muted)]">
        <span class="font-semibold uppercase tracking-wider">Cash Flow Trend</span>
        <div class="flex flex-wrap items-center gap-4">
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--gain)] inline-block"></span>
            Income
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--loss)] inline-block"></span>
            Expenses
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--accent)] inline-block"></span>
            Investments
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--warning)] inline-block"></span>
            Liabilities
          </span>
        </div>
      </div>
      <BarTrendChart data={chartPoints} granularity={cfGranularity} />
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">Period</th>
              <th class="py-2.5 px-4 text-right font-medium">Income</th>
              <th class="py-2.5 px-4 text-right font-medium">Expenses</th>
              <th class="py-2.5 px-4 text-right font-medium">Investments</th>
              <th class="py-2.5 px-4 text-right font-medium">Net</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each [...activeFlowRows].reverse() as row}
              {@const net = row.income - row.expenses - row.tax}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2.5 px-4 font-medium">
                  {cfGranularity === "yearly"
                    ? dayjs(row.date).format("YYYY")
                    : dayjs(row.date).format("MMM YYYY")}
                </td>
                <td class="py-2.5 px-4 text-right text-[var(--gain)]">{formatMoney(row.income)}</td>
                <td class="py-2.5 px-4 text-right text-[var(--loss)]">
                  {formatMoney(row.expenses)}
                </td>
                <td class="py-2.5 px-4 text-right">{formatMoney(row.investment)}</td>
                <td
                  class="py-2.5 px-4 text-right font-semibold {net >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {formatMoney(net)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Monthly Breakdown by Category -->
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold">
            Monthly Breakdown ({selectedMonth
              ? dayjs(`${selectedMonth}-01`).format("MMMM YYYY")
              : "—"})
          </h3>
          <p class="text-xs text-[var(--muted)]">
            Income, Expenses, Investments, Taxes, and Liabilities for the selected month
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2 text-xs">
          {#if availableMonths.length > 0}
            <select
              bind:value={selectedMonth}
              class="px-2.5 py-1 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            >
              {#each availableMonths as m}
                <option value={m}>{dayjs(`${m}-01`).format("MMM YYYY")}</option>
              {/each}
            </select>
          {/if}
          <div
            class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5"
          >
            {#each [{ v: 1, l: "Level 1" }, { v: 2, l: "Level 2" }, { v: "all", l: "All" }] as opt}
              <button
                type="button"
                onclick={() => (breakdownLevel = opt.v as 1 | 2 | "all")}
                class="px-2.5 py-1 rounded-md cursor-pointer {breakdownLevel === opt.v
                  ? 'bg-[var(--surface)] font-semibold'
                  : 'text-[var(--muted)]'}"
              >
                {opt.l}
              </button>
            {/each}
          </div>
        </div>
      </div>

      {#if monthlyNetByCurrency.length > 0}
        <div
          class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/60 px-4 py-3 flex flex-wrap items-center justify-between gap-3"
        >
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Monthly Net Flow (Income − Expenses − Taxes)
          </span>
          <div class="flex flex-wrap items-center gap-4 text-sm font-semibold tabular-nums">
            {#each monthlyNetByCurrency as n}
              <span class={n.amount >= 0 ? "text-[var(--gain)]" : "text-[var(--loss)]"}>
                {formatMoney(n.amount, n.commodity)}
              </span>
            {/each}
          </div>
        </div>
      {/if}

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each [{ title: "Income", rows: monthlyBreakdown.incomes, color: "text-[var(--gain)]" }, { title: "Expenses", rows: monthlyBreakdown.expenses, color: "text-[var(--loss)]" }, { title: "Investments", rows: monthlyBreakdown.investments, color: "text-[var(--accent)]" }, { title: "Taxes", rows: monthlyBreakdown.taxes, color: "text-[var(--warning)]" }, { title: "Liabilities", rows: monthlyBreakdown.liabilities, color: "text-[var(--muted)]" }] as sec}
          {#if sec.rows.length > 0}
            {@const secTotals = sumByCommodity(sec.rows)}
            <div
              class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/40 p-4 space-y-2.5"
            >
              <div
                class="flex items-center justify-between gap-2 border-b border-[var(--border)] pb-2"
              >
                <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                  {sec.title}
                </span>
                <div class="text-right text-sm font-semibold tabular-nums {sec.color}">
                  {#each secTotals as st}
                    <div>{formatMoney(st.amount, st.commodity)}</div>
                  {/each}
                </div>
              </div>
              <div class="divide-y divide-[var(--border)]/60 text-xs tabular-nums">
                {#each sec.rows as r}
                  <div class="py-1.5 flex justify-between gap-2">
                    <span class="truncate">{r.account}</span>
                    <span class="font-medium shrink-0">
                      {formatMoney(r.quantity, r.commodity)}
                    </span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        {/each}
      </div>
    </div>
  {:else if activeTab === "statement"}
    {#if !currentStatement}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
      >
        No income statement data for this financial year.
      </div>
    {:else}
      {@const diff =
        (currentStatement.endingBalance || 0) - (currentStatement.startingBalance || 0)}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Starting Net Worth
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(currentStatement.startingBalance || 0)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Ending Net Worth
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(currentStatement.endingBalance || 0)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Net Change
          </div>
          <div
            class="text-2xl font-semibold tabular-nums mt-1 {diff >= 0
              ? 'text-[var(--gain)]'
              : 'text-[var(--loss)]'}"
          >
            {formatMoney(diff)}
          </div>
        </div>
      </div>

      <!-- Waterfall Bridge Graph (Horizontal, delta steps only with top scale) -->
      {@const bridgeSteps = statementWaterfall.filter((s) => s.kind !== "total")}
      {#if bridgeSteps.length > 0}
        {@const rawMin = Math.min(...bridgeSteps.flatMap((s) => [s.start, s.end]))}
        {@const rawMax = Math.max(...bridgeSteps.flatMap((s) => [s.start, s.end]))}
        {@const rawSpan = Math.max(1, rawMax - rawMin)}
        {@const wfMin = rawMin - rawSpan * 0.04}
        {@const wfMax = rawMax + rawSpan * 0.04}
        {@const wfSpan = Math.max(1, wfMax - wfMin)}
        {@const wfW = 1040}
        {@const wfH = 228}
        {@const padT = 28}
        {@const padB = 14}
        {@const padL = 104}
        {@const padR = 116}
        {@const rowH = (wfH - padT - padB) / bridgeSteps.length}
        {@const barH = Math.min(20, rowH * 0.62)}
        {@const plotW = wfW - padL - padR}
        {@const xAt = (v: number) => padL + ((v - wfMin) / wfSpan) * plotW}
        {@const wfTicks = [0, 0.25, 0.5, 0.75, 1].map((t) => wfMin + t * wfSpan)}
        {@const formatWfTick = (val: number) =>
          Math.abs(val) >= 10000000
            ? `${(val / 10000000).toFixed(2)}Cr`
            : Math.abs(val) >= 100000
              ? `${(val / 100000).toFixed(1)}L`
              : Math.abs(val) >= 1000
                ? `${(val / 1000).toFixed(0)}K`
                : `${Math.round(val)}`}
        {@const startStep = statementWaterfall[0]}
        {@const endStep = statementWaterfall[statementWaterfall.length - 1]}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
            <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">
              Net Worth Bridge ({selectedYear})
            </span>
            {#if startStep && endStep}
              <span class="text-[var(--muted)] tabular-nums">
                Start {formatMoney(startStep.end)} → End {formatMoney(endStep.end)}
              </span>
            {/if}
          </div>
          <div class="overflow-x-auto">
            <svg viewBox="0 0 {wfW} {wfH}" class="w-full min-w-[600px] h-56 overflow-visible">
              <!-- Top X-Axis Scale & Vertical Gridlines -->
              <line x1={padL} x2={wfW - padR} y1={padT} y2={padT} stroke="var(--border)" />
              {#each wfTicks as tick}
                {@const tx = xAt(tick)}
                <line
                  x1={tx}
                  x2={tx}
                  y1={padT}
                  y2={wfH - padB}
                  stroke="var(--border)"
                  stroke-dasharray="3 3"
                  opacity="0.6"
                />
                <text
                  x={tx}
                  y={padT - 8}
                  text-anchor="middle"
                  font-size="10"
                  fill="var(--muted)"
                  class="tabular-nums"
                >
                  {formatWfTick(tick)}
                </text>
              {/each}

              <!-- Start & End Reference Markers -->
              {#if startStep}
                {@const sx = xAt(startStep.end)}
                <line
                  x1={sx}
                  x2={sx}
                  y1={padT}
                  y2={wfH - padB}
                  stroke="var(--accent)"
                  stroke-dasharray="2 2"
                  opacity="0.6"
                />
                <polygon
                  points="{sx - 4},{padT - 3} {sx + 4},{padT - 3} {sx},{padT + 3}"
                  fill="var(--accent)"
                />
              {/if}
              {#if endStep}
                {@const ex = xAt(endStep.end)}
                <line
                  x1={ex}
                  x2={ex}
                  y1={padT}
                  y2={wfH - padB}
                  stroke="var(--accent)"
                  stroke-dasharray="2 2"
                  opacity="0.6"
                />
                <polygon
                  points="{ex - 4},{wfH - padB + 3} {ex + 4},{wfH - padB + 3} {ex},{wfH - padB - 3}"
                  fill="var(--accent)"
                />
              {/if}

              {#each bridgeSteps as step, idx}
                {@const xStart = xAt(step.start)}
                {@const xEnd = xAt(step.end)}
                {@const x1 = Math.min(xStart, xEnd)}
                {@const x2 = Math.max(xStart, xEnd)}
                {@const y = padT + idx * rowH + (rowH - barH) / 2}
                {@const fill = step.kind === "pos" ? "var(--gain)" : "var(--loss)"}
                {#if idx < bridgeSteps.length - 1}
                  {@const nextY = padT + (idx + 1) * rowH + (rowH - barH) / 2}
                  <line
                    x1={xEnd}
                    x2={xEnd}
                    y1={y + barH}
                    y2={nextY}
                    stroke="var(--border)"
                    stroke-dasharray="3 3"
                  />
                {/if}
                <text
                  x={padL - 12}
                  y={y + barH / 2 + 4}
                  text-anchor="end"
                  font-size="11.5"
                  fill="var(--muted)"
                >
                  {step.label}
                </text>
                <rect
                  x={x1}
                  {y}
                  width={Math.max(4, x2 - x1)}
                  height={barH}
                  {fill}
                  rx="4"
                  opacity="0.9"
                />
                <text
                  x={Math.max(x1 + 4, x2) + 8}
                  y={y + barH / 2 + 4}
                  text-anchor="start"
                  font-size="11"
                  fill="var(--muted)"
                  class="tabular-nums"
                >
                  {`${step.delta >= 0 ? "+" : ""}${formatMoney(step.delta)}`}
                </text>
              {/each}
            </svg>
          </div>
        </div>
      {/if}

      <!-- Detailed Account / Yearly Amounts Breakdown Table -->
      {#if statementGroups.length > 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div
            class="px-5 py-3 border-b border-[var(--border)] text-xs font-semibold uppercase tracking-wider text-[var(--muted)]"
          >
            Detailed Account Breakdown by Year
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
              >
                <tr>
                  <th class="py-2.5 px-4 font-medium">Account</th>
                  {#each statementYears as yr}
                    <th class="py-2.5 px-4 text-right font-medium">{yr}</th>
                  {/each}
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                <tr class="bg-[var(--surface-elevated)]/30 font-semibold">
                  <td class="py-2 px-4">Starting Balance</td>
                  {#each statementYears as yr}
                    <td class="py-2 px-4 text-right">
                      {formatMoney(yearlyStatements[yr]?.startingBalance || 0)}
                    </td>
                  {/each}
                </tr>
                {#each statementGroups as grp}
                  <tr
                    class="bg-[var(--surface-elevated)]/60 font-semibold text-xs uppercase tracking-wider text-[var(--muted)]"
                  >
                    <td class="py-2 px-4">{grp.label}</td>
                    {#each statementYears as yr}
                      {@const grpSum =
                        Object.values(yearlyStatements[yr]?.[grp.key] || {}).reduce(
                          (a, b) => a + (b || 0),
                          0
                        ) * grp.multiplier}
                      <td
                        class="py-2 px-4 text-right {grpSum > 0
                          ? 'text-[var(--gain)]'
                          : grpSum < 0
                            ? 'text-[var(--loss)]'
                            : ''}"
                      >
                        {Math.abs(grpSum) > 0.01 ? formatMoney(grpSum) : "—"}
                      </td>
                    {/each}
                  </tr>
                  {#each grp.accounts as acct}
                    <tr class="hover:bg-[var(--surface-elevated)]/40">
                      <td class="py-1.5 px-4 pl-7 text-xs">{acct}</td>
                      {#each statementYears as yr}
                        {@const val =
                          (yearlyStatements[yr]?.[grp.key]?.[acct] || 0) * grp.multiplier}
                        <td class="py-1.5 px-4 text-right text-xs text-[var(--muted)]">
                          {Math.abs(val) > 0.01 ? formatMoney(val) : ""}
                        </td>
                      {/each}
                    </tr>
                  {/each}
                {/each}
                <tr class="bg-[var(--surface-elevated)]/40 font-semibold">
                  <td class="py-2 px-4">Ending Balance</td>
                  {#each statementYears as yr}
                    <td class="py-2 px-4 text-right">
                      {formatMoney(yearlyStatements[yr]?.endingBalance || 0)}
                    </td>
                  {/each}
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    {/if}
  {:else if activeTab === "sankey"}
    <div class="space-y-5">
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex flex-wrap items-center justify-between gap-3 text-xs"
      >
        <div class="flex items-center gap-3">
          <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">Flow Filter</span
          >
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input type="checkbox" bind:checked={sankeyExpenseOnly} />
            Expenses only
          </label>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <input
            type="date"
            bind:value={sankeyFrom}
            oninput={() => (sankeyCustomRange = true)}
            class="px-2.5 py-1 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
          />
          <span class="text-[var(--muted)]">to</span>
          <input
            type="date"
            bind:value={sankeyTo}
            oninput={() => (sankeyCustomRange = true)}
            class="px-2.5 py-1 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
          />
          {#if sankeyCustomRange}
            <button
              type="button"
              onclick={() => setSankeyRangeForPeriod(sankeyPeriod)}
              class="px-2.5 py-1 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
            >
              Reset
            </button>
          {/if}
        </div>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5">
        {#if sankeyLoading}
          <div class="h-48 animate-pulse"></div>
        {:else}
          <SankeyChart links={sankeyLinks} />
        {/if}
      </div>

      {#if sankeyLinks.length > 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead
                class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
              >
                <tr>
                  <th class="py-2.5 px-4 font-medium">From</th>
                  <th class="py-2.5 px-4 font-medium">To</th>
                  <th class="py-2.5 px-4 text-right font-medium">Flow</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[var(--border)] tabular-nums">
                {#each sankeyLinks.slice(0, 40) as link}
                  <tr class="hover:bg-[var(--surface-elevated)]/50">
                    <td class="py-2 px-4 text-[var(--muted)]">{link.source}</td>
                    <td class="py-2 px-4 font-medium">{link.target}</td>
                    <td class="py-2 px-4 text-right font-semibold">{formatMoney(link.value)}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    </div>
  {:else if activeTab === "income"}
    <div class="space-y-5">
      <!-- Income Timeline Summary KPIs -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Gross Income
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
            {formatMoney(incomeSummary.gross)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Net Tax
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-1">
            {formatMoney(incomeSummary.tax)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            TTM Investment Income
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--accent)] mt-1">
            {formatMoney(ttmInvestmentIncome)}
          </div>
        </div>
      </div>

      <!-- Chart 1: Monthly Income Timeline -->
      {#if monthlyIncomeBarPoints.length > 0}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
          <div
            class="flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--muted)]"
          >
            <span class="font-semibold uppercase tracking-wider">Monthly Income Timeline</span>
            <div class="flex items-center gap-4">
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2.5 h-2.5 rounded-xs bg-[var(--gain)] inline-block"></span>
                Gross Income
              </span>
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2.5 h-2.5 rounded-xs bg-[var(--loss)] inline-block"></span>
                Tax
              </span>
            </div>
          </div>
          <BarTrendChart data={monthlyIncomeBarPoints} />
        </div>
      {/if}

      <!-- Charts 2, 3, 4: Financial Year Income Timelines -->
      {#if yearlyIncomeCards.length > 0}
        {@const maxGross = Math.max(
          1,
          ...yearlyIncomeCards.map((c) => Math.max(c.gross_income || 0, c.net_tax || 0))
        )}
        {@const maxNet = Math.max(1, ...yearlyIncomeCards.map((c) => c.net_income || 0))}
        {@const maxTax = Math.max(1, ...yearlyIncomeCards.map((c) => c.net_tax || 0))}
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <!-- Chart 2: Yearly Gross vs Tax -->
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Yearly Gross Income vs Tax
            </div>
            <div class="h-44 flex items-end gap-2 pt-4">
              {#each yearlyIncomeCards as yc}
                {@const gH = Math.max(2, ((yc.gross_income || 0) / maxGross) * 130)}
                {@const tH = Math.max(0, ((yc.net_tax || 0) / maxGross) * 130)}
                <div class="flex-1 flex flex-col items-center gap-1 min-w-0">
                  <div class="w-full flex items-end justify-center gap-1 h-[130px]">
                    <div
                      class="w-3.5 rounded-t bg-[var(--gain)]/85"
                      style="height: {gH}px"
                      title="Gross: {formatMoney(yc.gross_income || 0)}"
                    ></div>
                    <div
                      class="w-3.5 rounded-t bg-[var(--loss)]/85"
                      style="height: {tH}px"
                      title="Tax: {formatMoney(yc.net_tax || 0)}"
                    ></div>
                  </div>
                  <span class="text-[10px] text-[var(--muted)] truncate"
                    >{formatIncomeCardYear(yc)}</span
                  >
                </div>
              {/each}
            </div>
          </div>

          <!-- Chart 3: Yearly Net Income -->
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Yearly Net Income
            </div>
            <div class="h-44 flex items-end gap-2 pt-4">
              {#each yearlyIncomeCards as yc}
                {@const nH = Math.max(2, ((yc.net_income || 0) / maxNet) * 130)}
                <div class="flex-1 flex flex-col items-center gap-1 min-w-0">
                  <div class="w-full flex items-end justify-center h-[130px]">
                    <div
                      class="w-6 rounded-t bg-[var(--gain)]/85"
                      style="height: {nH}px"
                      title="Net Income: {formatMoney(yc.net_income || 0)}"
                    ></div>
                  </div>
                  <span class="text-[10px] text-[var(--muted)] truncate"
                    >{formatIncomeCardYear(yc)}</span
                  >
                </div>
              {/each}
            </div>
          </div>

          <!-- Chart 4: Yearly Net Tax -->
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Yearly Net Tax
            </div>
            <div class="h-44 flex items-end gap-2 pt-4">
              {#each yearlyIncomeCards as yc}
                {@const txH = Math.max(2, ((yc.net_tax || 0) / maxTax) * 130)}
                <div class="flex-1 flex flex-col items-center gap-1 min-w-0">
                  <div class="w-full flex items-end justify-center h-[130px]">
                    <div
                      class="w-6 rounded-t bg-[var(--loss)]/85"
                      style="height: {txH}px"
                      title="Net Tax: {formatMoney(yc.net_tax || 0)}"
                    ></div>
                  </div>
                  <span class="text-[10px] text-[var(--muted)] truncate"
                    >{formatIncomeCardYear(yc)}</span
                  >
                </div>
              {/each}
            </div>
          </div>
        </div>
      {/if}

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
        {#if yearlyIncomeCards.length === 0}
          <div class="p-8 text-center text-sm text-[var(--muted)]">
            No yearly income timeline data available.
          </div>
        {:else}
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">Year</th>
                <th class="py-2.5 px-4 text-right font-medium">Gross Income</th>
                <th class="py-2.5 px-4 text-right font-medium">Tax</th>
                <th class="py-2.5 px-4 text-right font-medium">Net Income</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each yearlyIncomeCards as yc}
                {@const fyLabel = formatIncomeCardYear(yc)}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2.5 px-4 font-medium">{fyLabel}</td>
                  <td class="py-2.5 px-4 text-right">{formatMoney(yc.gross_income || 0)}</td>
                  <td class="py-2.5 px-4 text-right text-[var(--loss)]">
                    {formatMoney(yc.net_tax || 0)}
                  </td>
                  <td class="py-2.5 px-4 text-right font-semibold text-[var(--gain)]">
                    {formatMoney(yc.net_income || 0)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    </div>
  {:else if activeTab === "expenses"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Spending by Category ({expenseTrendGranularity === "yearly" ? "Yearly" : "Monthly"} Stacked)
        </span>
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {expenseTrendGranularity === 'monthly'
              ? 'bg-[var(--surface)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (expenseTrendGranularity = "monthly")}
          >
            Monthly
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {expenseTrendGranularity === 'yearly'
              ? 'bg-[var(--surface)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (expenseTrendGranularity = "yearly")}
          >
            Yearly
          </button>
        </div>
      </div>
      <StackedBarChart columns={expenseStackedColumns} />
    </div>

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Total Spent ({dayjs(`${selectedMonth}-01`).format("MMMM YYYY")})
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-0.5">
            {formatMoney(selectedMonthTotal)}
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <div
            class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
          >
            {#each [{ id: "breakdown", label: "Breakdown" }, { id: "heatmap", label: "Heatmap" }, { id: "trends", label: "MoM / YoY" }] as v}
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {expenseView === v.id
                  ? 'bg-[var(--surface)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (expenseView = v.id as ExpenseView)}
              >
                {v.label}
              </button>
            {/each}
          </div>

          {#if expenseView === "breakdown"}
            <div
              class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
            >
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {groupDepth === 1
                  ? 'bg-[var(--surface)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (groupDepth = 1)}
              >
                Top Level
              </button>
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {groupDepth === 2
                  ? 'bg-[var(--surface)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (groupDepth = 2)}
              >
                Detailed
              </button>
            </div>
          {/if}
        </div>
      </div>

      {#if expenseView === "breakdown"}
        {#if groupedExpenseCategories.length === 0}
          <p class="text-sm text-[var(--muted)]">No expenses recorded in this month.</p>
        {:else}
          <div class="space-y-3">
            {#each groupedExpenseCategories as cat}
              {@const pct =
                selectedMonthTotal > 0 ? Math.min(100, (cat.amount / selectedMonthTotal) * 100) : 0}
              <div class="space-y-1">
                <div class="flex justify-between text-xs">
                  <span class="font-medium">{cat.name.replace(/^Expenses:/, "")}</span>
                  <span class="tabular-nums">
                    <strong>{formatMoney(cat.amount)}</strong>
                    <span class="text-[var(--muted)] ml-1">({pct.toFixed(1)}%)</span>
                  </span>
                </div>
                <div class="h-2 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                  <div class="h-full rounded-full bg-[var(--accent)]" style="width: {pct}%"></div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      {:else if expenseView === "heatmap"}
        <div class="grid grid-cols-7 gap-2 pt-1">
          {#each Array.from({ length: daysInSelectedMonth }, (_, i) => i + 1) as day}
            {@const spent = dailyExpenseMap.get(day) || 0}
            {@const intensity = spent > 0 ? Math.max(0.18, spent / maxDailySpend) : 0}
            <div
              class="rounded-lg border border-[var(--border)] p-2 text-xs tabular-nums"
              style={spent > 0
                ? `background-color: color-mix(in srgb, var(--accent) ${Math.round(intensity * 45)}%, var(--surface))`
                : ""}
              title="Day {day}: {formatMoney(spent)}"
            >
              <div class="text-[var(--muted)] text-[10px]">Day {day}</div>
              <div class="font-medium mt-0.5 truncate">{spent > 0 ? formatMoney(spent) : "—"}</div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
          <div class="rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] p-4">
            <div class="text-xs text-[var(--muted)]">
              vs Previous Month ({dayjs(`${momComparison.prevMonth}-01`).format("MMM YYYY")})
            </div>
            <div class="text-lg font-semibold tabular-nums mt-1">
              {formatMoney(momComparison.prevTotal)}
              {#if momComparison.momDeltaPct !== null}
                <span
                  class="text-xs ml-2 {momComparison.momDeltaPct <= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {momComparison.momDeltaPct >= 0 ? "+" : ""}{momComparison.momDeltaPct.toFixed(1)}%
                </span>
              {/if}
            </div>
          </div>

          <div class="rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] p-4">
            <div class="text-xs text-[var(--muted)]">
              vs Same Month Last Year ({dayjs(`${momComparison.yoyMonth}-01`).format("MMM YYYY")})
            </div>
            <div class="text-lg font-semibold tabular-nums mt-1">
              {formatMoney(momComparison.yoyTotal)}
              {#if momComparison.yoyDeltaPct !== null}
                <span
                  class="text-xs ml-2 {momComparison.yoyDeltaPct <= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {momComparison.yoyDeltaPct >= 0 ? "+" : ""}{momComparison.yoyDeltaPct.toFixed(1)}%
                </span>
              {/if}
            </div>
          </div>
        </div>

        <!-- MoM % Delta Chart -->
        {#if momMonthlySeries.length > 0}
          {@const maxAbsPct = Math.max(10, ...momMonthlySeries.map((m) => Math.abs(m.deltaPct)))}
          <div class="pt-3 space-y-2">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Month-over-Month Spending Change (%)
            </div>
            <div class="grid grid-cols-6 sm:grid-cols-12 gap-2 items-end h-28 pt-2">
              {#each momMonthlySeries as m}
                {@const barH = Math.max(4, Math.min(64, (Math.abs(m.deltaPct) / maxAbsPct) * 64))}
                <div
                  class="flex flex-col items-center justify-end gap-1 h-full"
                  title="{m.label}: {m.deltaPct.toFixed(1)}%"
                >
                  <span
                    class="text-[10px] tabular-nums {m.deltaPct <= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {m.deltaPct >= 0 ? "+" : ""}{m.deltaPct.toFixed(0)}%
                  </span>
                  <div
                    class="w-4 rounded-t {m.deltaPct <= 0
                      ? 'bg-[var(--gain)]/85'
                      : 'bg-[var(--loss)]/85'}"
                    style="height: {barH}px"
                  ></div>
                  <span class="text-[10px] text-[var(--muted)] truncate">{m.label.slice(0, 3)}</span
                  >
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- YoY Multi-Year 12-Month Overlay Chart -->
        {#if yoyYearSeries.length > 0}
          {@const yoyMax = Math.max(1, ...yoyYearSeries.flatMap((y) => y.months))}
          <div class="pt-4 border-t border-[var(--border)] space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Year-over-Year Monthly Trajectory (Jan – Dec)
              </span>
              <div class="flex flex-wrap items-center gap-3 text-xs tabular-nums">
                {#each yoyYearSeries as ys}
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-2.5 h-0.5 inline-block" style="background: {ys.color}"></span>
                    <strong>{ys.year}</strong> ({formatMoney(ys.total)})
                  </span>
                {/each}
              </div>
            </div>
            <div class="overflow-x-auto">
              <svg viewBox="0 0 680 180" class="w-full min-w-[480px] h-44 overflow-visible">
                {#each yoyYearSeries as ys}
                  {@const pts = ys.months
                    .map((val, idx) => `${30 + (idx / 11) * 620},${150 - (val / yoyMax) * 130}`)
                    .join(" ")}
                  <polyline fill="none" stroke={ys.color} stroke-width="2" points={pts} />
                  {#each ys.months as val, idx}
                    <circle
                      cx={30 + (idx / 11) * 620}
                      cy={150 - (val / yoyMax) * 130}
                      r="3"
                      fill={ys.color}
                    >
                      <title>{ys.year} {dayjs().month(idx).format("MMM")}: {formatMoney(val)}</title
                      >
                    </circle>
                  {/each}
                {/each}
                {#each Array.from({ length: 12 }, (_, i) => i) as idx}
                  <text
                    x={30 + (idx / 11) * 620}
                    y="170"
                    text-anchor="middle"
                    font-size="10"
                    fill="var(--muted)"
                  >
                    {dayjs().month(idx).format("MMM")}
                  </text>
                {/each}
              </svg>
            </div>
          </div>
        {/if}

        <!-- MoM Category Variance Table -->
        {#if momCategoryVariance.length > 0}
          <div class="pt-4 border-t border-[var(--border)] space-y-2">
            <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
              Top Category Variance vs Previous Month
            </div>
            <div class="overflow-x-auto">
              <table class="w-full text-xs text-left">
                <thead
                  class="uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)]"
                >
                  <tr>
                    <th class="py-2 px-3 font-medium">Category</th>
                    <th class="py-2 px-3 text-right font-medium">Prev Month</th>
                    <th class="py-2 px-3 text-right font-medium">Selected</th>
                    <th class="py-2 px-3 text-right font-medium">Change</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)] tabular-nums">
                  {#each momCategoryVariance as row}
                    <tr>
                      <td class="py-1.5 px-3 font-medium">{row.name}</td>
                      <td class="py-1.5 px-3 text-right text-[var(--muted)]"
                        >{formatMoney(row.prev)}</td
                      >
                      <td class="py-1.5 px-3 text-right font-semibold">{formatMoney(row.cur)}</td>
                      <td
                        class="py-1.5 px-3 text-right {row.diff <= 0
                          ? 'text-[var(--gain)]'
                          : 'text-[var(--loss)]'}"
                      >
                        {row.diff >= 0 ? "+" : ""}{formatMoney(row.diff)}
                        {#if row.pct !== null}
                          <span class="ml-1">({row.pct >= 0 ? "+" : ""}{row.pct.toFixed(0)}%)</span>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/if}
      {/if}

      {#if dailyExpenses.length > 0 || dailyLoading}
        <div class="pt-4 border-t border-[var(--border)] space-y-2.5">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Daily Spending Breakdown ({dailyExpenses.length} entries)
          </div>
          {#if dailyLoading}
            <div class="h-16 animate-pulse rounded-lg bg-[var(--surface-elevated)]"></div>
          {:else}
            <div class="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2 text-xs tabular-nums">
              {#each dailyExpenses.slice(0, 18) as d}
                <div
                  class="p-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
                >
                  <div class="text-[var(--muted)]">{dayjs(d.date).format("DD MMM")}</div>
                  <div class="font-semibold mt-0.5">{formatMoney(Math.abs(d.amount || 0))}</div>
                  {#if d.category}
                    <div class="text-[10px] text-[var(--muted)] truncate">{d.category}</div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {:else if activeTab === "budget"}
    <div class="space-y-4">
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex items-center justify-between"
      >
        <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Available for Budgeting
        </span>
        <span class="text-lg font-semibold tabular-nums">
          {formatMoney(availableForBudgeting)}
        </span>
      </div>

      {#if currentBudgetAccounts.length === 0}
        <div
          class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
        >
          No budget entries configured for {dayjs(`${selectedMonth}-01`).format("MMM YYYY")}.
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each currentBudgetAccounts as b}
            {@const pct = b.forecast > 0 ? Math.min(100, (b.actual / b.forecast) * 100) : 0}
            <div
              class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2.5"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm font-medium truncate">
                  {b.account.replace(/^Expenses:/, "")}
                </span>
                <span
                  class="text-xs font-semibold tabular-nums {b.available >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {b.available >= 0 ? "Left" : "Over"}: {formatMoney(Math.abs(b.available))}
                </span>
              </div>
              <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                <div
                  class="h-full rounded-full {b.available >= 0
                    ? 'bg-[var(--gain)]'
                    : 'bg-[var(--loss)]'}"
                  style="width: {pct}%"
                ></div>
              </div>
              <div class="flex justify-between text-xs text-[var(--muted)] tabular-nums">
                <span>Spent: {formatMoney(b.actual)}</span>
                <span>Budget: {formatMoney(b.forecast)}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Recurring Bills &amp; Income ({recurringItems.length})
        </span>
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {recurringView === 'calendar'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (recurringView = "calendar")}
          >
            Calendar Grid
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {recurringView === 'list'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (recurringView = "list")}
          >
            List
          </button>
        </div>
      </div>

      {#if recurringItems.length === 0}
        <div
          class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
        >
          No recurring transactions configured.
        </div>
      {:else if recurringView === "calendar"}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="grid grid-cols-7 gap-2 text-xs">
            {#each Array.from({ length: 31 }, (_, i) => i + 1) as day}
              {@const itemsForDay = recurringItems.filter((it) => {
                if (!it.next_date) return false;
                return dayjs(it.next_date).date() === day;
              })}
              <div
                class="min-h-16 p-2 rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/40"
              >
                <div class="font-semibold text-[var(--muted)] tabular-nums">{day}</div>
                {#each itemsForDay as it}
                  <div
                    class="mt-1 px-1.5 py-0.5 rounded bg-[var(--accent)]/15 text-[var(--accent)] truncate text-[10px] font-medium"
                  >
                    {it.name || it.payee || it.account || "Bill"}
                  </div>
                {/each}
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="divide-y divide-[var(--border)]">
            {#each recurringItems as item}
              <div class="px-4 py-3 flex items-center justify-between gap-3 text-sm">
                <div class="min-w-0">
                  <div class="font-medium truncate">
                    {item.name || item.payee || item.narration || item.account || "Recurring item"}
                  </div>
                  <div class="text-xs text-[var(--muted)]">
                    {item.frequency || "Recurring"}
                    {item.next_date ? ` · Next: ${item.next_date}` : ""}
                  </div>
                </div>
                {#if item.amount !== undefined}
                  <span class="font-semibold tabular-nums">{formatMoney(item.amount)}</span>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>
