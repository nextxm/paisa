---
description: How to build the Paisa application.
---

# Building Paisa

Paisa can be built as a standalone binary with the frontend assets embedded or compiled separately.

## Standard Build

To build both the frontend and the Go binary:
// turbo
```powershell
make install
```

This will:
1. Build the frontend (`npm run build`).
2. Build the Go binary (`go build`).
3. Install the Go binary to your `GOPATH/bin`.

## Individual Builds

### Backend (Go)
To build the Go binary only:
// turbo
```powershell
go build .
```

To build for specific platforms (Windows, for example):
// turbo
```powershell
make windows
```

### Frontend (SvelteKit)
To build the static frontend assets:
// turbo
```powershell
npm run build
```

## Documentation Build
To build the MkDocs documentation:
// turbo
```powershell
mkdocs build
```
