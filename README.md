# gh-depdash

`gh-depdash` shows the latest successful deployment per stable environment for a target repository.

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
gh depdash example-owner/example-repo
gh depdash --repo example-owner/example-repo --verbose
gh depdash --repo example-owner/example-repo --json
gh depdash --repo example-owner/example-repo --plans
```
