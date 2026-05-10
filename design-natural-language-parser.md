# Natural Language Transaction Parser – Design & Implementation Plan

## Executive Summary

Add a **Natural Language Parser** endpoint that accepts free-form text descriptions of transactions (e.g., "20 Apr, bought 15$ groceries using bmo cc from no frills"), extracts transaction elements, fills in defaults from config, and creates a ledger transaction via the existing `POST /api/add/transaction` flow.

This feature leverages **existing infrastructure**:
- TF-IDF account matching (`internal/prediction/tf_idf.go` + `src/lib/template_helpers.ts`)
- Date/amount parsing from import templates
- Structured transaction creation endpoint (`AddTransactionHandler`)
- Flexible config system (`paisa.yaml`)

---

## 1. Parser Architecture

### 1.1 Component Structure

```
internal/parser/
├── parser.go                  # Main parsing logic (Go)
│   ├── ParseTransaction()      # Entry point
│   ├── extractAmount()         # Regex + heuristics
│   ├── extractDate()           # Flexible date parsing
│   ├── extractPayee()          # Residual text / quoted strings
│   └── extractAccountHints()   # Keywords like "using", "from", "via"
│
├── nlp_patterns.go            # Regex patterns + keyword mappings
│   ├── patterns for amounts    # $15, 15.00, 15$ etc.
│   ├── patterns for dates      # "Apr 20", "20/4", "2026-04-20", etc.
│   ├── account keyword maps    # "cc" → liability hints, "cash" → asset
│   └── payee extractors        # "from <merchant>", "at <location>"
│
└── parser_test.go             # Unit tests for each scenario

src/lib/
├── nlp_parser.ts              # Frontend TypeScript helpers (optional)
│   ├── normalizeText()         # Trim, lowercase, split
│   ├── extractCurrency()       # If text contains "15 USD" etc.
│   └── UI component hooks      # For preview + feedback
```

### 1.2 Data Flow

```
User Input Text
    ↓
[ParseTransaction] → extracts date, amount, payee, account hints
    ↓
[Account Matching via TF-IDF] → from_account, to_account guesses
    ↓
[Config Defaults] → fill missing currency, commodity
    ↓
[Validate] → ensure date/amount/accounts
    ↓
[AddTransactionRequest] → POST /api/add/transaction
    ↓
Ledger Transaction Created ✓
```

---

## 2. Configuration Defaults

Extend `paisa.yaml` with optional defaults section:

```yaml
# Optional: Natural language parser defaults
parser_defaults:
  # Default transaction metadata
  default_account_from: "Assets:Checking"        # If from_account not inferred
  default_account_to: "Expenses:Unknown"         # If to_account not inferred
  
  # Account category keywords for smarter matching
  account_keywords:
    # Assets
    assets:
      - checking
      - saving
      - cash
      - wallet
      - debit
    # Liabilities
    liabilities:
      - credit
      - cc
      - visa
      - mastercard
      - amex
      - loan
      - mortgage
    # Expenses
    expenses:
      - groceries
      - food
      - restaurant
      - gas
      - fuel
      - shopping
```

**OR** simpler approach: hardcode common keywords + allow override.

---

## 3. Parsing Logic: Detailed Algorithm

### 3.1 Tokenization & Preprocessing

```go
input := "20 Apr, bought 15$ groceries using bmo cc from no frills"
// Normalize
text := normalize(input)  // lowercase, trim, standardize symbols
// → "20 apr bought 15 dollars groceries using bmo cc from no frills"
```

### 3.2 Extraction Steps (Sequential)

**Step 1: Extract Date**
```
Patterns: [YYYY-MM-DD, DD/MM/YYYY, D-Mon, Mon D, D Mon YYYY, "today", "yesterday"]
Example: "20 Apr" → 2026-04-20 (current year, using TimeZone)
Fallback: config.DefaultDate OR today
```

