# Data Synchronization

Paisa needs to periodically pull new data from external price providers and your journal file.

## Background Sync

When using `paisa serve`, Paisa runs periodic background syncs to ensure your prices and journal entries are kept fresh. This ensures your dashboard always displays the latest available market data and recent journal modifications without requiring a manual refresh.

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
