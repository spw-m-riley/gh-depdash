# gh-depdash

`gh-depdash` shows the latest successful deployment per stable environment for a target repository.

In an interactive terminal, running `gh depdash` with no repo target opens a searchable repository picker and then a deployment browser. The deployment browser shows a short commit SHA (first 7 characters) alongside the branch and date for each successful deployment. Explicit repo targets default to JSON output; use `--verbose` to get the plain-table format instead.

By default it reports the stable environments only. `/Plan` environments are hidden unless `--plans` is passed.

`--verbose` and `--json` are operator-facing inspection modes:

- `--verbose` adds latest-attempt status and log URL context to the table output.
- `--json` emits stable JSON field names for downstream inspection or scripting.


## Releases

Releases are automated with shared workflows from `matt-riley/matt-riley-ci`.

- Push conventional commits to `main`.
- Release Please opens or updates the release PR and manages `CHANGELOG.md`.
- Merging the release PR creates the GitHub Release and version tag.
- The release workflow runs GoReleaser to attach platform-specific `gh-depdash` extension binaries and `checksums.txt` to that release.

## Examples

```bash
gh depdash
gh depdash example-owner/example-repo
gh depdash --repo example-owner/example-repo --verbose
gh depdash --repo example-owner/example-repo --json
gh depdash --repo example-owner/example-repo --plans
```

## Behavior matrix

| Command | Behavior |
| --- | --- |
| `gh depdash` | Interactive repo picker + deployment browser on a TTY; actionable missing-repo error otherwise |
| `gh depdash owner/repo` | JSON output for that repo (explicit repo targets default to JSON) |
| `gh depdash --repo owner/repo --verbose` | Plain table output for that repo |
| `gh depdash --repo owner/repo --json` | JSON output for that repo |
| `gh depdash --repo owner/repo --plans` | Includes `/Plan` environments in the JSON output |
