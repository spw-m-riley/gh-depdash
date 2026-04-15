package githubapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestListEnvironments(t *testing.T) {
	client := newFixtureClient(t, fixtureResponse{
		path:       "repos/octo/example/environments",
		statusCode:  http.StatusOK,
		body:        loadTestdata(t, "environments.json"),
	})

	got, err := client.ListEnvironments("octo", "example")
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v, want nil", err)
	}

	want := []Environment{{Name: "Development"}, {Name: "Test"}, {Name: "UAT"}, {Name: "Production"}}
	if !slices.Equal(got, want) {
		t.Fatalf("ListEnvironments() = %#v, want %#v", got, want)
	}
}

func TestListDeploymentsByEnvironment(t *testing.T) {
	tests := []struct {
		name       string
		environment string
		fixture    string
		want       []Deployment
	}{
		{
			name:        "development",
			environment: "development",
			fixture:     "deployments-development-success.json",
			want: []Deployment{
				{ID: 1001, Ref: "feature/dev-stable", SHA: "1111111111111111111111111111111111111111", CreatedAt: "2026-04-14T09:05:00Z"},
			},
		},
		{
			name:        "test",
			environment: "test",
			fixture:     "deployments-test-success.json",
			want: []Deployment{
				{ID: 2001, Ref: "feature/test-stable", SHA: "2222222222222222222222222222222222222222", CreatedAt: "2026-04-14T10:05:00Z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFixtureClient(t, fixtureResponse{
				path:        "repos/octo/example/deployments",
				query:       url.Values{"environment": {tt.environment}, "per_page": {"10"}},
				statusCode:  http.StatusOK,
				body:        loadTestdata(t, tt.fixture),
			})

			got, err := client.ListDeployments("octo", "example", tt.environment)
			if err != nil {
				t.Fatalf("ListDeployments() error = %v, want nil", err)
			}

			if !slices.Equal(got, tt.want) {
				t.Fatalf("ListDeployments() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestListDeploymentStatuses(t *testing.T) {
	tests := []struct {
		name         string
		deploymentID int64
		fixture      string
		want         []DeploymentStatus
	}{
		{
			name:         "success",
			deploymentID: 4361877783,
			fixture:      "statuses-uat-success.json",
			want: []DeploymentStatus{
				{State: "success", CreatedAt: "2026-04-14T08:45:00Z", LogURL: "https://logs.example/uat-success"},
			},
		},
		{
			name:         "waiting",
			deploymentID: 4001,
			fixture:      "statuses-production-waiting.json",
			want: []DeploymentStatus{
				{State: "waiting", CreatedAt: "2026-04-14T11:05:00Z", LogURL: "https://logs.example/prod-waiting"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFixtureClient(t, fixtureResponse{
				path:       fmt.Sprintf("repos/octo/example/deployments/%d/statuses", tt.deploymentID),
				query:      url.Values{"per_page": {"10"}},
				statusCode: http.StatusOK,
				body:       loadTestdata(t, tt.fixture),
			})

			got, err := client.ListDeploymentStatuses("octo", "example", tt.deploymentID)
			if err != nil {
				t.Fatalf("ListDeploymentStatuses() error = %v, want nil", err)
			}

			if !slices.Equal(got, tt.want) {
				t.Fatalf("ListDeploymentStatuses() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestClientPropagatesHTTPErrors(t *testing.T) {
	client := newFixtureClient(t, fixtureResponse{
		path:       "repos/octo/example/deployments/4361927516/statuses",
		query:      url.Values{"per_page": {"10"}},
		statusCode:  http.StatusInternalServerError,
		body:       loadTestdata(t, "statuses-uat-error.json"),
	})

	_, err := client.ListDeploymentStatuses("octo", "example", 4361927516)
	if err == nil {
		t.Fatal("ListDeploymentStatuses() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "list deployment statuses for 4361927516") {
		t.Fatalf("error %q does not include endpoint context", err)
	}
}

type fixtureResponse struct {
	path       string
	query      url.Values
	statusCode int
	body       []byte
}

type fixtureRoundTripper struct {
	t        *testing.T
	response fixtureResponse
}

func (rt fixtureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.t.Helper()

	if req.Method != http.MethodGet {
		rt.t.Fatalf("unexpected method %q, want GET", req.Method)
	}
	if got := req.URL.Path; got != "/"+rt.response.path {
		rt.t.Fatalf("unexpected path %q, want %q", got, "/"+rt.response.path)
	}
	gotQuery := req.URL.Query()
	if !slices.Equal(gotQuery["per_page"], rt.response.query["per_page"]) || gotQuery.Get("environment") != rt.response.query.Get("environment") {
		rt.t.Fatalf("unexpected query %q, want %q", gotQuery.Encode(), rt.response.query.Encode())
	}

	return &http.Response{
		StatusCode: rt.response.statusCode,
		Status:     http.StatusText(rt.response.statusCode),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(rt.response.body)),
		Request:    req,
	}, nil
}

func newFixtureClient(t *testing.T, response fixtureResponse) Client {
	t.Helper()

	rest, err := api.NewRESTClient(api.ClientOptions{
		Host:        "github.com",
		AuthToken:   "test-token",
		Transport:   fixtureRoundTripper{t: t, response: response},
		SkipDefaultHeaders: true,
	})
	if err != nil {
		t.Fatalf("NewRESTClient() error = %v", err)
	}

	return &RESTClient{client: rest}
}

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}
