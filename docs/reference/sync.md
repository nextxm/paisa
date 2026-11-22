---
description: "Synchronizing journals, market prices, portfolios, and external integrations in Paisa"
---

# Data Synchronization

Paisa keeps its SQLite read model synchronized with your plain-text journal files and external market price providers.

## Manual & Background Sync

Trigger synchronization anytime from the top-right action bar:

- **Sync Journal**: Re-parses your journal files and updates postings and snapshots.
- **Update Prices**: Fetches latest quotes from configured [price providers](./commodities.md). Use **Force Refresh** on **Ledger → Commodity Prices** (`/ledger/price`) to pass `force_prices: true` and rebuild full price history.
- **Update Mutual Fund Portfolios**: Updates underlying fund portfolio compositions.

### Async Sync & Job Tracking

`POST /api/sync` returns `202 Accepted` immediately with a job ID (`{ "job_id": "<uuid>" }`). Work runs asynchronously in a SQLite-backed queue so in-flight jobs survive process restarts, while the UI subscribes to Server-Sent Events (`GET /api/jobs/stream`) for live progress updates.

#### Lazy Snapshot Refresh

To minimize wall-clock sync latency, Paisa marks cached read-model snapshots (`dashboard`, `projection`, `investment_income`) dirty when journal or price data changes and lazily rebuilds each snapshot on first access (or prioritizes the active view via `active_snapshot`).

#### Incremental Price Sync Optimization

Price providers use incremental fetching to keep updates fast:

- **Yahoo Finance** (`com-yahoo`) passes incremental `period1` timestamps since the last cached price date.
- **Alpha Vantage** (`co-alphavantage`) switches to `outputsize=compact` when incremental history is within 80 days.
- Database writes use batched SQLite upserts.

#### Sync History Overlay

Click the clock-rotate-left icon in the top-right action bar to open **Sync History**. It displays jobs in reverse-chronological order with status badges, timestamps, live-ticking wall-clock durations for active jobs, commodity progress counts, error snippets, and expandable step details.

Clicking **Clear history** calls `POST /api/jobs/clear` to remove terminal (completed/failed) jobs from both the server queue and the UI.

### Automatic Freshness & External File Change Detection

The top-right action bar shows live freshness indicators:

- **Update Prices icon**: Turns **amber** when the last price sync is $\ge 24$ hours old and **red** when it is $\ge 48$ hours old (or never run).
- **Sync Journal icon**: Turns **amber** whenever journal files change on disk. Paisa runs a background `JournalWatcher` (comparing file modification times and SHA-256 hashes across your main journal and all `include`d files) while the UI polls `GET /api/journal/status` every 30 seconds and on tab focus (`visibilitychange`), surfacing external edits made in your editor or via Git without needing a manual check.

## Scheduled Sync via CLI

Schedule automated syncs outside the web UI using `paisa update`:

```bash
# Sync journal, commodities, portfolios, and CII index
paisa update

# Sync individual targets
paisa update --journal      # -j
paisa update --commodity    # -c
paisa update --portfolio    # -p
```

### Automation Examples

- **Windows**: Use **Task Scheduler** to run `paisa.exe update --commodity` daily.
- **Linux / macOS**: Add a cron job (`crontab -e`): `0 18 * * * /path/to/paisa update --commodity`.
- **Docker**: Execute from host cron: `docker exec <container_name> paisa update -c`.

## Firefly III Webhook Integration

Paisa can receive transaction webhooks from [Firefly III](https://www.firefly-iii.org/) and automatically append new entries to `add_journal_path`.

### Setup

1. In `paisa.yaml`, set `add_journal_path` (Paisa creates parent directories and the file automatically if missing; make sure your main journal `include`s it):

    ```yaml
    add_journal_path: /home/john/Documents/paisa/added.ledger
    ```

2. In Firefly III, create a webhook pointing at `http://<paisa-host>:7500/api/webhooks/firefly` with the **After transaction creation** trigger (`STORE_TRANSACTION` events are processed; other webhook event types are ignored).
3. Run **Sync Journal** (or scheduled `paisa update --journal`) to ingest the appended entries.

## Firefly III Reconciliation (Labs)

!!! example "Labs Feature"
    Hidden by default. Enable `labs.firefly_reconcile: true` in `paisa.yaml` (or under **Tools → Configuration → Tools → Labs**).

When enabled, **Tools → Reconciliation** (`/more/reconciliation`; Classic: **More → Reconciliation**) compares your Paisa account balances against Firefly III and highlights discrepancies.

```yaml
firefly:
  url: https://firefly.example.com
  token: your-personal-access-token
  ignore_accounts:
    - Assets:Cash

labs:
  firefly_reconcile: true
```
