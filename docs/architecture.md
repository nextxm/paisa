# Architecture and Design Document

## Overview

Paisa is a personal finance management tool designed to track finances, visualize expenses, and manage portfolios. It operates as a local server application with a web-based user interface and supports a desktop mode via Wails. It emphasizes privacy by running locally and storing data in a local SQLite database.

## Technology Stack

### Backend
- **Language**: Go (Golang) 1.24+
- **Web Framework**: Gin (High-performance HTTP web framework)
- **ORM**: GORM (with SQLite driver)
- **CLI Framework**: Cobra (for command-line interface commands like `serve`, `update`)
- **Desktop Wrapper**: Wails (for bundling as a native desktop app)
- **Logging**: Logrus (Structured logging)
- **Configuration**: YAML-based configuration

### Frontend
- **Framework**: SvelteKit (App framework based on Svelte)
- **Build Tool**: Vite (Next Generation Frontend Tooling)
- **Runtime**: Node.js (for development/build)
- **Styling**: 
    - Tailwind CSS (Utility-first CSS framework)
    - Bulma (CSS framework, seemingly used for legacy or specific components)
    - SCSS (Sassy CSS)
- **State Management**: Svelte Stores
- **Charting**: D3.js (for complex data visualizations)
- **Editor**: CodeMirror (for editing ledger files)

### Documentation
- **Tool**: MkDocs (Markdown-based documentation generator)

## High-Level Architecture

Paisa utilizes a monolithic client-server architecture, typically running on the user's local machine.

1.  **Client (Frontend)**: A Single Page Application (SPA) built with SvelteKit. It communicates with the backend via a RESTful API.
2.  **Server (Backend)**: A Go binary that serves two purposes:
    -   **Static File Server**: Serves the compiled frontend assets.
    -   **API Server**: Exposes endpoints for data retrieval and manipulation.
3.  **Database**: A local SQLite database (`paisa.db` likely) managed via GORM.
4.  **Desktop Wrapper**: Wails acts as a bridge, running the Go backend and rendering the frontend in a system interaction view (WebView2 on Windows, WebKit on macOS/Linux).

## Backend Architecture (`internal`)

The backend logic is strictly organized by domain within the `internal` directory.

### Entry Points (`cmd`)
-   `root.go`: Initializes the CLI, logger, and configuration.
-   `serve.go`: Starts the web server.

### Core Domain Modules
-   **`accounting`**: Implements double-entry accounting logic.
-   **`ledger`**: Handles parsing and management of plain text ledger files.
-   **`transaction`**: Manages financial transactions.
-   **`portfolio`**: Logic for investment portfolio tracking.
-   **`taxation`**: Computes tax implications.
-   **`prediction`**: Financial forecasting implementation (likely using ARIMA).
-   **`xirr`**: Extended Internal Rate of Return calculations for investments.

### Infrastructure & Adapters
-   **`server`**: 
    -   Sets up the Gin router (`server.go`).
    -   Defines API handlers (e.g., `budget.go`, `expense.go`, `income.go`) that map HTTP requests to domain logic.
    -   Implements `TokenAuthMiddleware` for API authentication using `X-Auth` header.
-   **`model`**: Defines the database schema using GORM structs.
-   **`scraper`**: Modules for scraping or importing financial data from various sources (Banks, PDF statements, etc.).
-   **`config`**: Manages application configuration (loading `paisa.yaml`).

## Frontend Architecture (`src`)

The frontend is structured as a standard SvelteKit application.

### Routing (`src/routes`)
-   **`(app)`**: The main protected application area.
    -   `dashboard`: Main overview.
    -   `assets`, `liabilities`: Balance sheet items.
    -   `income`, `expense`: Income statement items.
    -   `cash_flow`: Cash flow analysis.
    -   `ledger`: Review and edit ledger entries.
    -   `more`: Settings and tools.
-   **`login`**: Authentication page.

### Key Components
-   **`src/lib`**: Reusable UI components and helper functions.
-   **`src/store.ts`**: Global state management (likely for user session, theme, etc.).
-   **`app.scss` / `app.html`**: Global styles and entry point.

## Data Flow

1.  **User Action**: User interacts with the UI (e.g., adds a transaction).
2.  **API Call**: Frontend sends a POST request to `/api/editor/save` (or similar).
3.  **Authentication**: `TokenAuthMiddleware` validates the request.
4.  **Handler**: The request is routed to the specific handler in `internal/server`.
5.  **Domain Logic**: The handler calls the appropriate domain service (e.g., `accounting`, `ledger`).
6.  **Persistence**: GORM writes changes to the SQLite database.
7.  **Response**: Updated state or success message is returned to the frontend.

