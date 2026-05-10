# Natural Language Parser – Real-World Examples

## Example Collection & Expected Output

This document shows what the parser should produce for various real-world inputs.

---

## 1. Simple Expense (User's Example)

```
INPUT:
20 Apr, bought 15$ groceries using bmo cc from no frills

PARSED OUTPUT:
├─ date: 2026-04-20
├─ amount: 15.00
├─ currency: CAD (inferred)
├─ payee: No Frills
├─ from_hint: "bmo cc"
├─ to_hint: "groceries"
├─ from_account: "Liabilities:CAD:BMO:CC" (92% confidence)
├─ to_account: "Expenses:Groceries" (89% confidence)
└─ overall_confidence: 0.92

LEDGER OUTPUT:
2026-04-20 * "No Frills"
  Liabilities:CAD:BMO:CC  -15.00 CAD
  Expenses:Groceries       15.00 CAD
```

---

## 2. Explicit Currency & Amount

```
INPUT:
paid 100 USD for dinner at Luigi's using amex card

PARSED OUTPUT:
├─ date: 2026-05-09 (today)
├─ amount: 100.00
├─ currency: USD
├─ payee: Luigi's
├─ from_hint: "amex"
├─ to_hint: "dinner"
├─ from_account: "Liabilities:USD:AMEX" (0.88)
├─ to_account: "Expenses:Dining" (0.85)
└─ overall_confidence: 0.88

LEDGER OUTPUT:
2026-05-09 * "Luigi's"
  Liabilities:USD:AMEX  -100.00 USD
  Expenses:Dining        100.00 USD
```

---

## 3. Transfer Between Accounts

```
INPUT:
transferred 500 from checking to savings

PARSED OUTPUT:
├─ date: 2026-05-09
├─ amount: 500.00
├─ currency: INR (config default)
├─ payee: Transfer
├─ from_hint: "checking"
├─ to_hint: "savings"
├─ from_account: "Assets:Checking" (0.95)
├─ to_account: "Assets:Savings" (0.93)
└─ overall_confidence: 0.95

LEDGER OUTPUT:
2026-05-09 * "Transfer"
  Assets:Checking  -500.00 INR
  Assets:Savings    500.00 INR
```

---

## 4. Income / Salary

```
INPUT:
received salary 50000 from employer on 1st May

PARSED OUTPUT:
├─ date: 2026-05-01
├─ amount: 50000.00
├─ currency: INR
├─ payee: Employer
├─ from_hint: "salary" / "employer"
├─ to_hint: (none, inferred as income)
├─ from_account: "Income:Salary" (0.92)
├─ to_account: "Assets:Checking" (0.90)
├─ direction: Income
└─ overall_confidence: 0.91

LEDGER OUTPUT:
2026-05-01 * "Employer"
  Income:Salary       -50000.00 INR
  Assets:Checking      50000.00 INR
```

---

## 5. Investment Purchase

```
INPUT:
bought 10 shares of APPLE at 150 USD each using brokerage

PARSED OUTPUT:
├─ date: 2026-05-09
├─ quantity: 10
├─ commodity: AAPL
├─ unit_price: 150.00
├─ total_amount: 1500.00
├─ currency: USD
├─ payee: APPLE
├─ from_hint: "brokerage"
├─ from_account: "Assets:Brokerage" (0.87)
├─ to_account: "Assets:Investments:AAPL" (0.85)
└─ overall_confidence: 0.88

LEDGER OUTPUT:
2026-05-09 * "APPLE"
  Assets:Brokerage    -1500.00 USD
  Assets:Investments   10 AAPL @ 150.00 USD
```

---

## 6. Multi-Currency Exchange

```
INPUT:
exchanged 100 USD to EUR at rate 1.08

PARSED OUTPUT:
├─ date: 2026-05-09
├─ amount_from: 100.00
├─ amount_to: 108.00
├─ currency_from: USD
├─ currency_to: EUR
├─ exchange_rate: 1.08
├─ from_account: "Assets:Checking:USD" (0.80)
├─ to_account: "Assets:Checking:EUR" (0.78)
└─ overall_confidence: 0.80

LEDGER OUTPUT:
2026-05-09 * "Currency Exchange"
  Assets:Checking:USD  -100.00 USD
  Assets:Checking:EUR   108.00 EUR @ 1.08 EUR/USD
```

---

## 7. Bill Payment (Rent)

