# Natural Language Parser – Phase 1 Implementation Roadmap

**Status**: ✅ Ready to implement  
**Scope**: Phase 1 (Core Parser) + Phase 2 (Frontend & ML Logging) in parallel  
**Timeline**: 3-4 weeks  
**Target Ship Date**: End of May 2026

---

## 📋 What's Already Done (Design & Scaffolding)

✅ **Core Parser Package** (`internal/parser/`)
- `parser.go` – Main ParseTransaction() entry point with 8-step pipeline
- `keywords.go` – Keyword matching + config loading
- `confidence.go` – Confidence scoring algorithm with thresholds
- `nlp_patterns.go` – Regex patterns for date/amount/payee extraction
- `parser_test.go` – 10 scenario tests + edge cases

✅ **ML Training Model** (`internal/model/`)
- `parser_training_log.go` – Database schema + logging functions

✅ **Design Documentation**
- `design-natural-language-parser.md` – Complete specification (15 sections)
- `PARSER-IMPLEMENTATION-DECISIONS.md` – Technical decisions (this file)
- `PARSER-DECISIONS-FINAL.md` – Feature decisions summary
- `PARSER-IMPLEMENTATION-GUIDE.md` – Code templates
- `PARSER-EXAMPLES.md` – Real-world examples
- `PARSER-EXECUTIVE-SUMMARY.md` – Stakeholder overview

---

## 🛠️ Phase 1: Core Parser Implementation (Weeks 1-2)

### Step 1: Complete `internal/parser/parser.go`

**Implement these 8-step pipeline functions**:

1. **`normalizeText(text string) string`** (~50 lines)
   - Convert to lowercase
   - Trim whitespace
   - Expand abbreviations (cc → credit card, atm → cash withdrawal)
   - Remove extra punctuation
   - Return normalized text

2. **`extractDate(text string) (time.Time, float64, error)`** (~100 lines)
   - Use regex patterns from `nlp_patterns.go`
   - Try ISO format first (2026-05-10)
   - Try month+day format (May 10, 10 May)
   - Try relative dates (today, yesterday)
   - Default to time.Now() if not found
   - Return: date, confidence (0-1), error

3. **`extractAmount(text string) (decimal.Decimal, string, float64, error)`** (~100 lines)
   - Match dollar prefix: `$15.50` → (15.50, "USD")
   - Match suffix: `15 USD`, `15 INR` → (15, "USD"/"INR")
   - Match word form: `fifteen dollars` → (15, "USD")
   - Default currency from `config.GetConfig().DefaultCurrency`
   - Return: amount, currency, confidence, error

4. **`extractPayee(text string) (string, float64)`** (~80 lines)
   - Remove action words (bought, paid, transferred)
   - Remove amounts
   - Extract remaining nouns as payee
   - Match against custom payees from keywords config
   - Return: payee name, confidence

5. **`extractHints(text string) (fromHint, toHint string)`** (~80 lines)
   - Search for prepositions: "from", "using", "to", "into"
   - Extract account hints after prepositions
   - Separate "from" and "to" hints
   - Examples: "from checking" → fromHint="checking"
   - Return: (fromHint, toHint)

6. **`matchAccounts(hint string, direction string) (string, float64)`** (~100 lines)
   - Query all accounts from database
   - Integrate with existing `internal/prediction/tf_idf.go`
   - Compute cosine similarity for hint vs each account
   - Return best match + confidence score
   - If confidence <0.75, note for suggestions

7. **`determineDirection(fromHint, toHint string) (string, float64)`** (~60 lines)
   - Check for expense keywords in text (bought, paid, spent)
   - Check for income keywords (received, earned, sold)
   - Check for transfer keywords (moved, transferred)
   - Return: direction ("expense"/"income"/"transfer"), confidence

8. **`computeConfidence(scores ConfidenceScores) float64`** (Already implemented in confidence.go)
   - Weighted average across fields
   - Weights: amount(0.30), from(0.25), to(0.25), payee(0.15), date(0.05)

