---
description: How to set up the development environment for Paisa.
---

# Development Environment Setup

Paisa is a client-server application with a Go backend and a SvelteKit frontend.

## Prerequisites

- Go 1.22+
- Node.js 18+
- `npm` or `bun` (project uses `bun` for some tests/scripts)
- `make`

## Steps

1. Install Go dependencies:
   ```bash
   go mod download
   ```

2. Install Node dependencies:
   ```bash
   npm install
   ```

3. (Optional) Run the development environment with concurrent backend and frontend:
   // turbo
   ```powershell
   make develop
   ```

4. If you want to run the server only:
   // turbo
   ```powershell
   make serve
   ```

5. For frontend development only:
   // turbo
   ```powershell
   npm run dev
   ```

## Helpful Commands

- **Initialize demo data**: `go build && ./paisa init && ./paisa update`
- **Watch frontend changes**: `npm run build:watch`
