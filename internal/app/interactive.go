package app

import (
	"context"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

// RepositoryItem represents a repository in the interactive picker.
type RepositoryItem struct {
	FullName    string
	Description string
}

// DeploymentItem represents a deployment environment for the TUI.
type DeploymentItem = output.ViewRow

// LoadInitialRepositories fetches the first page of repositories for the picker.
func LoadInitialRepositories(ctx context.Context, client githubapi.Client) ([]RepositoryItem, error) {
	return loadRepositoryPage(ctx, client, 1, 30)
}

// LoadMoreRepositories fetches additional repository pages for the picker.
func LoadMoreRepositories(ctx context.Context, client githubapi.Client, page, perPage int) ([]RepositoryItem, error) {
	return loadRepositoryPage(ctx, client, page, perPage)
}

// LoadDeploymentsForRepo fetches deployment rows for a selected repository,
// reusing the existing buildRows orchestration.
func LoadDeploymentsForRepo(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]DeploymentItem, error) {
	rows, _, err := buildRows(ctx, client, owner, repo, includePlans, verbose)
	if err != nil {
		return nil, err
	}
	return output.ToViewRows(rows), nil
}

func loadRepositoryPage(ctx context.Context, client githubapi.Client, page, perPage int) ([]RepositoryItem, error) {
	repos, err := client.ListRepositories(page, perPage)
	if err != nil {
		return nil, err
	}

	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		desc := ""
		if repo.Description != nil {
			desc = *repo.Description
		}
		items = append(items, RepositoryItem{
			FullName:    repo.FullName,
			Description: desc,
		})
	}
	return items, nil
}