**Implementation Order**:
1. Start with `normalizeText()` (simplest)
2. `extractDate()` and `extractAmount()` (use regex patterns)
3. `extractPayee()` (simpler matching)
4. `extractHints()` (straightforward string extraction)
5. `matchAccounts()` (integrate with TF-IDF)
6. `determineDirection()` (keyword-based logic)
7. Integration & refinement

**Testing During Implementation**:
```bash
# Test individual components
go test -v ./internal/parser -run TestParseTransaction_SimpleExpense

# Test all scenarios
go test -v ./internal/parser -run TestParseTransaction_

# Benchmark
go test -bench=BenchmarkParseTransaction ./internal/parser
```

### Step 2: Complete `internal/parser/keywords.go`

- Implement `KeywordScore()` function (word boundary checking)
- Implement `ExtractPaymentMethodHint()` (CC vs debit vs cash)
- Implement `ExtractTransactionTypeHint()` (expense vs income vs transfer)
- Implement `FindCustomPayee()` (match against user config)
- Load config customizations from `paisa.yaml` in `LoadKeywordsFromConfig()`

**Testing**:
```bash
go test -v ./internal/parser -run TestKeywordMatching
```

### Step 3: Complete `internal/parser/confidence.go`

- Implement `ComputeConfidence()` function (weighted average formula)
- Helper functions already have stubs:
  - `AccountMatchConfidenceFor(cosineSimilarity)`
  - `DateConfidenceFor(patternType)`
  - `AmountConfidenceFor(clarity)`
  - `DirectionConfidenceFor(markers)`

**Testing**:
```bash
go test -v ./internal/parser -run TestConfidence
```

### Step 4: Complete `internal/parser/nlp_patterns.go`

- Compile regex patterns in `CompilePatterns()` function
- Patterns are already defined but verify they work:
  - Date patterns (ISO, month names, relative)
  - Amount patterns (dollar prefix, suffix, words)
  - Account hints, payment methods, payees
- Add pattern matching helper functions

**Testing**:
```bash
go test -v ./internal/parser -run TestRegexPatterns
```

### Step 5: Implement All Unit Tests

- Fill in the TODO assertions in `parser_test.go`
- Implement test fixture mocking for TF-IDF matching
- Test all 10 scenarios:
  1. ✅ Simple expense
  2. ✅ Income deposit
  3. ✅ Transfer
  4. ✅ Ambiguous amount
  5. ✅ Missing date
  6. ✅ Multi-currency
  7. ✅ Ambiguous account
  8. ✅ Refund scenario
  9. ✅ Cash withdrawal
  10. ✅ Custom payee

**Testing**:
```bash
# Run all parser tests
go test -v ./internal/parser

# Run with coverage
go test -cover ./internal/parser
```

### Step 6: Performance Optimization

- Run benchmarks: `go test -bench ./internal/parser`
- Target: <500ms per transaction (p99)
- Optimize regex compilation (compile once at startup)
- Cache TF-IDF lookups if needed
- Profile with `pprof` if slow

### Step 7: Configuration Schema Update

Update `internal/config/config.go` and the JSON schema with parser options:

```go
type ParserConfig struct {
  AutoCreateConfidenceThreshold float64 `yaml:"auto_create_confidence_threshold"`
  ShowSuggestionsBelow         float64 `yaml:"show_suggestions_below"`
  MaxSuggestions               int     `yaml:"max_suggestions"`
  Keywords                     KeywordConfig `yaml:"keywords"`
  Defaults                     struct {
    ExpenseAccount string
    IncomeAccount  string
    FromAccount    string
    ToAccount      string
  } `yaml:"defaults"`
}
```

---

## 🎨 Phase 2: API & Frontend (Weeks 3-4, Parallel with Phase 1)

### Step 8: Create API Handlers (`internal/server/parser_handlers.go`)

**Endpoint 1: POST `/api/parser/parse` (Preview)**
```
Request: { "text": "20 Apr bought 15$ groceries" }
Response: {
  "success": true,
  "extracted": {
    "date": "2026-04-20",
    "amount": "15.00",
    "currency": "USD",
    "payee": "groceries",
    "from_account": "Liabilities:CC",
    "to_account": "Expenses:Groceries",
    "confidence": { ... }
  },
  "suggestions": [
    { "field": "from_account", "suggestions": [...] }
  ]
}
```

