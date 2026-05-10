# Natural Language Parser – Implementation Decisions

**Date**: May 10, 2026  
**Status**: Technical decisions finalized for Phase 1-2 concurrent implementation

---

## Implementation Scope & Approach

### 1. **Scope: Phase 1 + Phase 2 Concurrent** ✅

**Decision**: Implement parser core (Phase 1) and frontend UI (Phase 2) in parallel

**Rationale**:
- Frontend can develop based on API contract while parser core is being built
- Both teams don't block each other
- Faster time to MVP (ship both together in ~3-4 weeks)
- UI can drive parser requirements (confidence display, suggestions)

**Timeline**:
```
Week 1-2: Parser API (Phase 1) + UI mockups (Phase 2)
Week 3: Integration testing + refinement
Week 4: Bug fixes, perf optimization, shipping
```

**Deliverables at Ship**:
- ✅ Parser API fully functional
- ✅ Svelte modal component integrated
- ✅ ML training data collection active
- ✅ Interactive disambiguation UI working
- ✅ Full regression test coverage

---

### 2. **NLP Library Choice: Go Standard Library `regexp`** ✅

**Decision**: Use `regexp` (Go stdlib) + custom parsing logic. NO external NLP library.

#### Comparison Table

| Aspect | Go `regexp` | External Lib (e.g., `regexp2`, NLP lib) |
|--------|------------|----------------------------------------|
| **Binary Size** | ~0KB (built-in) | +1-15MB |
| **Performance** | ⚡ Fastest (~μs) | Slower (1-5x) |
| **Dependencies** | 0 (stdlib only) | 1+ external |
| **Build Time** | Fast | Slower (more compile) |
| **Pattern Support** | RE2 (no lookahead) | More features (lookahead, Unicode) |
| **Maintenance** | Go team | Community/vendor |
| **Docker Size** | Smaller | Larger image |
| **Deployment** | Simple | More validation needed |

#### Pattern Complexity Analysis

Our patterns are **simple to moderate**:
- ✅ Dates: `\d{1,2}\s+(Jan|Feb|...|Dec|\d{1,2}|\d{4})`
- ✅ Amounts: `\d+\.?\d*\s*(INR|USD|EUR|...)?`
- ✅ Payees: word sequences + common markers (bought, paid, transferred)
- ✅ Accounts: colon-separated hierarchies `Assets:Bank:Checking`
- ❌ Lookahead/lookbehind: Not needed
- ❌ Named groups: Can use positional groups + slicing

#### Why `regexp` is Sufficient

1. **No lookahead needed**: We extract left-to-right linearly
2. **Simple grouping**: Positional groups work fine
3. **Performance critical**: High throughput (many users parsing daily)
4. **Binary size matters**: Paisa ships as single Go binary + desktop (Wails)
5. **TF-IDF is the heavy lifting**: NLP work is in account matching, not regex

#### Cost/Benefit Analysis

| External Lib Type | Benefit | Cost | Verdict |
|------------------|---------|------|---------|
| `regexp2` (better regex) | Named groups, lookahead | +2MB binary, slower | ❌ Not worth it |
| General NLP lib (e.g., go-nlp) | Tokenization helpers | +5-10MB, complex API | ❌ Overkill |
| ML framework (TensorFlow Go) | Pre-trained models | +50MB, infrastructure | ❌ Way overkill |
| Custom regex + helpers | Maintainability | None (our approach) | ✅ Best choice |

**Recommendation**: **Go `regexp` + custom helper functions**
- No dependencies
- Fast
- Binary stays ~15-20MB (not ~25-40MB with external libs)
- Full control over pattern evolution
- Easy to debug (patterns are visible in code)

---

### 3. **Confidence Threshold: Auto-Create at >0.85** ✅

**Decision**: Automatically create transaction if overall confidence ≥ 0.85. No confirmation dialog needed.

**Implementation**:
```
POST /api/parser/create-transaction?auto_create=true (default)
  if confidence >= 0.85: 
    → Create entry + log to DB → return success
  if confidence < 0.85:
    → Show suggestions + return preview (UI handles confirmation)
```

