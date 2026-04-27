import dayjs from "dayjs";
import _ from "lodash";
import type { Transaction } from "$lib/utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type FilterType = "account" | "amount" | "date" | "tag";

export type AccountOperator = "=" | "!=" | "contains" | "starts_with";
export type AmountOperator = "=" | "!=" | ">" | "<" | ">=" | "<=";
export type DateOperator = "=" | ">" | "<" | ">=" | "<=";
export type TagOperator = "=" | "!=" | "is_set" | "is_not_set";

export type FilterOperator = AccountOperator | AmountOperator | DateOperator | TagOperator;

export type LogicOperator = "AND" | "OR";

export interface Filter {
  id: string;
  type: FilterType;
  operator: FilterOperator;
  value: string;
}

export type TransactionPredicate = (transaction: Transaction) => boolean;

// ---------------------------------------------------------------------------
// Operator definitions per filter type
// ---------------------------------------------------------------------------

export const OPERATORS_BY_TYPE: Record<FilterType, { value: FilterOperator; label: string }[]> = {
  account: [
    { value: "=", label: "equals" },
    { value: "!=", label: "not equals" },
    { value: "contains", label: "contains" },
    { value: "starts_with", label: "starts with" }
  ],
  amount: [
    { value: "=", label: "=" },
    { value: "!=", label: "≠" },
    { value: ">", label: ">" },
    { value: "<", label: "<" },
    { value: ">=", label: "≥" },
    { value: "<=", label: "≤" }
  ],
  date: [
    { value: "=", label: "on" },
    { value: ">", label: "after" },
    { value: "<", label: "before" },
    { value: ">=", label: "on or after" },
    { value: "<=", label: "on or before" }
  ],
  tag: [
    { value: "=", label: "equals" },
    { value: "!=", label: "not equals" },
    { value: "is_set", label: "is set" },
    { value: "is_not_set", label: "is not set" }
  ]
};

export function defaultOperator(type: FilterType): FilterOperator {
  return OPERATORS_BY_TYPE[type][0].value;
}

// ---------------------------------------------------------------------------
// Predicate builders
// ---------------------------------------------------------------------------

function matchAccount(op: AccountOperator, value: string): (account: string) => boolean {
  const lower = value.toLowerCase();
  switch (op) {
    case "=":
      return (a) => a === value;
    case "!=":
      return (a) => a !== value;
    case "contains":
      return (a) => a.toLowerCase().includes(lower);
    case "starts_with":
      return (a) => a.toLowerCase().startsWith(lower);
  }
}

function matchAmount(op: AmountOperator, value: number): (amount: number) => boolean {
  switch (op) {
    case "=":
      return (a) => a === value;
    case "!=":
      return (a) => a !== value;
    case ">":
      return (a) => a > value;
    case "<":
      return (a) => a < value;
    case ">=":
      return (a) => a >= value;
    case "<=":
      return (a) => a <= value;
  }
}

function matchDate(op: DateOperator, value: dayjs.Dayjs): (date: dayjs.Dayjs) => boolean {
  switch (op) {
    case "=":
      return (d) => d.isSame(value, "day");
    case ">":
      return (d) => d.isAfter(value, "day");
    case "<":
      return (d) => d.isBefore(value, "day");
    case ">=":
      return (d) => d.isSame(value, "day") || d.isAfter(value, "day");
    case "<=":
      return (d) => d.isSame(value, "day") || d.isBefore(value, "day");
  }
}

export function buildPredicate(filter: Filter): TransactionPredicate {
  switch (filter.type) {
    case "account": {
      const matcher = matchAccount(filter.operator as AccountOperator, filter.value);
      return (t) => _.some(t.postings, (p) => matcher(p.account));
    }
    case "amount": {
      const num = parseFloat(filter.value);
      if (isNaN(num)) return () => true;
      const matcher = matchAmount(filter.operator as AmountOperator, num);
      return (t) => _.some(t.postings, (p) => matcher(p.amount));
    }
    case "date": {
      const date = dayjs(filter.value);
      if (!date.isValid()) return () => true;
      const matcher = matchDate(filter.operator as DateOperator, date);
      return (t) => matcher(t.date);
    }
    case "tag": {
      if (filter.operator === "is_set") {
        return (t) => _.some(t.postings, (p) => p.tag_recurring !== "");
      }
      if (filter.operator === "is_not_set") {
        return (t) => _.every(t.postings, (p) => p.tag_recurring === "");
      }
      const tagValue = filter.value;
      if (filter.operator === "=") {
        return (t) => _.some(t.postings, (p) => p.tag_recurring === tagValue);
      }
      // !=
      return (t) => _.every(t.postings, (p) => p.tag_recurring !== tagValue);
    }
  }
}

export function combinePredicate(filters: Filter[], logic: LogicOperator): TransactionPredicate {
  if (filters.length === 0) return () => true;
  const predicates = filters.map(buildPredicate);
  if (logic === "AND") {
    return (t) => _.every(predicates, (p) => p(t));
  } else {
    return (t) => _.some(predicates, (p) => p(t));
  }
}

// ---------------------------------------------------------------------------
// Display helpers
// ---------------------------------------------------------------------------

export function operatorLabel(op: FilterOperator): string {
  switch (op) {
    case "=":
      return "=";
    case "!=":
      return "≠";
    case ">":
      return ">";
    case "<":
      return "<";
    case ">=":
      return "≥";
    case "<=":
      return "≤";
    case "contains":
      return "contains";
    case "starts_with":
      return "starts with";
    case "is_set":
      return "is set";
    case "is_not_set":
      return "not set";
    default:
      return op;
  }
}

export function filterLabel(filter: Filter): string {
  if (filter.operator === "is_set") return `${filter.type} is set`;
  if (filter.operator === "is_not_set") return `${filter.type} not set`;
  return `${filter.type} ${operatorLabel(filter.operator)} ${filter.value}`;
}
