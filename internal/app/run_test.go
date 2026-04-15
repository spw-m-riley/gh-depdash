package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

func TestRunDefaultTable(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environments: []githubapi.Environment{
				{Name: "Production"},
				{Name: "Development"},
			},
			deployments: map[string][]githubapi.Deployment{
				"Development": {
					{
						ID:        101,
						Ref:       "feature/dev-stable",
						CreatedAt: "2024-03-14T09:26:00Z",
					},
				},
				"Production": {
					{
						ID:        201,
						Ref:       "main",
						CreatedAt: "2024-03-14T10:00:00Z",
					},
				},
			},
			statuses: map[int64][]githubapi.DeploymentStatus{
				101: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T09:30:00Z",
						LogURL:    "https://example.com/dev",
					},
				},
				201: {
					{
						State:     "queued",
						CreatedAt: "2024-03-14T10:05:00Z",
					},
				},
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"octo/example"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	want := "Env | Branch | Date\nDevelopment | feature/dev-stable | 2024-03-14\nProduction | — | —\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPlansPreservesPlanRows(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environments: []githubapi.Environment{
				{Name: "Development/Plan"},
				{Name: "Production"},
				{Name: "Development"},
			},
			deployments: map[string][]githubapi.Deployment{
				"Development": {
					{
						ID:        111,
						Ref:       "feature/dev-stable",
						CreatedAt: "2024-03-14T09:26:00Z",
					},
				},
				"Development/Plan": {
					{
						ID:        112,
						Ref:       "feature/dev-plan",
						CreatedAt: "2024-03-14T09:28:00Z",
					},
				},
				"Production": {
					{
						ID:        113,
						Ref:       "main",
						CreatedAt: "2024-03-14T10:00:00Z",
					},
				},
			},
			statuses: map[int64][]githubapi.DeploymentStatus{
				111: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T09:30:00Z",
					},
				},
				112: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T09:32:00Z",
					},
				},
				113: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T10:05:00Z",
					},
				},
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"--repo", "octo/example", "--plans"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	want := "Env | Branch | Date\nDevelopment | feature/dev-stable | 2024-03-14\nDevelopment/Plan | feature/dev-plan | 2024-03-14\nProduction | main | 2024-03-14\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunJSON(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environments: []githubapi.Environment{
				{Name: "Production"},
			},
			deployments: map[string][]githubapi.Deployment{
				"Production": {
					{
						ID:        301,
						Ref:       "main",
						CreatedAt: "2024-03-14T09:26:00Z",
					},
				},
			},
			statuses: map[int64][]githubapi.DeploymentStatus{
				301: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T09:30:00Z",
						LogURL:    "https://example.com/prod",
					},
				},
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"--repo", "octo/example", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	var decoded []output.ViewRow
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	want := []output.ViewRow{{
		Environment: "Production",
		Branch:      "main",
		Date:        "2024-03-14",
		Status:      "success",
		LogURL:      "https://example.com/prod",
	}}
	if !slices.Equal(decoded, want) {
		t.Fatalf("decoded output = %#v, want %#v", decoded, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunMissingRepoError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stderr.String(), "missing repo target") {
		t.Fatalf("stderr = %q, want missing repo target guidance", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunAuthenticationError(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return nil, errors.New("authentication token not found for host github.com")
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"octo/example"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stderr.String(), "gh authentication unavailable") {
		t.Fatalf("stderr = %q, want authentication guidance", stderr.String())
	}
}

func TestRunRepositoryAccessDeniedError(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environmentsErr: &api.HTTPError{
				StatusCode: 403,
				Message:    "Forbidden",
				RequestURL: mustParseURL(t, "https://api.github.com/repos/octo/example/environments"),
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"octo/example"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stderr.String(), "repository access denied") {
		t.Fatalf("stderr = %q, want repository access denied guidance", stderr.String())
	}
}

func TestRunNoEnvironmentsFoundError(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"octo/example"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stderr.String(), "no environments found") {
		t.Fatalf("stderr = %q, want no environments found guidance", stderr.String())
	}
}

func TestRunNoSuccessfulDeploymentsError(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environments: []githubapi.Environment{{Name: "Production"}},
			deployments: map[string][]githubapi.Deployment{
				"Production": {
					{
						ID:        401,
						Ref:       "main",
						CreatedAt: "2024-03-14T09:26:00Z",
					},
				},
			},
			statuses: map[int64][]githubapi.DeploymentStatus{
				401: {
					{
						State:     "queued",
						CreatedAt: "2024-03-14T09:30:00Z",
					},
				},
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"octo/example"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stderr.String(), "no successful deployments") {
		t.Fatalf("stderr = %q, want no successful deployments guidance", stderr.String())
	}
}

func TestRunPartialEnvironmentFailure(t *testing.T) {
	restore := stubNewGitHubClient(t, func() (githubapi.Client, error) {
		return fixtureClient{
			environments: []githubapi.Environment{
				{Name: "Development"},
				{Name: "UAT"},
			},
			deployments: map[string][]githubapi.Deployment{
				"Development": {
					{
						ID:        501,
						Ref:       "feature/dev-stable",
						CreatedAt: "2024-03-14T09:26:00Z",
					},
				},
			},
			deploymentErrs: map[string]error{
				"UAT": errors.New("backend unavailable"),
			},
			statuses: map[int64][]githubapi.DeploymentStatus{
				501: {
					{
						State:     "success",
						CreatedAt: "2024-03-14T09:30:00Z",
						LogURL:    "https://example.com/dev",
					},
				},
			},
		}, nil
	})
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"--repo", "octo/example", "--verbose"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(stdout.String(), "Development | feature/dev-stable | 2024-03-14 | success | https://example.com/dev") {
		t.Fatalf("stdout = %q, want successful environment row", stdout.String())
	}
	if !strings.Contains(stdout.String(), "UAT | — | — | — | —") {
		t.Fatalf("stdout = %q, want blank UAT row to remain visible", stdout.String())
	}
	if !strings.Contains(stderr.String(), "partial per-environment fetch failure") {
		t.Fatalf("stderr = %q, want partial failure guidance", stderr.String())
	}
}

func stubNewGitHubClient(t *testing.T, fn func() (githubapi.Client, error)) func() {
	t.Helper()

	previous := newGitHubClient
	newGitHubClient = fn

	return func() {
		newGitHubClient = previous
	}
}

type fixtureClient struct {
	environments    []githubapi.Environment
	environmentsErr error
	deployments     map[string][]githubapi.Deployment
	deploymentErrs  map[string]error
	statuses        map[int64][]githubapi.DeploymentStatus
	statusErrs      map[int64]error
}

func (c fixtureClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	if c.environmentsErr != nil {
		return nil, c.environmentsErr
	}
	return slices.Clone(c.environments), nil
}

func (c fixtureClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	if err := c.deploymentErrs[environment]; err != nil {
		return nil, err
	}
	return slices.Clone(c.deployments[environment]), nil
}

func (c fixtureClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	if err := c.statusErrs[deploymentID]; err != nil {
		return nil, err
	}
	return slices.Clone(c.statuses[deploymentID]), nil
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return parsed
}
