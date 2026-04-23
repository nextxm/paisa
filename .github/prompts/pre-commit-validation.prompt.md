---
mode: ask
description: Run the expected touched-file validation sequence before handoff or commit
---

Validate the current changes before handoff or commit.

Requirements:

1. Identify the touched files and group them by area.
2. For important behavior changes, confirm whether a focused regression or unit test was added or updated. If not, call that out.
3. Run the smallest meaningful validation commands first:
   - Go: `gofmt` on touched `.go` files, then the narrowest relevant `go test` package(s).
   - Frontend: Prettier on touched `src/**` files, then `npm run check`; run scoped ESLint on touched frontend files when practical.
4. Only escalate to broader commands like `make lint`, `go test ./...`, or full regression tests if the change spans multiple subsystems or the narrow checks are insufficient.
5. If repo-wide lint or formatting fails due to unrelated pre-existing issues, do not fix unrelated files automatically. Report that clearly and separate scoped pass/fail from repo-wide pass/fail.

When you respond:

- List the commands you ran.
- State which checks passed.
- State any remaining warnings, unrelated failures, or missing tests.
- Keep the summary concise and action-oriented.

Project references:

- Main agent guidance: [.github/copilot-instructions.md](../copilot-instructions.md)
- Quick checklist: [AGENTS.md](../../AGENTS.md)
- Test selection helper: [.github/skills/test-selection/SKILL.md](../skills/test-selection/SKILL.md)
