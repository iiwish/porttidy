package model

import "time"

const (
	SafetySafeToCleanup = "safe_to_cleanup"
	SafetyNeedsConfirm  = "needs_confirmation"
	SafetyBlocked       = "blocked"
)

// Process represents a single system process
type Process struct {
	PID             int       `json:"pid"`
	PPID            int       `json:"ppid"`
	Name            string    `json:"name"`
	Cmdline         string    `json:"cmdline"`
	CWD             string    `json:"cwd"`
	User            string    `json:"user"`
	Ports           []int     `json:"ports"`
	IsDev           bool      `json:"is_dev"`
	IsOrphan        bool      `json:"is_orphan"`
	IsSystem        bool      `json:"is_system"`
	SafetyLevel     string    `json:"safety_level"`
	CanForceCleanup bool      `json:"can_force_cleanup"`
	MatchReason     string    `json:"match_reason,omitempty"`
	OrphanReason    string    `json:"orphan_reason,omitempty"`
	BlockedReason   string    `json:"blocked_reason,omitempty"`
	StartTime       time.Time `json:"start_time"`
}

// Project groups processes by their working directory
type Project struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Processes []Process `json:"processes"`
}

// ScanResult is the output of a scan operation
type ScanResult struct {
	Projects []Project `json:"projects"`
	Summary  Summary   `json:"summary"`
}

// Summary provides aggregate statistics
type Summary struct {
	Total         int   `json:"total"`
	Orphans       int   `json:"orphans"`
	PortsOccupied []int `json:"ports_occupied"`
	ProjectsCount int   `json:"projects_count"`
}

// KillResult tracks the outcome of a kill operation
type KillResult struct {
	PID    int    `json:"pid"`
	Status string `json:"status"` // "killed", "already_dead", "failed"
	Error  string `json:"error,omitempty"`
}