## Security
-   **Authentication**: Custom token-based authentication mechanism.
-   **Local-First**: Data is stored locally, reducing external attack vectors.
-   **Readonly Mode**: Configuration option to prevent data modification via the UI.

## API Reference

The backend exposes a RESTful API under the `/api` prefix.

### Authentication
Authentication is enforced via `TokenAuthMiddleware` if `UserAccounts` are configured in `paisa.yaml`. Two header formats are accepted:

#### Session Token (primary)
1.  `POST /api/auth/login` with `{"username": "...", "password": "..."}` → returns `{"token": "<uuid>", "expires_at": "...", "username": "..."}`
2.  Subsequent requests: `X-Auth: <session-token>` (UUID, no colon)

#### Legacy Credential Header
-   **Header**: `X-Auth: username:plaintext-password`
-   **Mechanism**: The middleware SHA-256-hashes the plaintext password and compares it to `sha256:<hash>` stored in `paisa.yaml`.
-   Configure user accounts in `paisa.yaml`:
    ```yaml
    user_accounts:
      - username: myuser
        password: sha256:<sha256-of-plaintext-password>
    ```
    Generate the hash: `echo -n "mypassword" | sha256sum` (Linux/macOS)

#### Rate Limiting
6 requests per minute (burst 3) per IP. Failed auth attempts consume quota.

### Endpoints

**Authentication**
-   `POST /api/auth/login`: Exchange username/password for a session token.

**Configuration & System**
-   `GET /api/ping`: Health check.
-   `GET /api/config`: Get system configuration.
-   `POST /api/config`: Update system configuration.
-   `POST /api/init`: Initialize demo data (only if not readonly).
-   `POST /api/sync`: Sync data (Journal, Prices, Portfolios).

**Dashboard & Reports**
-   `GET /api/dashboard`: Dashboard summary data.
-   `GET /api/networth`: Net worth history.
-   `GET /api/assets/balance`: Current asset balances.
-   `GET /api/investment`: Investment portfolio summary.
-   `GET /api/gain`: Overall capital gains/losses.
-   `GET /api/gain/:account`: Capital gains for a specific account.
-   `GET /api/income`: Income analysis.
-   `GET /api/expense`: Expense analysis. Response includes `expenses`, `month_wise`, `year_wise`, `graph` (hierarchy Sankey per FY), and `income_graph` (income-allocation Sankey per FY).
-   `GET /api/budget`: Budget status.
-   `GET /api/cash_flow`: Cash flow statement.
-   `GET /api/income_statement`: Income statement details.
-   `GET /api/recurring`: Recurring transaction analysis.
-   `GET /api/allocation`: Asset allocation.
-   `GET /api/portfolio_allocation`: Portfolio allocation.
-   `GET /api/diagnosis`: data health checks.
-   `GET /api/logs`: Server logs.

**Ledger & Transactions**
-   `GET /api/ledger`: Raw ledger entries.
-   `GET /api/transaction`: Transaction list.
-   `GET /api/transaction/balanced`: Balanced postings.
-   `GET /api/editor/files`: List available ledger files.
-   `POST /api/editor/file`: specific ledger file content.
-   `POST /api/editor/save`: Save changes to a ledger file.
-   `POST /api/editor/validate`: Validate ledger file syntax.
-   `GET /api/sheets/files`: List sheet files.
-   `POST /api/sheets/file`: Get sheet content.
-   `POST /api/sheets/save`: Save sheet content.

**Market Data (Prices)**
-   `GET /api/price`: Get stored prices.
-   `POST /api/price/delete`: Clear price cache.
-   `GET /api/price/providers`: List price providers.
-   `POST /api/price/providers/delete/:provider`: Clear cache for a provider.
-   `POST /api/price/autocomplete`: Autocomplete for price symbols.

**Tax & Planning**
-   `GET /api/harvest`: Tax loss harvesting opportunities.
-   `GET /api/capital_gains`: Capital gains report.
-   `GET /api/schedule_al`: Schedule AL (Assets and Liabilities) report.

**Liabilities**
-   `GET /api/liabilities/interest`: Loan interest analysis.
-   `GET /api/liabilities/balance`: Liability balances.
-   `GET /api/liabilities/repayment`: Repayment schedule.

**Templates & Goals**
-   `GET /api/templates`: List templates.
-   `POST /api/templates/upsert`: Create/Update details.
-   `POST /api/templates/delete`: Delete template.
-   `GET /api/goals`: List goals.
-   `GET /api/goals/:type/:name`: Specific goal details.

**Credit Cards**
-   `GET /api/credit_cards`: List credit cards.
-   `GET /api/credit_cards/:account`: Specific credit card details.
