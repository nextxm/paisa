---
description: "How to configure credit cards in Paisa"
---

# Credit Cards

Paisa allows you to track your credit card bills and payments. Let's
say you have a liability account called
`Liabilities:CreditCard:Freedom` to track your real world credit card,
you can configure it in Paisa as follows:

```yaml
credit_cards:
    - account: Liabilities:CreditCard:Freedom #(1)!
      credit_limit: 150000 #(2)!
      statement_end_day: 8 #(3)!
      due_day: 20 #(4)!
      network: visa #(5)!
      number: "0007" #(6)!
      expiration_date: "2029-05-01" #(7)!
```

1. Account name
2. Credit limit of the card
3. The day of the month when the statement is generated
4. The day of the month when the payment is due
5. Card network (`visa`, `mastercard`, `dinersclub`, `amex`, `rupay`, `jcb`, or `discover`)
6. The last 4 digits of the card number
7. The expiration date of the card

Configure cards in **Tools → Configuration → Finance → Credit Cards** (`/more/config/credit-cards`) by clicking the :fontawesome-solid-circle-plus: icon, or add them directly under `credit_cards:` in `paisa.yaml`.

Once configured, your cards appear on **Net Worth → Liabilities & Debt → Credit Cards** (Classic: **Liabilities → Credit Cards**, `/liabilities/credit_cards`), displaying amount due, payment due date, and credit limit utilization. Select any card to open its drill-down page (`/liabilities/credit_cards/[slug]`) for available credit, statement cycle history, bill payment status, and a year-wise spend chart.
