package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/iiwish/porttidy/internal/pathutil"
	"github.com/iiwish/porttidy/internal/platform"
	"github.com/iiwish/porttidy/pkg/config"
	"github.com/iiwish/porttidy/pkg/model"
	"github.com/shirou/gopsutil/v4/process"
)

var selfPID = os.Getpid()
var selfPPID = os.Getppid()

// Scanner scans for development processes
type Scanner struct {
	cfg *config.Config
}

// New creates a new Scanner
func New(cfg *config.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Scan performs a full scan and returns the result
func (s *Scanner) Scan(opts ScanOptions) (*model.ScanResult, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
	}

	// Bulk port lookup: 300x faster than per-process p.Connections()
	portMap, _ := platform.GetAllPorts()

	all := make([]model.Process, 0)
	for _, p := range procs {
		info, ok := s.extractProcessInfo(p, portMap)
		if !ok {
			continue
		}

		// Never target ourselves
		if info.PID == selfPID || info.PID == selfPPID {
			continue
		}

		// Filter: must be in target dirs
		if !s.inTargetDir(info.CWD) {
			continue
		}

		info.IsOrphan, info.OrphanReason = s.orphanStatus(info)
		s.classifyProcess(&info)

		if !opts.IncludeAll && !info.IsDev {
			continue
		}

		if info.IsSystem && !opts.IncludeSystem {
			continue
		}

		// Filter: orphan only
		if opts.OrphanOnly && !info.IsOrphan {
			continue
		}

		// Filter: specific port
		if opts.Port > 0 {
			hasPort := false
			for _, port := range info.Ports {
				if port == opts.Port {
					hasPort = true
					break
				}
			}
			if !hasPort {
				continue
			}
		}

		// Filter: specific PID
		if opts.PID > 0 && info.PID != opts.PID {
			continue
		}

		// Filter: minimum age
		if opts.Since > 0 {
			age := time.Since(info.StartTime)
			if age < opts.Since {
				continue
			}
		}

		all = append(all, info)
	}

	// Group by project
	projects := s.groupByProject(all)

	// Build summary
	summary := model.Summary{
		Total:         len(all),
		ProjectsCount: len(projects),
		PortsOccupied: make([]int, 0),
	}

	seenPorts := make(map[int]bool)
	for _, p := range all {
		if p.IsOrphan {
			summary.Orphans++
		}
		for _, port := range p.Ports {
			if !seenPorts[port] {
				summary.PortsOccupied = append(summary.PortsOccupied, port)
				seenPorts[port] = true
			}
		}
	}

	return &model.ScanResult{
		Projects: projects,
		Summary:  summary,
	}, nil
}

// ScanOptions controls scan behavior
type ScanOptions struct {
	OrphanOnly    bool
	Port          int
	PID           int
	Since         time.Duration
	IncludeAll    bool // include non-dev processes
	IncludeSystem bool // include system apps
}

func (s *Scanner) extractProcessInfo(p *process.Process, portMap map[int32][]int) (model.Process, bool) {
	pid := p.Pid

	name, err := p.Name()
	if err != nil {
		return model.Process{}, false
	}

	cmdline, err := p.Cmdline()
	if err != nil {
		cmdline = name
	}

	ppid, err := p.Ppid()
	if err != nil {
		ppid = 0
	}

	cwd, err := platform.GetCWD(pid)
	if err != nil {
		return model.Process{}, false
	}

	ports := portMap[pid]

	username := ""
	if uids, err := p.Uids(); err == nil && len(uids) > 0 {
		username = fmt.Sprintf("%d", uids[0])
	}

	createTime, err := p.CreateTime()
	startTime := time.Now()
	if err == nil && createTime > 0 {
		startTime = time.UnixMilli(createTime)
	}

	return model.Process{
		PID:       int(pid),
		PPID:      int(ppid),
		Name:      name,
		Cmdline:   cmdline,
		CWD:       cwd,
		User:      username,
		Ports:     ports,
		StartTime: startTime,
	}, true
}

func (s *Scanner) inTargetDir(cwd string) bool {
	return pathutil.AnyContains(s.cfg.TargetDirs, cwd)
}

func (s *Scanner) classifyProcess(info *model.Process) {
	info.IsSystem = s.isSystemApp(*info)
	info.IsDev = false
	info.CanForceCleanup = false
	info.SafetyLevel = model.SafetyBlocked
	info.CleanupDecision = model.CleanupBlocked
	info.MatchReason = ""
	info.BlockedReason = ""

	if info.PID == selfPID || info.PID == selfPPID {
		info.BlockedReason = "current porttidy process or parent shell"
		return
	}

	if info.IsSystem {
		info.BlockedReason = "known system, editor, terminal, browser, or agent process"
		return
	}

	if isKnownAgentRuntime(*info) {
		info.IsSystem = true
		info.BlockedReason = "known coding-agent runtime"
		return
	}

	if s.isIgnoredDir(info.CWD) {
		info.CleanupDecision = model.CleanupIgnored
		info.BlockedReason = "ignored by user policy"
		return
	}

	matched, reason := s.matchDevProcess(*info)
	if !matched {
		if isBroadRuntime(*info) {
			info.SafetyLevel = model.SafetyNeedsConfirm
			info.CleanupDecision = model.CleanupAsk
			info.MatchReason = "broad runtime without specific dev-server signature"
			return
		}
		info.BlockedReason = "no specific dev-server signature"
		return
	}

	info.IsDev = true
	info.MatchReason = reason
	if info.IsOrphan {
		info.SafetyLevel = model.SafetySafeToCleanup
		info.CleanupDecision = model.CleanupAuto
		info.CanForceCleanup = true
		return
	}

	info.SafetyLevel = model.SafetyNeedsConfirm
	info.CleanupDecision = model.CleanupAsk
}

