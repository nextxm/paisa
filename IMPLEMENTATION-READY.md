# Natural Language Parser – Implementation Ready ✅

**Date**: May 10, 2026  
**Status**: All planning, design, and scaffolding complete. Ready for code implementation.  
**Timeline**: 3-4 weeks for Phase 1-2 (parser core + frontend)

---

## 📊 What's Been Completed

### ✅ Full Design Specification
- 10 realistic transaction scenarios with full examples
- 8-step parsing pipeline architecture
- Confidence scoring algorithm with thresholds
- ML training data collection strategy
- Interactive UI design for disambiguation
- API endpoint specifications with request/response examples
- Database schema for training logs
- Configuration schema for `paisa.yaml`

### ✅ Code Scaffolding (Ready to Implement)

**Go Parser Package** (`internal/parser/`):
```
parser.go (420 lines)
  ├── ParseTransaction() - Main entry point
  ├── 8-step pipeline (normalizeText, extractDate, extractAmount, etc)
  ├── buildSuggestions() - For low-confidence fields
  └── LogToTrainingDatabase() - For Phase 2 ML logging

keywords.go (180 lines)
  ├── DefaultKeywords() - Built-in keyword set
  ├── LoadKeywordsFromConfig() - Load user customizations
  ├── KeywordScore() - Word boundary matching
  ├── ExtractPaymentMethodHint() - CC vs debit vs cash
  ├── ExtractTransactionTypeHint() - Expense vs income vs transfer
  └── FindCustomPayee() - Match against user config

confidence.go (250 lines)
  ├── ComputeConfidence() - Weighted average scoring
  ├── AccountMatchConfidenceFor() - TF-IDF score conversion
  ├── DateConfidenceFor() - Date extraction confidence
  ├── AmountConfidenceFor() - Amount clarity scoring
  ├── DirectionConfidenceFor() - Transaction type confidence
  ├── ShouldShowSuggestions() - Threshold logic
  └── Threshold constants (auto-create: 0.85, show suggestions: <0.75)

nlp_patterns.go (120 lines)
  ├── CompilePatterns() - Regex patterns for extraction
  ├── DatePatterns - ISO, month names, relative dates
  ├── AmountPatterns - Prefix ($), suffix (USD), words
  ├── KeywordPatterns - Payment methods, transaction types
  └── CurrencyPatterns - Code mappings

parser_test.go (350 lines)
  ├── 10 scenario tests (all from design doc)
  ├── Confidence computation tests
  ├── Keyword matching tests
  ├── Regex pattern tests
  ├── Edge case tests (empty input, long input, no amount)
  └── Benchmark test (<500ms target)
```

**Database Model** (`internal/model/`):
```
parser_training_log.go (220 lines)
  ├── ParserTrainingLog struct
  ├── CreateParserTrainingLog() - Insert logging
  ├── GetParserTrainingLogs() - Retrieve for analysis
  ├── GetCorrectedLogs() - User corrections only
  ├── AnalyzeConfidence() - Accuracy analysis
  ├── PruneOldLogs() - Data retention
  ├── ExportTrainingData() - CSV/JSON export
  └── HidePersonalData() - Anonymization for ML
```

### ✅ Implementation Guides

**7 Documentation Files**:
1. **PHASE-1-ROADMAP.md** (850+ lines) - Step-by-step implementation guide
   - 14 detailed implementation steps
   - Code snippets for each function
   - Testing strategy
   - Quality checklist
   - Deployment plan

2. **PARSER-IMPLEMENTATION-DECISIONS.md** (600+ lines) - Technical architecture
   - NLP library analysis (why stdlib regex)
   - Confidence threshold decisions
   - Keywords configuration
   - Database schema
   - Risk mitigation

3. **design-natural-language-parser.md** (1800+ lines) - Complete specification
   - 15 comprehensive sections
   - 10 scenarios with ledger examples
   - 3 architecture diagrams (Mermaid)
   - API design with JSON examples
   - ML training & interactive UI designs