**Step 2: Extract Amount**
```
Patterns: [$15, 15$, 15 dollars, 15 USD, $15.00, 15.99]
Regex: /\$?\d+\.?\d*\$?|\d+\.?\d*\s*(dollars?|usd|eur|inr|£|€|₹)/i
Example: "15$" → amount: "15.00", currency: (inferred or default)
Edge cases: 
  - "15$" = $15 positive
  - "(15)" = -15 negative
  - Multiple amounts? Use largest, warn user
```

**Step 3: Extract Payee**
```
Heuristics:
  a) Check for quoted text: "from 'No Frills'" → payee = "No Frills"
  b) Look for location keywords: "at Costco" → payee = "Costco"
  c) Extract between "from" and next verb: "from no frills to checking" 
     → payee = "no frills"
  d) Residual nouns not in account keywords
Example: "no frills" from "using bmo cc from no frills" → payee
```

**Step 4: Extract Account Hints**
```
Use keyword patterns:
  "using X" / "via X" / "with X"  → from_account hint
  "from X" / "out of X"             → from_account hint  
  "to Y" / "into Y"                 → to_account hint
  "for Y" / "for expenses:Y"        → to_account hint

Example: "using bmo cc" → from_account hint = "bmo cc"
         "groceries" (implicit expense category)
```

**Step 5: Match Accounts via TF-IDF**
```
from_account_hint: "bmo cc" + posting data → TF-IDF search
  → "Liabilities:CAD:BMO:CC" (high score)
  
to_account_hint: "groceries" + posting data → TF-IDF search
  → "Expenses:Groceries" (high score)

If no match: use config defaults (default_account_from, default_account_to)
```

**Step 6: Infer Transaction Direction**
```
Heuristics:
  - If from is liability + to is expense → expense (most common)
  - If from is asset + to is income → income
  - Otherwise: ask user / use config default direction

Example:
  from="Liabilities:CAD:BMO:CC", to="Expenses:Groceries"
  → Expense transaction (liability decreases, expense increases)
```

---

## 4. Scenario Coverage

### Scenario 1: Simple Expense (User's Example)
```
Input: "20 Apr, bought 15$ groceries using bmo cc from no frills"

Extracted:
  date: 2026-04-20
  amount: 15.00
  payee: "No Frills"
  from_hint: "bmo cc"
  to_hint: "groceries"

Matched:
  from: "Liabilities:CAD:BMO:CC" (TF-IDF on "bmo cc")
  to: "Expenses:Groceries" (TF-IDF on "groceries")
  currency: "CAD" (from config or inferred)

Transaction:
  2026-04-20 * "No Frills"
    Liabilities:CAD:BMO:CC  -15.00 CAD
    Expenses:Groceries      15.00 CAD
```

### Scenario 2: Transfer Between Accounts
```
Input: "transferred 500 from checking to savings account"

Extracted:
  date: today
  amount: 500.00
  from_hint: "checking"
  to_hint: "savings"

Matched:
  from: "Assets:Checking" (TF-IDF)
  to: "Assets:Savings" (TF-IDF)
  currency: "INR" (from config)

Transaction:
  2026-05-09 * "Transfer"
    Assets:Checking     -500.00 INR
    Assets:Savings      500.00 INR
```

### Scenario 3: Multi-Currency Exchange
```
Input: "paid 100 USD for 1 EUR transfer"

Extracted:
  amount: 100.00
  amount_to: 1.00
  currency_from: USD
  currency_to: EUR

Matched:
  from: "Assets:Checking:USD" (inferred)
  to: "Assets:Checking:EUR" (inferred)

Transaction:
  2026-05-09 * "Currency Exchange"
    Assets:Checking:USD  -100.00 USD
    Assets:Checking:EUR  1.00 EUR @ 100 USD
```

### Scenario 4: Missing Date (Defaults to Today)
```
Input: "bought groceries for 25 dollars"

Extracted:
  date: 2026-05-09 (today)
  amount: 25.00
  from_hint: (none)
  to_hint: "groceries"

Matched:
  from: (config default) "Assets:Checking"
  to: "Expenses:Groceries"

Transaction:
  2026-05-09 * "Unknown"
    Assets:Checking     -25.00 INR
    Expenses:Groceries   25.00 INR
```