```
INPUT:
paid rent 20000 on 1st for landlord

PARSED OUTPUT:
├─ date: 2026-05-01
├─ amount: 20000.00
├─ currency: INR
├─ payee: Landlord
├─ from_hint: (none)
├─ to_hint: "rent"
├─ from_account: "Assets:Checking" (default, 0.70)
├─ to_account: "Expenses:Rent" (0.92)
├─ warnings: ["No payment method specified; using default account"]
└─ overall_confidence: 0.81

LEDGER OUTPUT:
2026-05-01 * "Landlord"
  Assets:Checking     -20000.00 INR
  Expenses:Rent        20000.00 INR
```

---

## 8. Refund / Reversal

```
INPUT:
refund 35 dollars for returned item from amazon

PARSED OUTPUT:
├─ date: 2026-05-09
├─ amount: 35.00 (negative amount detected)
├─ currency: USD
├─ payee: Amazon
├─ type: Refund (inferred)
├─ from_account: "Expenses:Shopping" (0.88)
├─ to_account: "Assets:Checking" (0.90)
└─ overall_confidence: 0.89

LEDGER OUTPUT:
2026-05-09 * "Amazon"
  Expenses:Shopping   -35.00 USD
  Assets:Checking      35.00 USD
```

---

## 9. With Transaction Tags

```
INPUT:
paid insurance 5000 #recurring=monthly #period=1m on 1st may

PARSED OUTPUT:
├─ date: 2026-05-01
├─ amount: 5000.00
├─ currency: INR
├─ payee: Insurance
├─ from_account: "Assets:Checking" (default)
├─ to_account: "Expenses:Insurance" (0.91)
├─ tags: {recurring: "monthly", period: "1m"}
└─ overall_confidence: 0.88

LEDGER OUTPUT:
2026-05-01 * "Insurance"
  ; recurring=monthly, period=1m
  Assets:Checking        -5000.00 INR
  Expenses:Insurance      5000.00 INR
```

---

## 10. Minimal Input

```
INPUT:
100

PARSED OUTPUT:
├─ date: 2026-05-09 (today, default)
├─ amount: 100.00
├─ currency: INR (config default)
├─ payee: Unknown
├─ from_account: "Assets:Checking" (config default)
├─ to_account: "Expenses:Unknown" (config default)
├─ warnings: ["Minimal input; used defaults for all fields"]
└─ overall_confidence: 0.45

LEDGER OUTPUT:
2026-05-09 * "Unknown"
  Assets:Checking      -100.00 INR
  Expenses:Unknown      100.00 INR
```

---

## 11. Ambiguous Amount (⚠️ Warning)

```
INPUT:
on 15 Apr got salary 5000 and spent 100 on lunch

PARSED OUTPUT:
├─ date: 2026-04-15
├─ amount: 5000.00 (largest selected)
├─ currency: INR
├─ payee: Salary
├─ from_account: "Income:Salary" (inferred)
├─ to_account: "Assets:Checking" (inferred)
├─ warnings: ["Multiple amounts detected: 5000, 100. Using largest (5000)."]
└─ overall_confidence: 0.70 (lowered due to ambiguity)

ACTION: Show warning to user, suggest alternatives or split
```

---

## 12. Explicit Account Names (High Precision)

```
INPUT:
moved 1000 from Assets:SBI:Savings to Expenses:Medical

PARSED OUTPUT:
├─ date: 2026-05-09
├─ amount: 1000.00
├─ currency: INR
├─ payee: Transfer
├─ from_hint: "Assets:SBI:Savings"
├─ to_hint: "Expenses:Medical"
├─ from_account: "Assets:SBI:Savings" (exact match, 1.00)
├─ to_account: "Expenses:Medical" (exact match, 1.00)
└─ overall_confidence: 1.00 (perfect)

LEDGER OUTPUT:
2026-05-09 * "Transfer"
  Assets:SBI:Savings   -1000.00 INR
  Expenses:Medical      1000.00 INR
```

---

## 13. Credit Card Statement Entry

```
INPUT:
on 28 Apr swiped visa for 2499 at Amazon.in

PARSED OUTPUT:
├─ date: 2026-04-28
├─ amount: 2499.00
├─ currency: INR
├─ payee: Amazon.in
├─ from_hint: "visa"
├─ to_hint: "shopping" (inferred from Amazon)
├─ from_account: "Liabilities:Visa" (0.94)
├─ to_account: "Expenses:Shopping" (0.91)
└─ overall_confidence: 0.92

LEDGER OUTPUT:
2026-04-28 * "Amazon.in"
  Liabilities:Visa    -2499.00 INR
  Expenses:Shopping    2499.00 INR
```

---

## 14. Investment Dividend

