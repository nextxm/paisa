
$repo = "nextxm/paisa"
$milestone = "Performance & Scalability Improvements"

$issues = @(
    @{
        title = "Optimize Database Writes with Batch Inserts"
        body = "The current 'UpsertAll' implementation deletes all postings and re-inserts them one by one, which is slow for large datasets.`n`nApproach: Use GORM's 'CreateInBatches' to perform batch inserts during sync.`n`nExpected Result: Significant reduction in time taken for journal synchronization."
    },
    @{
        title = "Leverage SQL-Level Aggregations for Balance Queries"
        body = "Balance calculations currently fetch all postings into memory and aggregate them in Go code.`n`nApproach: Refactor account balance queries to use SQL 'GROUP BY' and 'SUM' to reduce data transfer and memory overhead.`n`nExpected Result: Faster API response times for dashboard and balance reports."
    },
    @{
        title = "Implement Materialized Summary Tables for Account Balances"
        body = "Calculating current balances from full transaction history is O(N).`n`nApproach: Create a summary table to store pre-calculated account balances, updated during the sync process.`n`nExpected Result: O(1) balance lookups for improved UI responsiveness."
    },
    @{
        title = "Optimize Running Balance Calculation using SQL Window Functions"
        body = "The current Go-based day-by-day iteration for historical balances is inefficient for multi-year datasets.`n`nApproach: Use SQLite window functions (SUM() OVER) to calculate running totals directly in the database.`n`nExpected Result: Drastic performance improvement for net worth and historical balance charts."
    },
    @{
        title = "Asynchronous Post-Sync Processing and UI Feedback"
        body = "Heavy post-sync tasks like XIRR calculation currently block the sync response or the UI.`n`nApproach: Ensure all post-sync heavy lifting (XIRR warming, summary updates) is fully backgrounded and provide better 'syncing' status in the UI.`n`nExpected Result: Immediate UI feedback after sync without waiting for expensive calculations."
    },
    @{
        title = "Incremental Journal Sync and Change Tracking"
        body = "Re-parsing the entire journal on any change is wasteful for large files.`n`nApproach: Investigate methods to track changes in journal files (hashing transactions or using file offsets) to perform partial updates in SQLite.`n`nExpected Result: Reduced CPU and I/O load during frequent small journal edits."
    }
)

foreach ($issue in $issues) {
    Write-Host "Creating issue: $($issue.title)"
    gh issue create --repo $repo --title $issue.title --body $issue.body --milestone $milestone
}
