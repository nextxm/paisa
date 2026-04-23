# Test Selection

Use this skill when you need to decide which tests and validation commands to run for a proposed or completed change.

## Goal

Choose the smallest meaningful validation set for the files and behavior being changed, then state the exact commands to run.

## Process

1. Identify the owning area from the changed files.
2. Prefer the narrowest behavior-scoped or package-scoped test first.
3. Add formatting, lint, and type checks for touched files before suggesting broader repo-wide commands.
4. Escalate to broader checks only when the change crosses boundaries or the narrow check cannot give confidence.

## Paisa Mapping

### Go backend

- For a single backend package under `internal/**`, start with `go test ./<package>`.
- For parser or ledger behavior changes, prefer tests in the owning package such as `go test ./internal/ledger`.
- For pricing, market, or provider changes, prefer `go test ./internal/service`, `go test ./internal/scraper/...`, or the most specific subpackage.
- For API handler changes, prefer `go test ./internal/server`.

### Frontend

- For `src/**` changes, run Prettier on touched files first, then `npm run check`.
- If the change is isolated, also run scoped ESLint on touched frontend files when practical.
- For import/template helper behavior under `src/lib/**`, consider the relevant Bun test file in that area when one exists.

### Cross-cutting changes

- If the change updates Go structs mirrored in TypeScript, run both the relevant Go test package and `npm run check`.
- If the change affects sync, pricing, or API responses across subsystems, consider `go test ./...` and the regression suite only after narrow checks pass.

## Output Template

Return:

- The changed area you identified.
- The smallest required test command(s).
- The touched-file format/lint/type checks.
- Any broader optional checks worth running if risk remains.

## Notes

- Do not recommend repo-wide lint first unless the change is broad.
- If repo-wide formatting/lint is already failing for unrelated files, say so explicitly and keep the required checks scoped to touched files.
- Link to [.github/copilot-instructions.md](../../../.github/copilot-instructions.md) for deeper project conventions instead of duplicating them.
