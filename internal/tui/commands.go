package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gh-depdash/internal/deployments"
	"gh-depdash/internal/githubapi"
)

func loadRepoPage(ctx context.Context, client githubapi.Client) tea.Cmd {
	return func() tea.Msg {
		return repoPageFailedMsg{err: "repository discovery not yet implemented"}
	}
}

func loadDeployments(ctx context.Context, client githubapi.Client, repo string, includePlans, verbose bool) tea.Cmd {
	return func() tea.Msg {
		owner, repoName, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || repoName == "" {
			return deploymentsFatalErrorMsg{err: fmt.Sprintf("invalid repo target %q: expected <owner/repo>", repo)}
		}

		orderingService := deployments.Service{Client: orderingClient{base: client}}
		orderedRows, err := orderingService.BuildRows(ctx, owner, repoName, includePlans)
		if err != nil {
			return deploymentsFatalErrorMsg{err: classifyBuildError(owner, repoName, err)}
		}
		if len(orderedRows) == 0 {
			return deploymentsFatalErrorMsg{err: fmt.Sprintf("no environments found for %s/%s", owner, repoName)}
		}

		rows := make([]deployments.Row, 0, len(orderedRows))
		partialFailures := make([]string, 0)

		for _, orderedRow := range orderedRows {
			service := deployments.Service{Client: singleEnvironmentClient{
				base:        client,
				environment: orderedRow.Environment,
			}}

			builtRows, err := service.BuildRows(ctx, owner, repoName, true)
			if err != nil {
				partialFailures = append(partialFailures, fmt.Sprintf("%s: %v", orderedRow.Environment, err))
				if verbose {
					rows = append(rows, deployments.Row{Environment: orderedRow.Environment})
				}
				continue
			}
			if len(builtRows) == 0 {
				if verbose {
					rows = append(rows, deployments.Row{Environment: orderedRow.Environment})
				}
				continue
			}

			rows = append(rows, builtRows[0])
		}

		return deploymentsLoadedMsg{rows: rows, partialFailures: partialFailures}
	}
}

func classifyBuildError(owner, repo string, err error) string {
	return fmt.Sprintf("error loading deployments for %s/%s: %v", owner, repo, err)
}

type orderingClient struct {
	base githubapi.Client
}

func (c orderingClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return c.base.ListEnvironments(owner, repo)
}

func (c orderingClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return nil, nil
}

func (c orderingClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return nil, nil
}

func (c orderingClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return c.base.ListRepositories(page, perPage)
}

type singleEnvironmentClient struct {
	base        githubapi.Client
	environment string
}

func (c singleEnvironmentClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	envs, err := c.base.ListEnvironments(owner, repo)
	if err != nil {
		return nil, err
	}
	for _, env := range envs {
		if env.Name == c.environment {
			return []githubapi.Environment{env}, nil
		}
	}
	return nil, nil
}

func (c singleEnvironmentClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return c.base.ListDeployments(owner, repo, environment)
}

func (c singleEnvironmentClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return c.base.ListDeploymentStatuses(owner, repo, deploymentID)
}

func (c singleEnvironmentClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return c.base.ListRepositories(page, perPage)
}
