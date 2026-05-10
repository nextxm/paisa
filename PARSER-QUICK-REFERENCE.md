# Natural Language Parser – Quick Reference Card

## 📋 Document Navigation

### Start Here
👉 **[PARSER-EXECUTIVE-SUMMARY.md](PARSER-EXECUTIVE-SUMMARY.md)** – 5 min read  
High-level overview, key benefits, success metrics

### Comprehensive Design
📖 **[design-natural-language-parser.md](design-natural-language-parser.md)** – 30 min read  
Full specification with architecture, 10 scenarios, phased rollout

### Implementation Details
💻 **[PARSER-IMPLEMENTATION-GUIDE.md](PARSER-IMPLEMENTATION-GUIDE.md)** – 20 min read  
Step-by-step code guide, file structure, Phase 1–3 tasks

### Real-World Examples
🔍 **[PARSER-EXAMPLES.md](PARSER-EXAMPLES.md)** – 15 min read  
15 parsing examples with expected output & confidence scores

---

## 🚀 Feature in 30 Seconds

**User types**:
```
20 Apr, bought 15$ groceries using bmo cc from no frills
```

**System returns**:
```json
{
  "extracted": {
    "date": "2026-04-20",
    "amount": "15.00",
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries"
  },
  "confidence": 0.92
}
```

**User confirms** → **Ledger transaction created** ✅

---

## 🏗️ Architecture Overview

```
Text Input
  ↓
Parser (Go)  ← extracts date, amount, payee, account hints
  ↓
TF-IDF Matcher ← matches accounts using cosine similarity
  ↓
Confidence Scoring ← per-field confidence (0.0–1.0)
  ↓
API Response ← with extracted fields + warnings
  ↓
Frontend Preview (optional) ← user can edit before commit
  ↓
Create Transaction ← append to journal + sync
```

---

## 📊 Coverage: 10 Scenarios

| # | Scenario | Status |
|---|----------|--------|
| 1 | Simple expense | ✅ |
| 2 | Transfer between accounts | ✅ |
| 3 | Multi-currency exchange | ✅ |
| 4 | Missing date (defaults to today) | ✅ |
| 5 | Multiple amounts (warning) | ✅ |
| 6 | Explicit account names (high precision) | ✅ |
| 7 | Income transaction | ✅ |
| 8 | Investment purchase | ✅ |
| 9 | Minimal input (all defaults) | ✅ |
| 10 | With transaction tags | ✅ |

---

## 🔌 API Endpoints

### POST /api/parser/parse
**Preview mode** – Parse without creating  
```bash
curl -X POST http://localhost:7500/api/parser/parse \
  -H "X-Auth: <token>" \
  -H "Content-Type: application/json" \
  -d '{"text": "20 Apr, bought 15$ groceries using bmo cc"}'
```

**Response** (excerpt):
```json
{
  "success": true,
  "extracted": {
    "date": "2026-04-20",
    "amount": "15.00",
    "currency": "CAD",
    "payee": "No Frills",
    "from_account": "Liabilities:CAD:BMO:CC",
    "to_account": "Expenses:Groceries"
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

### POST /api/parser/create-transaction
**Create mode** – Parse + create + sync  
```bash
curl -X POST http://localhost:7500/api/parser/create-transaction \
  -H "X-Auth: <token>" \
  -H "Content-Type: application/json" \
  -d '{"text": "20 Apr, bought 15$ groceries using bmo cc"}'
```

**Response**:
```json
{
  "success": true,
  "entry": "2026-04-20 * \"No Frills\"\n  Liabilities:CAD:BMO:CC  -15.00 CAD\n  Expenses:Groceries      15.00 CAD"
}
```

---

## 🔑 Key Extraction Steps

1. **Normalize**: lowercase, trim, standardize symbols
2. **Date**: regex patterns + defaults to today
3. **Amount**: currency symbols, decimal, words (dollars, USD, etc.)
4. **Payee**: quoted text, location keywords, residual
5. **Hints**: "using X" → from_account, "for Y" → to_account
6. **Match**: TF-IDF cosine similarity on stored postings
7. **Infer**: direction (expense, income, transfer)
8. **Score**: confidence 0.0–1.0 per field

---

## ⚙️ Configuration

Optional extension to `paisa.yaml`:

```yaml
parser_defaults:
  default_account_from: "Assets:Checking"
  default_account_to: "Expenses:Unknown"
  enable_warnings: true
  min_confidence_threshold: 0.75
