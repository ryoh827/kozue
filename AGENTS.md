# Repository Guidelines

## Project Structure & Module Organization
The CLI entrypoint lives in `cmd/koz/main.go`, wiring Cobra setup into internal packages. Core logic belongs under `internal/cli`, where each subcommand should stay small and focused. Keep shared helpers private unless cross-package reuse is essential. Couple specs live in `hoge.md`; treat it as the source of truth for workflows, schema expectations, and safety switches when extending features. Tests should sit beside their sources as `_test.go` files, and module metadata remains in `go.mod`/`go.sum`.

## Worktree Safety & UX Principles
`koz` exists to make `git worktree` safer. Default behaviors must be non-destructive: honor the `--force --i-understand` guard rails, respect automatic path selection (`.<repo>-koz/<branch>`), and reuse helpers for sanitizing branch names. When outputting tables or JSON, stick to schema_version=1 until the spec says otherwise. Propagate actionable error messages (`status dirty`, `locked`, `ahead`) so users know what to fix.

## Build, Test, and Development Commands
- `go run ./cmd/koz --help` regenerates help text and validates Cobra wiring.
- `go build ./cmd/koz` produces the binary against Go 1.23+ without installing.
- `go test ./...` executes all packages; add `-cover` to gauge coverage before merging.
- `KOZ_GIT_TIMEOUT=6s go test ./...` exercises timeout overrides referenced in `hoge.md`.

## Coding Style & Naming Conventions
Run `go fmt ./...` (or your editor’s gofmt integration) before commits—tabs, not spaces. Package names stay lower-case (`cli`, `gitx`). New exported identifiers should exist only when public use is intended; favor unexported helpers. Cobra commands need descriptive `Use`, `Short`, and `Long` strings consistent with the safety-first tone; the `add` command now auto-configures upstreams when `origin/<branch>` exists, so expose opt-outs via flags instead of ad-hoc overrides. Prefer `context.Context` plumbing when executing git.

## Testing Guidelines
Stick to Go’s standard testing library. Mirror production filenames with `_test.go` siblings (`internal/cli/root_test.go`). Use table-driven cases for parser logic (porcelain parsing, sanitizer collisions) and assert substrings over entire help messages to keep tests resilient. Run `go test ./...` locally before opening a PR and add regressions for every bug fix, especially around guard rails or path resolution.

## Commit & Pull Request Guidelines
History follows Conventional Commits (`feat:`, `fix:`, `refactor:`). Keep each commit scoped and include generated artifacts when relevant. PRs should summarize motivation, link issues, and list manual verification commands (e.g., `go run ./cmd/koz list`). Mention spec touchpoints from `hoge.md` when behaviors change, and attach CLI output excerpts if they differ from previous versions. Always confirm CI status and note future follow-ups explicitly.