### Scenario 5: Ambiguous Amount / Multiple Numbers
```
Input: "on 15 Apr got salary 5000 and spent 100 on lunch"

Issue: Multiple amounts detected (5000 and 100)
Solution A: Use largest amount (5000) and ask user
Solution B: Warn "multiple amounts, please clarify"
Solution C: Create multiple transactions (requires confirmation)

Best UX: API returns `confidence: low` + asks for confirmation via UI
```

### Scenario 6: Quoted/Explicit Account Names
```
Input: "paid 50 from Assets:CheckingAccount to Expenses:Groceries"

Extracted:
  from_hint: "Assets:CheckingAccount"
  to_hint: "Expenses:Groceries"

Matching:
  from: Exact match → "Assets:CheckingAccount"
  to: Exact match → "Expenses:Groceries"
  
No TF-IDF needed if explicit account names detected.
```

### Scenario 7: Income Transaction
```
Input: "deposited 3000 salary from employer"

Extracted:
  amount: 3000.00
  from_hint: "salary" / "employer"
  to_hint: (none)

Heuristics:
  Keywords like "salary", "income", "payment from", "invoice paid"
  → infer as income (to Asset, from Income)

Transaction:
  2026-05-09 * "Employer"
    Income:Salary          -3000.00 INR
    Assets:Checking        3000.00 INR
```

### Scenario 8: Investment Purchase
```
Input: "bought 10 AAPL shares at 150 each using bmo cc"

Extracted:
  quantity: 10
  price: 150.00
  commodity: "AAPL"
  total_amount: 1500.00

Matched:
  from: "Liabilities:CAD:BMO:CC"
  to: "Assets:Brokerage:AAPL"

Transaction:
  2026-05-09 * "AAPL Purchase"
    Liabilities:CAD:BMO:CC  -1500.00 CAD
    Assets:Brokerage        10 AAPL @ 150.00 CAD
```

### Scenario 9: Minimal Input (Uses All Defaults)
```
Input: "150"

Extracted:
  amount: 150.00
  (everything else uses config defaults)

Transaction:
  2026-05-09 * "Unknown"
    Assets:Checking         -150.00 INR  (or default)
    Expenses:Unknown        150.00 INR
```

### Scenario 10: With Transaction Tags/Notes
```
Input: "paid rent 1000 on 1st may #recurring=monthly #period=1m"

Extracted:
  date: 2026-05-01
  amount: 1000.00
  from_hint: (none)
  to_hint: "rent"
  tags: { recurring: "monthly", period: "1m" }

Transaction (with metadata):
  2026-05-01 * "Rent"
    Assets:Checking     -1000.00 INR
    Expenses:Rent       1000.00 INR
    ; recurring=monthly, period=1m
```

---

## 5. API Design

### Endpoint: `POST /api/parser/parse`

**Request:**
```json
{
  "text": "20 Apr, bought 15$ groceries using bmo cc from no frills",
  "preview_only": false
}
```

**Response (Success):**
```json
{
  "success": true,
  "extracted": {
    "date": "2026-04-20",
    "payee": "No Frills",
    "amount": "15.00",
    "currency": "CAD",
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries",
    "raw_hints": {
      "from_hint": "bmo cc",
      "to_hint": "groceries"
    }
  },
  "confidence": {
    "overall": 0.92,
    "date": 0.95,
    "amount": 0.98,
    "from_account": 0.87,
    "to_account": 0.89
  },
  "suggestions": [],
  "warnings": [],
  "transaction": {
    "date": "2026-04-20",
    "payee": "No Frills",
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries",
    "amount": "15.00",
    "commodity": "CAD"
  }
}
```

**Note**: Confidence scores are logged for ML training dataset. When user confirms a transaction, the actual account match is recorded for future model improvement.

