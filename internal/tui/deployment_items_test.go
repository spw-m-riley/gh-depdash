package tui

import (
	"strings"
	"testing"

	"gh-depdash/internal/output"
)

func TestRenderDeploymentItemSuccessfulRowIncludesShortSHA(t *testing.T) {
	item := newDeploymentItem(output.ViewRow{
		Environment: "Production",
		Branch:      "main",
		SHA:         "1234567890abcdef",
		Date:        "2024-01-15",
	})

	view := renderDeploymentItem(item, false)

	if !strings.Contains(view, "main • 1234567 • 2024-01-15") {
		t.Fatalf("view = %q, want compact branch/SHA/date", view)
	}
}

func TestRenderDeploymentItemShortSHAFallsBackToFullValue(t *testing.T) {
	item := newDeploymentItem(output.ViewRow{
		Environment: "Production",
		Branch:      "main",
		SHA:         "abc12",
		Date:        "2024-01-15",
	})

	view := renderDeploymentItem(item, false)

	if !strings.Contains(view, "main • abc12 • 2024-01-15") {
		t.Fatalf("view = %q, want full short SHA value", view)
	}
}

func TestRenderDeploymentItemNonSuccessRowDoesNotRenderSHA(t *testing.T) {
	item := newDeploymentItem(output.ViewRow{
		Environment: "Test",
		SHA:         "1234567890abcdef",
		Status:      "in_progress",
	})

	view := renderDeploymentItem(item, false)

	if !strings.Contains(view, "in progress") {
		t.Fatalf("view = %q, want unchanged non-success status", view)
	}
	if strings.Contains(view, "1234567") {
		t.Fatalf("view = %q, want no SHA for non-success rows", view)
	}
}