```

If not specified, hardcoded defaults are used.

---

## 📈 Confidence Scores

| Range | Interpretation | Action |
|-------|-----------------|--------|
| 0.90–1.00 | Excellent | Auto-create if user requests |
| 0.80–0.89 | Good | Preview + create immediately |
| 0.70–0.79 | Fair | Preview + warn + allow edit |
| 0.60–0.69 | Low | Require explicit confirmation |
| <0.60 | Very Low | Ask user to clarify/fix |

---

## 📁 File Structure (After Implementation)

```
internal/parser/                       (NEW)
├── parser.go                          Main parsing logic (500 lines)
├── nlp_patterns.go                    Regex + keywords (200 lines)
├── confidence.go                      Scoring logic (100 lines)
└── parser_test.go                     Unit tests (300 lines)

internal/server/
└── parser_handlers.go                 (NEW) API endpoints (100 lines)

src/lib/components/
└── NLParser.svelte                    (NEW, Phase 2) Modal component (200 lines)

tests/
└── regression.test.ts                 (updated) Parser endpoint tests
```

---

## 🧪 Testing Checklist

- [ ] Unit tests pass: `go test internal/parser/...`
- [ ] All 10 scenarios parse correctly
- [ ] Confidence scores are accurate (>0.85 correlation)
- [ ] TF-IDF integration works with live DB
- [ ] API response time <500ms
- [ ] Regression tests pass: `bun test tests`
- [ ] Frontend UI renders correctly (Phase 2)
- [ ] Zero breaking changes to existing API

---

## 📅 Implementation Timeline

**Phase 1** (1–2 weeks): Parser core
- ✅ `internal/parser/` package
- ✅ API endpoint: `/api/parser/parse`
- ✅ Unit tests (10 scenarios)
- 🎯 Deliverable: ~90% accuracy, preview-only mode

**Phase 2** (1 week): Frontend + Create
- ✅ Svelte modal component
- ✅ API endpoint: `/api/parser/create-transaction`
- ✅ Integration with transaction UI
- 🎯 Deliverable: Full end-to-end flow

**Phase 3** (Future): Enhancements
- ML confidence scoring
- Interactive disambiguation
- Batch import
- Mobile UI optimization

---

## 🎯 Success Criteria

- ✅ Parse all 10 scenarios correctly
- ✅ Confidence scores reliable (>0.8 correlation with accuracy)
- ✅ <500ms API response time
- ✅ All regression tests pass
- ✅ Zero breaking changes
- ✅ Clear user feedback (confidence + warnings)
- ✅ Editable preview before commit

---

## ❓ FAQ

**Q: Will this break existing transactions?**  
A: No. This is purely additive. Existing `/api/add/transaction` endpoint unchanged.

**Q: What if the parser guesses wrong?**  
A: User sees preview with confidence scores and can edit any field before confirming.

**Q: Does it require external NLP libraries?**  
A: No. Uses only regex patterns + existing TF-IDF system.

**Q: Can users customize parsing rules?**  
A: Phase 1: No (hardcoded patterns). Phase 3: Yes (via config).

**Q: Does it work offline?**  
A: Yes. Parsing is local; TF-IDF index is built from in-memory postings.

**Q: Mobile-friendly?**  
A: Phase 1: Yes (text input works). Phase 2+: Optional UI optimization.

---

## 🔗 Related Systems

| System | Integration Point |
|--------|-------------------|
| TF-IDF Account Matching | `internal/prediction/tf_idf.go` |
| Transaction Sync | `model.SyncJournal()` |
| Ledger CLI | Via existing `AddTransactionHandler` |
| Authentication | Standard `X-Auth` header |
| Config System | `paisa.yaml` + JSON schema |

---

## 📞 Questions & Decisions

Before starting Phase 1:

1. **Scope**: Core parser only, or include Phase 2 UI?
2. **NLP Library**: Use Go standard library regex or add external lib?
3. **Confidence Threshold**: Auto-create at >0.85 or require explicit confirmation?
4. **Keywords**: Hardcode keyword maps or allow user customization (Phase 3)?
5. **Batch Processing**: Support multiple transactions in one input (Phase 3)?

---

## 📝 Document Index

| File | Purpose |
|------|---------|
| [PARSER-EXECUTIVE-SUMMARY.md](#) | Overview + metrics |
| [design-natural-language-parser.md](#) | Complete spec |
| [PARSER-IMPLEMENTATION-GUIDE.md](#) | Code templates |
| [PARSER-EXAMPLES.md](#) | 15 real examples |
| [PARSER-QUICK-REFERENCE.md](#) | This file |

---

## ✨ Status

**Design Phase**: ✅ Complete  
**Ready for Coding**: ✅ Yes  
**Estimated Effort**: 3–4 weeks full stack, 1–2 weeks Phase 1  
**Complexity**: Medium (NLP + TF-IDF integration)  
**Risk Level**: Low (additive feature, no schema changes)  

---

**Last Updated**: May 9, 2026  
**Design Owner**: Team  
**Status**: Ready for Implementation  
