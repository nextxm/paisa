# Phase 5 Query Plan Tuning (long-history journals)

This phase targets SQLite posting scans that dominate dashboard/projection input derivation on 20+ year ledgers.

## Added indexes

Migration **v12** adds:

- `idx_postings_forecast_date` on `(forecast, date)`
- `idx_postings_forecast_account_date` on `(forecast, account, date)`

Both indexes are created with `CREATE INDEX IF NOT EXISTS`, so migration is safe for fresh installs and existing databases.

## Query-plan verification

Use SQLite `EXPLAIN QUERY PLAN` for representative predicates:

```sql
EXPLAIN QUERY PLAN
SELECT * FROM postings
WHERE forecast = 0 AND account LIKE 'Income:%' AND date >= ? AND date < ?
ORDER BY date ASC, amount DESC, account ASC;

EXPLAIN QUERY PLAN
SELECT * FROM postings
WHERE forecast = 0 AND date < ?
ORDER BY date ASC, amount DESC, account ASC;
```

Expected plan output includes:

- `idx_postings_forecast_account_date` for account-prefix + date-range scans
- `idx_postings_forecast_date` for forecast + date-range scans

Automated coverage exists in `internal/model/migration/migration_test.go` (`TestV12Migration_ExplainUsesPostingReadIndexes`).

## Rollout and rollback strategy

### Rollout

1. Deploy binary containing migration v12.
2. On startup, `migration.RunMigrations` applies v12 and records schema version `12`.
3. Validate via:
   - `SELECT version FROM schema_versions ORDER BY version DESC LIMIT 1;`
   - `SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_postings_forecast%';`

### Rollback

If rollback is required:

1. `DROP INDEX IF EXISTS idx_postings_forecast_account_date;`
2. `DROP INDEX IF EXISTS idx_postings_forecast_date;`
3. Optionally keep `schema_versions` at 12 and run older binaries in read-only mode, or ship a follow-up migration if strict downgrade support is required.

No table shape or data payload changes are introduced in v12; rollback only removes secondary indexes.
