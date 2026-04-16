package tui

import (
	"context"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"gh-depdash/internal/deployments"
	"gh-depdash/internal/githubapi"
)

type phase int

const (
	phaseRepoLoading phase = iota
	phaseRepoPicker
	phaseDeploymentLoading
	phaseDeploymentBrowser
	phaseFatalError
)

type Model struct {
	phase           phase
	ctx             context.Context
	client          githubapi.Client
	repoSpinner     spinner.Model
	deploySpinner   spinner.Model
	fatalError      string
	repoPage        []string
	selectedRepo    string
	deploymentRows  []deployments.Row
	partialFailures []string
	includePlans    bool
	verbose         bool
}

func NewModel(ctx context.Context, client githubapi.Client, includePlans, verbose bool) Model {
	repoSpinner := spinner.New()
	repoSpinner.Spinner = spinner.Dot
	repoSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	deploySpinner := spinner.New()
	deploySpinner.Spinner = spinner.Dot
	deploySpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		phase:         phaseRepoLoading,
		ctx:           ctx,
		client:        client,
		repoSpinner:   repoSpinner,
		deploySpinner: deploySpinner,
		includePlans:  includePlans,
		verbose:       verbose,
	}
}

func (m Model) Init() tea.Cmd {
	switch m.phase {
	case phaseRepoLoading:
		return tea.Batch(m.repoSpinner.Tick, loadRepoPage(m.ctx, m.client))
	case phaseDeploymentLoading:
		return tea.Batch(m.deploySpinner.Tick, loadDeployments(m.ctx, m.client, m.selectedRepo, m.includePlans, m.verbose))
	default:
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case repoPageLoadedMsg:
		m.repoPage = msg.repos
		m.phase = phaseRepoPicker
		return m, nil

	case repoPageFailedMsg:
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case deploymentsLoadedMsg:
		m.deploymentRows = msg.rows
		m.partialFailures = msg.partialFailures
		m.phase = phaseDeploymentBrowser
		return m, nil

	case deploymentsPartialFailureMsg:
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case deploymentsFatalErrorMsg:
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case spinner.TickMsg:
		switch m.phase {
		case phaseRepoLoading:
			var cmd tea.Cmd
			m.repoSpinner, cmd = m.repoSpinner.Update(msg)
			return m, cmd
		case phaseDeploymentLoading:
			var cmd tea.Cmd
			m.deploySpinner, cmd = m.deploySpinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.phase {
	case phaseRepoLoading:
		return renderRepoLoading(m.repoSpinner)
	case phaseRepoPicker:
		return renderRepoPicker(m.repoPage)
	case phaseDeploymentLoading:
		return renderDeploymentLoading(m.deploySpinner, m.selectedRepo)
	case phaseDeploymentBrowser:
		return renderDeploymentBrowser(m.deploymentRows, m.partialFailures, m.verbose)
	case phaseFatalError:
		return renderFatalError(m.fatalError)
	default:
		return ""
	}
}

func NewModelForDirectRepo(ctx context.Context, client githubapi.Client, repo string, includePlans, verbose bool) Model {
	m := NewModel(ctx, client, includePlans, verbose)
	m.phase = phaseDeploymentLoading
	m.selectedRepo = repo
	return m
}

var newProgram = func(m Model, opts ...tea.ProgramOption) *tea.Program {
	return tea.NewProgram(m, opts...)
}

func Run(ctx context.Context, client githubapi.Client, includePlans, verbose bool, stdout, stderr io.Writer) error {
	m := NewModel(ctx, client, includePlans, verbose)
	opts := []tea.ProgramOption{
		tea.WithOutput(stdout),
	}
	p := newProgram(m, opts...)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if fm, ok := finalModel.(Model); ok && fm.phase == phaseFatalError {
		if stderr != nil {
			_, _ = io.WriteString(stderr, fm.fatalError+"\n")
		}
		return fmt.Errorf("%s", fm.fatalError)
	}

	return nil
}
