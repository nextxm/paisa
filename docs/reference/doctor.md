---
description: "Data Health (Doctor) diagnoses accounting discrepancies, missing prices, balance anomalies, and duplicate entries in your ledger"
---

# Data Health (Doctor)

The **Data Health** system (also known as **Doctor**) continuously evaluates your ledger journal for accounting discrepancies, missing market prices, allocation gaps, and potential anomalies. It categorizes findings by severity, explains why each matters, and links directly to the ledger editor or configuration page for triage.

Access Data Health via **Tools → Data Health** (`/more/doctor`) in Grouped navigation (or **More → Data Health** in Classic navigation), or via the numeric badge shown next to **Data Health** when active findings exist (`/more/doctor-v2` and `/more/doctor-v3` redirect to `/more/doctor`).

---

## 1. Overview & Health Summary

At the top of the Data Health screen, the **Health Summary** card summarizes ledger hygiene metrics:

- **Fix Now (`fix_count`)**: Severe accounting discrepancies (e.g., negative cash/bank balances, unexpected credit/debit postings).
- **To Review (`review_count + info_count`)**: Potential errors or informational gaps requiring review (missing exchange prices, unit price mismatches, unmapped asset allocation accounts, potential duplicate transactions, statistical outliers).
- **Dismissed (`dismissed_count`)**: Findings intentionally suppressed with an optional reason note.

Segmented filter buttons let you switch between **Fix Now**, **To Review**, **Dismissed**, and **All Items**, paired with a real-time search box to filter by account name, payee, or problem description.

---

## 2. Automated Diagnostic Rules

Paisa evaluates automated audits across eight rule categories:

| Rule ID | Name | Severity | Problem Description | Recommended Fix |
| :--- | :--- | :--- | :--- | :--- |
| `negative_balance` | **Negative Asset Balance** | `Fix` | An asset account running balance dipped below zero at some point in time. | Correct preceding ledger entries or add opening balances. |
| `credit_entry` *(config: `non_credit_account`)* | **Unexpected Debit in Income Account** | `Fix` | An income account (excluding `Income:CapitalGains:%`) has an unexpected positive/debit entry (`> 0.01`). | Check the posting sign (`Income:*` credits should be negative). |
| `debit_entry` *(config: `non_debit_account`)* | **Unexpected Credit in Expense Account** | `Fix` | An expense account has an unexpected negative/credit entry (`< -0.01`). | Check the posting sign (`Expenses:*` debits should be positive). |
| `exchange_price_missing` | **Missing Commodity Price** | `Review` | A non-currency commodity lacks a conversion price to `default_currency`. | Add price directives or configure a price provider for the commodity. |
| `unit_price_mismatch` | **Unit Price Mismatch** | `Review` | A recorded purchase/sale price differs significantly from the external market reference rate for that date. | Verify units and total cost against trade confirmations. |
| `asset_allocation_missing` | **Asset Accounts Missing Allocation Target** | `Info` | Active asset accounts are not matched by any configured `allocation_targets` rule. | Map the unassigned accounts under **Net Worth → Target Asset Allocation**. |
| `duplicate_transaction` | **Potential Duplicate Transactions** | `Review` | Two postings on the same account with identical rounded default-currency amount occurred within 48 hours of each other. | Inspect both entries in the editor; delete the duplicate or dismiss if legitimate. |
| `outlier_transaction` | **Statistical Outlier Postings** | `Review` | On accounts with at least 5 default-currency postings, an entry deviates by $\ge 3\sigma$ from the account mean. | Verify whether the amount has a typo/misplaced decimal or is a valid one-off expense. |

---

## 3. Triage & Direct Actions

Each finding card provides structured evidence and actions:

### Contextual Evidence
- **Posting Details**: Date, Payee, Account, and Amount.
- **Why It Matters**: Explanation of how the issue affects balances, reports, or projections.
- **Direct Deep-Link**: Clicking an evidence entry or **Open in Editor** opens the exact file and line number in the integrated **Ledger Editor** (`/ledger/editor/<filename>#<line>`).

### Dismiss & Memory
If an alert is intentional (such as an annual insurance premium flagged as an outlier, or two valid identical charges within 48 hours):

1. Click **Dismiss** on the finding card and optionally enter a rationale note.
2. Dismissals are stored via `POST /api/doctor/dismiss` (and duplicate dismissals automatically sync with `duplicate_suppressions` so they stay suppressed).
3. Switch to the **Dismissed** tab and click **Undismiss** (`POST /api/doctor/undismiss`) to restore a finding at any time.

---

## 4. API & Configuration Reference (`paisa.yaml`)

Doctor exposes `GET /api/doctor/findings`, `GET /api/doctor/dismissed`, `POST /api/doctor/dismiss`, and `POST /api/doctor/undismiss`.

Diagnostic rules can be enabled, disabled, or scoped in `paisa.yaml` under `doctor:` or in the UI under **Tools → Configuration → Tools → Doctor Rules** (`/more/config/doctor`):

```yaml
doctor:
  negative_balance:
    enabled: "yes"
    pattern:
      - "Assets:%"

  non_credit_account:
    enabled: "yes"
    pattern:
      - "Income:%"

  non_debit_account:
    enabled: "yes"
    pattern:
      - "Expenses:%"

  exchange_price_missing:
    enabled: "yes"

  unit_price_mismatch:
    enabled: "yes"

  asset_allocation_missing:
    enabled: "yes"
    pattern:
      - "Assets:%"
```

- **`enabled`** *(string enum: `"yes"` or `"no"`)*: Toggle rule evaluation on or off.
- **`pattern`** *(array of strings)*: SQL `LIKE` patterns (`%` wildcard, e.g., `Assets:%`, `Income:%`, `Expenses:%`) specifying which accounts the check evaluates.