**Response (With Warnings & Suggestions):**
```json
{
  "success": true,
  "extracted": { ... },
  "confidence": { "overall": 0.62, ... },
  "suggestions": [
    {
      "field": "from_account",
      "current": "Liabilities:BMO:XYZ",
      "candidates": [
        {"account": "Liabilities:CAD:BMO:CC", "score": 0.88},
        {"account": "Liabilities:BMO:Debit", "score": 0.65},
        {"account": "Liabilities:BMO:Personal", "score": 0.58}
      ],
      "reason": "Low confidence match (0.65). Did you mean one of these?"
    }
  ],
  "warnings": [
    "Multiple amounts detected (100, 500). Using largest (500)."
  ],
  "transaction": { ... }
}
```

**Interactive Disambiguation**: If confidence for any field is <0.75, `suggestions` array contains top 3 matching alternatives. User can click to select alternative from UI.

**Response (Error):**
```json
{
  "success": false,
  "error": "Could not extract amount from input text",
  "extracted": {
    "date": "2026-05-09",
    "payee": "Some text"
  }
}
```

### Endpoint: `POST /api/parser/create-transaction`

**Request:**
```json
{
  "text": "20 Apr, bought 15$ groceries using bmo cc from no frills",
  "confirm": true  // require explicit confirmation
}
```

**Response:**
```json
{
  "success": true,
  "message": "Transaction created successfully",
  "entry": "2026-04-20 * \"No Frills\"\n  Liabilities:CAD:BMO:CC  -15.00 CAD\n  Expenses:Groceries      15.00 CAD"
}
```

---

## 6. Frontend UI

### Option A: Modal / Dialog Component
```
[Transaction Input Box]
"20 Apr, bought 15$ groceries using bmo cc from no frills"

[Parse Preview]
┌─────────────────────────┐
│ Date: 2026-04-20        │
│ Payee: No Frills        │
│ From: Liabilities:...CC │ ⚠️ (87% confidence)
│ To: Expenses:Groceries  │ ✓ (89% confidence)
│ Amount: 15.00 CAD       │
│ Confidence: 92%         │
└─────────────────────────┘

[⚠️ Warnings (if any)]
[Cancel] [Edit] [Create]
```

### Option B: Inline Parser (Quick Add Enhancement)
Replace current "Quick Add" form with:
```
1. Text input field (natural language)
2. Live parsing preview (as you type)
3. Editable extracted fields (override if needed)
4. Create button
```

### Option C: Separate Page/Route
`/ledger/parse` - dedicated page for complex transactions

---

## 7. Implementation Phases

### Phase 1: Core Parser (Go)
- [ ] `internal/parser/parser.go` – Main parsing logic
- [ ] `internal/parser/nlp_patterns.go` – Regex patterns + keywords
- [ ] `internal/parser/parser_test.go` – Unit tests (all 10 scenarios)
- [ ] Endpoint: `POST /api/parser/parse` (preview only)
- [ ] Config schema extension (optional defaults)

### Phase 2: Integration & UI
- [ ] `POST /api/parser/create-transaction` endpoint
- [ ] Frontend component (modal/dialog)
  - [ ] Show confidence scores for each field
  - [ ] Display suggestions for low-confidence matches
  - [ ] Allow user to pick from top 3 alternatives
- [ ] Log parsing results + user corrections to DB (for ML training)
- [ ] Integration with existing Quick Add / Transaction page
- [ ] Error handling & user feedback

### Phase 3: ML & Future Enhancements (Future)
- [ ] Analyze stored confidence scores + user corrections
- [ ] Train ML model on parsing history
- [ ] Improve confidence scoring algorithm over time
- [ ] Consider multi-language support (if user base grows)
- [ ] Batch parsing enhancements

---

## 8. Testing Strategy

