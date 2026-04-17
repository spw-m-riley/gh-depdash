package tui

import (
	"context"
	"errors"
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
