---
description: Workflow for adding a new API endpoint in the Paisa application.
---

# Adding a New API Endpoint

This workflow guides you through the process of adding a new RESTful API endpoint to the Paisa backend and exposing it to the frontend.

## Step 1: Define the API Endpoint in the Backend

Most API handlers are defined in `internal/server/`.

1. **Locate the appropriate handler file**: Choose a file that matches the domain (e.g., `budget.go`, `expense.go`). If none match, create a new one.
2. **Implement the handler function**: Use Gin's context to parse the request and return JSON results.
3. **Register the route**: Add your new handler function to `internal/server/server.go` within its relevant group.

### Example Handler Implementation

```go
func (s *Server) GetNewFeature(c *gin.Context) {
    // 1. Fetch data using a domain service
    data, err := s.accountingService.GetNewData()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 2. Return data as JSON
    c.JSON(http.StatusOK, data)
}
```

### Example Route Registration

```go
// In internal/server/server.go
api.GET("/new_feature", s.GetNewFeature)
```

## Step 2: Implement Domain Logic (if needed)

If the logic doesn't exist yet, implement it in the corresponding domain package inside `internal/`.
- `internal/accounting/`: Accounting logic.
- `internal/ledger/`: Ledger file manipulation.
- `internal/portfolio/`: Investment tracking.

## Step 3: Consume the API in the Frontend

Paisa's frontend uses SvelteKit and standard `fetch` or specific stores for state management.

1. **Create/Update the data-fetching function**: Wrap your API call in a reusable function in `src/lib/` or within a Svelte component.
2. **State Management**: If the data is global, consider adding it to `src/store.ts`.

### Example Frontend Fetch

```javascript
async function fetchNewData() {
    const response = await fetch('/api/new_feature', {
        headers: { 'X-Auth': sessionToken }
    });
    return await response.json();
}
```

## Step 4: Verify and Test

1. **Build and Run**: Use `make develop` to start the backend and frontend in development mode.
2. **Manual Test**: Use `curl` or a browser to visit the new API endpoint directly (if authenticated).
3. **Automated Test**: Add a new test case in `internal/server/integration_test.go` or a dedicated test file.

// turbo
```powershell
go test ./internal/server/... -run TestNewFeature
```