**Endpoint 2: POST `/api/parser/create-transaction` (Create + Log)**
```
Request: { "text": "...", "selected_suggestions": { ... } }
Response: {
  "success": true,
  "entry": "2026-04-20 groceries\n  Liabilities:CC  -15 USD\n  Expenses:Groceries  15 USD",
  "log_id": 123
}
```

**Implementation**:
1. Create `ParseRequest`, `ParseResponse` types
2. Create `ParseHandler(db)` gin handler
3. Create `CreateTransactionHandler(db)` gin handler
4. Handle confidence threshold checks
5. Call `parser.ParseTransaction()` from handlers
6. Return structured JSON responses
7. Use existing `RespondError()` for errors

### Step 9: Implement ML Logging

**In `CreateTransactionHandler`**:
```go
// After user confirms transaction
go logParsingResult(db, result, userConfirmedFrom, userConfirmedTo)
```

**Create `logParsingResult()` function**:
- Non-blocking (async with `go`)
- Insert into `parser_training_log` table
- Log: input, predictions, confidence scores, actual accounts, corrections

### Step 10: Register API Routes

In `internal/server/server.go`:
```go
api := router.Group("/api")
api.POST("/parser/parse", ParseHandler(db))

writeGroup := api.Group("")
writeGroup.Use(ReadonlyMiddleware)
writeGroup.POST("/parser/create-transaction", CreateTransactionHandler(db))
```

### Step 11: Database Migration

Create migration for `parser_training_log` table:
```sql
CREATE TABLE parser_training_log (
  id INTEGER PRIMARY KEY,
  created_at DATETIME,
  input_text TEXT,
  predicted_date TEXT,
  predicted_amount DECIMAL,
  ...
  user_corrected BOOLEAN,
  INDEX idx_created_at (created_at),
  INDEX idx_corrected (user_corrected)
);
```

### Step 12: Create Svelte Modal Component

**File**: `src/lib/components/NLParser.svelte`

**Features**:
- Text input field
- "Parse" button → calls `/api/parser/parse`
- Live preview of extracted fields
- Confidence indicators (green/yellow/red)
- Editable fields
- Suggestions dropdown for low-confidence fields
- "Confirm & Create" button → calls `/api/parser/create-transaction`
- Loading states, error handling, success toast

**Skeleton**:
```svelte
<script lang="ts">
  import { ajax } from "$lib/utils";
  
  let input = "";
  let preview = $state(null);
  let loading = $state(false);
  
  async function parse() {
    loading = true;
    try {
      preview = await ajax("/api/parser/parse", { text: input });
    } finally {
      loading = false;
    }
  }
  
  async function create() {
    // Call /api/parser/create-transaction with selected values
  }
</script>

<div class="nlparser-modal">
  <!-- Input area -->
  <!-- Preview area with confidence indicators -->
  <!-- Suggestions for low-confidence fields -->
  <!-- Confirmation buttons -->
</div>
```

### Step 13: Integration Testing

Create regression test fixtures:
```bash
tests/fixture/
├── parser/
│   ├── simple_expense.json
│   ├── income_deposit.json
│   ├── transfer.json
│   └── ...
```

**Test file**: `tests/regression.test.ts`
- POST `/api/parser/parse` → verify extracted fields
- POST `/api/parser/create-transaction` → verify transaction created
- Verify `parser_training_log` records inserted

### Step 14: Manual E2E Testing

1. Start dev server: `make develop`
2. Test parser modal in UI
3. Try 10 scenarios from design doc
4. Verify confidence scores displayed
5. Test suggestions for low-confidence fields
6. Verify transactions created correctly
7. Check `parser_training_log` table filled

---

## ✅ Quality Checklist (Before Ship)

