---
description: "The Analysis page shows you what securities you own along with their amount and percentage. It also categorizes the securities by their industry and rating."
---

# Analysis

Ledger represents everything except currency as commodities. A
commodity could represent physical object like Gold or certificates
like Stock, Bond etc. Some commodities like Mutual Fund is a container
for Securities. When you own a mutual fund unit, you indirectly own
these securities.

The Analysis page unwraps the container and shows you what securities
you own along with their amount and percentage. It also categorizes
the securities by their industry and rating.

!!! example "Experimental"
    The data that powers this page comes from various sources and might
    not be 100% accurate. Before you make any decision based on this
    information, double check via different source.

## Year-over-Year Comparison

The **Year-over-Year (YoY)** page at `/analysis/yoy` lets you compare
spending and income trends across multiple calendar years side-by-side.

### Configuring the Year Range

Use the year-range selector at the top of the page to choose between
2 and 5 years of history.

### Charts

| Chart | Description |
|-------|-------------|
| **Spending YoY** | Monthly spending per year, Jan–Dec aligned.  Supports line and grouped-bar toggle. |
| **Income YoY** | Monthly income per year, Jan–Dec aligned. |
| **Category YoY** | Per-expense-category monthly totals across selected years (line/bar toggle). |

Hovering over a data point shows a tooltip with the month-level
amounts for all selected years.  A legend identifies each year by
colour.

### YoY Insights

Below the charts, computed insights highlight the year with the
highest/lowest total spend, the month with the greatest variance
across years, and the overall YoY change percentage.

### CSV Export

Click the **Export CSV** button to download the currently displayed
dataset (spending or income, for the selected years and categories) as
a comma-separated file.
