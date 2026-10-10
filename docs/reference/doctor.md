---
description: "Data Health (Doctor) diagnoses accounting discrepancies, missing prices, balance anomalies, and duplicate entries in your ledger"
---

# Data Health (Doctor)

The **Data Health** system (also known as **Doctor**) continuously evaluates your ledger journal for accounting discrepancies, data quality issues, and potential anomalies. It automatically categorizes issues, quantifies their severity, explains why they matter, and provides direct links into the ledger editor or relevant configuration pages for swift triage.

Access Data Health via **More → Data Health** (`/more/doctor`) or by clicking the health status indicator badge in the top navigation bar.

---

## 1. Overview & Health Summary

At the top of the Data Health screen, the **Health Summary** card displays current ledger hygiene metrics:

* **Needs Fixing (Critical / Fix)**: Severe accounting discrepancies that violate fundamental rules (e.g. negative cash/bank balances, improper credit/debit postings).
* **To Review (Review & Info)**: Potential errors or anomalies requiring human inspection (e.g. statistical spending outliers, potential duplicate transactions, missing market price mappings).
* **Dismissed**: Findings intentionally suppressed with an optional reason note.

Tabbed filters let you toggle between **Fix**, **Review**, **Dismissed**, and **All**, alongside a real-time text filter to search issues by account name, payee, or problem description.

---

## 2. Automated Diagnostic Rules

Paisa's engine runs automated audits across eight distinct categories:

| Rule ID | Name | Severity | Problem Description | Recommended Fix |
| :--- | :--- | :--- | :--- | :--- |
| `negative_balance` | **Negative Asset Balance** | `Fix` | An asset account running balance dipped below zero at some point in time. | Correct the preceding ledger entries or adjust opening balances. |
| `non_credit_account` | **Credit in Debit-Only Account** | `Fix` | Unexpected negative/credit postings in asset/expense accounts that distort balances. | Inspect the transaction sign and flip credit/debit legs as appropriate. |
| `non_debit_account` | **Debit in Credit-Only Account** | `Fix` | Unexpected positive/debit postings in income or liability accounts. | Review transaction sign conventions. |
| `exchange_price_missing` | **Missing Price Directives** | `Review` | A multi-commodity transaction lacks conversion rates or historical price directives (`P`). | Add explicit exchange prices or configure a commodity price provider. |
| `unit_price_mismatch` | **Commodity Price Mismatch** | `Review` | Recorded purchase/sale price differs drastically from the market reference rate fetched for that date. | Verify unit counts and total amounts against statement trade confirms. |
| `asset_allocation_missing` | **Asset Accounts Missing Allocation Target** | `Info` | Active asset accounts are not mapped to any configured target allocation bucket. | Map the unassigned account(s) under **Assets → Allocation**. |
| `duplicate` | **Potential Duplicate Transactions** | `Review` | Two postings with matching amounts, commodities, and accounts occurred within 2 days of each other. | Inspect both entries in the editor; delete duplicate or dismiss if legitimate. |
| `outlier` | **Statistical Outlier Postings** | `Review` | An entry amount deviates significantly ($> 3\sigma$) above the historical average for that account. | Confirm whether the amount has a misplaced decimal or represents a valid one-off transaction. |

---

## 3. Triage & Direct Actions

Each finding card provides structured contextual evidence and actionable workflows:

### Contextual Evidence
* **Posting Details**: Date, Payee, Account, and exact numerical Amount.
* **Why It Matters**: Concise financial explanation of how this entry affects your statements, reports, and tax calculations.
* **Direct Deep-Link**: Clicking an evidence entry or the **Open in Editor** action navigates directly to the exact file and line number in the integrated **Ledger Editor** (`/ledger/editor/<filename>#<line>`).

### Dismiss & Memory
If an alert is an intentional transaction (e.g., a planned large annual tuition payment flagged as an outlier, or two valid identical subscriptions on the same date):
* Click **Dismiss** on the finding card.
* Enter an optional note explaining the rationale (e.g., "Annual insurance premium, verified").
* The finding moves to the **Dismissed** tab and will no longer count against the active badge count.
* You can revert dismissals at any time from the **Dismissed** tab using **Undismiss**.

---

## 4. Configuration Reference (`paisa.yaml`)

Doctor diagnostic rules can be selectively enabled, disabled, or tuned in `paisa.yaml` under the `doctor` key or through **Configuration → Tools → Doctor Rules**:

```yaml
doctor:
  negative_balance:
    enabled: true
    pattern:
      - "Assets:*"

  non_credit_account:
    enabled: true
    pattern:
      - "Expenses:*"
      - "Assets:*"

  non_debit_account:
    enabled: true
    pattern:
      - "Income:*"
      - "Liabilities:*"

  exchange_price_missing:
    enabled: true

  unit_price_mismatch:
    enabled: true

  asset_allocation_missing:
    enabled: true
```

* **`enabled`** *(boolean)*: Toggle the rule evaluation on or off.
* **`pattern`** *(array of strings)*: Glob patterns specifying which accounts the check applies to.
