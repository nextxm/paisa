# Natural Language Parser – Design Decisions Finalized ✅

**Date**: May 10, 2026  
**Status**: Design complete, ready for Phase 1 implementation  

---

## Confirmed Implementation Decisions

### ✅ Features to Include in MVP

#### 1. **Store Parsing Confidence Scores for ML Training**
- **Decision**: YES
- **Implementation**: 
  - Create `parser_training_log` table in SQLite
  - Log on every successful parse + user confirmation
  - Store: input text, predicted accounts, confidence scores, actual user-confirmed accounts
  - Asynchronous logging (non-blocking)
- **Benefits**: 
  - Enables future ML model improvements (Phase 3)
  - Measure confidence accuracy over time
  - Track which hints lead to correct matches
- **Privacy**: Local database only, no cloud sync

#### 2. **Interactive Disambiguation UI for Low Confidence Matches**
- **Decision**: YES
- **Implementation**:
  - When confidence <0.75, show top 3 alternative account matches
  - User can click to select different option
  - Selection updates preview immediately
  - User selection recorded for ML training
- **Benefits**:
  - Transparent uncertainty handling
  - User always has final say
  - Captures user intent for training data
  - Reduces friction (no need to type full account name)
- **UX Mockup**:
  ```
  From Account: Liabilities:BMO:XYZ ⚠️ (65% confidence)
  
  Suggestions:
    1. ☑️ Liabilities:CAD:BMO:CC (88% match)
    2. ○ Liabilities:BMO:Debit (65% match)
    3. ○ Liabilities:BMO:Personal (58% match)
  ```

---

### ❌ Features Deferred (Not MVP)

#### 1. **Multi-Language Support** (e.g., "20 Avr" for French)
- **Decision**: NOT for Phase 1-2
- **Reasoning**: Start with English; expand if user base grows
- **Timeline**: Consider for Phase 3+ if requested

#### 2. **Voice Input → Text → Parsing**
- **Decision**: NOT for Phase 1-2
- **Reasoning**: Adds complexity; text parsing is already powerful
- **Timeline**: Consider future integration with speech-to-text APIs

#### 3. **Batch Import** (CSV, Slack, Telegram, message logs)
- **Decision**: NOT for Phase 1-2
- **Reasoning**: Single transaction parsing solves primary use case
- **Timeline**: Can be added in Phase 3 if user demand exists

---

## Phase Breakdown

### Phase 1: Parser Core (1-2 weeks)
**Output**: Parser API, ~90% accuracy, confidence scoring

**Tasks**:
- [ ] Create `internal/parser/` package
- [ ] Implement 8-step extraction pipeline
- [ ] TF-IDF account matching integration
- [ ] Confidence scoring logic
- [ ] `POST /api/parser/parse` endpoint (preview only)
- [ ] Unit tests (all 10 scenarios)
- [ ] Edge case handling
- [ ] Performance optimization (<500ms)

**Deliverable**: Working parser API users can test

---

### Phase 2: Frontend & ML Logging (1 week)
**Output**: Complete feature with UI, ML training data collection

**Tasks**:
- [ ] Create `parser_training_log` database table
- [ ] Implement ML logging in CreateTransactionHandler
- [ ] Build Svelte modal component
- [ ] Implement interactive disambiguation UI
- [ ] Display confidence indicators + suggestions
- [ ] `POST /api/parser/create-transaction` endpoint
- [ ] Integration with existing transaction UI
- [ ] Regression tests
- [ ] UX testing

**Deliverable**: Production-ready feature with full UI

---

### Phase 3: ML Improvements & Enhancements (Future)
**Output**: Smarter matching, extended capabilities

**Potential Tasks**:
- [ ] Analyze stored confidence scores + user corrections
- [ ] Train supervised ML model
- [ ] A/B test new models vs. TF-IDF baseline
- [ ] Add multi-language date parsing (if demand)
- [ ] Batch transaction import (if demand)
- [ ] Advanced confidence scoring

**Trigger**: After 1-2 months of production usage + collected training data

---

## Impact on Design Documents

### Updated Sections:
1. **design-natural-language-parser.md** (PRIMARY)
   - Added Section 10: ML Training Data Collection
   - Added Section 11: Interactive Disambiguation UI
   - Updated Section 14: Confirmed Design Decisions
   - Updated Phase 2 tasks to include ML + UI

