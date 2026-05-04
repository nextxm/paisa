# Paisa Agent Guide

Start with [.github/copilot-instructions.md](.github/copilot-instructions.md) for the project-specific architecture, workflows, and backend/frontend contract details.

## Quick Rules

- Prefer the owning package or route when making changes; avoid broad repo edits when a local fix is possible.
- For important behavior changes, add or update the nearest focused regression or unit test before considering the task complete.
- Always review and update corresponding tests (e.g., `make regen` for fixtures) when features or UI are updated to prevent regression failures.
- Validate the touched slice first:
  - Go: `gofmt` on touched files, then the narrowest relevant `go test` package(s).
  - Frontend: Prettier on touched `src/**` files, then `npm run check`; run scoped ESLint on touched files when practical.
- Escalate to broader validation (`make lint`, `go test ./...`, regression tests) when the change spans multiple subsystems or the narrow check is insufficient.
- If repo-wide lint or formatting fails because of pre-existing unrelated issues, report that clearly and still ensure touched files pass scoped checks.

## Useful References

- Architecture and workflows: [.github/copilot-instructions.md](.github/copilot-instructions.md)
- User config behavior: [docs/reference/config.md](docs/reference/config.md)
- Pricing and multi-currency context: [docs/reference/multi-currency-prices-rfc.md](docs/reference/multi-currency-prices-rfc.md)
- Engineering review expectations: [docs/reference/engineering-review.md](docs/reference/engineering-review.md)

## Suggested Customizations

- Use the `test-selection` skill when deciding the smallest meaningful validation command for a change.
- Use the `pre-commit-validation` prompt before handoff or commit to run the expected touched-file checks in order.
