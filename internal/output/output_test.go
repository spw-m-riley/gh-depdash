package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gh-depdash/internal/deployments"
)

func TestRenderDefaultTable(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Staging",
			Branch:      "feature/login",
			Date:        time.Date(2024, time.March, 14, 9, 26, 0, 0, time.UTC),
			HasSuccess:  true,
		},
	}

	got := RenderTable(rows, false)

	want := "Env | Branch | Date\nStaging | feature/login | 2024-03-14\n"
	if got != want {
		t.Fatalf("unexpected table output\nwant:\n%sgot:\n%s", want, got)
	}
}

func TestRenderVerboseTableIncludesStatusAndLogURL(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			Branch:      "main",
			Date:        time.Date(2024, time.March, 14, 9, 26, 0, 0, time.UTC),
			Status:      "success",
			LogURL:      "https://example.com/log",
			HasSuccess:  true,
		},
	}

	got := RenderTable(rows, true)

	want := "Env | Branch | Date | Status | Log URL\nProduction | main | 2024-03-14 | success | https://example.com/log\n"
	if got != want {
		t.Fatalf("unexpected verbose table output\nwant:\n%sgot:\n%s", want, got)
	}
}

func TestRenderJSON(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			Branch:      "main",
			Date:        time.Date(2024, time.March, 14, 9, 26, 0, 0, time.UTC),
			Status:      "success",
			LogURL:      "https://example.com/log",
			HasSuccess:  true,
		},
	}

	got, err := RenderJSON(rows)
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var decoded []ViewRow
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("RenderJSON returned invalid json: %v", err)
	}

	want := []ViewRow{
		{
			Environment: "Production",
			Branch:      "main",
			Date:        "2024-03-14",
			Status:      "success",
			LogURL:      "https://example.com/log",
		},
	}

	if len(decoded) != len(want) || decoded[0] != want[0] {
		t.Fatalf("unexpected decoded json\nwant: %#v\ngot: %#v", want, decoded)
	}
}

func TestRenderBlankProductionRow(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			HasSuccess:  false,
		},
	}

	got := RenderTable(rows, false)

	if !strings.Contains(got, "Production | — | —") {
		t.Fatalf("expected blank production row to render em dashes, got:\n%s", got)
	}
}

func TestRenderBlankVerboseProductionRow(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			HasSuccess:  false,
		},
	}

	got := RenderTable(rows, true)

	if !strings.Contains(got, "Production | — | — | — | —") {
		t.Fatalf("expected blank verbose production row to render em dashes, got:\n%s", got)
	}
}

func TestRenderVerboseTablePreservesLatestAttemptContextWithoutSuccess(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			Status:      "failure",
			LogURL:      "https://example.com/failed-run",
			HasSuccess:  false,
		},
	}

	got := RenderTable(rows, true)

	want := "Env | Branch | Date | Status | Log URL\nProduction | — | — | failure | https://example.com/failed-run\n"
	if got != want {
		t.Fatalf("unexpected verbose table output for no-success row\nwant:\n%sgot:\n%s", want, got)
	}
}

func TestRenderJSONPreservesLatestAttemptContextWithoutSuccess(t *testing.T) {
	rows := []deployments.Row{
		{
			Environment: "Production",
			Status:      "failure",
			LogURL:      "https://example.com/failed-run",
			HasSuccess:  false,
		},
	}

	got, err := RenderJSON(rows)
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var decoded []ViewRow
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("RenderJSON returned invalid json: %v", err)
	}

	want := []ViewRow{
		{
			Environment: "Production",
			Status:      "failure",
			LogURL:      "https://example.com/failed-run",
		},
	}

	if len(decoded) != len(want) || decoded[0] != want[0] {
		t.Fatalf("unexpected decoded json for no-success row\nwant: %#v\ngot: %#v", want, decoded)
	}
}
