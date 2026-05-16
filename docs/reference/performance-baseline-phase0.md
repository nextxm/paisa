# Phase 0 Performance Baseline (config/dashboard/projection)

This baseline captures endpoint latency and SQL workload before performance refactors.

## Reproducible run steps

From repo root:

```bash
go run ./cmd/perfbaseline --iterations 30 --warmup 5 --years 20
```

The harness:

- creates a temporary SQLite database
- seeds a synthetic **20-year monthly dataset**
- calls:
  - `GET /api/config`
  - `GET /api/dashboard`
  - `GET /api/networth/projection`
- collects:
  - request latency (p50, p95)
  - SQL query count
  - total SQL time

Telemetry comes from response headers emitted by the handlers:

- `X-Paisa-Perf-Latency-Ms`
- `X-Paisa-Perf-SQL-Count`
- `X-Paisa-Perf-SQL-Time-Ms`

## Baseline results (Phase 0)

Captured at `2026-05-16T13:14:42Z` on:

- Go `go1.24.13`
- `linux/amd64`
- 4 vCPUs

Dataset: synthetic 20 years (`5` warmup + `30` measured samples per endpoint)

| endpoint | p50 latency (ms) | p95 latency (ms) | total sql queries | total sql time (ms) | avg sql queries/request | avg sql time/request (ms) |
|---|---:|---:|---:|---:|---:|---:|
| `/api/config` | 1.24 | 2.10 | 60 | 3.94 | 2.00 | 0.13 |
| `/api/dashboard` | 106.77 | 120.31 | 570 | 2432.93 | 19.00 | 81.10 |
| `/api/networth/projection` | 32.71 | 35.73 | 120 | 406.03 | 4.00 | 13.53 |

## Before/after comparison template

Use this checklist for each optimization phase:

- [ ] Run the same harness command (`cmd/perfbaseline`) with same flags.
- [ ] Record machine/runtime context (OS, arch, CPU count, Go version).
- [ ] Compare p50/p95 for all 3 endpoints.
- [ ] Compare total SQL query count per endpoint.
- [ ] Compare total SQL time per endpoint.
- [ ] Flag any regression (>10% p95 increase) with root-cause notes.
- [ ] Link PR/commit implementing the phase optimization.
