package output

import (
	"encoding/json"
	"os"

	"github.com/iiwish/porttidy/pkg/model"
)

// PrintJSON prints scan results as JSON
func PrintJSON(result *model.ScanResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// PrintKillResult prints kill results as JSON
func PrintKillResult(results []model.KillResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
