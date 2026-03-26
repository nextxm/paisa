## Feature Request

### Current Behavior
The Cashflow → Yearly chart currently displays all transactions, including transfers between accounts. This creates visual noise with internal account movements.

### Desired Behavior
Add a setting/toggle option to hide transfers between accounts in the Cashflow yearly chart. When enabled, the chart should display only:
- **Income** (inflows from external sources)
- **Expenses** (outflows to external sources)

### Use Case
This provides a cleaner, more meaningful view of actual cash inflow/outflow, filtering out internal account movements that don't represent real income or expenses.

## Acceptance Criteria
- [ ] Cashflow yearly chart has a toggle/switch for "Show Transfers" option
- [ ] When toggle is OFF, transfers between accounts are excluded from the chart display
- [ ] When toggle is ON, all transactions including transfers are shown (current behavior)
- [ ] The toggle state persists across page reloads (stored in user preferences/localStorage)
- [ ] Chart data is correctly recalculated when toggle is switched
- [ ] UI clearly indicates whether transfers are included or excluded
- [ ] Income and Expenses remain visible and accurate regardless of transfer filtering
- [ ] No performance degradation when toggling filter on/off

## Tests

### Manual Testing Checklist
- [ ] Verify toggle appears on Cashflow yearly chart
- [ ] Verify toggling OFF hides transfer transactions
- [ ] Verify toggling ON shows transfer transactions again
- [ ] Verify income totals remain unchanged when filter is toggled
- [ ] Verify expense totals remain unchanged when filter is toggled
- [ ] Verify toggle state persists after page refresh
- [ ] Verify toggle state persists after logout/login
- [ ] Test with multiple currencies to ensure filtering works correctly
- [ ] Test with complex transfer chains (A→B→C) to ensure all are filtered

### Automated Tests (Unit/Integration)
- [ ] Test filter function correctly identifies transfer transactions
- [ ] Test chart calculation with transfers included vs excluded
- [ ] Test persistence of toggle preference to storage
- [ ] Test toggle state restoration on app load
- [ ] Test with edge cases: zero-balance transfers, self-transfers, matched transfers
- [ ] Regression tests: verify existing cashflow calculations unaffected

### Implementation Suggestion
- Add a toggle switch or filtering option in the Cashflow yearly chart UI
- Allow users to toggle "Show Transfers" on/off (default: on to maintain current behavior)
- When disabled, exclude transactions where both posting accounts are transfers (internal movements)