### Unit Tests (`parser_test.go`)
```go
TestParseSimpleExpense() // Scenario 1
TestParseTransfer()      // Scenario 2
TestParseMultiCurrency() // Scenario 3
TestParseMissingDate()   // Scenario 4
TestParseMultipleAmounts() // Scenario 5
TestParseExplicitAccounts() // Scenario 6
TestParseIncome()        // Scenario 7
TestParseInvestment()    // Scenario 8
TestParseMinimalInput()  // Scenario 9
TestParseWithTags()      // Scenario 10
```

### Regression Tests (`tests/regression.test.ts`)
Add new test cases for the `/api/parser/*` endpoints:
```typescript
test('POST /api/parser/parse returns extracted transaction', async () => {
  const { extracted } = await fetch('/api/parser/parse', {
    text: "20 Apr, bought 15$ groceries using bmo cc from no frills"
  }).json();
  
  expect(extracted.date).toBe('2026-04-20');
  expect(extracted.amount).toBe('15.00');
  expect(extracted.from_account).toContain('BMO');
  expect(extracted.to_account).toContain('Groceries');
});
```

### Edge Case Tests
- Empty input
- Malformed dates
- Currency mismatch (user has INR, enters USD)
- Account ambiguity (multiple good matches)
- Special characters / unicode

---

## 9. Security & Validation

1. **Input Sanitization**
   - Reject inputs > 1000 chars (DOS prevention)
   - Filter HTML/script tags
   - Validate account names match existing accounts (or use defaults)

2. **Authorization**
   - Require `X-Auth` header (token or legacy)
   - Subject to `ReadonlyMiddleware` rate limits

3. **Data Validation**
   - Date must be valid and within reasonable range (e.g., past 10 years)
   - Amount must be positive (or support negative with `-` prefix)
   - Accounts must exist in system

---

## 10. ML Training Data Collection

### Overview
To enable future ML model improvements, the system will **automatically log parsing results and user feedback**.

### Data Captured
For each parse + user confirmation:
```json
{
  "timestamp": "2026-05-09T10:30:00Z",
  "input_text": "20 Apr, bought 15$ groceries using bmo cc from no frills",
  "extracted": {
    "date": "2026-04-20",
    "amount": "15.00",
    "payee": "No Frills",
    "from_hint": "bmo cc",
    "to_hint": "groceries"
  },
  "predicted": {
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries"
  },
  "confidence": {
    "overall": 0.92,
    "from_account": 0.87,
    "to_account": 0.89
  },
  "actual": {
    "from_account": "Liabilities:CAD:BMO:CC",  // user may correct this
    "to_account": "Expenses:Groceries"
  },
  "user_corrected": false  // true if user changed anything
}
```

### Storage
- Create `parser_training_log` table in SQLite
- Store one row per parse + user confirmation
- Non-blocking: log asynchronously after transaction created
- Retention: Keep indefinitely (useful for future model training)

### Privacy & Compliance
- Only store in user's local database (no cloud sync)
- User has full control (can delete via admin panel)
- No PII beyond transaction data already in ledger
- No external transmission (Phase 3 may enable opt-in sharing)

### Future Use (Phase 3)
- Analyze patterns: which hints lead to which accounts?
- Train supervised ML model on input text → account mapping
- Measure precision/recall vs. TF-IDF baseline
- Potentially improve confidence scoring over time
- A/B test new models against user feedback

---

## 11. Interactive Disambiguation UI (Phase 2)

### Low-Confidence Matching Strategy
When TF-IDF confidence for an account is **<0.75**, show suggestions:

```
From Account: Liabilities:BMO:XYZ ⚠️ (65% confidence)

Suggestions (pick one):
  1. ☑️ Liabilities:CAD:BMO:CC (88% match)
  2. ○ Liabilities:BMO:Debit (65% match)
  3. ○ Liabilities:BMO:Personal (58% match)
  
  [Use selected] [Skip & keep current]
```

### Implementation
- Return `suggestions` array in API response (top 3 matches + scores)
- Frontend displays clickable alternatives
- User selection updates extracted field in preview
- Treat user selection as confirmation for ML training

