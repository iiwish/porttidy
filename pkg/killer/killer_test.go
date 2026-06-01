package killer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/iiwish/porttidy/pkg/model"
)

func TestKillBlocksTargetsThatAreNotSafeForCleanup(t *testing.T) {
	k := New()
	results := k.Kill([]model.Process{
		{
			PID:             os.Getpid(),
			SafetyLevel:     model.SafetyNeedsConfirm,
			CanForceCleanup: false,
		},
	}, false)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Status != "failed" {
		t.Fatalf("status = %q, want failed", results[0].Status)
	}
	if !strings.Contains(results[0].Error, "cannot kill self") {
		t.Fatalf("error = %q, want self-protection error", results[0].Error)
	}
}

func TestKillBlocksExistingUnsafeProcessBeforeSignal(t *testing.T) {
	proc := startKillerHelper(t)
	defer stopProcess(proc)

	k := New()
	results := k.Kill([]model.Process{
		{
			PID:             proc.Pid,
			SafetyLevel:     model.SafetyNeedsConfirm,
			CanForceCleanup: false,
		},
	}, false)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Status != "failed" {
		t.Fatalf("status = %q, want failed", results[0].Status)
	}
	if !strings.Contains(results[0].Error, "not marked safe") {
		t.Fatalf("error = %q, want safety failure", results[0].Error)
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("helper process was signaled or exited unexpectedly: %v", err)
	}
}

func TestValidateTargetSafety(t *testing.T) {
	tests := []struct {
		name       string
		allow      bool
		process    model.Process
		wantErrSub string
	}{
		{
			name:  "safe process passes",
			allow: false,
			process: model.Process{
				PID:             1234,
				SafetyLevel:     model.SafetySafeToCleanup,
				CanForceCleanup: true,
			},
		},
		{
			name:  "needs confirmation fails",
			allow: false,
			process: model.Process{
				PID:             1234,
				SafetyLevel:     model.SafetyNeedsConfirm,
				CanForceCleanup: false,
			},
			wantErrSub: "not marked safe",
		},
		{
			name:  "system process fails even in unsafe mode",
			allow: true,
			process: model.Process{
				PID:             1234,
				SafetyLevel:     model.SafetySafeToCleanup,
				CanForceCleanup: true,
				IsSystem:        true,
			},
			wantErrSub: "system app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Killer{allowUnsafe: tt.allow}
			err := k.validateTarget(tt.process)
			if tt.wantErrSub == "" {
				if err != "" {
					t.Fatalf("validateTarget() = %q, want empty", err)
				}
				return
			}
			if !strings.Contains(err, tt.wantErrSub) {
				t.Fatalf("validateTarget() = %q, want substring %q", err, tt.wantErrSub)
			}
		})
	}
}

func TestValidateTargetRechecksCurrentCWD(t *testing.T) {
	targetDir := t.TempDir()
	otherDir := t.TempDir()

	proc := startKillerHelperInDir(t, targetDir)
	defer stopProcess(proc)

	process := model.Process{
		PID:             proc.Pid,
		SafetyLevel:     model.SafetySafeToCleanup,
		CanForceCleanup: true,
	}

	if err := New(targetDir).validateTarget(process); err != "" {
		t.Fatalf("validateTarget() = %q, want empty for current cwd in target dir", err)
	}

	err := New(otherDir).validateTarget(process)
	if !strings.Contains(err, "cwd is no longer inside target dirs") {
		t.Fatalf("validateTarget() = %q, want cwd containment failure", err)
	}
}

func TestValidateTargetRechecksCurrentCWDWithSymlinkedTarget(t *testing.T) {
	tmp := t.TempDir()
	realDir := filepath.Join(tmp, "real")
	linkDir := filepath.Join(tmp, "link")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	proc := startKillerHelperInDir(t, realDir)
	defer stopProcess(proc)

	process := model.Process{
		PID:             proc.Pid,
		SafetyLevel:     model.SafetySafeToCleanup,
		CanForceCleanup: true,
	}

	if err := New(linkDir).validateTarget(process); err != "" {
		t.Fatalf("validateTarget() = %q, want empty for symlinked target dir", err)
	}
}

func TestKillerHelper(t *testing.T) {
	if os.Getenv("PORTTIDY_KILLER_HELPER") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func startKillerHelper(t *testing.T) *os.Process {
	t.Helper()
	return startKillerHelperInDir(t, "")
}

func startKillerHelperInDir(t *testing.T, dir string) *os.Process {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestKillerHelper")
	cmd.Env = append(os.Environ(), "PORTTIDY_KILLER_HELPER=1")
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	t.Cleanup(func() {
		_ = cmd.Wait()
	})

	return cmd.Process
}

func stopProcess(proc *os.Process) {
	if proc == nil {
		return
	}
	_ = proc.Signal(os.Interrupt)
	time.Sleep(50 * time.Millisecond)
	_ = proc.Kill()
}