func (s *Scanner) isDevProcess(info model.Process) bool {
	matched, _ := s.matchDevProcess(info)
	return matched
}

func (s *Scanner) matchDevProcess(info model.Process) (bool, string) {
	// Shell wrappers (bash -c, sh -c) should not be flagged as dev processes
	// just because their inline script contains a dev command.
	// Only their actual child process (the real dev server) should match.
	if isShellWrapper(info.Name) && strings.Contains(info.Cmdline, " -c ") {
		return false, ""
	}

	for _, sig := range s.cfg.DevSignatures {
		sig = strings.TrimSpace(strings.ToLower(sig))
		if sig == "" || isBroadSignature(sig) {
			continue
		}
		if commandMatchesSignature(info.Cmdline, sig) {
			return true, "matched specific dev-server signature: " + sig
		}
	}
	for _, sig := range s.cfg.UserSignatures {
		sig = strings.TrimSpace(strings.ToLower(sig))
		if sig == "" {
			continue
		}
		if commandMatchesSignature(info.Cmdline, sig) {
			return true, "matched user cleanup signature: " + sig
		}
	}
	return false, ""
}

func (s *Scanner) isIgnoredDir(cwd string) bool {
	return pathutil.AnyContains(s.cfg.IgnoreDirs, cwd)
}

func isShellWrapper(name string) bool {
	wrappers := []string{"bash", "sh", "zsh", "dash", "csh", "tcsh", "fish", "cmd", "powershell"}
	lower := strings.ToLower(name)
	for _, w := range wrappers {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

func (s *Scanner) isSystemApp(info model.Process) bool {
	if isKnownAgentRuntime(info) {
		return true
	}

	name := strings.ToLower(info.Name)
	cmdline := strings.ToLower(info.Cmdline)
	for _, denied := range s.cfg.Denylist {
		denied = strings.ToLower(strings.TrimSpace(denied))
		if denied == "" {
			continue
		}
		// Match by process name prefix (e.g. "Code" matches "Code Helper")
		// and by command path for app-bundled helpers.
		if strings.HasPrefix(name, denied) || strings.Contains(cmdline, "/"+denied+".app/") {
			return true
		}
	}
	return false
}

func (s *Scanner) isOrphan(info model.Process) bool {
	orphan, _ := s.orphanStatus(info)
	return orphan
}

func (s *Scanner) orphanStatus(info model.Process) (bool, string) {
	// PPID == 1 means adopted by init
	if info.PPID == 1 {
		return true, "ppid=1"
	}

	// Or parent process no longer exists
	if info.PPID > 0 && !platform.IsProcessAlive(int32(info.PPID)) {
		return true, "parent process not alive"
	}

	return false, ""
}

func (s *Scanner) groupByProject(processes []model.Process) []model.Project {
	groups := make(map[string][]model.Process)
	for _, p := range processes {
		groups[p.CWD] = append(groups[p.CWD], p)
	}

	projects := make([]model.Project, 0, len(groups))
	for cwd, procs := range groups {
		name := filepath.Base(cwd)
		projects = append(projects, model.Project{
			Name:      name,
			Path:      cwd,
			Processes: procs,
		})
	}

	return projects
}

var tokenSplitRE = regexp.MustCompile(`[^\w.@+-]+`)

func commandTokens(cmdline string) []string {
	parts := tokenSplitRE.Split(strings.ToLower(cmdline), -1)
	tokens := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		part = strings.Trim(part, `"'`)
		if part == "" {
			continue
		}
		tokens = append(tokens, part)
		base := strings.ToLower(filepath.Base(part))
		if base != "" && base != part {
			tokens = append(tokens, base)
		}
	}
	return tokens
}

func commandMatchesSignature(cmdline, sig string) bool {
	tokens := commandTokens(cmdline)
	if len(tokens) == 0 {
		return false
	}

	sigTokens := commandTokens(sig)
	if len(sigTokens) == 0 {
		return false
	}

	for i := 0; i <= len(tokens)-len(sigTokens); i++ {
		matched := true
		for j, sigToken := range sigTokens {
			if tokens[i+j] != sigToken {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func isBroadSignature(sig string) bool {
	switch sig {
	case "node", "python", "python3", "npm", "pnpm", "yarn", "bun", "deno", "tsx", "ts-node", "npm run", "npm exec", "npx":
		return true
	default:
		return false
	}
}

func isBroadRuntime(info model.Process) bool {
	name := strings.ToLower(filepath.Base(info.Name))
	cmdline := strings.ToLower(info.Cmdline)
	broad := []string{"node", "python", "python3", "npm", "pnpm", "yarn", "bun", "deno", "tsx", "ts-node"}
	for _, runtime := range broad {
		if name == runtime || commandMatchesSignature(cmdline, runtime) {
			return true
		}
	}
	return false
}

func isKnownAgentRuntime(info model.Process) bool {
	name := strings.ToLower(info.Name)
	cmdline := strings.ToLower(info.Cmdline)

	needles := []string{
		"node_repl",
		"codex app-server",
		"/codex.app/",
		"claude-code",
		"claude code",
		"/claude.app/",
		"opencode",
		"/cursor.app/",
		"cursor helper",
	}
	for _, needle := range needles {
		if strings.Contains(name, needle) || strings.Contains(cmdline, needle) {
			return true
		}
	}
	return false
}
