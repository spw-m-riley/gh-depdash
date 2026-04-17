package tui

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

func TestLoadRepoPageAvoidsDuplicateRepositoryPrefix(t *testing.T) {
	msg := loadRepoPage(context.Background(), &errorClient{err: "list repositories: network error"})()

	failedMsg, ok := msg.(repoPageFailedMsg)
	if !ok {
		t.Fatalf("message type = %T, want repoPageFailedMsg", msg)
	}
	want := "failed to list repositories: network error"
	if failedMsg.err != want {
		t.Fatalf("repoPageFailedMsg.err = %q, want %q", failedMsg.err, want)
	}
}

func TestLoadMoreReposAvoidsDuplicateRepositoryPrefix(t *testing.T) {
	msg := loadMoreRepos(context.Background(), &errorClient{err: "list repositories: network error"}, 1, 7)()

	failedMsg, ok := msg.(moreReposFailedMsg)
	if !ok {
		t.Fatalf("message type = %T, want moreReposFailedMsg", msg)
	}
	if failedMsg.sessionID != 7 {
		t.Fatalf("moreReposFailedMsg.sessionID = %d, want 7", failedMsg.sessionID)
	}
	want := "failed to load more repositories: network error"
	if failedMsg.err != want {
		t.Fatalf("moreReposFailedMsg.err = %q, want %q", failedMsg.err, want)
	}
}

func TestLoadRepoPageDoesNotAssumeMoreFromExactPageSize(t *testing.T) {
	msg := loadRepoPage(context.Background(), repoPageClient{
		pages: map[int][]githubapi.Repository{
			1: makeRepositories(perPage),
		},
	})()

	loadedMsg, ok := msg.(repoPageLoadedMsg)
	if !ok {
		t.Fatalf("message type = %T, want repoPageLoadedMsg", msg)
	}
	if loadedMsg.hasMore {
		t.Fatal("repoPageLoadedMsg.hasMore = true, want false for an exact final page")
	}
}

func TestLoadMoreReposDoesNotAssumeMoreFromExactPageSize(t *testing.T) {
	msg := loadMoreRepos(context.Background(), repoPageClient{
		pages: map[int][]githubapi.Repository{
			2: makeRepositories(perPage),
		},
	}, 1, 7)()

	loadedMsg, ok := msg.(moreReposLoadedMsg)
	if !ok {
		t.Fatalf("message type = %T, want moreReposLoadedMsg", msg)
	}
	if loadedMsg.hasMore {
		t.Fatal("moreReposLoadedMsg.hasMore = true, want false for an exact final page")
	}
}

func TestLoadDeploymentsPreservesActionableErrors(t *testing.T) {
	previous := loadDeploymentsForRepo
	loadDeploymentsForRepo = func(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]output.ViewRow, []string, error) {
		return nil, nil, errors.New("repository access denied for octo/example: verify the repository exists and your gh auth can read it")
	}
	t.Cleanup(func() {
		loadDeploymentsForRepo = previous
	})

	msg := loadDeployments(context.Background(), &stubClient{}, "octo/example", false, false)()

	fatalMsg, ok := msg.(deploymentsFatalErrorMsg)
	if !ok {
		t.Fatalf("message type = %T, want deploymentsFatalErrorMsg", msg)
	}
	want := "repository access denied for octo/example: verify the repository exists and your gh auth can read it"
	if fatalMsg.err != want {
		t.Fatalf("deploymentsFatalErrorMsg.err = %q, want %q", fatalMsg.err, want)
	}
}

func TestLoadDeploymentsRejectsInvalidRepoTarget(t *testing.T) {
	msg := loadDeployments(context.Background(), &stubClient{}, "octo/example/extra", false, false)()

	fatalMsg, ok := msg.(deploymentsFatalErrorMsg)
	if !ok {
		t.Fatalf("message type = %T, want deploymentsFatalErrorMsg", msg)
	}
	want := `invalid repo target "octo/example/extra": expected <owner/repo>`
	if fatalMsg.err != want {
		t.Fatalf("deploymentsFatalErrorMsg.err = %q, want %q", fatalMsg.err, want)
	}
}

type repoPageClient struct {
	pages   map[int][]githubapi.Repository
	hasMore map[int]bool
}

func (c repoPageClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return nil, fmt.Errorf("unexpected ListEnvironments call")
}

func (c repoPageClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return nil, fmt.Errorf("unexpected ListDeployments call")
}

func (c repoPageClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return nil, fmt.Errorf("unexpected ListDeploymentStatuses call")
}

func (c repoPageClient) ListRepositories(page, perPage int) (githubapi.RepositoryPage, error) {
	return githubapi.RepositoryPage{
		Repositories: c.pages[page],
		HasMore:      c.hasMore[page],
	}, nil
}

func makeRepositories(count int) []githubapi.Repository {
	repos := make([]githubapi.Repository, 0, count)
	for i := 0; i < count; i++ {
		repos = append(repos, githubapi.Repository{FullName: fmt.Sprintf("octo/repo-%d", i)})
	}
	return repos
}
