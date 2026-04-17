package tui

import (
	"context"
	"errors"
	"testing"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

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
