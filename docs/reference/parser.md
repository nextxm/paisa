# Natural Language Parser API

This document describes the Phase 2 backend APIs for natural-language transaction parsing.

## Endpoints

### `POST /api/parser/parse`
Parses free-form transaction text and returns structured suggestions with confidence scores.

Request body:

```json
{
  "text": "20 Apr spent $15 at grocery store using credit card"
}
```

Response body:

```json
{
  "result": {
    "date": "2026-04-20T00:00:00Z",
    "amount": "15",
    "currency": "USD",
    "payee": "grocery store",
    "from_account": "Liabilities:CreditCard:Visa",
    "to_account": "Expenses:Groceries",
    "direction": "expense",
    "confidence": {
      "date": 0.8,
      "amount": 0.95,
      "payee": 0.8,
      "from_account": 0.72,
      "to_account": 0.81,
      "direction": 0.85,
      "overall": 0.82
    },
    "suggestions": [],
    "warnings": []
  },
  "auto_create": false,
  "requires_confirmation": true
}
```

### `POST /api/parser/create-transaction`
Parses free-form text, applies optional user overrides, appends a transaction to `add_journal_path`, triggers journal sync, and asynchronously logs training data.

Request body:

```json
{
  "text": "paid $20 for lunch using debit",
  "date": "2026-05-10",
  "payee": "Lunch",
  "from_account": "Assets:Checking",
  "to_account": "Expenses:Dining",
  "amount": "20",
  "commodity": "USD",
  "suggestion_used": 1,
  "time_to_confirm_ms": 1200
}
```

Response body:

```json
{
  "success": true,
  "entry": "2026/05/10 * Lunch\n  Assets:Checking  -20 USD\n  Expenses:Dining   20 USD\n",
  "final_transaction": {
    "date": "2026-05-10",
    "payee": "Lunch",
    "from_account": "Assets:Checking",
    "to_account": "Expenses:Dining",
    "amount": "20",
    "commodity": "USD"
  },
  "parser_confidence": {
    "overall": 0.82
  },
  "parser_suggestions": []
}
```

## Error Handling

Both endpoints use the standard error envelope:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "..."
  }
}
```

Possible codes include:
- `INVALID_REQUEST` for malformed JSON or invalid field values
- `INTERNAL_ERROR` for unexpected server-side failures
- `READONLY` when write endpoint is blocked by readonly mode
- `UNAUTHORIZED` when auth fails

## Notes

- `POST /api/parser/parse` is a read operation and does not mutate data.
- `POST /api/parser/create-transaction` is a write operation and is protected by readonly middleware.
- On low-confidence parse results, clients should show account suggestions and allow user correction before create.
