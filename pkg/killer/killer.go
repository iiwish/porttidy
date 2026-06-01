package killer

import (
	"fmt"
	"os"
	"time"

	"github.com/iiwish/porttidy/internal/pathutil"
	"github.com/iiwish/porttidy/internal/platform"
	"github.com/iiwish/porttidy/pkg/model"
)

// Killer handles process termination
type Killer struct {
	allowUnsafe bool
	targetDirs  []string
}

// New creates a new Killer
func New(targetDirs ...string) *Killer {
	return &Killer{targetDirs: targetDirs}
}

// NewUnsafe creates a killer for explicitly confirmed expert operations.
func NewUnsafe(targetDirs ...string) *Killer {
	return &Killer{allowUnsafe: true, targetDirs: targetDirs}
}

// Kill terminates the given processes
func (k *Killer) Kill(processes []model.Process, dryRun bool) []model.KillResult {
	var results []model.KillResult

	for _, p := range processes {
		if dryRun {
			results = append(results, model.KillResult{
				PID:    p.PID,
				Status: "dry_run",
			})
			continue
		}

		result := k.killOne(p)
		results = append(results, result)
	}

	return results
}

func (k *Killer) killOne(p model.Process) model.KillResult {
	// Never kill ourselves or our parent shell
	if p.PID == os.Getpid() || p.PID == os.Getppid() {
		return model.KillResult{
			PID:    p.PID,
			Status: "failed",
			Error:  "cannot kill self or parent shell",
		}
	}

	// Verify process still exists
	if !platform.IsProcessAlive(int32(p.PID)) {
		return model.KillResult{PID: p.PID, Status: "already_dead"}
	}

	if err := k.validateTarget(p); err != "" {
		return model.KillResult{
			PID:    p.PID,
			Status: "failed",
			Error:  err,
		}
	}

	// Send SIGTERM, wait briefly, then SIGKILL
	if err := platform.TerminateProcess(int32(p.PID)); err != nil {
		return model.KillResult{PID: p.PID, Status: "failed", Error: fmt.Sprintf("sigterm: %v", err)}
	}

	// Wait up to 1 second for graceful shutdown
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		if !platform.IsProcessAlive(int32(p.PID)) {
			return model.KillResult{PID: p.PID, Status: "killed"}
		}
	}

	// SIGKILL
	if err := platform.KillProcess(int32(p.PID)); err != nil {
		return model.KillResult{PID: p.PID, Status: "failed", Error: fmt.Sprintf("sigkill: %v", err)}
	}

	// Give SIGKILL a moment
	time.Sleep(100 * time.Millisecond)

	// Verify
	if platform.IsProcessAlive(int32(p.PID)) {
		return model.KillResult{PID: p.PID, Status: "failed", Error: "process still alive after sigkill"}
	}

	return model.KillResult{PID: p.PID, Status: "killed"}
}

func (k *Killer) validateTarget(p model.Process) string {
	if !k.allowUnsafe && !p.CanForceCleanup {
		return "process is not marked safe for cleanup"
	}

	if !k.allowUnsafe && p.SafetyLevel != model.SafetySafeToCleanup {
		return "process safety level is not safe_to_cleanup"
	}

	if !k.allowUnsafe && p.CleanupDecision != "" && p.CleanupDecision != model.CleanupAuto {
		return "process cleanup decision is not auto_cleanup"
	}

	if p.IsSystem {
		return "process is marked as system app"
	}

	if len(k.targetDirs) > 0 {
		cwd, err := platform.GetCWD(int32(p.PID))
		if err != nil {
			return fmt.Sprintf("could not re-check process cwd: %v", err)
		}
		if !pathutil.AnyContains(k.targetDirs, cwd) {
			return "process cwd is no longer inside target dirs"
		}
	}

	return ""
}
