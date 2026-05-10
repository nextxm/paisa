# Natural Language Transaction Parser – Design Complete ✅

## Summary

I've created a **comprehensive design** for a natural language transaction parser feature for Paisa. Users can enter free-form text like:

```
"20 Apr, bought 15$ groceries using bmo cc from no frills"
```

And the system automatically:
1. Extracts date, amount, payee, and account hints
2. Matches accounts using TF-IDF (leverages existing system)
3. Returns confidence scores for each field
4. Creates a ledger transaction with one click

---

## 📚 Documentation Package

I've created **4 comprehensive documents** + **3 architecture diagrams**:

### 📄 Documents (Read in This Order)

1. **[PARSER-QUICK-REFERENCE.md](PARSER-QUICK-REFERENCE.md)** ⭐ START HERE
   - 2-minute overview
   - Quick API examples
   - 10-second feature summary
   - FAQ & quick links

2. **[PARSER-EXECUTIVE-SUMMARY.md](PARSER-EXECUTIVE-SUMMARY.md)**
   - High-level design & benefits
   - 10 realistic scenarios covered
   - Success metrics & risk mitigation
   - Rollout strategy
   - **~5 pages**

3. **[design-natural-language-parser.md](design-natural-language-parser.md)**
   - Complete technical specification
   - Detailed parsing algorithm (8 steps)
   - Full scenario walkthroughs
   - API design with JSON examples
   - Testing strategy & phase breakdown
   - **~15 pages**

4. **[PARSER-IMPLEMENTATION-GUIDE.md](PARSER-IMPLEMENTATION-GUIDE.md)**
   - Step-by-step implementation instructions
   - Code templates for all major components
   - File structure & organization
   - Phase 1–3 tasks with checkboxes
   - Config schema extension
   - **~12 pages**

5. **[PARSER-EXAMPLES.md](PARSER-EXAMPLES.md)**
   - 15 real-world parsing examples
   - Input → extracted output → ledger transaction
   - Confidence score explanations
   - Edge cases & how they're handled
   - Performance targets
   - **~8 pages**

### 📊 Architecture Diagrams (Mermaid)

All embedded in the documents:

1. **System Architecture** – Components & data flow
2. **Sequence Diagram** – API interaction timeline
3. **Extraction Pipeline** – Step-by-step parsing algorithm

---

## 🎯 Key Features

✅ **Parses 10 Different Scenarios**
- Simple expenses
- Transfers
- Multi-currency
- Income/salary
- Investments
- Missing data (uses defaults)
- Ambiguous input (warnings)
- And more...

✅ **Reuses Existing Paisa Infrastructure**
- TF-IDF account matching (proven system)
- Ledger sync pipeline
- Transaction formatting
- Auth/security layers
- **Zero breaking changes**

✅ **Smart Confidence Scoring**
- Per-field confidence (date: 0.95, amount: 0.98, accounts: 0.87, etc.)
- Overall confidence (weighted average)
- Transparent uncertainty handling
- User can preview & edit before committing

✅ **Graceful Degradation**
- Missing date? → Use today
- Missing account? → Use config defaults
- Ambiguous match? → Show warning + alternatives
- No required fields missing → Clear error message

✅ **Production-Ready Design**
- Security: Same auth as existing endpoints
- Rate limiting: Included
- Error handling: Standardized
- Testing: 10 scenarios + edge cases
- Performance: <500ms target

---

## 🏗️ Architecture Summary

```
User Input (natural language)
    ↓
Parser Engine (Go) – Extracts: date, amount, payee, account hints
    ↓
TF-IDF Account Matcher – Matches accounts via cosine similarity (existing system)
    ↓
Config Defaults – Fills missing currency, accounts
    ↓
Confidence Scoring – Per-field scores (0.0–1.0)
    ↓
API Response – { extracted, confidence, warnings }
    ↓
Optional: Frontend Preview – Show results, allow edits
    ↓
Create Transaction – Append to journal, sync to DB
    ↓
✅ Ledger Transaction Created
```

---

## 📋 Implementation Plan

