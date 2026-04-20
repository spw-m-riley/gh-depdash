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
	perPage = 30
)

var loadDeploymentsForRepo = func(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]output.ViewRow, []string, error) {
	return nil, nil, fmt.Errorf("deployment loader not initialized")
}

func SetDeploymentLoader(fn func(context.Context, githubapi.Client, string, string, bool, bool) ([]output.ViewRow, []string, error)) {
	loadDeploymentsForRepo = fn
}

func loadRepoPage(ctx context.Context, client githubapi.Client) tea.Cmd {
	return func() tea.Msg {
		page, err := client.ListRepositories(1, perPage)
		if err != nil {
			return repoPageFailedMsg{err: formatRepositoryLoadError("failed to list repositories", err)}
		}

		return repoPageLoadedMsg{repos: page.Repositories, hasMore: page.HasMore}
	}
}

func loadMoreRepos(ctx context.Context, client githubapi.Client, currentPage, sessionID int) tea.Cmd {
	return func() tea.Msg {
		nextPage := currentPage + 1
		page, err := client.ListRepositories(nextPage, perPage)
		if err != nil {
			return moreReposFailedMsg{sessionID: sessionID, err: formatRepositoryLoadError("failed to load more repositories", err)}
		}

		return moreReposLoadedMsg{sessionID: sessionID, repos: page.Repositories, hasMore: page.HasMore}
	}
}

func loadDeployments(ctx context.Context, client githubapi.Client, repo string, includePlans, verbose bool) tea.Cmd {
	return func() tea.Msg {
		owner, repoName, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || repoName == "" || strings.Contains(repoName, "/") {
			return deploymentsFatalErrorMsg{err: fmt.Sprintf("invalid repo target %q: expected <owner/repo>", repo)}
		}

		items, partialFailures, err := loadDeploymentsForRepo(ctx, client, owner, repoName, includePlans, verbose)
		if err != nil {
			return deploymentsFatalErrorMsg{err: err.Error()}
		}

		return deploymentsLoadedMsg{rows: items, partialFailures: partialFailures}
	}
}

func formatRepositoryLoadError(prefix string, err error) string {
	message := strings.TrimPrefix(err.Error(), "list repositories: ")
	return fmt.Sprintf("%s: %s", prefix, message)
}
