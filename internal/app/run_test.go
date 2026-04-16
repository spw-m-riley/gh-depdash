package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
	restoreTTY := stubIsInteractiveTTY(t, false)
	defer restoreTTY()

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

func TestRunMissingRepoWithJSONError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"--json"}, &stdout, &stderr)
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

func TestRunInteractiveLaunch(t *testing.T) {
	restoreTTY := stubIsInteractiveTTY(t, true)
	defer restoreTTY()

	interactiveCalled := false
	restoreInteractive := stubRunInteractive(t, func(stdout, stderr io.Writer) error {
		interactiveCalled = true
		_, _ = io.WriteString(stdout, "interactive mode\n")
		return nil
	})
	defer restoreInteractive()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if !interactiveCalled {
		t.Fatal("runInteractive was not called")
	}
	if !strings.Contains(stdout.String(), "interactive mode") {
		t.Fatalf("stdout = %q, want interactive mode output", stdout.String())
	}
}

func TestRunInteractiveError(t *testing.T) {
	restoreTTY := stubIsInteractiveTTY(t, true)
	defer restoreTTY()

	restoreInteractive := stubRunInteractive(t, func(stdout, stderr io.Writer) error {
		return errors.New("TUI startup failed")
	})
	defer restoreInteractive()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "TUI startup failed") {
		t.Fatalf("error = %v, want TUI startup failed", err)
	}
	if !strings.Contains(stderr.String(), "TUI startup failed") {
		t.Fatalf("stderr = %q, want TUI startup failed written to stderr", stderr.String())
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

func stubIsInteractiveTTY(t *testing.T, interactive bool) func() {
	t.Helper()

	previous := isInteractiveTTY
	isInteractiveTTY = func() bool {
		return interactive
	}

	return func() {
		isInteractiveTTY = previous
	}
}

func stubRunInteractive(t *testing.T, fn func(stdout, stderr io.Writer) error) func() {
	t.Helper()

	previous := runInteractive
	runInteractive = fn

	return func() {
		runInteractive = previous
	}
}

type fixtureClient struct {
	environments    []githubapi.Environment
	environmentsErr error
	deployments     map[string][]githubapi.Deployment
	deploymentErrs  map[string]error
	statuses        map[int64][]githubapi.DeploymentStatus
	statusErrs      map[int64]error
	repositories    []githubapi.Repository
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

func (c fixtureClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	if c.repositories == nil {
		return nil, nil
	}
	return slices.Clone(c.repositories), nil
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return parsed
}

func TestLoadInitialRepositories(t *testing.T) {
	desc1 := "First repo description"
	desc2 := "Second repo description"
	client := fixtureClient{
		repositories: []githubapi.Repository{
			{
				FullName:    "octo/first",
				Description: &desc1,
			},
			{
				FullName:    "octo/second",
				Description: &desc2,
			},
		},
	}

	items, err := LoadInitialRepositories(context.Background(), client)
	if err != nil {
		t.Fatalf("LoadInitialRepositories() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("LoadInitialRepositories() returned %d items, want 2", len(items))
	}

	want := []RepositoryItem{
		{FullName: "octo/first", Description: "First repo description"},
		{FullName: "octo/second", Description: "Second repo description"},
	}

	for i, item := range items {
		if item != want[i] {
			t.Errorf("item[%d] = %+v, want %+v", i, item, want[i])
		}
	}
}

func TestLoadInitialRepositoriesHandlesNilDescription(t *testing.T) {
	client := fixtureClient{
		repositories: []githubapi.Repository{
			{
				FullName:    "octo/nodesc",
				Description: nil,
			},
		},
	}

	items, err := LoadInitialRepositories(context.Background(), client)
	if err != nil {
		t.Fatalf("LoadInitialRepositories() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("LoadInitialRepositories() returned %d items, want 1", len(items))
	}

	if items[0].Description != "" {
		t.Errorf("items[0].Description = %q, want empty string", items[0].Description)
	}
}

func TestLoadMoreRepositories(t *testing.T) {
	desc := "Paged repo"
	client := fixtureClient{
		repositories: []githubapi.Repository{
			{
				FullName:    "octo/paged",
				Description: &desc,
			},
		},
	}

	items, err := LoadMoreRepositories(context.Background(), client, 2, 50)
	if err != nil {
		t.Fatalf("LoadMoreRepositories() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("LoadMoreRepositories() returned %d items, want 1", len(items))
	}

	want := RepositoryItem{FullName: "octo/paged", Description: "Paged repo"}
	if items[0] != want {
		t.Errorf("items[0] = %+v, want %+v", items[0], want)
	}
}

func TestLoadDeploymentsForRepo(t *testing.T) {
	client := fixtureClient{
		environments: []githubapi.Environment{
			{Name: "Production"},
		},
		deployments: map[string][]githubapi.Deployment{
			"Production": {
				{
					ID:        201,
					Ref:       "main",
					CreatedAt: "2024-03-14T10:00:00Z",
				},
			},
		},
		statuses: map[int64][]githubapi.DeploymentStatus{
			201: {
				{
					State:     "success",
					CreatedAt: "2024-03-14T10:05:00Z",
					LogURL:    "https://example.com/log",
				},
			},
		},
	}

	items, partialFailures, err := LoadDeploymentsForRepo(context.Background(), client, "octo", "example", false, false)
	if err != nil {
		t.Fatalf("LoadDeploymentsForRepo() error = %v", err)
	}

	if len(partialFailures) != 0 {
		t.Errorf("LoadDeploymentsForRepo() returned %d partial failures, want 0", len(partialFailures))
	}

	if len(items) != 1 {
		t.Fatalf("LoadDeploymentsForRepo() returned %d items, want 1", len(items))
	}

	want := DeploymentItem{
		Environment: "Production",
		Branch:      "main",
		Date:        "2024-03-14",
		Status:      "success",
		LogURL:      "https://example.com/log",
	}

	if items[0] != want {
		t.Errorf("items[0] = %+v, want %+v", items[0], want)
	}
}

func TestLoadDeploymentsForRepoPreservesPartialFailures(t *testing.T) {
	client := fixtureClient{
		environments: []githubapi.Environment{
			{Name: "Development"},
			{Name: "UAT"},
			{Name: "Production"},
		},
		deployments: map[string][]githubapi.Deployment{
			"Development": {
				{
					ID:        101,
					Ref:       "feature/dev",
					CreatedAt: "2024-03-14T09:00:00Z",
				},
			},
			"Production": {
				{
					ID:        301,
					Ref:       "main",
					CreatedAt: "2024-03-14T12:00:00Z",
				},
			},
		},
		deploymentErrs: map[string]error{
			"UAT": errors.New("environment temporarily unavailable"),
		},
		statuses: map[int64][]githubapi.DeploymentStatus{
			101: {
				{
					State:     "success",
					CreatedAt: "2024-03-14T09:05:00Z",
					LogURL:    "https://example.com/dev-log",
				},
			},
			301: {
				{
					State:     "success",
					CreatedAt: "2024-03-14T12:05:00Z",
					LogURL:    "https://example.com/prod-log",
				},
			},
		},
	}

	items, partialFailures, err := LoadDeploymentsForRepo(context.Background(), client, "octo", "example", false, true)
	if err != nil {
		t.Fatalf("LoadDeploymentsForRepo() error = %v, want nil", err)
	}

	if len(partialFailures) != 1 {
		t.Fatalf("LoadDeploymentsForRepo() returned %d partial failures, want 1", len(partialFailures))
	}

	if !strings.Contains(partialFailures[0], "UAT") {
		t.Errorf("partial failure %q does not mention UAT", partialFailures[0])
	}
	if !strings.Contains(partialFailures[0], "temporarily unavailable") {
		t.Errorf("partial failure %q does not contain error message", partialFailures[0])
	}

	if len(items) != 3 {
		t.Fatalf("LoadDeploymentsForRepo() returned %d items, want 3", len(items))
	}

	wantDev := DeploymentItem{
		Environment: "Development",
		Branch:      "feature/dev",
		Date:        "2024-03-14",
		Status:      "success",
		LogURL:      "https://example.com/dev-log",
	}
	if items[0] != wantDev {
		t.Errorf("items[0] = %+v, want %+v", items[0], wantDev)
	}

	wantUAT := DeploymentItem{
		Environment: "UAT",
	}
	if items[1] != wantUAT {
		t.Errorf("items[1] = %+v, want blank UAT row %+v", items[1], wantUAT)
	}

	wantProd := DeploymentItem{
		Environment: "Production",
		Branch:      "main",
		Date:        "2024-03-14",
		Status:      "success",
		LogURL:      "https://example.com/prod-log",
	}
	if items[2] != wantProd {
		t.Errorf("items[2] = %+v, want %+v", items[2], wantProd)
	}
}


