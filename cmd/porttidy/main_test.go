package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/iiwish/porttidy/pkg/model"
)

func TestSelectTargetsSafeOnly(t *testing.T) {
	result := &model.ScanResult{
		Projects: []model.Project{
			{
				Name: "app",
				Path: "/Users/me/self/app",
				Processes: []model.Process{
					{PID: 1001, CWD: "/Users/me/self/app", CanForceCleanup: true},
					{PID: 1002, CWD: "/Users/me/self/app", CanForceCleanup: false},
				},
			},
		},
	}

	got := selectTargets(result, targetSelection{SafeOnly: true})
	if len(got) != 1 {
		t.Fatalf("len(targets) = %d, want 1; targets: %#v", len(got), got)
	}
	if got[0].PID != 1001 {
		t.Fatalf("PID = %d, want 1001", got[0].PID)
	}
}

func TestSelectTargetsDirFilter(t *testing.T) {
	result := &model.ScanResult{
		Projects: []model.Project{
			{
				Name: "projects",
				Path: "/Users/me/self",
				Processes: []model.Process{
					{PID: 1001, CWD: "/Users/me/self/app-a", CanForceCleanup: true},
					{PID: 1002, CWD: "/Users/me/self/app-b", CanForceCleanup: true},
				},
			},
		},
	}

	got := selectTargets(result, targetSelection{DirFilter: "app-b", SafeOnly: true})
	if len(got) != 1 {
		t.Fatalf("len(targets) = %d, want 1; targets: %#v", len(got), got)
	}
	if got[0].PID != 1002 {
		t.Fatalf("PID = %d, want 1002", got[0].PID)
	}
}

func TestExecuteTargetsDryRunReturnsCandidatesExitCode(t *testing.T) {
	previousJSONMode := jsonMode
	jsonMode = false
	defer func() {
		jsonMode = previousJSONMode
	}()

	var err error
	captureStdout(t, func() {
		err = executeTargets([]model.Process{
			{
				PID:             1001,
				Name:            "node",
				CWD:             "/Users/me/self/app",
				SafetyLevel:     model.SafetySafeToCleanup,
				CanForceCleanup: true,
				MatchReason:     "matched specific dev-server signature: vite",
				OrphanReason:    "ppid=1",
			},
		}, true, false, false, nil)
	})

	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("executeTargets returned %T, want exitCodeError", err)
	}
	if exitErr.code != exitCandidatesFound {
		t.Fatalf("exit code = %d, want %d", exitErr.code, exitCandidatesFound)
	}
}

func TestExecuteTargetsNoCandidatesReturnsZero(t *testing.T) {
	previousJSONMode := jsonMode
	jsonMode = false
	defer func() {
		jsonMode = previousJSONMode
	}()

	captureStdout(t, func() {
		if err := executeTargets(nil, true, false, false, nil); err != nil {
			t.Fatalf("executeTargets returned error for no candidates: %v", err)
		}
	})
}

func TestRootCommandDoesNotExposeCompletionYet(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			t.Fatal("completion command should stay deferred until shell completion is intentionally supported")
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
