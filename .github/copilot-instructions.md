# Copilot Instructions

## Build, test, and lint commands

- Build all packages: `go build ./...`
- Run the full test suite: `go test ./...`
- Run a single app test: `go test ./internal/app -run '^TestRunJSON$'`
- Run a single deployments test: `go test ./internal/deployments -run '^TestBuildRowsUsesLatestSuccessfulDeploymentForUAT$'`
- Run a single output test: `go test ./internal/output -run '^TestRenderVerboseTablePreservesLatestAttemptContextWithoutSuccess$'`
- No repository-specific lint command or lint config is present today; do not invent one in changes or instructions.

## High-level architecture

- `main.go` is only a thin entrypoint; all CLI behavior runs through `internal/app.Run`.
- `internal/app` is the orchestration layer. It parses CLI options, resolves the `<owner/repo>` target, creates the GitHub client, builds deployment rows, renders output, and converts backend failures into actionable stderr messages.
- Deployment lookup is intentionally a two-pass flow in `internal/app` + `internal/deployments`:
  1. an ordering pass lists environments and applies the repo’s environment ordering/filtering rules;
  2. a per-environment pass fetches deployments and statuses one environment at a time so the final output preserves that ordering and can surface partial per-environment failures.
- `internal/deployments.Service` owns the core domain rules: `/Plan` environments are hidden unless `--plans` is set, environments sort as `Development`, `Test`, `UAT`, `Production`, then everything else alphabetically, and the latest **last-known-good** deployment is the first deployment whose newest status is `success` or `inactive`.
- `internal/githubapi` is a thin wrapper around `github.com/cli/go-gh/v2/pkg/api`. It talks directly to the GitHub REST endpoints for environments, deployments, and deployment statuses and keeps the HTTP/path/query logic out of the domain layer.
- `internal/output` is responsible for the stable presentation contract. Table output is a simple pipe-delimited text format, while JSON output uses `output.ViewRow` with stable field names such as `logUrl`.

## Key conventions

- Accept repository targets either positionally (`gh depdash owner/repo`) or via `--repo owner/repo`. If both are supplied, they must match exactly.
- Treat `/Plan` as a naming convention, not a separate environment model. Filtering and sort order are driven by the environment name suffix.
- `deployments.Row.HasSuccess` controls whether `Branch` and `Date` are populated. When there is no successful deployment, keep `Branch`/`Date` blank while still preserving the latest attempt’s `Status` and `LogURL` for verbose and JSON output.
- `inactive` is treated as a historical success alongside `success`; queued/waiting/in-progress/failure states are not.
- User-facing errors are intentionally written to stderr as actionable messages and then returned as errors from `app.Run`. Preserve that behavior when changing error handling.
- Tests prefer lightweight local seams instead of heavy mocking frameworks:
  - `internal/app` swaps package-level vars like `newGitHubClient`;
  - service tests use small fixture clients implementing `githubapi.Client`;
  - REST client tests use a custom `http.RoundTripper` with JSON fixtures from `testdata/`.
- When changing rendering or row-building behavior, update tests across the affected layers, not just one package: `internal/deployments`, `internal/output`, and `internal/app` each lock in part of the end-to-end contract.
