package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"gh-depdash/internal/output"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	docStyle = lipgloss.NewStyle().Margin(1, 2)
)

func renderRepoLoading(s spinner.Model) string {
	return fmt.Sprintf("\n%s Loading repositories...\n\n", s.View())
}

func renderDeploymentLoading(s spinner.Model, repo string) string {
	return fmt.Sprintf("\n%s Loading deployments for %s...\n\n", s.View(), repo)
}

func renderDeploymentBrowser(rows []output.ViewRow, partialFailures []string, verbose bool) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Deployment Status"))
	b.WriteString("\n\n")

	if len(rows) == 0 && len(partialFailures) == 0 {
		b.WriteString(subtleStyle.Render("No deployments found"))
		b.WriteString("\n")
	} else {
		for _, row := range rows {
			item := newDeploymentItem(row)
			b.WriteString(renderDeploymentItem(item, verbose))
			b.WriteString("\n\n")
		}
	}

	if len(partialFailures) > 0 {
		b.WriteString(errorStyle.Render("Partial failures:"))
		b.WriteString("\n")
		for _, failure := range partialFailures {
			b.WriteString(errorStyle.Render("  • "))
			b.WriteString(failure)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(subtleStyle.Render("Press 'b' to go back, 'q' or ctrl+c to quit"))
	b.WriteString("\n")
	return b.String()
}

func renderFatalError(err string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("Error:"))
	b.WriteString(" ")
	b.WriteString(err)
	b.WriteString("\n\n")
	return b.String()
}