4. **PARSER-EXECUTIVE-SUMMARY.md** (500+ lines) - Stakeholder overview
5. **PARSER-EXAMPLES.md** (400+ lines) - Real-world examples
6. **PARSER-QUICK-REFERENCE.md** (300+ lines) - Quick lookup
7. **PARSER-DECISIONS-FINAL.md** (400+ lines) - Feature scope & timeline

---

## 🎯 Implementation Path (Ready to Start)

### Phase 1: Core Parser (Weeks 1-2)

**14 Implementation Steps**:
1. Complete `parser.go` 8-step pipeline
2. Complete `keywords.go` functions
3. Complete `confidence.go` functions
4. Complete `nlp_patterns.go` patterns
5. Implement all unit tests
6. Performance optimization
7. Configuration schema update

Each step has:
- ✅ File location
- ✅ Lines of code estimate
- ✅ Testing instructions
- ✅ Code examples in PHASE-1-ROADMAP.md

### Phase 2: API & Frontend (Weeks 3-4, Parallel)

**7 Implementation Steps**:
8. Create API handlers (`parser_handlers.go`)
9. Implement ML logging function
10. Register API routes
11. Create database migration
12. Create Svelte modal component
13. Integration testing
14. Manual E2E testing

---

## 🔑 Technical Decisions (Confirmed)

| Decision | Choice | Why |
|----------|--------|-----|
| **Scope** | Phase 1 + 2 together | Faster MVP, no blocking |
| **NLP Library** | Go stdlib `regexp` | No dependencies, fast, small binary (+0KB) |
| **Auto-Create Threshold** | 0.85 confidence | Balances automation + accuracy |
| **Keywords** | Configurable in `paisa.yaml` | User customization from Day 1 |
| **Batch Processing** | Not for Phase 1-2 | MVP focus: single transaction |
| **ML Training** | ✅ Collect from Day 1 | Enable Phase 3 improvements |
| **Interactive UI** | ✅ Show suggestions <0.75 | Transparent + user control |

---

## 📁 Document Map

**Quick Navigation**:
- **Start here**: `PHASE-1-ROADMAP.md` (14 steps with code)
- **Full spec**: `design-natural-language-parser.md` (15 sections, diagrams)
- **Technical decisions**: `PARSER-IMPLEMENTATION-DECISIONS.md` (tech analysis)
- **Examples**: `PARSER-EXAMPLES.md` (15 real examples)
- **Quick ref**: `PARSER-QUICK-REFERENCE.md` (cheat sheet)
- **Executive summary**: `PARSER-EXECUTIVE-SUMMARY.md` (stakeholder overview)
- **Feature scope**: `PARSER-DECISIONS-FINAL.md` (timeline & scope)

**In Workspace Root**: `/d/Git/paisa/paisa/PARSER-*.md`

---

## ✨ What's New in Codebase

### New Packages
- `internal/parser/` – Complete parser package (4 files + tests)
- `internal/model/parser_training_log.go` – ML training schema

### New Endpoints (Phase 2)
- `POST /api/parser/parse` – Preview only (no side effects)
- `POST /api/parser/create-transaction` – Create + log

### New Config Section (paisa.yaml)
```yaml
parser:
  auto_create_confidence_threshold: 0.85
  show_suggestions_below: 0.75
  max_suggestions: 3
  keywords:
    transaction_markers:
      expense: ["bought", "paid", "spent"]
      income: ["received", "earned"]
      transfer: ["moved", "transferred"]
    custom_payees:
      "Acme Corp": "Expenses:Retail"
  defaults:
    expense_account: "Expenses:Uncategorized"
    from_account: "Assets:Checking"
```

### New Database Table
```sql
parser_training_log (with 20+ columns for prediction + actual + confidence)
```

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- Node.js/Bun
- SQLite3
- Paisa environment setup (`make develop`)

### Start Implementation
1. Open `PHASE-1-ROADMAP.md`
2. Start with Step 1: Complete `internal/parser/parser.go`
3. Follow each step sequentially
4. Run tests after each step
5. Code samples provided in each step

### Build & Test
```bash
# Build parser package
go build ./internal/parser

# Run all tests
go test -v ./internal/parser

# Benchmark
go test -bench=Benchmark ./internal/parser

# Coverage
go test -cover ./internal/parser

# Full system
make test
```

---

## 📊 Code Statistics

