//go:build darwin

package scanner

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/iiwish/porttidy/pkg/config"
	"github.com/iiwish/porttidy/pkg/model"
)

const integrationDevSignature = "porttidy-test-dev-server"

func TestScanFindsOrphanGoHTTPServer(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "server.pid")
	portFile := filepath.Join(tmp, "server.port")

	launcher := exec.Command(os.Args[0], "-test.run=TestGoHTTPServerHelper", "--", integrationDevSignature)
	launcher.Dir = tmp
	launcher.Env = append(os.Environ(),
		"PORTTIDY_TEST_HELPER=launcher",
		"PORTTIDY_TEST_CWD="+tmp,
		"PORTTIDY_TEST_PID_FILE="+pidFile,
		"PORTTIDY_TEST_PORT_FILE="+portFile,
	)

	if output, err := launcher.CombinedOutput(); err != nil {
		t.Fatalf("launcher failed: %v\n%s", err, output)
	}

	pid := readIntFileEventually(t, pidFile, 3*time.Second)
	port := readIntFileEventually(t, portFile, 3*time.Second)

	defer func() {
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)
			_ = proc.Signal(os.Kill)
		}
	}()

	s := New(&config.Config{
		TargetDirs:    []string{tmp},
		DevSignatures: []string{integrationDevSignature},
		Denylist:      config.DefaultDenylist,
	})

	var found *model.Process
	var lastResult *model.ScanResult
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result, err := s.Scan(ScanOptions{IncludeAll: true, IncludeSystem: true})
		if err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		lastResult = result
		if p := findProcess(result, pid); p != nil {
			found = p
			if p.IsOrphan && p.CanForceCleanup {
				break
			}
		}
		time.Sleep(150 * time.Millisecond)
	}

	if found == nil {
		t.Fatalf("Go HTTP server pid %d was not found; diagnostics:\n%s\nlast scan: %s",
			pid, processDiagnostics(pid), describeScan(lastResult))
	}
	if !found.IsDev {
		t.Fatalf("IsDev = false, want true; process: %#v", *found)
	}
	if !found.IsOrphan {
		t.Fatalf("IsOrphan = false, want true; process: %#v", *found)
	}
	if found.SafetyLevel != model.SafetySafeToCleanup {
		t.Fatalf("SafetyLevel = %q, want %q; process: %#v", found.SafetyLevel, model.SafetySafeToCleanup, *found)
	}
	if !found.CanForceCleanup {
		t.Fatalf("CanForceCleanup = false, want true; process: %#v", *found)
	}
	if !hasPort(found.Ports, port) {
		t.Fatalf("Ports = %v, want port %d; process: %#v", found.Ports, port, *found)
	}
	if !strings.Contains(found.MatchReason, integrationDevSignature) {
		t.Fatalf("MatchReason = %q, want it to mention %q", found.MatchReason, integrationDevSignature)
	}

	orphanResult, err := s.Scan(ScanOptions{OrphanOnly: true})
	if err != nil {
		t.Fatalf("orphan scan failed: %v", err)
	}
	if p := findProcess(orphanResult, pid); p == nil {
		t.Fatalf("orphan scan did not include pid %d; process from full scan: %#v", pid, *found)
	}
}

func TestGoHTTPServerHelper(t *testing.T) {
	mode := os.Getenv("PORTTIDY_TEST_HELPER")
	if mode == "" {
		return
	}

	switch mode {
	case "launcher":
		runLauncherHelper()
	case "server":
		runServerHelper()
	default:
		fmt.Fprintf(os.Stderr, "unknown PORTTIDY_TEST_HELPER mode %q\n", mode)
		os.Exit(2)
	}
}

func runLauncherHelper() {
	cwd := os.Getenv("PORTTIDY_TEST_CWD")
	pidFile := os.Getenv("PORTTIDY_TEST_PID_FILE")
	portFile := os.Getenv("PORTTIDY_TEST_PORT_FILE")
	if cwd == "" || pidFile == "" || portFile == "" {
		fmt.Fprintln(os.Stderr, "missing helper env")
		os.Exit(2)
	}

	server := exec.Command(os.Args[0], "-test.run=TestGoHTTPServerHelper", "--", integrationDevSignature)
	server.Dir = cwd
	server.Env = append(os.Environ(),
		"PORTTIDY_TEST_HELPER=server",
		"PORTTIDY_TEST_CWD="+cwd,
		"PORTTIDY_TEST_PORT_FILE="+portFile,
	)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start server helper: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(server.Process.Pid)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write pid file: %v\n", err)
		os.Exit(2)
	}

	os.Exit(0)
}

func runServerHelper() {
	portFile := os.Getenv("PORTTIDY_TEST_PORT_FILE")
	if portFile == "" {
		fmt.Fprintln(os.Stderr, "missing port file env")
		os.Exit(2)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(2)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	if err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write port file: %v\n", err)
		os.Exit(2)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	}

	done := make(chan struct{})
	go func() {
		_ = server.Serve(ln)
		close(done)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	select {
	case <-signals:
		_ = server.Close()
	case <-time.After(30 * time.Second):
		_ = server.Close()
	case <-done:
	}

	os.Exit(0)
}

func readIntFileEventually(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			value, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				return value
			}
			lastErr = err
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("could not read int file %s: %v", path, lastErr)
	return 0
}

func findProcess(result *model.ScanResult, pid int) *model.Process {
	for _, project := range result.Projects {
		for _, proc := range project.Processes {
			if proc.PID == pid {
				p := proc
				return &p
			}
		}
	}
	return nil
}

func hasPort(ports []int, want int) bool {
	for _, port := range ports {
		if port == want {
			return true
		}
	}
	return false
}

func describeScan(result *model.ScanResult) string {
	if result == nil {
		return "<nil>"
	}
	var b strings.Builder
	for _, project := range result.Projects {
		for _, proc := range project.Processes {
			fmt.Fprintf(&b, "pid=%d ppid=%d name=%q cwd=%q is_dev=%v is_orphan=%v safety=%q match=%q block=%q cmd=%q\n",
				proc.PID, proc.PPID, proc.Name, proc.CWD, proc.IsDev, proc.IsOrphan,
				proc.SafetyLevel, proc.MatchReason, proc.BlockedReason, proc.Cmdline)
		}
	}
	if b.Len() == 0 {
		return "<empty>"
	}
	return b.String()
}

func processDiagnostics(pid int) string {
	var b strings.Builder
	for _, args := range [][]string{
		{"ps", "-p", strconv.Itoa(pid), "-o", "pid,ppid,stat,command"},
		{"lsof", "-a", "-d", "cwd", "-p", strconv.Itoa(pid), "-Fn"},
		{"lsof", "-a", "-iTCP", "-sTCP:LISTEN", "-P", "-n", "-p", strconv.Itoa(pid), "-Fpcn"},
	} {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		fmt.Fprintf(&b, "$ %s\n%s", strings.Join(args, " "), out)
		if err != nil {
			fmt.Fprintf(&b, "error: %v\n", err)
		}
	}
	return b.String()
}
