package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gh-depdash/internal/deployments"
	"gh-depdash/internal/githubapi"
)

func TestModelInitPhaseRepoLoading(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	if m.phase != phaseRepoLoading {
		t.Errorf("NewModel phase = %v, want %v", m.phase, phaseRepoLoading)
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd, want non-nil batch command")
	}
}

func TestModelInitPhaseDeploymentLoading(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModelForDirectRepo(ctx, client, "owner/repo", false, false)

	if m.phase != phaseDeploymentLoading {
		t.Errorf("NewModelForDirectRepo phase = %v, want %v", m.phase, phaseDeploymentLoading)
	}
	if m.selectedRepo != "owner/repo" {
		t.Errorf("NewModelForDirectRepo selectedRepo = %q, want %q", m.selectedRepo, "owner/repo")
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd, want non-nil batch command")
	}
}

func TestModelUpdateRepoPageLoaded(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	msg := repoPageLoadedMsg{repos: []string{"owner/repo1", "owner/repo2"}}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseRepoPicker {
		t.Errorf("phase after repoPageLoadedMsg = %v, want %v", um.phase, phaseRepoPicker)
	}
	if len(um.repoPage) != 2 {
		t.Errorf("repoPage length = %d, want 2", len(um.repoPage))
	}
	if cmd != nil {
		t.Errorf("cmd after repoPageLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelUpdateRepoPageFailed(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	msg := repoPageFailedMsg{err: "network error"}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseFatalError {
		t.Errorf("phase after repoPageFailedMsg = %v, want %v", um.phase, phaseFatalError)
	}
	if um.fatalError != "network error" {
		t.Errorf("fatalError = %q, want %q", um.fatalError, "network error")
	}
	if cmd == nil {
		t.Error("cmd after repoPageFailedMsg = nil, want tea.Quit")
	}
}

func TestModelUpdateDeploymentsLoaded(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentLoading

	msg := deploymentsLoadedMsg{
		rows: []deployments.Row{
			{Environment: "Production"},
		},
		partialFailures: []string{"Dev: timeout"},
	}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseDeploymentBrowser {
		t.Errorf("phase after deploymentsLoadedMsg = %v, want %v", um.phase, phaseDeploymentBrowser)
	}
	if len(um.deploymentRows) != 1 {
		t.Errorf("deploymentRows length = %d, want 1", len(um.deploymentRows))
	}
	if len(um.partialFailures) != 1 {
		t.Errorf("partialFailures length = %d, want 1", len(um.partialFailures))
	}
	if cmd != nil {
		t.Errorf("cmd after deploymentsLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelUpdateDeploymentsFatalError(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentLoading

	msg := deploymentsFatalErrorMsg{err: "auth failed"}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseFatalError {
		t.Errorf("phase after deploymentsFatalErrorMsg = %v, want %v", um.phase, phaseFatalError)
	}
	if um.fatalError != "auth failed" {
		t.Errorf("fatalError = %q, want %q", um.fatalError, "auth failed")
	}
	if cmd == nil {
		t.Error("cmd after deploymentsFatalErrorMsg = nil, want tea.Quit")
	}
}

func TestModelView(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	tests := []struct {
		name  string
		phase phase
	}{
		{"repo loading", phaseRepoLoading},
		{"repo picker", phaseRepoPicker},
		{"deployment loading", phaseDeploymentLoading},
		{"deployment browser", phaseDeploymentBrowser},
		{"fatal error", phaseFatalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.phase = tt.phase
			if tt.phase == phaseDeploymentLoading {
				m.selectedRepo = "owner/repo"
			}
			if tt.phase == phaseFatalError {
				m.fatalError = "test error"
			}
			view := m.View()
			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

type stubClient struct{}

func (c *stubClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return []githubapi.Environment{{Name: "Production"}}, nil
}

func (c *stubClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return []githubapi.Deployment{{ID: 1, Ref: "main"}}, nil
}

func (c *stubClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return []githubapi.DeploymentStatus{{State: "success"}}, nil
}

func (c *stubClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return []githubapi.Repository{{FullName: "owner/repo"}}, nil
}

func TestNewProgramSeam(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	called := false
	previous := newProgram
	newProgram = func(model Model, opts ...tea.ProgramOption) *tea.Program {
		called = true
		return previous(model, opts...)
	}
	defer func() { newProgram = previous }()

	_ = newProgram(m)
	if !called {
		t.Error("newProgram seam was not invoked")
	}
}
