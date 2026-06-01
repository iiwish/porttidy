package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iiwish/porttidy/pkg/model"
)

// PrintTable prints scan results in a TTY-friendly format
func PrintTable(result *model.ScanResult) {
	if len(result.Projects) == 0 {
		fmt.Println("✅ 没有发现开发进程。")
		return
	}

	// Fixed-width columns to stay within 80 chars
	const (
		wPID     = 7
		wProject = 16
		wCmd     = 28
		wPort    = 9
		wState   = 8
		wReason  = 30
		gap      = 2
	)

	format := fmt.Sprintf("%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%s\n",
		wPID+gap, wProject+gap, wCmd+gap, wPort+gap, wState+gap)

	// Header
	fmt.Printf(format, "PID", "Project", "Command", "Port", "State", "Reason")
	fmt.Println(strings.Repeat("─", wPID+wProject+wCmd+wPort+wState+wReason+gap*5))

	for _, proj := range result.Projects {
		for _, p := range proj.Processes {
			ports := "—"
			if len(p.Ports) > 0 {
				var ps []string
				for _, port := range p.Ports {
					ps = append(ps, fmt.Sprintf("%d", port))
				}
				ports = strings.Join(ps, ",")
			}

			shortCwd := ShortenPath(p.CWD, wProject)
			shortCmd := ShortenCmd(p.Cmdline, wCmd)
			state := SafetyLabel(p)
			reason := ShortenText(ProcessReason(p), wReason)

			fmt.Printf(format,
				fmt.Sprintf("%d", p.PID),
				shortCwd,
				shortCmd,
				ports,
				state,
				reason)
		}
	}

	fmt.Println(strings.Repeat("─", wPID+wProject+wCmd+wPort+wState+wReason+gap*5))
	fmt.Printf("%d processes found (%d orphans) across %d projects\n",
		result.Summary.Total, result.Summary.Orphans, result.Summary.ProjectsCount)

	if len(result.Summary.PortsOccupied) > 0 {
		fmt.Printf("Ports occupied: %v\n", result.Summary.PortsOccupied)
	}
}

// ShortenPath abbreviates a path for display
func ShortenPath(p string, max int) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(p, home) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	if len(p) > max {
		p = "…" + p[len(p)-max+1:]
	}
	return p
}

// ShortenCmd abbreviates a command for display
func ShortenCmd(cmd string, max int) string {
	// Remove common node_modules paths
	cmd = strings.ReplaceAll(cmd, filepath.Join(os.Getenv("HOME"), "daas"), "~/daas")
	cmd = strings.ReplaceAll(cmd, filepath.Join(os.Getenv("HOME"), "self"), "~/self")

	if len(cmd) > max {
		return cmd[:max-1] + "…"
	}
	return cmd
}

// SafetyLabel returns the compact display label for a process safety level.
func SafetyLabel(p model.Process) string {
	switch p.CleanupDecision {
	case model.CleanupAuto:
		return "auto"
	case model.CleanupAsk:
		return "ask"
	case model.CleanupIgnored:
		return "ignored"
	case model.CleanupBlocked:
		return "blocked"
	}

	switch p.SafetyLevel {
	case model.SafetySafeToCleanup:
		return "auto"
	case model.SafetyNeedsConfirm:
		return "ask"
	case model.SafetyBlocked:
		return "blocked"
	default:
		if p.CanForceCleanup {
			return "auto"
		}
		if p.IsOrphan {
			return "orphan"
		}
		return "-"
	}
}

// ProcessReason returns the shortest useful explanation for a process decision.
func ProcessReason(p model.Process) string {
	var parts []string

	if p.MatchReason != "" {
		parts = append(parts, trimReasonPrefix(p.MatchReason))
	}
	if p.OrphanReason != "" {
		parts = append(parts, p.OrphanReason)
	} else if p.IsOrphan {
		parts = append(parts, "orphan")
	}
	if p.BlockedReason != "" {
		parts = append(parts, p.BlockedReason)
	}

	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "; ")
}

func trimReasonPrefix(reason string) string {
	reason = strings.TrimPrefix(reason, "matched specific dev-server signature: ")
	reason = strings.TrimPrefix(reason, "matched user cleanup signature: ")
	reason = strings.TrimPrefix(reason, "broad runtime without specific dev-server signature")
	if reason == "" {
		return "broad runtime"
	}
	return reason
}

// ShortenText abbreviates a string for fixed-width table display.
func ShortenText(text string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(text) > max {
		return text[:max-1] + "…"
	}
	return text
}