**Confidence Floor Definition**:
- **0.90-1.00**: High confidence → Auto-create
- **0.75-0.89**: Medium confidence → Show 1-2 alternatives, ask user
- **0.50-0.74**: Low confidence → Show top 3, require selection
- **<0.50**: Very low → Show all matching accounts, suggest manual entry

**Benefits**:
1. Fast path for high-confidence parses (most cases ~85%+)
2. User doesn't see friction for confident matches
3. Rare ambiguities handled with suggestions
4. Reduces latency (no round-trip dialog)

**Config Option** (for conservative users):
```yaml
# paisa.yaml
parser:
  auto_create_confidence_threshold: 0.85  # adjustable per user preference
  require_confirmation_below: 0.75        # always ask below this
```

---

### 4. **Keywords: Configurable (Phase 1 Support)** ✅

**Decision**: Allow keyword customization via `paisa.yaml` from Phase 1.

**Phase 1 Implementation**:

Add to `paisa.yaml` schema:
```yaml
parser:
  keywords:
    transaction_markers:
      expense: ["bought", "paid", "spent", "withdrew", "transferred out"]
      income: ["received", "sold", "earned", "transferred in"]
      transfer: ["moved", "transferred", "sent to", "from"]
    
    payment_methods:
      cc: ["credit card", "cc", "amex", "visa", "mastercard"]
      debit: ["debit", "chequing", "debit card"]
      cash: ["cash", "handed"]
    
    # User can add custom keywords (e.g., company-specific terms)
    custom_payees:
      "Acme Corp": "Liabilities:CorporateCard:Acme"
      "Starbucks": "Expenses:Dining:Coffee"
```

**How It Works**:

1. **Load at startup**: `parser.NewKeywordMatcher(config.GetConfig())`
2. **Hints extraction**: Match user input against keywords to detect patterns
3. **Override default weights**: Users can adjust keyword importance
4. **Override built-ins**: Custom keywords take precedence

**Example**:
```
Input: "20 Apr, bought 15$ groceries using my amex from no frills"

Keywords detected:
  - "bought" → expense (weight: 0.9)
  - "amex" → credit card payment method (weight: 0.8)
  - "no frills" → matches custom mapping or TF-IDF
  
Confidence boost:
  - If "amex" always points to "Liabilities:Amex:CC", boost that match
  - Custom payees like "no frills" → "Expenses:Groceries" override TF-IDF
```

**Storage**:
- Built-in defaults in code: `internal/parser/keywords.go`
- User customizations loaded from `paisa.yaml`
- Hot-reload on config change? (Phase 2 enhancement)

**Phase 3 Enhancement** (not MVP):
- Web UI to add/remove keywords
- Suggestion learning (auto-add frequently-corrected keywords)
- Per-account alias configuration

---

### 5. **Batch Processing: Deferred (Phase 3+)** ✅

**Decision**: NOT for Phase 1-2. Single transaction per request.

**Rationale**:
- MVP focus: Get single-transaction parsing rock-solid
- API design is simpler (one input → one output)
- Testing is easier (no state machine, transaction ordering)
- User can call API multiple times if needed
- Phase 3: Can batch API calls on frontend

**Future Batch Approach** (when implemented):
```
POST /api/parser/create-transactions (plural)
{
  transactions: [
    { text: "20 Apr bought 15$ groceries" },
    { text: "21 Apr paid 50$ rent" }
  ]
}
→ Returns { success: true, created: [entry1, entry2], failed: [] }
```

---

## Code Organization for Phase 1-2 Parallel Work

### **Track A: Parser Core (Phase 1)**

**Files to create**:
```
internal/parser/
  ├── parser.go                 # Main ParseTransaction() entry point
  ├── keywords.go               # Keyword definitions + matcher
  ├── nlp_patterns.go           # Regex patterns for extraction
  ├── confidence.go             # Scoring logic + thresholds
  ├── parser_test.go            # Unit tests (10 scenarios)
  └── fixtures/                 # Test data
      ├── scenario_expense.txt
      ├── scenario_transfer.txt
      └── ...
```