### Phase 1: Parser Core (1–2 weeks)
```go
internal/parser/
├── parser.go              // Main parsing logic
├── nlp_patterns.go        // Regex + keyword maps
├── confidence.go          // Scoring logic
└── parser_test.go         // Unit tests (10 scenarios)
```
- API endpoint: `/api/parser/parse` (preview only)
- ~800 lines of Go code
- Full test coverage
- **Output: 90% accuracy, preview mode**

### Phase 2: Frontend (1 week)
```svelte
src/lib/components/
└── NLParser.svelte        // Modal component
```
- API endpoint: `/api/parser/create-transaction`
- Modal with live preview
- Editable extracted fields
- **Output: Full end-to-end feature**

### Phase 3: Enhancements (Future)
- ML confidence scoring
- Interactive disambiguation
- Batch import
- Custom patterns per user
- Mobile optimization

---

## 🔑 Key Design Decisions

1. **Two-endpoint approach**
   - `/api/parser/parse` – Preview (no side effects)
   - `/api/parser/create-transaction` – Create + sync

2. **Reuse TF-IDF system**
   - Existing account matching proven & tested
   - Leverage `internal/prediction/tf_idf.go`
   - No new dependencies needed

3. **Confidence scoring**
   - Per-field (date, amount, from_account, to_account)
   - Overall weighted average
   - Transparent uncertainty → user edits if needed

4. **Graceful degradation**
   - Config defaults for missing fields
   - Warnings for low-confidence matches
   - Preview + edit before commit

5. **No schema changes**
   - Purely additive feature
   - Uses existing `AddTransactionRequest` structure
   - Zero breaking changes

---

## 🧪 Testing Coverage

**10 Scenarios** ✅
- Simple expense
- Transfer
- Multi-currency
- Missing date
- Multiple amounts
- Explicit accounts
- Income
- Investment
- Minimal input
- With tags

**Edge Cases** ✅
- Empty/blank input
- Malformed dates
- Missing required fields
- Ambiguous accounts
- Unicode characters
- Very old/future dates

**Performance** ✅
- <500ms API response time
- <300ms TF-IDF matching
- <50ms parsing overhead

---

## 📊 Confidence Score Ranges

| Range | Interpretation | Action |
|-------|---|---|
| **0.90–1.00** | Excellent | Auto-create if requested |
| **0.80–0.89** | Good | Preview + create |
| **0.70–0.79** | Fair | Preview + warn + edit |
| **0.60–0.69** | Low | Require confirmation |
| **<0.60** | Very Low | Ask user to clarify |

---

## 💡 Real-World Examples

### Example 1: User's Original Input
```
INPUT: "20 Apr, bought 15$ groceries using bmo cc from no frills"

PARSED:
├─ date: 2026-04-20 (confidence: 0.95)
├─ amount: 15.00 CAD (confidence: 0.98)
├─ payee: No Frills (confidence: 0.90)
├─ from_account: Liabilities:CAD:BMO:CC (confidence: 0.92)
├─ to_account: Expenses:Groceries (confidence: 0.89)
└─ overall_confidence: 0.92 ✅

LEDGER:
2026-04-20 * "No Frills"
  Liabilities:CAD:BMO:CC  -15.00 CAD
  Expenses:Groceries       15.00 CAD
```

### Example 2: Minimal Input
```
INPUT: "100"

PARSED:
├─ date: 2026-05-09 (today, default)
├─ amount: 100.00 (confidence: 0.95)
├─ from_account: Assets:Checking (default)
├─ to_account: Expenses:Unknown (default)
└─ overall_confidence: 0.45 (using defaults)

LEDGER:
2026-05-09 * "Unknown"
  Assets:Checking      -100.00 INR
  Expenses:Unknown     100.00 INR
```

### Example 3: Income
```
INPUT: "received salary 50000 from employer on 1st May"

PARSED:
├─ date: 2026-05-01 (confidence: 0.95)
├─ amount: 50000.00 (confidence: 0.98)
├─ from_account: Income:Salary (confidence: 0.92)
├─ to_account: Assets:Checking (confidence: 0.90)
└─ overall_confidence: 0.91 ✅

LEDGER:
2026-05-01 * "Employer"
  Income:Salary       -50000.00 INR
  Assets:Checking      50000.00 INR
```

