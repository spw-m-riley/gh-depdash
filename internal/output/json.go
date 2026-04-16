package output

import (
	"encoding/json"

	"gh-depdash/internal/deployments"
)

func RenderJSON(rows []deployments.Row) ([]byte, error) {
	return json.Marshal(ToViewRows(rows))
}
