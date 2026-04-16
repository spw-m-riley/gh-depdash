package app

import (
	"context"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

// LoadDeploymentsForRepo fetches deployment rows for a selected repository,
// reusing the existing buildRows orchestration. Returns deployment items,
// per-environment partial failure messages, and any fatal error.
func LoadDeploymentsForRepo(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]output.ViewRow, []string, error) {
	rows, partialFailures, err := buildRows(ctx, client, owner, repo, includePlans, verbose)
	if err != nil {
		return nil, nil, err
	}
	return output.ToViewRows(rows), partialFailures, nil
}