See [PARSER-EXAMPLES.md](PARSER-EXAMPLES.md) for 15 complete examples.

---

## ✨ Advantages

**For Users:**
- ⚡ Faster transaction entry (type vs. fill form)
- 🎯 Natural, conversational input
- 🔍 Smart account matching
- ⚙️ Configurable defaults
- 👁️ Preview before commit

**For Paisa:**
- 🔄 Reuses existing TF-IDF
- 🏗️ No schema changes
- 🔐 Same auth/security
- 📊 Confidence scores → future ML
- 🚀 Phased rollout possible

---

## 🚀 Getting Started

### Next Steps:
1. **Review** the 5 documents in order (30 min total)
2. **Discuss** any design trade-offs or concerns
3. **Decide** on Phase 1 scope & timeline
4. **Create** GitHub issues for Phase 1 tasks
5. **Start** implementation with unit tests

### Quick Links:
- **Start here**: [PARSER-QUICK-REFERENCE.md](PARSER-QUICK-REFERENCE.md)
- **Management overview**: [PARSER-EXECUTIVE-SUMMARY.md](PARSER-EXECUTIVE-SUMMARY.md)
- **Technical spec**: [design-natural-language-parser.md](design-natural-language-parser.md)
- **Code guide**: [PARSER-IMPLEMENTATION-GUIDE.md](PARSER-IMPLEMENTATION-GUIDE.md)
- **Examples**: [PARSER-EXAMPLES.md](PARSER-EXAMPLES.md)

---

## 📊 Document Stats

| Document | Pages | Purpose |
|----------|-------|---------|
| PARSER-QUICK-REFERENCE.md | 3 | Quick navigation + API examples |
| PARSER-EXECUTIVE-SUMMARY.md | 5 | High-level overview |
| design-natural-language-parser.md | 15 | Complete specification |
| PARSER-IMPLEMENTATION-GUIDE.md | 12 | Step-by-step code |
| PARSER-EXAMPLES.md | 8 | Real-world examples |
| **Total** | **~43** | **Comprehensive design package** |

---

## ✅ Design Status

| Item | Status |
|------|--------|
| Architecture | ✅ Complete |
| 10 Scenarios | ✅ Covered |
| API Design | ✅ Finalized |
| Edge Cases | ✅ Identified |
| Test Strategy | ✅ Defined |
| Implementation Plan | ✅ Ready |
| Risk Analysis | ✅ Done |
| Rollout Strategy | ✅ Proposed |
| **Overall** | **✅ Ready for Coding** |

---

## 🎯 Success Metrics

- ✅ Parse all 10 scenarios correctly
- ✅ Confidence scores >0.85 accuracy
- ✅ <500ms API response time
- ✅ All regression tests pass
- ✅ Zero breaking changes
- ✅ User can always preview before creating

---

## 📝 Questions?

See [PARSER-EXECUTIVE-SUMMARY.md](PARSER-EXECUTIVE-SUMMARY.md) "Open Questions" section for future enhancements:
- ML-based confidence?
- Voice input support?
- Batch processing?
- Custom regex patterns?

---

## 🙏 Feedback

This design is ready for:
1. ✅ Technical review
2. ✅ Architecture validation
3. ✅ Implementation kickoff
4. ✅ User feedback & iteration

---

**Design Created**: May 9, 2026  
**Status**: ✅ Complete & Ready for Implementation  
**Effort Estimate**: 3–4 weeks (full stack), 1–2 weeks (Phase 1 parser core)  
**Complexity**: Medium (NLP + TF-IDF integration)  
**Risk**: Low (additive feature, no breaking changes)  

---

## 🚀 Let's Build!

Start with [PARSER-QUICK-REFERENCE.md](PARSER-QUICK-REFERENCE.md) → then dive into [design-natural-language-parser.md](design-natural-language-parser.md) for the full picture.

Happy coding! 🎉
