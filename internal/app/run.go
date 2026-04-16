package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"golang.org/x/term"

	"gh-depdash/internal/cli"
	"gh-depdash/internal/deployments"
	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
	"gh-depdash/internal/tui"
)

var (
	parseOptions    = cli.Parse
	newGitHubClient = func() (githubapi.Client, error) {
		return githubapi.NewRESTClient()
	}
	isInteractiveTTY = func() bool {
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
	runInteractive = func(stdout, stderr io.Writer) error {
		ctx := context.Background()
		client, err := newGitHubClient()
		if err != nil {
			return fmt.Errorf("gh authentication unavailable: %w", err)
		}
		tui.SetDeploymentLoader(LoadDeploymentsForRepo)
		return tui.Run(ctx, client, false, false, stdout, stderr)
	}
)

func Run(args []string, stdout, stderr io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	if opts.Repo == "" {
		if opts.JSON {
			return writeActionableError(stderr, "missing repo target: pass <owner/repo> or use --repo <owner/repo>")
		}
		if isInteractiveTTY() {
			if err := runInteractive(stdout, stderr); err != nil {
				return writeActionableError(stderr, err.Error())
			}
			return nil
		}
		return writeActionableError(stderr, "missing repo target: pass <owner/repo> or use --repo <owner/repo>")
	}

	owner, repo, err := resolveRepoTarget(opts.Repo)
	if err != nil {
		return writeActionableError(stderr, err.Error())
	}

	client, err := newGitHubClient()
	if err != nil {
		return writeActionableError(stderr, authUnavailableMessage(err))
	}

	rows, partialFailures, err := buildRows(context.Background(), client, owner, repo, opts.IncludePlans, opts.Verbose)
	if err != nil {
		return writeActionableError(stderr, classifyBuildError(owner, repo, err))
	}

	rendered, err := renderRows(rows, opts.JSON, opts.Verbose)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(stdout, rendered); err != nil {
		return err
	}

	if len(partialFailures) > 0 {
		return writeActionableError(stderr, fmt.Sprintf("partial per-environment fetch failure: %s", strings.Join(partialFailures, "; ")))
	}
	if successfulRows(rows) == 0 {
		return writeActionableError(stderr, fmt.Sprintf("environment returned no successful deployments for %s/%s", owner, repo))
	}

	return nil
}

func resolveRepoTarget(target string) (string, string, error) {
	owner, repo, ok := strings.Cut(target, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("invalid repo target %q: expected <owner/repo>", target)
	}

	return owner, repo, nil
}

func buildRows(ctx context.Context, client githubapi.Client, owner, repo string, includePlans, verbose bool) ([]deployments.Row, []string, error) {
	orderingService := deployments.Service{Client: orderingClient{base: client}}
	orderedRows, err := orderingService.BuildRows(ctx, owner, repo, includePlans)
	if err != nil {
		return nil, nil, err
	}
	if len(orderedRows) == 0 {
		return nil, nil, errNoEnvironments
	}

	rows := make([]deployments.Row, 0, len(orderedRows))
	partialFailures := make([]string, 0)

	for _, orderedRow := range orderedRows {
		service := deployments.Service{Client: singleEnvironmentClient{
			base:        client,
			environment: orderedRow.Environment,
		}}

		builtRows, err := service.BuildRows(ctx, owner, repo, true)
		if err != nil {
			partialFailures = append(partialFailures, fmt.Sprintf("%s: %v", orderedRow.Environment, err))
			if verbose {
				rows = append(rows, deployments.Row{Environment: orderedRow.Environment})
			}
			continue
		}
		if len(builtRows) == 0 {
			if verbose {
				rows = append(rows, deployments.Row{Environment: orderedRow.Environment})
			}
			continue
		}

		rows = append(rows, builtRows[0])
	}

	return rows, partialFailures, nil
}

func renderRows(rows []deployments.Row, asJSON, verbose bool) (string, error) {
	if asJSON {
		payload, err := output.RenderJSON(rows)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	}

	return output.RenderTable(rows, verbose), nil
}

func successfulRows(rows []deployments.Row) int {
	count := 0
	for _, row := range rows {
		if row.HasSuccess {
			count++
		}
	}
	return count
}

func classifyBuildError(owner, repo string, err error) string {
	switch {
	case isAuthenticationError(err):
		return authUnavailableMessage(err)
	case isRepositoryAccessError(err):
		return fmt.Sprintf("repository access denied for %s/%s: verify the repository exists and your gh auth can read it", owner, repo)
	case errors.Is(err, errNoEnvironments):
		return fmt.Sprintf("no environments found for %s/%s", owner, repo)
	default:
		return err.Error()
	}
}

func authUnavailableMessage(err error) string {
	return fmt.Sprintf("gh authentication unavailable: run `gh auth login` or set GH_TOKEN (%v)", err)
}

func isAuthenticationError(err error) bool {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == 401 {
		return true
	}
	return strings.Contains(err.Error(), "authentication token not found")
}

func isRepositoryAccessError(err error) bool {
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == 403 || httpErr.StatusCode == 404
}

func writeActionableError(stderr io.Writer, message string) error {
	if stderr != nil {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return errors.New(message)
}

type orderingClient struct {
	base githubapi.Client
}

func (c orderingClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return c.base.ListEnvironments(owner, repo)
}

func (orderingClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return nil, nil
}

func (orderingClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return nil, nil
}

func (c orderingClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return c.base.ListRepositories(page, perPage)
}

type singleEnvironmentClient struct {
	base        githubapi.Client
	environment string
}

func (c singleEnvironmentClient) ListEnvironments(owner, repo string) ([]githubapi.Environment, error) {
	return []githubapi.Environment{{Name: c.environment}}, nil
}

func (c singleEnvironmentClient) ListDeployments(owner, repo, environment string) ([]githubapi.Deployment, error) {
	return c.base.ListDeployments(owner, repo, c.environment)
}

func (c singleEnvironmentClient) ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]githubapi.DeploymentStatus, error) {
	return c.base.ListDeploymentStatuses(owner, repo, deploymentID)
}

func (c singleEnvironmentClient) ListRepositories(page, perPage int) ([]githubapi.Repository, error) {
	return c.base.ListRepositories(page, perPage)
}

var errNoEnvironments = errors.New("no environments found")

