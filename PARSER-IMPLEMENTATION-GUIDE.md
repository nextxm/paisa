# Natural Language Parser – Quick Reference & Implementation Guide

## Quick Overview

**Goal**: Parse free-form text like `"20 Apr, bought 15$ groceries using bmo cc from no frills"` → automatic ledger transaction

**Key Insight**: Reuse existing Paisa infrastructure (TF-IDF account matching, date/amount helpers, transaction sync pipeline)

---

## File Structure

```
internal/parser/                    (NEW - Go parser package)
├── parser.go                       Main parsing logic
├── nlp_patterns.go                 Regex patterns + keyword maps
├── parser_test.go                  Unit tests (10 scenarios)
└── confidence.go                   Confidence scoring

internal/server/parser_handlers.go  (NEW - API endpoints)
├── POST /api/parser/parse
└── POST /api/parser/create-transaction

src/lib/components/
├── NLParser.svelte                 (NEW - Modal component)
└── (Integration with existing transaction UI)
```

---

## Phase 1: Core Parser (Go Implementation)

### Step 1: Create `internal/parser/parser.go`

```go
package parser

import (
    "time"
    "github.com/ananthakumaran/paisa/internal/config"
    "github.com/ananthakumaran/paisa/internal/query"
    "github.com/ananthakumaran/paisa/internal/prediction"
    "gorm.io/gorm"
)

type ParseResult struct {
    Date          time.Time             `json:"date"`
    Payee         string                `json:"payee"`
    Amount        decimal.Decimal       `json:"amount"`
    Currency      string                `json:"currency"`
    FromAccount   string                `json:"from_account"`
    ToAccount     string                `json:"to_account"`
    FromHint      string                `json:"from_hint"`   // debug
    ToHint        string                `json:"to_hint"`     // debug
    Confidence    ConfidenceScores      `json:"confidence"`
    Warnings      []string              `json:"warnings"`
}

type ConfidenceScores struct {
    Overall     float64 `json:"overall"`
    Date        float64 `json:"date"`
    Amount      float64 `json:"amount"`
    FromAccount float64 `json:"from_account"`
    ToAccount   float64 `json:"to_account"`
}

func ParseTransaction(text string, db *gorm.DB) (ParseResult, error) {
    // 1. Normalize
    normalized := normalize(text)
    
    // 2. Extract primitives
    dateResult := extractDate(normalized)
    amountResult := extractAmount(normalized)
    payee := extractPayee(normalized)
    
    // 3. Extract hints
    fromHint := extractFromHint(normalized)
    toHint := extractToHint(normalized)
    
    // 4. Match accounts
    fromAccount, fromConf := matchAccount(db, fromHint, "liability")
    toAccount, toConf := matchAccount(db, toHint, "expense")
    
    // 5. Build result
    result := ParseResult{
        Date:          dateResult.Date,
        Payee:         payee,
        Amount:        amountResult.Amount,
        Currency:      amountResult.Currency,
        FromAccount:   fromAccount,
        ToAccount:     toAccount,
        FromHint:      fromHint,
        ToHint:        toHint,
    }
    
    // 6. Confidence scoring
    result.Confidence = scoreConfidence(result, dateResult, amountResult)
    
    return result, nil
}
```

### Step 2: Create `internal/parser/nlp_patterns.go`

```go
package parser

import "regexp"

var (
    // Date patterns
    DatePatterns = []struct{
        Pattern string
        Layout  string
    }{
        {`(\d{4})-(\d{1,2})-(\d{1,2})`, "2006-01-02"},
        {`(\d{1,2})/(\d{1,2})/(\d{4})`, "02/01/2006"},
        {`(\d{1,2})\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)`, "02 Jan"},
        {`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})`, "Jan 02"},
    }
    
    // Amount patterns: $15, 15$, 15 dollars, 15 USD, 15.99, etc.
    AmountRegex = regexp.MustCompile(
        `\$?(\d+\.?\d*)\$?|\b(\d+\.?\d*)\s*(dollars?|usd|eur|gbp|inr|cad|aud|jpy|₹|€|£|\$)`,
    )
    
    // Preposition keywords
    FromKeywords = []string{"using", "via", "with", "from", "out of"}
    ToKeywords   = []string{"to", "into", "for"}
    PayeeKeywords = map[string]bool{
        "at": true, "from": true, "in": true,
    }
    
    // Account category keywords
    AssetKeywords     = []string{"checking", "savings", "cash", "wallet", "debit", "bank"}
    LiabilityKeywords = []string{"credit", "cc", "visa", "mastercard", "amex", "loan", "debt"}
    ExpenseKeywords   = []string{"groceries", "food", "restaurant", "gas", "shopping", "rent", "bill"}
    IncomeKeywords    = []string{"salary", "income", "bonus", "payment", "wage", "refund"}
)

func normalize(text string) string {
    // lowercase, trim, remove extra spaces
}

func extractDate(text string) DateResult {
    // Try patterns, return first match or default to today
}

func extractAmount(text string) AmountResult {
    // Regex match, extract amount + currency
}

func extractPayee(text string) string {
    // Remove account hints, return residual text as payee
}

func extractFromHint(text string) string {
    // Look for "using X", "from X", "via X" keywords
}

func extractToHint(text string) string {
    // Look for "to X", "into X", "for X" keywords
}

func matchAccount(db *gorm.DB, hint string, preferredType string) (string, float64) {
    // Use TF-IDF with preference bias
    // If no match, fallback to config defaults
}
```