2. **PARSER-IMPLEMENTATION-GUIDE.md**
   - Phase 2 renamed to "API Integration & ML Logging"
   - Phase 3 renamed to "Frontend Component & Interactive Disambiguation"
   - Added ML logging code examples
   - Added suggestions API response structure

3. **PARSER-EXECUTIVE-SUMMARY.md**
   - Phase breakdown updated with ML logging

4. **PARSER-EXAMPLES.md**
   - Add example responses showing suggestions field (for Phase 2)

---

## Key Metrics & Success Criteria

### Performance Targets
- API response time: <500ms
- ML training data quality: 100% coverage
- Confidence score accuracy: >0.85 correlation with actual match quality

### Feature Completeness
- ✅ Parse all 10 scenarios correctly
- ✅ Interactive disambiguation UI
- ✅ ML training data collection
- ✅ Confidence indicators in UI
- ✅ Zero breaking changes

### MVP Requirements
- Parse 90%+ accuracy on test set
- Confidence scoring works reliably
- UI provides clear feedback
- ML data being collected for Phase 3

---

## Timeline

| Phase | Duration | Start | End | Deliverable |
|-------|----------|-------|-----|-------------|
| Phase 1 | 1-2 weeks | Week 1 | Week 2-3 | Parser API |
| Phase 2 | 1 week | Week 3-4 | Week 4 | Full Feature + UI |
| Phase 3 | Future | Month 2+ | TBD | ML Improvements |

**Total MVP**: 2-4 weeks  
**Full Feature with UI**: 3-4 weeks

---

## What Changed from Initial Design

### Additions
1. **ML Training Data Collection**
   - New `parser_training_log` table
   - Automatic logging on every transaction created
   - Planned use for Phase 3 improvements

2. **Interactive Disambiguation UI**
   - API returns suggestions for low-confidence matches
   - Frontend shows clickable alternatives
   - User selections contribute to training data

### Deferred (Was questioned)
- Custom regex patterns per user → Not MVP
- Multi-language → Future consideration
- Voice input → Future consideration
- Batch import → Future consideration

---

## Risk Assessment

| Risk | Probability | Mitigation |
|------|-------------|-----------|
| Low ML data quality | Low | Phase 3 can validate and filter |
| User confusion with suggestions | Low | Clear UI labeling + confidence %s |
| Database growth (training log) | Low | Can be pruned/archived if needed |
| Performance impact | Low | Async logging, no blocking |

---

## Next Steps

1. ✅ **Design finalized** (all decisions made)
2. 📅 **Create GitHub issues** for Phase 1 tasks
3. 🚀 **Start Phase 1 implementation** (parser core)
4. 🧪 **Build unit tests** in parallel
5. 📊 **Measure confidence** quality during Phase 1
6. 🎨 **Design Phase 2 UI** during Phase 1
7. 🚢 **Ship Phase 1** for early testing
8. 🔧 **Phase 2 implementation** (UI + ML logging)
9. 📤 **GA release** with full feature

---

## Questions Answered

| Question | Answer |
|----------|--------|
| Store confidence scores for ML? | ✅ YES |
| Multi-language support? | ❌ Not for MVP |
| Custom regex patterns? | ❌ Not for MVP |
| Voice input? | ❌ Not for MVP |
| Batch import? | ❌ Not for MVP |
| Interactive disambiguation? | ✅ YES |

---

## Reference Documents

- [design-natural-language-parser.md](design-natural-language-parser.md) – Full spec (updated)
- [PARSER-EXECUTIVE-SUMMARY.md](PARSER-EXECUTIVE-SUMMARY.md) – Overview
- [PARSER-IMPLEMENTATION-GUIDE.md](PARSER-IMPLEMENTATION-GUIDE.md) – Code guide (updated)
- [PARSER-EXAMPLES.md](PARSER-EXAMPLES.md) – Examples
- [PARSER-QUICK-REFERENCE.md](PARSER-QUICK-REFERENCE.md) – Quick ref

---

**Design Status**: ✅ **FINALIZED**  
**Ready for Implementation**: ✅ **YES**  
**Implementation Can Start**: ✅ **IMMEDIATELY**  

