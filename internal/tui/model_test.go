package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
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

func TestModelRepoPicker(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	desc := "test repo"
	msg := repoPageLoadedMsg{
		repos: []githubapi.Repository{
			{FullName: "owner/repo1", Description: &desc, Private: false, UpdatedAt: "2024-01-01T00:00:00Z"},
			{FullName: "owner/repo2", Description: nil, Private: true, UpdatedAt: "2024-01-02T00:00:00Z"},
		},
		hasMore: true,
	}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseRepoPicker {
		t.Errorf("phase after repoPageLoadedMsg = %v, want %v", um.phase, phaseRepoPicker)
	}
	if um.repoPage != 1 {
		t.Errorf("repoPage = %d, want 1", um.repoPage)
	}
	if !um.repoHasMore {
		t.Error("repoHasMore = false, want true")
	}
	items := um.repoList.Items()
	if len(items) != 3 {
		t.Errorf("repoList items length = %d, want 3 (2 repos + load more)", len(items))
	}
	if cmd != nil {
		t.Errorf("cmd after repoPageLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelAppliesInitialWindowSizeWhenPickerLoads(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	resized, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatalf("cmd after WindowSizeMsg = %v, want nil", cmd)
	}

	desc := "test repo"
	updated, cmd := resized.(Model).Update(repoPageLoadedMsg{
		repos: []githubapi.Repository{
			{FullName: "owner/repo1", Description: &desc},
		},
		hasMore: false,
	})

	um := updated.(Model)
	if um.phase != phaseRepoPicker {
		t.Fatalf("phase after repoPageLoadedMsg = %v, want %v", um.phase, phaseRepoPicker)
	}

	h, v := docStyle.GetFrameSize()
	if got, want := um.repoList.Width(), 120-h; got != want {
		t.Fatalf("repoList.Width() = %d, want %d", got, want)
	}
	if got, want := um.repoList.Height(), 40-v; got != want {
		t.Fatalf("repoList.Height() = %d, want %d", got, want)
	}
	if cmd != nil {
		t.Fatalf("cmd after repoPageLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelLoadMoreRepos(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)

	desc := "first batch"
	msg1 := repoPageLoadedMsg{
		repos: []githubapi.Repository{
			{FullName: "owner/repo1", Description: &desc},
		},
		hasMore: true,
	}
	updated1, _ := m.Update(msg1)
	m = updated1.(Model)

	desc2 := "second batch"
	msg2 := moreReposLoadedMsg{
		repos: []githubapi.Repository{
			{FullName: "owner/repo2", Description: &desc2},
		},
		hasMore: false,
	}
	updated, cmd := m.Update(msg2)

	um := updated.(Model)
	if um.repoPage != 2 {
		t.Errorf("repoPage after moreReposLoadedMsg = %d, want 2", um.repoPage)
	}
	if um.repoHasMore {
		t.Error("repoHasMore = true, want false after final page")
	}
	items := um.repoList.Items()
	if len(items) != 2 {
		t.Errorf("repoList items length = %d, want 2 (2 repos, no load more)", len(items))
	}
	if cmd != nil {
		t.Errorf("cmd after moreReposLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelLoadMoreReposAllowsAdditionalPages(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker
	m.repoPage = 4
	m.repoHasMore = true

	items := make([]list.Item, 0, 121)
	for i := 0; i < 120; i++ {
		repoName := fmt.Sprintf("owner/repo-%d", i)
		items = append(items, repoItem{repo: githubapi.Repository{FullName: repoName}})
	}
	items = append(items, loadMoreItem{})
	m.repoList.SetItems(items)
	m.repoList.Select(len(items) - 1)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	um := updated.(Model)
	if !um.repoLoadingMore {
		t.Fatal("repoLoadingMore = false, want true after requesting another page")
	}
	if cmd == nil {
		t.Fatal("cmd after load-more enter = nil, want non-nil")
	}
}

func TestModelLoadMoreReposIgnoresDuplicateEnterWhileLoading(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker
	m.repoPage = 1
	m.repoHasMore = true
	m.repoLoadingMore = true

	items := []list.Item{
		repoItem{repo: githubapi.Repository{FullName: "owner/repo-1"}},
		loadMoreItem{},
	}
	m.repoList.SetItems(items)
	m.repoList.Select(len(items) - 1)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	um := updated.(Model)
	if !um.repoLoadingMore {
		t.Fatal("repoLoadingMore = false, want true while duplicate load-more is ignored")
	}
	if cmd != nil {
		t.Fatalf("cmd after duplicate load-more enter = %v, want nil", cmd)
	}
}

func TestModelRepoSelectionInvalidatesInFlightLoadMore(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker
	m.repoPickerSession = 1
	m.repoLoadingMore = true

	desc := "test"
	items := repoItemsFromRepositories([]githubapi.Repository{
		{FullName: "owner/repo1", Description: &desc},
	}, false)
	m.repoList.SetItems(items)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	um := updated.(Model)
	if um.phase != phaseDeploymentLoading {
		t.Fatalf("phase after repo selection = %v, want %v", um.phase, phaseDeploymentLoading)
	}
	if um.repoPickerSession != 2 {
		t.Fatalf("repoPickerSession = %d, want 2 after invalidating in-flight load-more requests", um.repoPickerSession)
	}
	if um.repoLoadingMore {
		t.Fatal("repoLoadingMore = true, want false after leaving repo picker")
	}
	if cmd == nil {
		t.Fatal("cmd after repo selection = nil, want deployment loading command")
	}
}

func TestModelIgnoresLateMoreReposLoadedAfterSelection(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentLoading
	m.selectedRepo = "owner/repo-1"
	m.repoPickerSession = 1
	m.repoPage = 1
	m.repoHasMore = true

	items := []list.Item{
		repoItem{repo: githubapi.Repository{FullName: "owner/repo-1"}},
		loadMoreItem{},
	}
	m.repoList.SetItems(items)

	updated, cmd := m.Update(moreReposLoadedMsg{
		sessionID: 1,
		repos: []githubapi.Repository{
			{FullName: "owner/repo-2"},
		},
		hasMore: true,
	})

	um := updated.(Model)
	if um.phase != phaseDeploymentLoading {
		t.Fatalf("phase after late moreReposLoadedMsg = %v, want %v", um.phase, phaseDeploymentLoading)
	}
	if um.selectedRepo != "owner/repo-1" {
		t.Fatalf("selectedRepo = %q, want %q", um.selectedRepo, "owner/repo-1")
	}
	if um.repoPage != 1 {
		t.Fatalf("repoPage = %d, want 1", um.repoPage)
	}
	if um.repoLoadingMore {
		t.Fatal("repoLoadingMore = true, want false after leaving repo picker")
	}
	if cmd != nil {
		t.Fatalf("cmd after late moreReposLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelIgnoresLateMoreReposFailedAfterSelection(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentLoading
	m.selectedRepo = "owner/repo-1"
	m.repoPickerSession = 1

	updated, cmd := m.Update(moreReposFailedMsg{sessionID: 1, err: "network timeout"})

	um := updated.(Model)
	if um.phase != phaseDeploymentLoading {
		t.Fatalf("phase after late moreReposFailedMsg = %v, want %v", um.phase, phaseDeploymentLoading)
	}
	if um.selectedRepo != "owner/repo-1" {
		t.Fatalf("selectedRepo = %q, want %q", um.selectedRepo, "owner/repo-1")
	}
	if um.repoLoadingMore {
		t.Fatal("repoLoadingMore = true, want false after late pagination failure is ignored")
	}
	if um.fatalError != "" {
		t.Fatalf("fatalError = %q, want empty", um.fatalError)
	}
	if cmd != nil {
		t.Fatalf("cmd after late moreReposFailedMsg = %v, want nil", cmd)
	}
}

func TestModelIgnoresStaleMoreReposLoadedAfterReturningToPicker(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker
	m.repoPickerSession = 2
	m.repoPage = 1
	m.repoHasMore = true
	m.repoLoadingMore = true

	items := []list.Item{
		repoItem{repo: githubapi.Repository{FullName: "owner/repo-1"}},
		loadMoreItem{},
	}
	m.repoList.SetItems(items)

	updated, cmd := m.Update(moreReposLoadedMsg{
		sessionID: 1,
		repos: []githubapi.Repository{
			{FullName: "owner/repo-2"},
		},
		hasMore: true,
	})

	um := updated.(Model)
	if um.phase != phaseRepoPicker {
		t.Fatalf("phase after stale moreReposLoadedMsg = %v, want %v", um.phase, phaseRepoPicker)
	}
	if um.repoPage != 1 {
		t.Fatalf("repoPage = %d, want 1 after ignoring stale response", um.repoPage)
	}
	if !um.repoLoadingMore {
		t.Fatal("repoLoadingMore = false, want true so stale success cannot clear the active session guard")
	}
	if len(um.repoList.Items()) != 2 {
		t.Fatalf("repoList items length = %d, want 2 after ignoring stale response", len(um.repoList.Items()))
	}
	if cmd != nil {
		t.Fatalf("cmd after stale moreReposLoadedMsg = %v, want nil", cmd)
	}
}

func TestModelIgnoresStaleMoreReposFailedAfterReturningToPicker(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker
	m.repoPickerSession = 2
	m.repoLoadingMore = true

	updated, cmd := m.Update(moreReposFailedMsg{sessionID: 1, err: "network timeout"})

	um := updated.(Model)
	if um.phase != phaseRepoPicker {
		t.Fatalf("phase after stale moreReposFailedMsg = %v, want %v", um.phase, phaseRepoPicker)
	}
	if um.fatalError != "" {
		t.Fatalf("fatalError = %q, want empty", um.fatalError)
	}
	if !um.repoLoadingMore {
		t.Fatal("repoLoadingMore = false, want true so stale failure cannot clear the active session guard")
	}
	if cmd != nil {
		t.Fatalf("cmd after stale moreReposFailedMsg = %v, want nil", cmd)
	}
}

func TestModelRepoSelection(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseRepoPicker

	desc := "test"
	items := repoItemsFromRepositories([]githubapi.Repository{
		{FullName: "owner/repo1", Description: &desc},
	}, false)
	m.repoList.SetItems(items)

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(keyMsg)

	um := updated.(Model)
	if um.phase != phaseDeploymentLoading {
		t.Errorf("phase after enter key = %v, want %v", um.phase, phaseDeploymentLoading)
	}
	if um.selectedRepo != "owner/repo1" {
		t.Errorf("selectedRepo = %q, want %q", um.selectedRepo, "owner/repo1")
	}
	if cmd == nil {
		t.Error("cmd after enter key = nil, want deployment loading command")
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
		rows: []output.ViewRow{
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

func TestModelDeploymentLoading(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModelForDirectRepo(ctx, client, "owner/repo", false, false)

	if m.phase != phaseDeploymentLoading {
		t.Errorf("phase = %v, want %v", m.phase, phaseDeploymentLoading)
	}

	view := m.View()
	if !strings.Contains(view, "Loading deployments") {
		t.Errorf("deployment loading view missing expected text, got: %s", view)
	}
	if !strings.Contains(view, "owner/repo") {
		t.Errorf("deployment loading view missing repo name, got: %s", view)
	}
}

func TestModelDeploymentBrowser(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentBrowser
	m.deploymentRows = []output.ViewRow{
		{Environment: "Production", Branch: "main", Date: "2024-01-15"},
		{Environment: "Test", Status: "in_progress"},
	}

	view := m.View()
	if !strings.Contains(view, "Production") {
		t.Errorf("deployment browser view missing Production, got: %s", view)
	}
	if !strings.Contains(view, "Test") {
		t.Errorf("deployment browser view missing Test, got: %s", view)
	}
	if !strings.Contains(view, "Press 'b' to go back") {
		t.Errorf("deployment browser view missing back navigation hint, got: %s", view)
	}
}

func TestModelBackNavigation(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentBrowser
	m.selectedRepo = "owner/repo"
	m.deploymentRows = []output.ViewRow{{Environment: "Production"}}
	m.partialFailures = []string{"Dev: error"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	updated, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("Update with 'b' key returned nil cmd, want backToRepoPickerMsg command")
	}

	result := cmd()
	if _, ok := result.(backToRepoPickerMsg); !ok {
		t.Errorf("cmd() returned %T, want backToRepoPickerMsg", result)
	}

	updated2, _ := updated.(Model).Update(result)
	um := updated2.(Model)

	if um.phase != phaseRepoPicker {
		t.Errorf("phase after back navigation = %v, want %v", um.phase, phaseRepoPicker)
	}
	if um.selectedRepo != "" {
		t.Errorf("selectedRepo after back navigation = %q, want empty", um.selectedRepo)
	}
	if len(um.deploymentRows) != 0 {
		t.Errorf("deploymentRows after back navigation = %d items, want 0", len(um.deploymentRows))
	}
	if len(um.partialFailures) != 0 {
		t.Errorf("partialFailures after back navigation = %d items, want 0", len(um.partialFailures))
	}
}

func TestModelPartialFailure(t *testing.T) {
	ctx := context.Background()
	client := &stubClient{}
	m := NewModel(ctx, client, false, false)
	m.phase = phaseDeploymentLoading

	msg := deploymentsLoadedMsg{
		rows: []output.ViewRow{
			{Environment: "Production", Branch: "main", Date: "2024-01-15"},
		},
		partialFailures: []string{"Development: timeout", "Test: network error"},
	}
	updated, cmd := m.Update(msg)

	um := updated.(Model)
	if um.phase != phaseDeploymentBrowser {
		t.Errorf("phase after partial failure = %v, want %v", um.phase, phaseDeploymentBrowser)
	}
	if len(um.deploymentRows) != 1 {
		t.Errorf("deploymentRows count = %d, want 1", len(um.deploymentRows))
	}
	if len(um.partialFailures) != 2 {
		t.Errorf("partialFailures count = %d, want 2", len(um.partialFailures))
	}
	if cmd != nil {
		t.Errorf("cmd after partial failure = %v, want nil", cmd)
	}

	view := um.View()
	if !strings.Contains(view, "Partial failures:") {
		t.Errorf("view missing partial failures section, got: %s", view)
	}
	if !strings.Contains(view, "Development: timeout") {
		t.Errorf("view missing Development failure, got: %s", view)
	}
	if !strings.Contains(view, "Test: network error") {
		t.Errorf("view missing Test failure, got: %s", view)
	}
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

func TestRunHandlesFatalErrorPhase(t *testing.T) {
	ctx := context.Background()
	client := &errorClient{err: "network error"}

	var stdout, stderr stubWriter

	err := Run(ctx, client, false, false, &stdout, &stderr)

	if err == nil {
		t.Fatal("Run() with fatal error phase returned nil, want error")
	}
	expectedErr := "failed to list repositories: network error"
	if err.Error() != expectedErr {
		t.Errorf("Run() error = %q, want %q", err.Error(), expectedErr)
	}
	if stderr.String() != expectedErr+"\n" {
		t.Errorf("stderr = %q, want message written", stderr.String())
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

func (c *stubClient) ListRepositories(page, perPage int) (githubapi.RepositoryPage, error) {
	desc := "test repo"
	return githubapi.RepositoryPage{
		Repositories: []githubapi.Repository{
			{FullName: "owner/repo", Description: &desc},
		},
	}, nil
}

type stubWriter struct {
	buf []byte
}

func (w *stubWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

func (w *stubWriter) String() string {
	return string(w.buf)
}

func (w *stubWriter) Len() int {
	return len(w.buf)
}

type errorClient struct {
	err string
}

func (c *errorClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return nil, fmt.Errorf("%s", c.err)
}

func (c *errorClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return nil, fmt.Errorf("%s", c.err)
}

func (c *errorClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return nil, fmt.Errorf("%s", c.err)
}

func (c *errorClient) ListRepositories(page, perPage int) (githubapi.RepositoryPage, error) {
	return githubapi.RepositoryPage{}, fmt.Errorf("%s", c.err)
}