### Code Quality
- [ ] All code formatted with `gofmt` (Go) and Prettier (TypeScript)
- [ ] All tests passing: `go test ./...`
- [ ] All regression tests passing: `bun test tests/regression.test.ts`
- [ ] No lint errors: `make lint`
- [ ] Performance <500ms per parse: `go test -bench ./internal/parser`

### Feature Completeness
- [ ] All 10 scenarios parse correctly
- [ ] Confidence scores computed accurately
- [ ] Suggestions shown for <0.75 confidence fields
- [ ] Auto-create at >0.85 confidence
- [ ] ML training data logged asynchronously
- [ ] User can customize keywords in `paisa.yaml`

### Documentation
- [ ] Update CHANGELOG.md with new feature
- [ ] Add config schema documentation
- [ ] Create user guide for parser feature
- [ ] Document API endpoints in README or docs

### Testing
- [ ] Unit tests: 25+ test cases
- [ ] Regression tests: All 10 scenarios
- [ ] Edge cases: Empty input, no amount, special characters
- [ ] Performance: <500ms p99 latency

---

## 📦 Package Structure (After Implementation)

```
internal/
├── parser/
│   ├── parser.go              # Main entry point
│   ├── keywords.go            # Keyword matching
│   ├── confidence.go          # Scoring logic
│   ├── nlp_patterns.go        # Regex patterns
│   └── parser_test.go         # 25+ test cases
│
├── model/
│   └── parser_training_log.go # ML training schema
│
└── server/
    ├── parser_handlers.go     # API handlers
    └── server.go              # Route registration

src/lib/components/
└── NLParser.svelte           # Svelte modal

tests/
├── regression.test.ts
└── fixture/parser/
    ├── simple_expense.json
    ├── income_deposit.json
    └── ...
```

---

## 🚀 Deployment & Rollout

### Phase 1 Ship (Core Parser)
- `POST /api/parser/parse` endpoint available
- Users can test parsing via API
- No UI yet (for testing)

### Phase 2 Ship (Frontend + ML)
- Svelte modal component integrated
- UI available in transaction creation flow
- ML training data collection active
- Ready for general user testing

### Rollout Strategy
1. **Week 1-2**: Internal testing of Phase 1 parser
2. **Week 2-3**: Phase 2 frontend development in parallel
3. **Week 3-4**: Integration & E2E testing
4. **Week 4**: Beta release to early users
5. **Week 5**: GA release with documentation

---

## 📊 Success Metrics

| Metric | Target | Validation |
|--------|--------|-----------|
| Parse accuracy (10 scenarios) | 100% | Unit tests + regression |
| Overall confidence score | >0.85 (auto-create) | Test suite |
| TF-IDF account matching | >0.80 avg score | Integration tests |
| API latency | <500ms p99 | Benchmark tests |
| ML training data collection | 100% of transactions | Log table validation |
| No regressions | All existing tests pass | `go test ./...` |
| Code coverage | >80% | `go test -cover` |

---

## 🛑 Known Limitations & Future Work

### Not in MVP (Phase 1-2)
- ❌ Multi-language date parsing (e.g., "20 Avr" French)
- ❌ Voice input → text parsing
- ❌ Batch import (multiple transactions at once)
- ❌ Custom regex patterns per user
- ❌ Supervised ML model training (Phase 3)

### Phase 3+ Enhancements
- Analyze training data + confidence scores
- Train supervised ML model
- Interactive disambiguation UI refinement
- Multi-language support
- Batch transaction import
- Performance optimization (caching, indexing)

---

## 📞 Getting Help

- **Design Questions**: See `design-natural-language-parser.md` (comprehensive spec)
- **Decision Questions**: See `PARSER-IMPLEMENTATION-DECISIONS.md` (this file)
- **Code Templates**: See `PARSER-IMPLEMENTATION-GUIDE.md` (code examples)
- **Real Examples**: See `PARSER-EXAMPLES.md` (expected outputs)
- **Architecture**: See `.github/copilot-instructions.md` (Paisa architecture)

---

**Status**: ✅ Ready to start implementation immediately  
**Next Action**: Begin with Step 1 implementation of `parser.go`  
**Estimated Effort**: 3-4 weeks for both Phase 1 and Phase 2

