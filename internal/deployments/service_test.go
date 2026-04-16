package deployments

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"gh-depdash/internal/githubapi"
)

func TestBuildRowsHidesPlanEnvironmentsByDefault(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", false)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	got := rowEnvironments(rows)
	want := []string{"Development", "Test", "UAT", "Production", "Sandbox"}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildRows() environments = %v, want %v", got, want)
	}
}

func TestBuildRowsKeepsPlanEnvironmentsWhenRequested(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", true)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	got := rowEnvironments(rows)
	want := []string{"Development", "Development/Plan", "Test", "Test/Plan", "UAT", "Production", "Sandbox"}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildRows() environments = %v, want %v", got, want)
	}
}

func TestBuildRowsUsesLatestSuccessfulDeploymentForUAT(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", false)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	row := findRow(t, rows, "UAT")
	if row.Branch != "release-3.1.0" {
		t.Fatalf("UAT Branch = %q, want %q", row.Branch, "release-3.1.0")
	}
	if row.Status != "success" {
		t.Fatalf("UAT Status = %q, want %q", row.Status, "success")
	}
	if !row.HasSuccess {
		t.Fatal("UAT HasSuccess = false, want true")
	}

	wantDate := mustParseTime(t, "2026-04-14T08:41:51Z")
	if !row.Date.Equal(wantDate) {
		t.Fatalf("UAT Date = %s, want %s", row.Date.Format(time.RFC3339), wantDate.Format(time.RFC3339))
	}
}

func TestBuildRowsTreatsInactiveAsHistoricalSuccess(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", false)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	row := findRow(t, rows, "Development")
	if row.Status != "inactive" {
		t.Fatalf("Development Status = %q, want %q", row.Status, "inactive")
	}
	if row.Branch != "feature/dev-stable" {
		t.Fatalf("Development Branch = %q, want %q", row.Branch, "feature/dev-stable")
	}
	if !row.HasSuccess {
		t.Fatal("Development HasSuccess = false, want true")
	}
}

func TestBuildRowsLeavesProductionBlankWhenOnlyWaitingAttemptExists(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", false)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	row := findRow(t, rows, "Production")
	if row.Branch != "" {
		t.Fatalf("Production Branch = %q, want blank", row.Branch)
	}
	if !row.Date.IsZero() {
		t.Fatalf("Production Date = %s, want zero time", row.Date.Format(time.RFC3339))
	}
	if row.Status != "queued" {
		t.Fatalf("Production Status = %q, want %q", row.Status, "queued")
	}
	if row.HasSuccess {
		t.Fatal("Production HasSuccess = true, want false")
	}
}

func TestBuildRowsSortsStableEnvironmentsInExpectedOrder(t *testing.T) {
	rows, err := newFixtureService(t).BuildRows(context.Background(), "octo", "example", false)
	if err != nil {
		t.Fatalf("BuildRows() error = %v, want nil", err)
	}

	got := rowEnvironments(rows)
	want := []string{"Development", "Test", "UAT", "Production", "Sandbox"}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildRows() environments = %v, want %v", got, want)
	}
}

type fixtureClient struct {
	environments []githubapi.Environment
	deployments  map[string][]githubapi.Deployment
	statuses     map[int64][]githubapi.DeploymentStatus
}

func (c fixtureClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return slices.Clone(c.environments), nil
}

func (c fixtureClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return slices.Clone(c.deployments[environment]), nil
}

func (c fixtureClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return slices.Clone(c.statuses[deploymentID]), nil
}

func (c fixtureClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return nil, nil
}

func newFixtureService(t *testing.T) Service {
	t.Helper()

	return Service{Client: fixtureClient{
		environments: []githubapi.Environment{
			{Name: "Sandbox"},
			{Name: "UAT"},
			{Name: "Production"},
			{Name: "Development/Plan"},
			{Name: "Development"},
			{Name: "Test/Plan"},
			{Name: "Test"},
		},
		deployments: map[string][]githubapi.Deployment{
			"Development":      loadDeploymentsFixture(t, "deployments-development.json"),
			"Test":             loadDeploymentsFixture(t, "deployments-test.json"),
			"UAT":              loadDeploymentsFixture(t, "deployments-uat.json"),
			"Production":       loadDeploymentsFixture(t, "deployments-production.json"),
			"Development/Plan": {{ID: 1101, Ref: "plan/dev", CreatedAt: "2026-04-14T06:05:00Z"}},
			"Test/Plan":        {{ID: 2101, Ref: "plan/test", CreatedAt: "2026-04-14T07:05:00Z"}},
			"Sandbox":          {{ID: 5101, Ref: "feature/sandbox", CreatedAt: "2026-04-14T12:05:00Z"}},
		},
		statuses: map[int64][]githubapi.DeploymentStatus{
			1001: {{State: "inactive", CreatedAt: "2026-04-14T09:05:00Z", LogURL: "https://logs.example/dev"}},
			2001: {{State: "success", CreatedAt: "2026-04-14T10:05:00Z", LogURL: "https://logs.example/test"}},
			2101: {{State: "success", CreatedAt: "2026-04-14T07:06:00Z", LogURL: "https://logs.example/test-plan"}},
			1101: {{State: "success", CreatedAt: "2026-04-14T06:06:00Z", LogURL: "https://logs.example/dev-plan"}},
			4361927516: {
				{State: "failure", CreatedAt: "2026-04-14T08:50:00Z", LogURL: "https://logs.example/uat-failed"},
				{State: "in_progress", CreatedAt: "2026-04-14T08:47:00Z", LogURL: "https://logs.example/uat-progress"},
			},
			4361877783: {{State: "success", CreatedAt: "2026-04-14T08:45:00Z", LogURL: "https://logs.example/uat-success"}},
			4001:       {{State: "queued", CreatedAt: "2026-04-14T11:05:00Z", LogURL: "https://logs.example/prod-queued"}},
			5101:       {{State: "success", CreatedAt: "2026-04-14T12:06:00Z", LogURL: "https://logs.example/sandbox"}},
		},
	}}
}

func loadDeploymentsFixture(t *testing.T, name string) []githubapi.Deployment {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var deployments []githubapi.Deployment
	if err := json.Unmarshal(data, &deployments); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
	}

	return deployments
}

func rowEnvironments(rows []Row) []string {
	environments := make([]string, 0, len(rows))
	for _, row := range rows {
		environments = append(environments, row.Environment)
	}
	return environments
}

func findRow(t *testing.T, rows []Row, environment string) Row {
	t.Helper()

	for _, row := range rows {
		if row.Environment == environment {
			return row
		}
	}

	t.Fatalf("row for environment %q not found", environment)
	return Row{}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}

	return parsed
}
