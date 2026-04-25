# Data Synchronization

Paisa needs to periodically pull new data from external price providers and your journal file.

## Manual Sync

When using the Paisa web interface, you can manually trigger a sync at any time using the action buttons in the navigation bar. This ensures your dashboard displays the latest available market data and recent journal modifications.

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