```
INPUT:
received dividend 1200 from ICICI Bank shares on 15 Apr

PARSED OUTPUT:
├─ date: 2026-04-15
├─ amount: 1200.00
├─ currency: INR
├─ payee: ICICI Bank
├─ from_hint: "dividend" / "ICICI Bank"
├─ from_account: "Income:Dividend" (0.93)
├─ to_account: "Assets:Checking" (0.89)
├─ type: Income
└─ overall_confidence: 0.91

LEDGER OUTPUT:
2026-04-15 * "ICICI Bank"
  Income:Dividend      -1200.00 INR
  Assets:Checking       1200.00 INR
```

---

## 15. Loan/Debt Repayment

```
INPUT:
paid 10000 towards education loan from checking account on 10th

PARSED OUTPUT:
├─ date: 2026-05-10
├─ amount: 10000.00
├─ currency: INR
├─ payee: Education Loan
├─ from_hint: "checking"
├─ to_hint: "education loan"
├─ from_account: "Assets:Checking" (0.91)
├─ to_account: "Liabilities:EducationLoan" (0.89)
├─ type: Liability Repayment
└─ overall_confidence: 0.90

LEDGER OUTPUT:
2026-05-10 * "Education Loan"
  Assets:Checking         -10000.00 INR
  Liabilities:EduLoan     10000.00 INR
```

---

## Confidence Score Reference

| Score Range | Interpretation | Action |
|-------------|-----------------|--------|
| 0.90–1.00 | Excellent | Auto-create (if requested) |
| 0.80–0.89 | Good | Preview + create |
| 0.70–0.79 | Fair | Preview + warn + allow edit |
| 0.60–0.69 | Low | Require confirmation |
| <0.60 | Very Low | Ask user to clarify |

---

## Common Extraction Challenges

### Challenge 1: Ambiguous Merchant Name
```
INPUT: "paid 500 to Reliance"
ISSUE: "Reliance" could be gas, electricity, retail, etc.
SOLUTION: Use largest category match; offer alternatives
CONFIDENCE: 0.65 (lower than normal)
```

### Challenge 2: Missing Key Field
```
INPUT: "spent on groceries"
ISSUE: Amount missing (required)
SOLUTION: Return error; ask user to provide amount
```

### Challenge 3: Unclear Account
```
INPUT: "transferred to SBI account"
ISSUE: Multiple SBI accounts might exist
SOLUTION: Show list of matches; ask user to pick one
```

### Challenge 4: Date in Different Format
```
INPUT: "5/15/2026"
ISSUE: Ambiguous: May 15 or 15 May?
SOLUTION: Check user's locale; use config timezone
CONFIDENCE: 0.80 (slightly lower due to ambiguity)
```

### Challenge 5: Sentence with Multiple Transactions
```
INPUT: "bought groceries for 300 and gas for 400"
ISSUE: Two transactions in one input
SOLUTION: Detect, split, and create two transactions or warn
```

---

## Edge Cases Handled

| Edge Case | Handling |
|-----------|----------|
| Empty input | Return error: "Input cannot be empty" |
| Only spaces | Return error: "Input is blank" |
| Huge amounts (9999999) | Accept; add note if >typical range |
| Negative amounts | Accept; interpret as opposite direction |
| Future dates | Accept; warn if >30 days in future |
| Very old dates | Accept; warn if >2 years old |
| Unicode payees (e.g., "café") | Fully supported |
| Currency mismatch | Warn if different from config default |
| Missing punctuation | Normalize; parse anyway |

---

## Performance Targets

| Operation | Target Time |
|-----------|-------------|
| Normalize & tokenize | <5ms |
| Extract primitives | <10ms |
| TF-IDF matching | <300ms (depends on DB size) |
| Build response | <10ms |
| **Total API response** | **<500ms** |

---

## Future Enhancement Examples

### Voice Input
```
VOICE: "20 April, bought 15 dollars groceries using BMO credit card from No Frills"
→ [Transcribe to text]
→ [Parse as normal]
→ [Create transaction]
```

### Batch Processing
```
INPUT FILE:
Line 1: 20 Apr, bought 15$ groceries using cc from no frills
Line 2: 25 Apr, paid 500 rent to landlord
Line 3: 30 Apr, salary 50000 from employer

→ [Parse 3 transactions]
→ [Show preview of all 3]
→ [Create all 3 in one operation]
```

### Context-Aware Suggestions
```
PREVIOUS: "bought groceries from No Frills on 20 Apr"
INPUT: "same store, 25$"

SUGGESTION:
Use same account pair as previous transaction?
→ Yes / No / Edit
```