| Component | Files | LOC | Status |
|-----------|-------|-----|--------|
| Parser core | 4 | 1,100+ | ✅ Scaffolded |
| Parser tests | 1 | 350+ | ✅ Scenarios ready |
| ML logging | 1 | 220+ | ✅ Schema designed |
| Documentation | 7 | 4,000+ | ✅ Complete |
| **Total** | 13 | **5,670+** | **Ready to code** |

---

## 🎬 Phase 1-2 Timeline

```
Week 1-2: Parser Implementation
  ├── Day 1-2: normalizeText, extractDate, extractAmount
  ├── Day 3: extractPayee, extractHints
  ├── Day 4: matchAccounts (TF-IDF integration)
  ├── Day 5: determineDirection, buildSuggestions
  ├── Day 6-7: Tests, performance tuning
  └── Test: go test ./internal/parser

Week 2-3: Parallel Phase 1 Refinement + Phase 2 Start
  ├── Phase 1: Perf optimization, edge cases
  └── Phase 2:
      ├── API handlers (parse + create)
      ├── ML logging function
      ├── Route registration
      └── Database migration

Week 3-4: Phase 2 Frontend + Integration
  ├── Svelte modal component
  ├── Integration with transaction creation
  ├── Regression tests
  └── Manual E2E testing

Week 4: Shipping
  ├── Final QA
  ├── Documentation
  ├── CHANGELOG update
  └── Release
```

---

## ✅ Quality Standards

### Before Ship
- [ ] All parser functions implemented
- [ ] All 10 scenarios passing tests
- [ ] Performance <500ms p99
- [ ] Coverage >80%
- [ ] No lint errors
- [ ] All regression tests pass
- [ ] API endpoints working
- [ ] ML training logs inserting
- [ ] Svelte component integrated
- [ ] E2E testing complete
- [ ] CHANGELOG updated

### Success Metrics
- Parse accuracy: 100% (10 scenarios)
- Overall confidence: >0.85 auto-create
- TF-IDF match quality: >0.80 avg
- API latency: <500ms p99
- ML data collection: 100% coverage

---

## 🔄 Continuous Development

### Phase 1-2 Dev Workflow
```bash
# Start dev server
make develop

# After code changes
go test ./internal/parser -v
npm run check  # TypeScript, ESLint, Prettier
make lint

# Before commit
go test ./...
bun test tests/regression.test.ts
```

---

## 📞 Reference Materials

All in `/d/Git/paisa/paisa/`:

1. **PHASE-1-ROADMAP.md** – Start here (step-by-step)
2. **design-natural-language-parser.md** – Full spec (architecture + scenarios)
3. **PARSER-IMPLEMENTATION-DECISIONS.md** – Tech decisions (NLP, thresholds, etc)
4. **PARSER-EXAMPLES.md** – Real examples with outputs
5. **PARSER-QUICK-REFERENCE.md** – Quick lookup
6. **.github/copilot-instructions.md** – Paisa architecture overview

---

## 🎉 Summary

**What You're Building**: A natural language transaction parser that converts text like "20 Apr, bought 15$ groceries using bmo cc" into structured ledger entries.

**MVP Features**:
- ✅ 8-step parsing pipeline
- ✅ Confidence scoring for each field
- ✅ Interactive suggestions for low-confidence matches
- ✅ Configurable keywords from `paisa.yaml`
- ✅ ML training data collection
- ✅ Auto-create at >0.85 confidence

**Technology Stack**:
- Go parser core (stdlib regex, no external deps)
- SQLite training log
- Svelte modal UI
- Existing TF-IDF for account matching

**Timeline**: 3-4 weeks (Phase 1-2 in parallel)

**Status**: ✅ All planning done. Code scaffolding complete. Ready to implement.

---

## 🚀 Ready to Start?

1. Read `PHASE-1-ROADMAP.md` (14 implementation steps)
2. Open `internal/parser/parser.go` in editor
3. Start implementing Step 1 functions
4. Run tests: `go test -v ./internal/parser`
5. Follow the roadmap sequentially

**Questions?** Check the reference materials above or the detailed docstrings in the code.

---

**All systems go.** Let's build this! 🚀
