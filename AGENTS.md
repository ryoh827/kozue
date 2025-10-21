# Repository Guidelines

## Project Structure & Module Organization
The CLI entrypoint lives in `cmd/koz/main.go`, which wires up the root command and delegates to internal packages. Shared command logic is organized under `internal/cli`, keeping the exported surface minimal while leaving room for new subcommands. Tests sit next to their sources as `_test.go` files; add new packages under `internal` when functionality does not need to be exported outside the module. Module and dependency metadata reside in `go.mod` and `go.sum`.

## Build, Test, and Development Commands
- `go run ./cmd/koz --help` prints the current command surface and is the fastest way to smoke-test changes.
- `go build ./cmd/koz` produces the binary with module-aware dependencies resolved.
- `go test ./...` runs the full suite; add `-cover` when you need coverage numbers before opening a pull request.

## Coding Style & Naming Conventions
Format Go source with `gofmt` (or `go fmt ./...`) prior to commit; the project assumes default Go tab indentation. Keep package names short and lower-case (`cli`, `worktree`). Exported identifiers should use CamelCase only when exposure outside the package is intentional; prefer unexported helpers otherwise. When adding Cobra commands, align `Use`, `Short`, and `Long` strings with the existing tone and ensure descriptions are actionable.

## Testing Guidelines
Follow Go’s standard `testing` package; mirror production filenames with `_test.go` siblings as shown in `internal/cli/root_test.go`. Table-driven tests are encouraged for multiple scenarios. Ensure help text and error messaging assertions stay resilient by using substring checks rather than full string equality. Run `go test ./...` locally before pushing; add regression tests for each bug fix.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commits (e.g., `feat:`, `fix:`) with concise English summaries; append short context in parentheses if needed. Keep commits scoped to a logical change and include any generated files. Pull requests should describe motivation, reference related issues, and note manual test commands (paste relevant `go` invocations). Screenshots are only needed when CLI output changes materially. Always confirm the suite passes in CI and call out follow-up work in the pull request body.
