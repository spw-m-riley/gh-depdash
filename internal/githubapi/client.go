package githubapi

import (
	"fmt"
	"net/url"

	"github.com/cli/go-gh/v2/pkg/api"
)

type RESTClient struct {
	client *api.RESTClient
}

func NewRESTClient() (*RESTClient, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}

	return &RESTClient{client: client}, nil
}

func (c *RESTClient) ListEnvironments(owner, repo string) ([]Environment, error) {
	var response struct {
		Environments []Environment `json:"environments"`
	}

	if err := c.client.Get(envPath(owner, repo), &response); err != nil {
		return nil, fmt.Errorf("list environments for %s/%s: %w", owner, repo, err)
	}

	return response.Environments, nil
}

func (c *RESTClient) ListDeployments(owner, repo, environment string) ([]Deployment, error) {
	var deployments []Deployment

	if err := c.client.Get(deploymentsPath(owner, repo, environment), &deployments); err != nil {
		return nil, fmt.Errorf("list deployments for %s: %w", environment, err)
	}

	return deployments, nil
}

func (c *RESTClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]DeploymentStatus, error) {
	var statuses []DeploymentStatus

	if err := c.client.Get(statusesPath(owner, repo, deploymentID), &statuses); err != nil {
		return nil, fmt.Errorf("list deployment statuses for %d: %w", deploymentID, err)
	}

	return statuses, nil
}

func (c *RESTClient) ListRepositories(page, perPage int) ([]Repository, error) {
	var repos []Repository

	if err := c.client.Get(reposPath(page, perPage), &repos); err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	return repos, nil
}

func envPath(owner, repo string) string {
	return fmt.Sprintf("repos/%s/%s/environments", owner, repo)
}

func deploymentsPath(owner, repo, environment string) string {
	query := url.Values{}
	query.Set("environment", environment)
	query.Set("per_page", "10")

	return fmt.Sprintf("repos/%s/%s/deployments?%s", owner, repo, query.Encode())
}

func statusesPath(owner, repo string, deploymentID int64) string {
	query := url.Values{}
	query.Set("per_page", "10")

	return fmt.Sprintf("repos/%s/%s/deployments/%d/statuses?%s", owner, repo, deploymentID, query.Encode())
}

func reposPath(page, perPage int) string {
	query := url.Values{}
	query.Set("sort", "updated")
	query.Set("direction", "desc")
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	query.Set("page", fmt.Sprintf("%d", page))

	return fmt.Sprintf("user/repos?%s", query.Encode())
}
