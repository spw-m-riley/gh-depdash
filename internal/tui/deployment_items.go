package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"gh-depdash/internal/output"
)

type deploymentItem struct {
	environment string
	branch      string
	sha         string
	date        string
	status      string
	logURL      string
}

func newDeploymentItem(row output.ViewRow) deploymentItem {
	return deploymentItem{
		environment: row.Environment,
		branch:      row.Branch,
		sha:         row.SHA,
		date:        row.Date,
		status:      row.Status,
		logURL:      row.LogURL,
	}
}

func renderDeploymentItem(item deploymentItem, verbose bool) string {
	var b strings.Builder

	envStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	b.WriteString(envStyle.Render(item.environment))

	if item.branch != "" && item.date != "" {
		b.WriteString("\n  ")
		b.WriteString(successStyle.Render("✓"))
		b.WriteString(" ")
		b.WriteString(item.branch)
		if item.sha != "" {
			b.WriteString(" • ")
			b.WriteString(shortSHA(item.sha))
		}
		b.WriteString(" • ")
		b.WriteString(subtleStyle.Render(item.date))
	} else if item.status != "" {
		b.WriteString("\n  ")
		statusIcon := "•"
		statusText := item.status
		statusColor := lipgloss.Color("241")

		switch item.status {
		case "success", "inactive":
			statusIcon = "✓"
			statusColor = lipgloss.Color("42")
		case "failure":
			statusIcon = "✗"
			statusColor = lipgloss.Color("196")
		case "in_progress":
			statusIcon = "⋯"
			statusColor = lipgloss.Color("226")
			statusText = "in progress"
		case "queued", "waiting":
			statusIcon = "○"
			statusColor = lipgloss.Color("246")
		}

		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		b.WriteString(statusStyle.Render(statusIcon))
		b.WriteString(" ")
		b.WriteString(statusText)
	}

	if verbose && item.logURL != "" {
		b.WriteString("\n  ")
		b.WriteString(subtleStyle.Render("Log: "))
		b.WriteString(item.logURL)
	}

	return b.String()
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
