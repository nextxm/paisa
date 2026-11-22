export interface QuickAddPostingValue {
  account: string;
  amount: string;
  commodity: string;
}

export interface QuickAddFormValues {
  date: string;
  payee: string;
  narration: string;
  fromAccount: string;
  toAccount: string;
  amount: string;
  commodity: string;
  toAmount?: string;
  toCommodity?: string;
  exchangeRate?: string;
  postings?: QuickAddPostingValue[];
}

export type SuggestionIndexMap = Record<string, number>;

export interface ParserSubmitContext {
  parserText: string;
  values: QuickAddFormValues;
  selectedSuggestionIndex: SuggestionIndexMap;
  parseStartedAt: number | null;
  nowMs: number;
}

function ensureString(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  return String(value);
}

export function parserFormOverrides(
  parsed: any,
  currentCommodity: string
): Partial<QuickAddFormValues> {
  return {
    payee: ensureString(parsed?.payee),
    fromAccount: ensureString(parsed?.from_account),
    toAccount: ensureString(parsed?.to_account),
    amount: ensureString(parsed?.amount),
    commodity: ensureString(parsed?.currency) || currentCommodity
  };
}

export function applySuggestionSelection(
  field: string,
  account: string,
  index: number,
  values: QuickAddFormValues,
  selectedSuggestionIndex: SuggestionIndexMap
): { values: QuickAddFormValues; selectedSuggestionIndex: SuggestionIndexMap } {
  const nextValues = { ...values };
  if (field === "from_account") {
    nextValues.fromAccount = account;
  }
  if (field === "to_account") {
    nextValues.toAccount = account;
  }

  return {
    values: nextValues,
    selectedSuggestionIndex: { ...selectedSuggestionIndex, [field]: index }
  };
}

export function selectedSuggestionUsed(selectedSuggestionIndex: SuggestionIndexMap): number {
  if (selectedSuggestionIndex.from_account !== undefined) {
    return selectedSuggestionIndex.from_account;
  }
  if (selectedSuggestionIndex.to_account !== undefined) {
    return selectedSuggestionIndex.to_account;
  }
  return -1;
}

export function buildQuickAddSubmitRequest(ctx: ParserSubmitContext): {
  endpoint: string;
  payload: Record<string, unknown>;
} {
  const usingParserFlow = !!ctx.parserText.trim();

  if (!usingParserFlow) {
    const payload: Record<string, unknown> = {
      date: ensureString(ctx.values.date),
      payee: ensureString(ctx.values.payee),
      narration: ensureString(ctx.values.narration),
      from_account: ensureString(ctx.values.fromAccount),
      to_account: ensureString(ctx.values.toAccount),
      amount: ensureString(ctx.values.amount),
      commodity: ensureString(ctx.values.commodity)
    };
    if (ctx.values.toAmount) payload.to_amount = ensureString(ctx.values.toAmount);
    if (ctx.values.toCommodity) payload.to_commodity = ensureString(ctx.values.toCommodity);
    if (ctx.values.exchangeRate) payload.exchange_rate = ensureString(ctx.values.exchangeRate);
    if (ctx.values.postings && ctx.values.postings.length > 0) {
      payload.postings = ctx.values.postings.map((p) => ({
        account: ensureString(p.account),
        amount: ensureString(p.amount),
        commodity: ensureString(p.commodity)
      }));
    }
    return {
      endpoint: "/api/transaction/add",
      payload
    };
  }

  return {
    endpoint: "/api/parser/create-transaction",
    payload: {
      text: ensureString(ctx.parserText),
      date: ensureString(ctx.values.date),
      payee: ensureString(ctx.values.payee),
      narration: ensureString(ctx.values.narration),
      from_account: ensureString(ctx.values.fromAccount),
      to_account: ensureString(ctx.values.toAccount),
      amount: ensureString(ctx.values.amount),
      commodity: ensureString(ctx.values.commodity),
      suggestion_used: selectedSuggestionUsed(ctx.selectedSuggestionIndex),
      time_to_confirm_ms: ctx.parseStartedAt ? ctx.nowMs - ctx.parseStartedAt : 0
    }
  };
}

export interface ParserUiState {
  parserText: string;
  parserResult: unknown | null;
  parserWarnings: string[];
  requiresConfirmation: boolean;
  selectedSuggestionIndex: SuggestionIndexMap;
  parseStartedAt: number | null;
}

export function clearedParserState(): ParserUiState {
  return {
    parserText: "",
    parserResult: null,
    parserWarnings: [],
    requiresConfirmation: false,
    selectedSuggestionIndex: {},
    parseStartedAt: null
  };
}
