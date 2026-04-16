package tui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
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
	repoList        list.Model
	repoPage        int
	repoHasMore     bool
	repoLoadingMore bool
	selectedRepo    string
	deploymentRows  []output.ViewRow
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

	delegate := itemDelegate{}
	repoList := list.New([]list.Item{}, delegate, 80, 20)
	repoList.Title = "Select a repository"
	repoList.SetShowStatusBar(false)
	repoList.SetFilteringEnabled(true)
	repoList.Styles.Title = titleStyle

	return Model{
		phase:         phaseRepoLoading,
		ctx:           ctx,
		client:        client,
		repoSpinner:   repoSpinner,
		deploySpinner: deploySpinner,
		repoList:      repoList,
		repoPage:      0,
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
		if m.phase == phaseRepoPicker {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				selected := m.repoList.SelectedItem()
				if selected == nil {
					return m, nil
				}
				switch item := selected.(type) {
				case repoItem:
					m.selectedRepo = item.repo.FullName
					m.phase = phaseDeploymentLoading
					return m, tea.Batch(
						m.deploySpinner.Tick,
						loadDeployments(m.ctx, m.client, m.selectedRepo, m.includePlans, m.verbose),
					)
				case loadMoreItem:
					if !m.repoHasMore || m.repoLoadingMore {
						return m, nil
					}
					m.repoLoadingMore = true
					return m, loadMoreRepos(m.ctx, m.client, m.repoPage)
				}
			}
		} else {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "b":
				if m.phase == phaseDeploymentBrowser {
					return m, func() tea.Msg { return backToRepoPickerMsg{} }
				}
			}
		}

	case tea.WindowSizeMsg:
		if m.phase == phaseRepoPicker {
			h, v := docStyle.GetFrameSize()
			m.repoList.SetSize(msg.Width-h, msg.Height-v)
		}

	case repoPageLoadedMsg:
		m.repoPage = 1
		m.repoHasMore = msg.hasMore
		m.repoLoadingMore = false
		items := repoItemsFromRepositories(msg.repos, msg.hasMore)
		m.repoList.SetItems(items)
		m.phase = phaseRepoPicker
		return m, nil

	case repoPageFailedMsg:
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case moreReposLoadedMsg:
		if m.phase != phaseRepoPicker {
			m.repoLoadingMore = false
			return m, nil
		}
		m.repoPage++
		m.repoHasMore = msg.hasMore
		m.repoLoadingMore = false
		currentItems := m.repoList.Items()
		var repoItems []list.Item
		for _, item := range currentItems {
			if _, ok := item.(loadMoreItem); !ok {
				repoItems = append(repoItems, item)
			}
		}
		newItems := repoItemsFromRepositories(msg.repos, msg.hasMore)
		repoItems = append(repoItems, newItems...)
		m.repoList.SetItems(repoItems)
		return m, nil

	case moreReposFailedMsg:
		if m.phase != phaseRepoPicker {
			m.repoLoadingMore = false
			return m, nil
		}
		m.repoLoadingMore = false
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case deploymentsLoadedMsg:
		m.deploymentRows = msg.rows
		m.partialFailures = msg.partialFailures
		m.phase = phaseDeploymentBrowser
		return m, nil

	case deploymentsFatalErrorMsg:
		m.fatalError = msg.err
		m.phase = phaseFatalError
		return m, tea.Quit

	case backToRepoPickerMsg:
		m.phase = phaseRepoPicker
		m.deploymentRows = nil
		m.partialFailures = nil
		m.selectedRepo = ""
		return m, nil

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

	if m.phase == phaseRepoPicker {
		var cmd tea.Cmd
		m.repoList, cmd = m.repoList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.phase {
	case phaseRepoLoading:
		return renderRepoLoading(m.repoSpinner)
	case phaseRepoPicker:
		return docStyle.Render(m.repoList.View())
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
