package output

import (
	"strings"

	"gh-depdash/internal/deployments"
)

type ViewRow struct {
	Environment string `json:"environment"`
	Branch      string `json:"branch"`
	SHA         string `json:"sha,omitempty"`
	Date        string `json:"date"`
	Status      string `json:"status,omitempty"`
	LogURL      string `json:"logUrl,omitempty"`
}

func RenderTable(rows []deployments.Row, verbose bool) string {
	viewRows := ToViewRows(rows)

	headers := []string{"Env", "Branch", "Date"}
	if verbose {
		headers = append(headers, "Status", "Log URL")
	}

	records := make([][]string, 0, len(viewRows)+1)
	records = append(records, headers)
	for _, row := range viewRows {
		record := []string{
			row.Environment,
			blankDash(row.Branch),
			blankDash(row.Date),
		}
		if verbose {
			record = append(record, blankDash(row.Status), blankDash(row.LogURL))
		}
		records = append(records, record)
	}

	return renderRows(records)
}

func ToViewRows(rows []deployments.Row) []ViewRow {
	viewRows := make([]ViewRow, 0, len(rows))
	for _, row := range rows {
		viewRow := ViewRow{
			Environment: row.Environment,
			Status:      row.Status,
			LogURL:      row.LogURL,
		}
		if row.HasSuccess {
			viewRow.Branch = row.Branch
			viewRow.SHA = row.SHA
			viewRow.Date = row.Date.Format(dateLayout)
		}
		viewRows = append(viewRows, viewRow)
	}
	return viewRows
}

func blankDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func renderRows(rows [][]string) string {
	var builder strings.Builder
	for _, row := range rows {
		builder.WriteString(strings.Join(row, " | "))
		builder.WriteByte('\n')
	}
	return builder.String()
}

const dateLayout = "2006-01-02"
