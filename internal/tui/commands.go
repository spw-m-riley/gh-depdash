package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

const (
	perPage        = 30
	maxReposToLoad = 100
)

var loadDeploymentsForRepo = func(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]output.ViewRow, []string, error) {
	return nil, nil, fmt.Errorf("deployment loader not initialized")
}

func SetDeploymentLoader(fn func(context.Context, githubapi.Client, string, string, bool, bool) ([]output.ViewRow, []string, error)) {
	loadDeploymentsForRepo = fn
}

func loadRepoPage(ctx context.Context, client githubapi.Client) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListRepositories(1, perPage)
		if err != nil {
			return repoPageFailedMsg{err: fmt.Sprintf("failed to list repositories: %v", err)}
		}

		hasMore := len(repos) >= perPage
		return repoPageLoadedMsg{repos: repos, hasMore: hasMore}
	}
}

func loadMoreRepos(ctx context.Context, client githubapi.Client, currentPage int) tea.Cmd {
	return func() tea.Msg {
		nextPage := currentPage + 1
		repos, err := client.ListRepositories(nextPage, perPage)
		if err != nil {
			return moreReposFailedMsg{err: fmt.Sprintf("failed to load more repositories: %v", err)}
		}

		hasMore := len(repos) >= perPage
		return moreReposLoadedMsg{repos: repos, hasMore: hasMore}
	}
}

func loadDeployments(ctx context.Context, client githubapi.Client, repo string, includePlans, verbose bool) tea.Cmd {
	return func() tea.Msg {
		owner, repoName, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || repoName == "" {
			return deploymentsFatalErrorMsg{err: fmt.Sprintf("invalid repo target %q: expected <owner/repo>", repo)}
		}

		items, partialFailures, err := loadDeploymentsForRepo(ctx, client, owner, repoName, includePlans, verbose)
		if err != nil {
			return deploymentsFatalErrorMsg{err: fmt.Sprintf("error loading deployments for %s/%s: %v", owner, repoName, err)}
		}

		return deploymentsLoadedMsg{rows: items, partialFailures: partialFailures}
	}
}
