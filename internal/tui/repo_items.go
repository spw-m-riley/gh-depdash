package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gh-depdash/internal/githubapi"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	descStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	privateStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	loadMoreStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).PaddingLeft(4)
)

type repoItem struct {
	repo githubapi.Repository
}

func (i repoItem) FilterValue() string {
	return i.repo.FullName
}

type loadMoreItem struct{}

func (i loadMoreItem) FilterValue() string {
	return ""
}

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 2 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	switch item := listItem.(type) {
	case repoItem:
		renderRepoItem(w, m, index, item)
	case loadMoreItem:
		renderLoadMoreItem(w, m, index)
	}
}

func renderRepoItem(w io.Writer, m list.Model, index int, item repoItem) {
	var title, desc string

	title = item.repo.FullName

	var parts []string
	if item.repo.Description != nil && *item.repo.Description != "" {
		if description := normalizeRepoDescription(*item.repo.Description); description != "" {
			parts = append(parts, description)
		}
	}

	if item.repo.Private {
		parts = append(parts, privateStyle.Render("private"))
	}

	if item.repo.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.repo.UpdatedAt); err == nil {
			parts = append(parts, fmt.Sprintf("updated %s", formatTimeAgo(t)))
		}
	}

	desc = descStyle.Render(strings.Join(parts, " • "))

	str := fmt.Sprintf("%s\n%s", title, desc)

	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("▸ "+str))
	} else {
		fmt.Fprint(w, itemStyle.Render(str))
	}
}

func renderLoadMoreItem(w io.Writer, m list.Model, index int) {
	str := "Load more repositories..."
	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("▸ "+str))
	} else {
		fmt.Fprint(w, loadMoreStyle.Render(str))
	}
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	d := now.Sub(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	if days < 365 {
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func repoItemsFromRepositories(repos []githubapi.Repository, hasMore bool) []list.Item {
	items := make([]list.Item, 0, len(repos)+1)
	for _, r := range repos {
		items = append(items, repoItem{repo: r})
	}
	if hasMore {
		items = append(items, loadMoreItem{})
	}
	return items
}

func normalizeRepoDescription(description string) string {
	return strings.Join(strings.Fields(description), " ")
}
