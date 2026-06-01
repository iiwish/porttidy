package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/iiwish/porttidy/pkg/model"
)

func TestPrintJSONIncludesAgentContractFields(t *testing.T) {
	result := &model.ScanResult{
		Projects: []model.Project{
			{
				Name: "app",
				Path: "/Users/me/self/app",
				Processes: []model.Process{
					{
						PID:             1001,
						PPID:            1,
						Name:            "node",
						Cmdline:         "node ./node_modules/.bin/vite --port 5173",
						CWD:             "/Users/me/self/app",
						Ports:           []int{5173},
						IsDev:           true,
						IsOrphan:        true,
						SafetyLevel:     model.SafetySafeToCleanup,
						CleanupDecision: model.CleanupAuto,
						CanForceCleanup: true,
						MatchReason:     "matched specific dev-server signature: vite",
						OrphanReason:    "ppid=1",
						StartTime:       time.Unix(100, 0),
					},
				},
			},
		},
		Summary: model.Summary{
			Total:         1,
			Orphans:       1,
			PortsOccupied: []int{5173},
			ProjectsCount: 1,
		},
	}

	raw := captureStdout(t, func() {
		if err := PrintJSON(result); err != nil {
			t.Fatalf("PrintJSON returned error: %v", err)
		}
	})

	var decoded struct {
		Projects []struct {
			Processes []struct {
				SafetyLevel     string `json:"safety_level"`
				CleanupDecision string `json:"cleanup_decision"`
				CanForceCleanup bool   `json:"can_force_cleanup"`
				MatchReason     string `json:"match_reason"`
				OrphanReason    string `json:"orphan_reason"`
			} `json:"processes"`
		} `json:"projects"`
		Summary struct {
			PortsOccupied []int `json:"ports_occupied"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("json unmarshal failed: %v\n%s", err, raw)
	}

	if len(decoded.Projects) != 1 || len(decoded.Projects[0].Processes) != 1 {
		t.Fatalf("decoded projects/processes shape is wrong: %#v", decoded)
	}

	proc := decoded.Projects[0].Processes[0]
	if proc.SafetyLevel != model.SafetySafeToCleanup {
		t.Fatalf("safety_level = %q, want %q", proc.SafetyLevel, model.SafetySafeToCleanup)
	}
	if proc.CleanupDecision != model.CleanupAuto {
		t.Fatalf("cleanup_decision = %q, want %q", proc.CleanupDecision, model.CleanupAuto)
	}
	if !proc.CanForceCleanup {
		t.Fatal("can_force_cleanup = false, want true")
	}
	if proc.MatchReason == "" {
		t.Fatal("match_reason is empty")
	}
	if proc.OrphanReason == "" {
		t.Fatal("orphan_reason is empty")
	}
	if len(decoded.Summary.PortsOccupied) != 1 || decoded.Summary.PortsOccupied[0] != 5173 {
		t.Fatalf("ports_occupied = %#v, want [5173]", decoded.Summary.PortsOccupied)
	}
}

func TestPrintJSONEmptyCollectionsAreArrays(t *testing.T) {
	result := &model.ScanResult{
		Projects: []model.Project{},
		Summary: model.Summary{
			PortsOccupied: []int{},
		},
	}

	raw := captureStdout(t, func() {
		if err := PrintJSON(result); err != nil {
			t.Fatalf("PrintJSON returned error: %v", err)
		}
	})

	if !json.Valid([]byte(raw)) {
		t.Fatalf("invalid json: %s", raw)
	}
	if !strings.Contains(raw, `"projects": []`) {
		t.Fatalf("projects should be an empty array, got:\n%s", raw)
	}
	if !strings.Contains(raw, `"ports_occupied": []`) {
		t.Fatalf("ports_occupied should be an empty array, got:\n%s", raw)
	}
}