### Benefits
- User always has final say (transparency)
- Captures user's actual intent (improves training data)
- Reduces friction (no need to type full account name)
- Enables learning over time (as ML improves, suggestions fewer needed)

---

## 12. Success Criteria

✓ Parse all 10 scenarios correctly  
✓ Confidence score accurately reflects extraction quality  
✓ All unit tests pass (`go test internal/parser/...`)  
✓ All regression tests pass (`bun test tests`)  
✓ Frontend UI provides clear feedback with confidence indicators  
✓ Interactive disambiguation UI shows top 3 suggestions for low-confidence matches  
✓ Confidence scores + user corrections logged to DB for ML training  
✓ TF-IDF integration working smoothly  
✓ Zero breaking changes to existing API  

---

## 13. Rollout Plan

1. **Development**: Phase 1 (parser core)
   - Implement parsing engine
   - Log confidence scores to DB
   - Unit tests + regression tests
   
2. **Internal Testing**: All scenarios + edge cases
   - Validate 90% accuracy target
   - Review confidence score quality

3. **Beta**: Phase 2 rollout with UI
   - Enable for opt-in users (config flag)
   - Gather feedback on disambiguation experience
   - Monitor ML training data collection

4. **GA**: Full rollout
   - Enable by default
   - Publicize feature
   - Gather community feedback

5. **Iterate**: Phase 3 based on feedback
   - Analyze training data
   - Consider ML model improvements
   - Plan future enhancements (multi-language, etc.)

---

## 14. Confirmed Design Decisions

### ✅ Implemented in MVP
- **Store parsing confidence scores for ML training**: YES
  - Log scores in database for future ML model training
  - Track actual vs. predicted account matches
  - Enable continuous model improvement
  
- **Interactive disambiguation UI (if low confidence)**: YES
  - Show suggestions when confidence <0.75
  - Allow user to pick from top 3 matches instead of just one
  - Learn from user corrections for future improvements

### ❌ Deferred (Future Phases)
- **Multi-language support** (e.g., "20 Avr" for French) – Not MVP
- **Voice input → text → parsing** – Not MVP
- **Batch import from message logs / Slack / Telegram** – Not MVP

---

## 15. Future Enhancement Ideas

After MVP + Phase 2, consider:
- Multi-language date parsing (once user base grows)
- Integration with third-party services (Slack, Telegram)
- Batch transaction import from CSV/structured logs
- Advanced ML model for confidence scoring
- User feedback loop training data collection

---

## Appendix: Keyword Reference

### Account Keywords (Hardcoded Defaults)

```go
var AssetKeywords = []string{
  "checking", "savings", "cash", "wallet", "account",
  "debit", "deposit", "bank", "sbi", "hdfc", "icici",
}

var LiabilityKeywords = []string{
  "credit", "cc", "visa", "mastercard", "amex",
  "loan", "mortgage", "debt", "borrow",
}

var ExpenseKeywords = []string{
  "groceries", "food", "restaurant", "gas", "fuel",
  "shopping", "clothes", "entertainment", "utilities",
  "rent", "bill", "medical", "doctor", "hospital",
}

var IncomeKeywords = []string{
  "salary", "income", "bonus", "payment", "refund",
  "wage", "stipend", "interest", "dividend",
}

var PrepositionKeywords = map[string]string{
  "using": "from_account",
  "via": "from_account",
  "with": "from_account",
  "from": "from_account",
  "out of": "from_account",
  "to": "to_account",
  "into": "to_account",
  "for": "to_account",
  "at": "payee",
  "in": "payee",
}
```

---

## Appendix: Example Config

```yaml
# paisa.yaml
journal_path: main.ledger
add_journal_path: added.ledger
db_path: paisa.db
default_currency: CAD

# (NEW) Natural language parser defaults
parser_defaults:
  default_account_from: "Assets:Checking"
  default_account_to: "Expenses:Unknown"
  enable_interactive_disambiguation: true
  confidence_threshold: 0.75  # warn if below this
```