### Step 3: Create `internal/parser/parser_test.go`

```go
package parser

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"
)

func TestParseSimpleExpense(t *testing.T) {
    db := openTestDB(t)
    seedSamplePostings(db)
    
    text := "20 Apr, bought 15$ groceries using bmo cc from no frills"
    result, err := ParseTransaction(text, db)
    
    require.NoError(t, err)
    assert.Equal(t, "2026-04-20", result.Date.Format("2006-01-02"))
    assert.Equal(t, "15.00", result.Amount.String())
    assert.Equal(t, "No Frills", result.Payee)
    assert.True(t, result.FromAccount contains "BMO")
    assert.True(t, result.ToAccount contains "Groceries")
}

func TestParseMissingDate(t *testing.T) {
    db := openTestDB(t)
    text := "bought 25 dollars groceries"
    result, _ := ParseTransaction(text, db)
    
    // Should default to today
    assert.Equal(t, time.Now().Format("2006-01-02"), result.Date.Format("2006-01-02"))
}

func TestParseMultipleAmounts(t *testing.T) {
    db := openTestDB(t)
    text := "on 15 Apr got salary 5000 and spent 100 on lunch"
    result, _ := ParseTransaction(text, db)
    
    // Should use largest amount
    assert.Equal(t, "5000.00", result.Amount.String())
    assert.NotEmpty(t, result.Warnings)
    assert.True(t, contains(result.Warnings, "multiple amounts"))
}

// ... 7 more test cases (scenarios 4-10)
```

---

## Phase 2: API Integration & ML Logging

### Step 4a: Add ML Training Log Table

```go
package server

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/ananthakumaran/paisa/internal/parser"
    "gorm.io/gorm"
)

type ParseRequest struct {
    Text        string `json:"text" binding:"required"`
    PreviewOnly bool   `json:"preview_only"`
}

type ParseResponse struct {
    Success   bool                    `json:"success"`
    Extracted parser.ParseResult      `json:"extracted"`
    Warnings  []string                `json:"warnings"`
    Confidence parser.ConfidenceScores `json:"confidence"`
    Error     string                  `json:"error,omitempty"`
}

func ParseHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ParseRequest
        if !BindJSONOrError(c, &req) {
            return
        }
        
        result, err := parser.ParseTransaction(req.Text, db)
        if err != nil {
            RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
            return
        }
        
        c.JSON(http.StatusOK, ParseResponse{
            Success:    true,
            Extracted:  result,
            Warnings:   result.Warnings,
            Confidence: result.Confidence,
        })
    }
}

func CreateTransactionHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ParseRequest
        if !BindJSONOrError(c, &req) {
            return
        }
        
        result, err := parser.ParseTransaction(req.Text, db)
        if err != nil {
            RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
            return
        }
        
        // Convert to AddTransactionRequest
        addReq := server.AddTransactionRequest{
            Date:       result.Date.Format("2006-01-02"),
            Payee:      result.Payee,
            FromAccount: result.FromAccount,
            ToAccount:  result.ToAccount,
            Amount:     result.Amount.String(),
            Commodity:  result.Currency,
        }
        
        // Use existing handler
        entryText, err := appendTransactionAndSync(db, addReq)
        if err != nil {
            RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
            return
        }
        
        // Log to ML training database (asynchronous)
        // This records: input text, predicted accounts, confidence scores,
        // and actual user-confirmed accounts for future ML training
        go logParsingResult(db, result, result.FromAccount, result.ToAccount)
        
        c.JSON(http.StatusOK, gin.H{
            "success": true,
            "entry":   entryText,
        })
    }
}
```

### Step 5: Register Routes in `internal/server/server.go`

```go
func Build(db *gorm.DB, embedFS bool) *gin.Engine {
    // ... existing code ...
    
    router.POST("/api/parser/parse", func(c *gin.Context) {
        ParseHandler(db)(c)
    })
    
    writeGroup.POST("/api/parser/create-transaction", func(c *gin.Context) {
        CreateTransactionHandler(db)(c)
    })
    
    // ... rest of code ...
}
```

---

## Phase 3: Frontend Component & Interactive Disambiguation

### Step 6: Create `src/lib/components/NLParser.svelte`

