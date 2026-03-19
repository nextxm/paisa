---
description: How to run tests and linters for Paisa.
---

# Testing and Linting

Paisa maintains a high code quality through Go unit tests, frontend component tests, and integration tests.

## Running All Tests

To run the complete test suite (frontend build + frontend tests + backend tests):
// turbo
```powershell
make test
```

## Running Specific Tests

### Backend (Go)
To run all backend tests:
// turbo
```powershell
go test ./...
```

To run a specific test:
// turbo
```powershell
go test -v ./internal/service/market_test.go
```

### Frontend (Svelte/Bun)
The project uses `bun` for frontend testing:
// turbo
```powershell
make jstest
```

## Linting

To check for style issues and lint errors:
// turbo
```powershell
make lint
```

## Regenerating Data
If you modify tests or common data, you might need to regenerate test fixtures:
// turbo
```powershell
make regen
```
