# Harness-neutral agent hooks

These scripts encode repository rules that AI coding harnesses would otherwise
discover only by failing CI. The scripts themselves are harness-agnostic; each
harness wires them up through its own adapter. There is currently no
vendor-neutral hook standard, so the contract is deliberately minimal:

* Scripts take at most one argument (a file path) and read nothing from stdin.
* Exit 0 means "nothing to report".
* Exit 1 means "finding" — the explanation is printed to stderr, phrased so an
  agent can act on it.
* Missing tools or unexpected repository state fail open (exit 0): a hook must
  never break an edit loop. CI remains the authoritative enforcement.

## Scripts

| Script | When to run | Purpose |
|---|---|---|
| `check-generated-file.sh <path>` | before a file edit (blocking) | Rejects hand-edits to generated trees (`zz_generated.*.go`, `bundle/`, `docs/api/`, `CHANGELOG.md`) and names the `make` target to use instead. |
| `check-go-file.sh <path>` | after a Go file edit (advisory) | Checks the SPDX license header and, when `bin/golangci-lint` exists, formatting (gofumpt/gci) of just that file. |

Changelog-entry presence is deliberately NOT a hook: `make precommit` checks it
(`hack/check-changelog-entry.sh`) so the nudge reaches every contributor and
agent, hooks or not.

Run the hook regression tests with `bash .agents/hooks/test-hooks.sh`.

## Adapters

* **Claude Code**: wired via the checked-in [`.claude/settings.json`](../../.claude/settings.json)
  (pre/post tool-use and stop hooks). Claude Code asks the user to approve
  project hooks on first use.
* **Other harnesses**: adapters are welcome, provided the logic stays in these
  scripts and the adapter only extracts the file path and maps exit codes.
