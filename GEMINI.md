# GEMINI.md

## Project Overview

**Paisa** is a Personal Finance Manager built on top of the [ledger](https://www.ledger-cli.org/) double-entry accounting tool. It features a Go-based backend providing a REST API and a SvelteKit-powered web interface. It can be run as a self-hosted web application or a standalone desktop application (using Wails).

### Core Technologies
- **Backend:** Go (1.24+), Gin (Web Framework), GORM (SQLite), Cobra (CLI), Wails (Desktop).
- **Frontend:** SvelteKit, Vite, TypeScript, TailwindCSS, Bulma.
- **Accounting:** Ledger-cli (data format and engine).
- **Build/Dev:** GNU Make, npm, Bun (used for some tests).

## Building and Running

The project uses a `Makefile` to orchestrate build and development tasks.

### Development
- **Concurrent Backend & Frontend:** `make develop` (Requires `concurrently` and `nodemon`)
- **Backend Only (Go):** `make serve`
- **Frontend Only (Vite):** `npm run dev`
- **Documentation:** `make docs` (Serves on `0.0.0.0:8000` using `mkdocs`)

### Testing and Linting
- **Full Test Suite:** `make test` (Builds JS, runs Bun tests, and Go tests)
- **Go Tests:** `go test ./...`
- **Frontend Tests:** `make jstest` (Uses Bun)
- **Linting:** `make lint` (Prettier, Svelte-check, and Gofmt)

### Installation/Build
- **Build and Install:** `make install` (Builds JS and installs Go binary)
- **Cross-compile Windows:** `make windows`
- **Regenerate Fixtures:** `make regen` (Useful when changing test data)

## Project Structure

- `cmd/`: CLI command definitions (using Cobra).
- `internal/`: Core Go business logic:
    - `accounting/`: Ledger integration.
    - `budget/`: Budgeting logic.
    - `model/`: Database and domain models.
    - `server/`: Gin API endpoints and middleware.
    - `service/`: Application services.
- `src/`: Frontend Svelte components, stores, and styles.
- `desktop/`: Wails-specific configuration and Go code for the desktop app.
- `docs/`: Markdown documentation for the project.
- `fixture/`: Ledger files and JSON samples used for testing and demos.
- `tests/`: Integration and regression tests.

## Development Conventions

- **Clean Architecture:** Domain logic is isolated in `internal/`.
- **API First:** The frontend communicates with the Go backend via a REST API defined in `internal/server/`.
- **Testing:** New features should include both Go unit tests and Svelte component tests where applicable.
- **Linting:** Strict linting is enforced via `make lint`. Ensure code is formatted with `gofmt` and `prettier`.
- **Data Format:** Ledger files are the source of truth for financial data. SQLite is used for caching and supplemental metadata.