**Key Functions**:
```go
// Main entry point
func ParseTransaction(text string, cfg *config.Config, db *gorm.DB) (*ParseResult, error)

// Field extraction (8 steps)
func normalizeText(s string) string
func extractDate(s string) (time.Time, float64, error)
func extractAmount(s string) (decimal.Decimal, float64, error)
func extractPayee(s string) (string, float64)
func extractHints(s string) (fromHint, toHint string)
func matchAccounts(hints string, db *gorm.DB) (account string, score float64)
func determineDirection(fromHint, toHint string) (bool, float64)
func computeConfidence(scores map[string]float64) float64

// Data structures
type ParseResult struct {
  Date          time.Time
  Amount        decimal.Decimal
  Currency      string
  Payee         string
  FromAccount   string
  ToAccount     string
  Confidence    ConfidenceScores
  Suggestions   []Suggestion  // For UI
  Warnings      []string
}

type ConfidenceScores struct {
  Date        float64
  Amount      float64
  Payee       float64
  FromAccount float64
  ToAccount   float64
  Direction   float64
  Overall     float64
}
```

### **Track B: API + Frontend (Phase 2)**

**Files to update/create**:
```
internal/server/
  ├── parser_handlers.go        # ParseHandler, CreateTransactionHandler
  ├── apierror.go               # Error responses (already exists)
  └── (register routes in server.go)

internal/model/
  ├── parser_training_log.go    # ML training data schema + logging

src/lib/
  ├── components/
  │   └── NLParser.svelte       # Modal component
  └── utils.ts                  # API call helpers (already exists)

tests/
  └── regression.test.ts        # `/api/parser/*` endpoint tests
```

**API Contract**:
```go
// Request
type ParseRequest struct {
  Text       string `json:"text"`
  PreviewOnly bool  `json:"preview_only"` // true = parse only, false = create
}

// Response (200 OK)
type ParseResponse struct {
  Success     bool
  Extracted   ParseResult      // all parsed fields
  Suggestions []SuggestionSet  // if confidence < 0.85 for any field
  Warnings    []string
  Entry       string           // resulting ledger entry (if created)
}

// Suggestion for interactive disambiguation
type Suggestion struct {
  Account string  `json:"account"`
  Score   float64 `json:"score"`    // 0.0-1.0
}

type SuggestionSet struct {
  Field       string        `json:"field"` // "from_account", "to_account", etc
  Suggestions []Suggestion  `json:"suggestions"`
}
```

---

## Database Changes (Phase 2)

### New Table: `parser_training_log`

```sql
CREATE TABLE parser_training_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  
  -- Input
  input_text TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  -- Predicted
  predicted_date TEXT,
  predicted_amount TEXT,
  predicted_currency TEXT,
  predicted_from_account TEXT,
  predicted_to_account TEXT,
  
  -- Confidence scores
  confidence_date REAL,
  confidence_amount REAL,
  confidence_from_account REAL,
  confidence_to_account REAL,
  confidence_overall REAL,
  
  -- User confirmation (filled after creation)
  actual_from_account TEXT,  -- null = user didn't correct, matches predicted
  actual_to_account TEXT,
  user_corrected BOOLEAN DEFAULT 0,  -- 1 if user changed suggestion
  correction_feedback TEXT  -- what user changed + why (future)
);

-- Index for analysis
CREATE INDEX idx_parser_training_date ON parser_training_log(created_at);
CREATE INDEX idx_parser_training_corrected ON parser_training_log(user_corrected);
```

---

## Configuration Schema Update (Phase 1)

Add to `paisa.yaml` JSON Schema + defaults:

```yaml
parser:
  # Confidence threshold for auto-creation
  auto_create_confidence_threshold: 0.85
  
  # Show suggestions UI below this threshold
  show_suggestions_below: 0.75
  
  # Maximum suggestions to show
  max_suggestions: 3
  
  # Customizable keywords
  keywords:
    transaction_markers:
      expense: ["bought", "paid", "spent"]
      income: ["received", "earned", "sold"]
      transfer: ["moved", "transferred"]
    
    payment_methods:
      cc: ["credit card", "amex", "visa"]
      debit: ["debit", "checking"]
      cash: ["cash"]
    
    # User-specific mappings
    custom_payees: {}
    
  # Default accounts when hints missing
  defaults:
    expense_account: "Expenses:Groceries"
    income_account: "Income:Salary"
    from_account: "Assets:Checking"
    to_account: "Expenses:Uncategorized"
```

---

## Quality Gates (Phase 1-2)

### Unit Tests (Phase 1)
- ✅ 10 scenario tests with confidence scores
- ✅ Edge cases (malformed dates, missing amounts, etc.)
- ✅ Keyword matching
- ✅ Confidence computation
- Run: `go test ./internal/parser/...`

### API Tests (Phase 2)
- ✅ `/api/parser/parse` preview endpoint
- ✅ `/api/parser/create-transaction` with auto-create
- ✅ Suggestions array when confidence < 0.85
- ✅ ML training log insertion
- Run: `go test ./internal/server/...`

### Regression Tests
- ✅ New `/api/parser/*` fixtures for all 10 scenarios
- ✅ Confidence scores validated
- ✅ Training log schema verified
- Run: `bun test tests/regression.test.ts`

### Performance
- ✅ Parse latency <500ms (p99)
- ✅ No DB performance impact
- Profile: `go test -bench ./internal/parser/...`

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Regex complexity grows | Keep patterns in `nlp_patterns.go`, well-documented |
| Confidence scoring unreliable | Unit tests + Phase 3 ML training validates |
| User confused by suggestions | Clear UI, show confidence %, explain score |
| False confidence > 0.85 | Conservative threshold, can be adjusted per user |
| ML training data grows large | Implement archival in Phase 3 (6-12 month rollover) |

---

## Implementation Checklist

### Phase 1: Parser Core
- [ ] Create `internal/parser/` package structure
- [ ] Implement `parser.go` with 8-step pipeline
- [ ] Regex patterns in `nlp_patterns.go`
- [ ] Keyword matcher with config support
- [ ] Confidence scoring in `confidence.go`
- [ ] Unit tests for 10 scenarios
- [ ] Performance profiling (<500ms)
- [ ] Update `paisa.yaml` schema
- [ ] Load keywords from config

### Phase 2: API & Frontend (Parallel)
- [ ] Create `parser_training_log` table migration
- [ ] Implement `parser_handlers.go` (Parse + CreateTransaction)
- [ ] Register API routes in `server.go`
- [ ] ML logging async function
- [ ] Svelte modal component `NLParser.svelte`
- [ ] Interactive disambiguation UI
- [ ] Confidence indicator display
- [ ] Suggestions click handler
- [ ] API integration tests
- [ ] Regression tests
- [ ] Manual E2E testing

### Shipping
- [ ] Lint + format all code
- [ ] All tests pass (unit + regression)
- [ ] Update CHANGELOG.md
- [ ] Documentation for users (config options)
- [ ] Confidence threshold docs
- [ ] Example outputs
- [ ] Ship Phase 1-2 together

---

## Summary: Key Technical Decisions

| Decision | Choice | Reasoning |
|----------|--------|-----------|
| Scope | Phase 1 + 2 parallel | Faster MVP, no blocking |
| NLP Library | Go stdlib `regexp` | No dependencies, fast, small binary |
| Auto-Create Threshold | 0.85 confidence | Balances automation + accuracy |
| Keywords | Configurable in YAML | User customization from Day 1 |
| Batch Processing | Phase 3+ | MVP focus: single transaction |

---

## References

- [design-natural-language-parser.md](design-natural-language-parser.md) – Full specification
- [PARSER-IMPLEMENTATION-GUIDE.md](PARSER-IMPLEMENTATION-GUIDE.md) – Code templates
- [PARSER-DECISIONS-FINAL.md](PARSER-DECISIONS-FINAL.md) – Feature decisions
- [.github/copilot-instructions.md](.github/copilot-instructions.md) – Paisa architecture

---

**Ready to Code**: ✅ YES  
**All Decisions Finalized**: ✅ YES  
**Start Implementation**: ✅ IMMEDIATELY
