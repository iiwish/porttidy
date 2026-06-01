//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v4/process"
)

// GetCWD returns the current working directory of a process on macOS.
// gopsutil's Cwd() may fail for some processes, so we fall back to lsof.
func GetCWD(pid int32) (string, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return "", err
	}

	cwd, err := p.Cwd()
	if err == nil && cwd != "" {
		return cwd, nil
	}

	// Fallback: use lsof
	out, err := exec.Command("lsof", "-a", "-d", "cwd", "-p", strconv.Itoa(int(pid)), "-Fn").Output()
	if err != nil {
		return "", fmt.Errorf("lsof failed: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n"), nil
		}
	}

	return "", fmt.Errorf("cwd not found for pid %d", pid)
}

// GetPorts returns the list of listening ports for a process.
// Deprecated: use GetAllPorts for bulk query (300x faster).
func GetPorts(pid int32) ([]int, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	conns, err := p.Connections()
	if err != nil {
		return nil, err
	}

	var ports []int
	seen := make(map[int]bool)
	for _, conn := range conns {
		if conn.Status == "LISTEN" && conn.Laddr.Port > 0 {
			if !seen[int(conn.Laddr.Port)] {
				ports = append(ports, int(conn.Laddr.Port))
				seen[int(conn.Laddr.Port)] = true
			}
		}
	}

	return ports, nil
}

// GetAllPorts returns a map of PID -> listening ports for all processes.
// Uses a single lsof call (~40ms) instead of per-process p.Connections() (~13s for 600 procs).
func GetAllPorts() (map[int32][]int, error) {
	out, err := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n", "-Fpcn").Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	result := make(map[int32][]int)
	var currentPID int32 = -1
	seen := make(map[int32]map[int]bool)

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		field := line[0]
		value := line[1:]

		switch field {
		case 'p':
			pid, err := strconv.Atoi(value)
			if err == nil {
				currentPID = int32(pid)
			}
		case 'n':
			if currentPID < 0 {
				continue
			}
			// Parse "*:8080" or "127.0.0.1:8080" or "[::]:8080"
			port := extractPort(value)
			if port > 0 {
				if seen[currentPID] == nil {
					seen[currentPID] = make(map[int]bool)
				}
				if !seen[currentPID][port] {
					seen[currentPID][port] = true
					result[currentPID] = append(result[currentPID], port)
				}
			}
		}
	}

	return result, nil
}

func extractPort(addr string) int {
	// Handle formats: "*:8080", "127.0.0.1:8080", "[::]:8080", "localhost:8080"
	// Find the last colon
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0
	}
	portStr := addr[idx+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}

// IsProcessAlive checks if a process is still running
func IsProcessAlive(pid int32) bool {
	_, err := process.NewProcess(pid)
	return err == nil
}

// TerminateProcess sends SIGTERM to a process.
func TerminateProcess(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGTERM)
}

// KillProcess sends SIGKILL to a process directly via syscall.
func KillProcess(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGKILL)
}