```svelte
<script lang="ts">
    import { ajax } from "$lib/utils";
    import * as toast from "bulma-toast";

    let input: string = $state("");
    let preview = $state(null);
    let loading = $state(false);
    let error = $state("");

    async function handleInput() {
        if (!input.trim()) {
            preview = null;
            return;
        }
        
        loading = true;
        error = "";
        
        try {
            const result = await ajax("/api/parser/parse", {
                method: "POST",
                body: JSON.stringify({ text: input }),
                background: true
            });
            
            preview = result;
            
            if (preview.warnings.length > 0) {
                toast.open({ message: preview.warnings[0], type: "is-warning" });
            }
        } catch (e) {
            error = e.message;
            preview = null;
        } finally {
            loading = false;
        }
    }

    async function handleCreate() {
        if (!preview) return;
        
        loading = true;
        try {
            await ajax("/api/parser/create-transaction", {
                method: "POST",
                body: JSON.stringify({ text: input })
            });
            
            toast.open({ message: "✓ Transaction created", type: "is-success" });
            input = "";
            preview = null;
        } catch (e) {
            toast.open({ message: e.message, type: "is-danger" });
        } finally {
            loading = false;
        }
    }
</script>

<div class="nl-parser">
    <textarea
        placeholder="e.g., 20 Apr, bought 15$ groceries using bmo cc from no frills"
        bind:value={input}
        onchange={handleInput}
        disabled={loading}
    />
    
    {#if loading}
        <p>Parsing...</p>
    {/if}
    
    {#if error}
        <div class="error">{error}</div>
    {/if}
    
    {#if preview}
        <div class="preview">
            <div class="field">
                <label>Date</label>
                <span>{preview.extracted.date}</span>
                <progress value={preview.confidence.date * 100} max="100"></progress>
            </div>
            <div class="field">
                <label>Amount</label>
                <span>{preview.extracted.amount} {preview.extracted.currency}</span>
                <progress value={preview.confidence.amount * 100} max="100"></progress>
            </div>
            <div class="field">
                <label>Payee</label>
                <span>{preview.extracted.payee}</span>
            </div>
            <div class="field">
                <label>From Account</label>
                <input type="text" bind:value={preview.extracted.from_account} />
                <progress value={preview.confidence.from_account * 100} max="100"></progress>
            </div>
            <div class="field">
                <label>To Account</label>
                <input type="text" bind:value={preview.extracted.to_account} />
                <progress value={preview.confidence.to_account * 100} max="100"></progress>
            </div>
            
            {#if preview.warnings.length > 0}
                <div class="warnings">
                    {#each preview.warnings as warning}
                        <p class="help is-warning">⚠️ {warning}</p>
                    {/each}
                </div>
            {/if}
            
            <button onclick={handleCreate} disabled={loading}>
                Create Transaction
            </button>
        </div>
    {/if}
</div>

<style>
    textarea {
        width: 100%;
        height: 80px;
    }
</style>
```

---

## Testing Checklist

- [ ] Unit tests pass: `go test internal/parser/...`
- [ ] Parser handles all 10 scenarios correctly
- [ ] API endpoints return proper error codes
- [ ] TF-IDF integration works with live database
- [ ] Regression tests pass: `bun test tests`
- [ ] Frontend renders preview correctly
- [ ] Transaction created matches expected format

---

## Config Extension (Optional for Phase 1)

```yaml
# paisa.yaml
parser_defaults:
  default_account_from: "Assets:Checking"
  default_account_to: "Expenses:Unknown"
  enable_warnings: true
```

In `internal/config/config.go`:
```go
type ParserDefaults struct {
    DefaultAccountFrom string `json:"default_account_from"`
    DefaultAccountTo   string `json:"default_account_to"`
    EnableWarnings     bool   `json:"enable_warnings"`
}

type Config struct {
    // ... existing fields ...
    ParserDefaults ParserDefaults `json:"parser_defaults"`
}
```

---

## Key Reuse Points

| Component | Source | Usage |
|-----------|--------|-------|
| Account matching | `internal/prediction/tf_idf.go` | `matchAccount()` |
| Date parsing | `src/lib/template_helpers.ts` | Ported to Go in `extractDate()` |
| Amount parsing | `src/lib/template_helpers.ts` | Ported to Go in `extractAmount()` |
| Transaction format | `internal/server/add.go` | `AddTransactionHandler` |
| Sync pipeline | `internal/model/model.go` | `SyncJournal()` |

---

## Success Metrics

- ✅ Parse "20 Apr, bought 15$ groceries using bmo cc from no frills" → correct transaction
- ✅ All 10 scenarios covered by tests
- ✅ Confidence scores accurate (>0.8 for high-quality matches)
- ✅ Zero regressions in existing features
- ✅ API response time <500ms
- ✅ User can edit extracted fields before creating

---

## Rollout Plan

**v1.0**: Parser core + API endpoints (Phase 1)  
**v1.1**: Frontend UI + integration (Phase 2)  
**v2.0**: ML improvements, batch processing (Phase 3)  

