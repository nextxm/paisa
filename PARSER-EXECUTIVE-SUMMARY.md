# Natural Language Transaction Parser – Executive Summary

## Feature Overview

**What**: Enable users to create ledger transactions by typing natural language descriptions.

**Example**:  
```
User Input: "20 Apr, bought 15$ groceries using bmo cc from no frills"

Generated Transaction:
2026-04-20 * "No Frills"
  Liabilities:CAD:BMO:CC  -15.00 CAD
  Expenses:Groceries      15.00 CAD
```

**Why**: Faster, more conversational transaction entry. Reduces friction for bulk entry and daily logging.

---

## Design Highlights

### 1. **Leverages Existing Infrastructure**
- ✅ TF-IDF account matching (`internal/prediction/tf_idf.go`)
- ✅ Ledger sync pipeline (`model.SyncJournal()`)
- ✅ Transaction formatting logic
- ✅ Authentication & authorization layers

### 2. **Covers 10 Realistic Scenarios**
1. Simple expense (user's example)
2. Transfer between accounts
3. Multi-currency exchange
4. Missing date (defaults to today)
5. Ambiguous amounts (warnings)
6. Explicit account names
7. Income transaction
8. Investment purchase
9. Minimal input (all defaults)
10. With transaction tags/notes

### 3. **Confidence Scoring**
Each extraction step scored 0.0–1.0:
- Overall confidence (weighted average)
- Per-field confidence (date, amount, accounts)
- Warnings for low confidence (<0.75)
- User can preview & edit before committing

### 4. **Graceful Degradation**
- Missing date? → Use today
- Missing account? → Use config defaults
- Ambiguous match? → Show warning + allow override
- Multiple amounts? → Use largest + warn

---

## Architecture

```
User Input (text)
    ↓
Parser Engine (Go)
  ├─ Normalize & tokenize
  ├─ Extract: date, amount, payee
  ├─ Extract: account hints via keywords
  └─ Score confidence
    ↓
TF-IDF Matcher
  ├─ Match from_account hint
  ├─ Match to_account hint
  └─ Return with confidence scores
    ↓
Config Defaults
  ├─ Fill missing currency
  ├─ Default accounts if no match
  └─ Infer transaction direction
    ↓
API Response
  ├─ Extracted fields
  ├─ Confidence scores
  ├─ Warnings & suggestions
  └─ Ready-to-create transaction
    ↓
Optional: Frontend Preview
  ├─ Show parsed data
  ├─ Allow edits
  ├─ Display confidence
    ↓
Transaction Creation
  ├─ Convert to ledger format
  ├─ Append to journal
  ├─ Sync to database
  └─ Return success
```

---

## API Endpoints

### `POST /api/parser/parse` – Preview Only
```
Request:  { "text": "20 Apr, bought 15$ groceries..." }

Response: {
  "success": true,
  "extracted": {
    "date": "2026-04-20",
    "amount": "15.00",
    "currency": "CAD",
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries",
    "payee": "No Frills"
  },
  "confidence": {
    "overall": 0.92,
    "date": 0.95,
    "amount": 0.98,
    "from_account": 0.87,
    "to_account": 0.89
  },
  "warnings": []
}
```

### `POST /api/parser/create-transaction` – Create & Sync
```
Request:  { "text": "20 Apr, bought 15$ groceries..." }

Response: {
  "success": true,
  "entry": "2026-04-20 * \"No Frills\"\n  Liabilities:CAD:BMO:CC  -15.00 CAD\n  Expenses:Groceries      15.00 CAD"
}
```

---

## Implementation Phases

### **Phase 1: Core Parser (1–2 weeks)**
- Go parser package: `internal/parser/parser.go`
- Regex patterns & keyword matching
- TF-IDF integration
- Unit tests (all 10 scenarios)
- API endpoint: `/api/parser/parse`
- **Deliverable**: Preview endpoint works, ~90% accuracy

### **Phase 2: Frontend & UX (1 week)**
- Svelte modal component
- Live parsing preview
- Editable fields before commit
- API endpoint: `/api/parser/create-transaction`
- Integration with transaction list
- **Deliverable**: Full UI + end-to-end flow

### **Phase 3: Enhancements (Future)**
- ML-based confidence scoring
- Interactive disambiguation (ask user when low confidence)
- Batch import (multiple lines/CSV)
- Custom regex patterns in config
- Mobile optimization
- Voice input support

---

## Key Extraction Steps

### 1. **Normalize** (lowercase, trim, standardize)
```
"20 Apr, bought 15$ groceries using bmo cc from no frills"
→ "20 apr bought 15 dollars groceries using bmo cc from no frills"
```

### 2. **Extract Date** (pattern matching + defaults)
```
Patterns: YYYY-MM-DD, DD-Mon, Mon-DD, "today", "yesterday"
Example: "20 Apr" → 2026-04-20 (using current year + timezone)
Fallback: today's date
```

### 3. **Extract Amount** (regex + currency detection)
```
Patterns: $15, 15$, 15 USD, 15.99, (15), etc.
Example: "15$" → amount=15.00, currency=CAD (from config)
Required: if missing, error
```

### 4. **Extract Payee** (location keywords + residual text)
```
Heuristics: quoted text, "at" keyword, residual nouns
Example: "from no frills" → payee="No Frills"
```

### 5. **Extract Account Hints** (keyword extraction)
```
From keywords: using, via, with, from, out of
To keywords: to, into, for
Example: "using bmo cc" → from_hint="bmo cc"
         "groceries" → to_hint="groceries"
```

### 6. **Match Accounts** (TF-IDF cosine similarity)
```
For each hint, search postings index:
  - "bmo cc" → Liabilities:CAD:BMO:CC (92% confidence)
  - "groceries" → Expenses:Groceries (89% confidence)
Fallback: config defaults if no good match
```

### 7. **Infer Direction** (transaction type)
```
Heuristics:
  Asset → Expense = expense (most common)
  Liability → Expense = expense (credit card purchase)
  Asset → Asset = transfer
  Income → Asset = income
```

### 8. **Score Confidence** (weighted average)
```
overall = avg(date, amount, from_account, to_account)
- High confidence (>0.85): auto-create if requested
- Medium (0.75–0.85): preview + warn + allow edit
- Low (<0.75): require user confirmation
```

---

## Configuration Extension

```yaml
# paisa.yaml (optional)
parser_defaults:
  # Fallback accounts if no match found
  default_account_from: "Assets:Checking"
  default_account_to: "Expenses:Unknown"
  
  # Enable/disable features
  enable_warnings: true
  min_confidence_threshold: 0.75
```

---

## File Structure (After Implementation)

```
internal/
├── parser/                       (NEW)
│   ├── parser.go                Main parsing logic
│   ├── nlp_patterns.go          Regex patterns + keywords
│   ├── confidence.go            Confidence scoring
│   └── parser_test.go           Unit tests (10 scenarios)
│
└── server/
    ├── parser_handlers.go       (NEW) API endpoints
    └── server.go                (modified) Register routes

src/lib/
├── components/
│   └── NLParser.svelte         (NEW) Frontend component
└── utils.ts                    (optional: shared helpers)

tests/
└── regression.test.ts          (add parser tests)
```

---

## Testing Coverage

**Unit Tests** (10 scenarios)
- ✅ Simple expense
- ✅ Transfer
- ✅ Multi-currency
- ✅ Missing date
- ✅ Multiple amounts
- ✅ Explicit accounts
- ✅ Income
- ✅ Investment
- ✅ Minimal input
- ✅ With tags

**Integration Tests**
- ✅ `/api/parser/parse` endpoint
- ✅ `/api/parser/create-transaction` endpoint
- ✅ TF-IDF integration
- ✅ Ledger sync pipeline
- ✅ Config defaults

**Edge Cases**
- Empty input
- Malformed dates
- Missing amount
- Ambiguous accounts
- Special characters
- Unicode payees

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Parse accuracy (10 scenarios) | 100% |
| Confidence score correlation | >0.85 |
| API response time | <500ms |
| TF-IDF match quality | >0.80 average |
| Zero breaking changes | ✅ |
| All regression tests pass | ✅ |

---

## Advantages & Benefits

**For Users:**
- ⚡ Faster transaction entry (type vs. fill form)
- 🎯 Conversational, natural input
- 🔍 Smart account matching (learns from history)
- ⚙️ Flexible defaults (configurable)
- 🛡️ Preview before commit (edit if needed)

**For Paisa:**
- 🔄 Reuses existing TF-IDF infrastructure
- 🏗️ No schema changes
- 🔐 Same auth/security as existing endpoints
- 📊 Confidence scores → future ML improvements
- 🚀 Can be shipped in phases

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Wrong account match | TF-IDF is proven; preview + edit before commit |
| Parsing edge cases | 10 scenario tests; extensible pattern system |
| Performance | Confidence scores pre-computed; <500ms target |
| User confusion | Clear UI feedback; warnings for low confidence |
| Breaking changes | Endpoints are additive; no schema modifications |

---

## Rollout & Adoption

**Phase 1 (Alpha)**: Parser API only  
→ Power users via direct API calls  
→ Gather feedback on extraction accuracy  

**Phase 2 (Beta)**: Frontend UI  
→ Integrate with existing Quick Add  
→ Monitor user behavior  

**Phase 3 (GA)**: Full rollout  
→ Marketing & documentation  
→ Community feedback loop  

---

## Future Enhancements (Not MVP)

- 🤖 Machine learning confidence scoring
- 💬 Interactive disambiguation (ask user when unsure)
- 📦 Batch import (multiple lines, CSV, email)
- 🎨 Custom regex patterns per user
- 🗣️ Voice input → text → parsing
- 📱 Mobile-optimized UI
- 🌍 Multi-language support
- 📚 Learning from corrections (feedback loop)

---

## Documents Reference

| Document | Purpose |
|----------|---------|
| `design-natural-language-parser.md` | Full design spec + 10 scenarios |
| `PARSER-IMPLEMENTATION-GUIDE.md` | Step-by-step implementation |
| `system-architecture-diagram` | Visual architecture |
| `sequence-diagram` | API interaction flow |
| `extraction-pipeline-diagram` | Parser algorithm steps |

---

## Getting Started

**Next Steps**:
1. Review the three design documents
2. Discuss any design trade-offs or concerns
3. Decide on Phase 1 scope (parser core only?)
4. Create GitHub issues for Phase 1 tasks
5. Start implementation with unit tests

**Questions?**
- How much NLP complexity do we want in Phase 1?
- Should we use any external NLP library (go-nlp, etc.)?
- What confidence threshold for auto-create?
- Mobile UI priority?

---

**Design Status**: ✅ Complete  
**Ready for Implementation**: ✅ Yes  
**Estimated Effort**: 3–4 weeks (full stack), 1–2 weeks (Phase 1 only)
