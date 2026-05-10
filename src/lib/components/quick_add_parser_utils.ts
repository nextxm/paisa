export interface QuickAddFormValues {
  date: string;
  payee: string;
  narration: string;
  fromAccount: string;
  toAccount: string;
  amount: string;
  commodity: string;
}

export type SuggestionIndexMap = Record<string, number>;

export interface ParserSubmitContext {
  parserText: string;
  values: QuickAddFormValues;
  selectedSuggestionIndex: SuggestionIndexMap;
  parseStartedAt: number | null;
  nowMs: number;
}

export function parserFormOverrides(
  parsed: any,
  currentCommodity: string
): Partial<QuickAddFormValues> {
  return {
    payee: parsed?.payee || "",
    fromAccount: parsed?.from_account || "",
    toAccount: parsed?.to_account || "",
    amount: parsed?.amount || "",
    commodity: parsed?.currency || currentCommodity
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
    return {
      endpoint: "/api/transaction/add",
      payload: {
        date: ctx.values.date,
        payee: ctx.values.payee,
        narration: ctx.values.narration,
        from_account: ctx.values.fromAccount,
        to_account: ctx.values.toAccount,
        amount: ctx.values.amount,
        commodity: ctx.values.commodity
      }
    };
  }

  return {
    endpoint: "/api/parser/create-transaction",
    payload: {
      text: ctx.parserText,
      date: ctx.values.date,
      payee: ctx.values.payee,
      narration: ctx.values.narration,
      from_account: ctx.values.fromAccount,
      to_account: ctx.values.toAccount,
      amount: ctx.values.amount,
      commodity: ctx.values.commodity,
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
