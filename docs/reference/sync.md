# Data Synchronization

Paisa needs to periodically pull new data from external price providers and your journal file.

## Manual Sync

When using the Paisa web interface, you can manually trigger a sync at any time using the action buttons in the navigation bar. This ensures your dashboard displays the latest available market data and recent journal modifications.

### Async Sync & Job Tracking

`POST /api/sync` returns `202 Accepted` immediately with a job ID
(`{ "job_id": "<uuid>" }`) rather than blocking until the sync
completes. The work is performed in the background and tracked in a
SQLite-backed queue, so in-flight jobs survive process restarts. The UI
subscribes to `GET /api/jobs/stream` (Server-Sent Events) for live
status/progress updates instead of polling.

#### Sync History Overlay

Click the clock-rotate-left icon in the header action bar to open the
**Sync History** overlay.  The overlay lists all background jobs in
reverse-chronological order and shows for each job:

- A colour-coded status badge (success / running / failed / pending).
- The creation, start, and finish timestamps and wall-clock duration.
- An error message snippet for failed jobs.
- An expandable list of per-step details.

A **Clear history** button resets the client-side view. The persisted
job queue remains available on the server and is replayed into the UI
stream after reconnect. The overlay can
be opened and closed at any time without interrupting an in-progress
sync.

### Price & Journal Freshness Indicators

The navigation bar shows visual freshness indicators at a glance:

- **Update Prices icon** – turns **amber** when the last price update
  is more than 24 hours ago and **red** when it is more than 48 hours
  ago.
- **Sync Journal icon** – turns **amber** whenever one or more journal
  files have been modified on disk since the last sync.

## Scheduled Sync via CLI

For automated periodic updates outside of the running application (e.g., pulling nightly mutual fund prices or syncing an external ledger file), you can schedule the `paisa update` command using your operating system's built-in tools.

```bash
# Sync journal, commodities, and portfolios
paisa update

# Sync only commodities
paisa update --commodity
```

### Automation Examples

- **Windows**: Use **Task Scheduler** to create a basic task that runs daily at a specific time, executing `paisa.exe update --commodity`.
- **Linux / macOS**: Add a cron job (`crontab -e`) to run it daily: 
  `0 18 * * * /path/to/paisa update --commodity`
- **Docker**: If hosting via Docker, use your host machine's cron to execute `docker exec <container_name> paisa update -c`.

## Firefly III Webhook Integration

Paisa can receive transaction webhooks from
[Firefly III](https://www.firefly-iii.org/) and automatically append
the imported transactions to your journal.

### Setup

1. In `paisa.yaml`, set `add_journal_path` to the file that should
   receive webhook-imported transactions:

    ```yaml
    add_journal_path: /home/john/Documents/paisa/added.ledger
    ```

2. In Firefly III, create a webhook pointing at:

    ```
    http://<paisa-host>:7500/api/webhooks/firefly
    ```

    Trigger: **After transaction creation** (or update/destroy as
    needed).

3. Paisa parses the incoming payload and appends well-formed ledger
   entries to `add_journal_path`.  Run **Sync Journal** afterward (or
   let the scheduled sync pick them up) to see the new transactions in
   the UI.

!!! note
    The `add_journal_path` file must already exist; Paisa will not
    create it automatically.

## Firefly III Reconciliation (Labs)

!!! example "Labs Feature"
    This feature is hidden by default.  Enable it with
    `labs.firefly_reconcile: true` in `paisa.yaml` (see
    [Configuration](config.md)).

When enabled, a balance-reconciliation tool lets you compare your
Paisa (Ledger) account balances with the corresponding account
balances in Firefly III.  Discrepancies are highlighted in the UI.

Configuration options in `paisa.yaml`:

```yaml
firefly:
  url: https://firefly.example.com   # Firefly III base URL
  token: your-personal-access-token  # Personal Access Token
  ignore_accounts:                   # Accounts to skip during reconciliation
    - Assets:Cash

labs:
  firefly_reconcile: true            # Enable the reconciliation tool
```
