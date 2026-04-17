package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"

	"gh-depdash/internal/githubapi"
)

func TestRenderRepoItemNormalizesDescriptionWhitespace(t *testing.T) {
	description := "first line\nsecond\tline\r\n  third"
	item := repoItem{repo: githubapi.Repository{
		FullName:    "owner/repo",
		Description: &description,
	}}

	model := list.New([]list.Item{item}, itemDelegate{}, 80, 20)

	var rendered strings.Builder
	renderRepoItem(&rendered, model, 1, item)

	output := rendered.String()
	if got := strings.Count(output, "\n"); got != 1 {
		t.Fatalf("newline count = %d, want 1 so repo items stay two lines tall", got)
	}
	if strings.ContainsAny(output, "\r\t") {
		t.Fatalf("output = %q, want normalized control whitespace removed", output)
	}
	if !strings.Contains(output, "first line second line third") {
		t.Fatalf("output = %q, want normalized single-line description", output)
	}
}
