package scanner

import (
	"strings"
	"testing"

	"github.com/iiwish/porttidy/pkg/config"
	"github.com/iiwish/porttidy/pkg/model"
)

func TestClassifyProcessSafetyLevels(t *testing.T) {
	s := New(&config.Config{
		TargetDirs:    []string{"/Users/me/self"},
		DevSignatures: config.DefaultDevSignatures,
		Denylist:      config.DefaultDenylist,
	})

	tests := []struct {
		name            string
		process         model.Process
		wantIsDev       bool
		wantSafety      string
		wantCanForce    bool
		wantBlockedPart string
	}{
		{
			name: "orphan vite server is safe cleanup candidate",
			process: model.Process{
				PID:      1001,
				PPID:     1,
				Name:     "node",
				Cmdline:  "node /Users/me/self/app/node_modules/.bin/vite --host 127.0.0.1 --port 5173",
				CWD:      "/Users/me/self/app",
				Ports:    []int{5173},
				IsOrphan: true,
			},
			wantIsDev:    true,
			wantSafety:   model.SafetySafeToCleanup,
			wantCanForce: true,
		},
		{
			name: "non orphan vite server requires confirmation",
			process: model.Process{
				PID:     1002,
				PPID:    900,
				Name:    "node",
				Cmdline: "node /Users/me/self/app/node_modules/.bin/vite --host 127.0.0.1 --port 5173",
				CWD:     "/Users/me/self/app",
				Ports:   []int{5173},
			},
			wantIsDev:    true,
			wantSafety:   model.SafetyNeedsConfirm,
			wantCanForce: false,
		},
		{
			name: "orphan python http server is safe cleanup candidate",
			process: model.Process{
				PID:      1003,
				PPID:     1,
				Name:     "python3",
				Cmdline:  "python3 -m http.server 8080",
				CWD:      "/Users/me/self/docs",
				Ports:    []int{8080},
				IsOrphan: true,
			},
			wantIsDev:    true,
			wantSafety:   model.SafetySafeToCleanup,
			wantCanForce: true,
		},
		{
			name: "codex node repl is blocked even when orphaned",
			process: model.Process{
				PID:      1004,
				PPID:     1,
				Name:     "node_repl",
				Cmdline:  "/Applications/Codex.app/Contents/Resources/node_repl",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:       false,
			wantSafety:      model.SafetyBlocked,
			wantCanForce:    false,
			wantBlockedPart: "agent",
		},
		{
			name: "codex app server is blocked even when command contains app resources",
			process: model.Process{
				PID:      1005,
				PPID:     1,
				Name:     "codex",
				Cmdline:  "/Applications/Codex.app/Contents/Resources/codex app-server --listen stdio://",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:       false,
			wantSafety:      model.SafetyBlocked,
			wantCanForce:    false,
			wantBlockedPart: "agent",
		},
		{
			name: "claude code runtime is blocked",
			process: model.Process{
				PID:      1006,
				PPID:     1,
				Name:     "claude",
				Cmdline:  "/Users/me/.claude/local/claude-code --stdio",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:       false,
			wantSafety:      model.SafetyBlocked,
			wantCanForce:    false,
			wantBlockedPart: "agent",
		},
		{
			name: "opencode runtime is blocked",
			process: model.Process{
				PID:      1007,
				PPID:     1,
				Name:     "opencode",
				Cmdline:  "/opt/homebrew/bin/opencode run",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:       false,
			wantSafety:      model.SafetyBlocked,
			wantCanForce:    false,
			wantBlockedPart: "agent",
		},
		{
			name: "cursor helper is blocked",
			process: model.Process{
				PID:      1008,
				PPID:     1,
				Name:     "Cursor Helper",
				Cmdline:  "/Applications/Cursor.app/Contents/Frameworks/Cursor Helper.app/Contents/MacOS/Cursor Helper",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:       false,
			wantSafety:      model.SafetyBlocked,
			wantCanForce:    false,
			wantBlockedPart: "agent",
		},
		{
			name: "generic node script is not force cleanup candidate",
			process: model.Process{
				PID:      1009,
				PPID:     1,
				Name:     "node",
				Cmdline:  "node scripts/build.js",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:    false,
			wantSafety:   model.SafetyNeedsConfirm,
			wantCanForce: false,
		},
		{
			name: "generic python script is not force cleanup candidate",
			process: model.Process{
				PID:      1010,
				PPID:     1,
				Name:     "python3",
				Cmdline:  "python3 scripts/task.py",
				CWD:      "/Users/me/self/app",
				IsOrphan: true,
			},
			wantIsDev:    false,
			wantSafety:   model.SafetyNeedsConfirm,
			wantCanForce: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.process
			s.classifyProcess(&got)

			if got.IsDev != tt.wantIsDev {
				t.Fatalf("IsDev = %v, want %v; process: %#v", got.IsDev, tt.wantIsDev, got)
			}
			if got.SafetyLevel != tt.wantSafety {
				t.Fatalf("SafetyLevel = %q, want %q; process: %#v", got.SafetyLevel, tt.wantSafety, got)
			}
			if got.CanForceCleanup != tt.wantCanForce {
				t.Fatalf("CanForceCleanup = %v, want %v; process: %#v", got.CanForceCleanup, tt.wantCanForce, got)
			}
			if tt.wantBlockedPart != "" && !strings.Contains(got.BlockedReason, tt.wantBlockedPart) {
				t.Fatalf("BlockedReason = %q, want it to contain %q", got.BlockedReason, tt.wantBlockedPart)
			}
		})
	}
}

func TestCommandMatchesSignatureIsTokenAware(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		sig     string
		want    bool
	}{
		{
			name:    "node_modules vite binary matches vite",
			cmdline: "node /Users/me/self/app/node_modules/.bin/vite --port 5173",
			sig:     "vite",
			want:    true,
		},
		{
			name:    "next dev matches exact command phrase",
			cmdline: "node /Users/me/self/app/node_modules/.bin/next dev",
			sig:     "next dev",
			want:    true,
		},
		{
			name:    "next build does not match next dev",
			cmdline: "node /Users/me/self/app/node_modules/.bin/next build",
			sig:     "next dev",
			want:    false,
		},
		{
			name:    "Resources does not match serve",
			cmdline: "/Applications/Codex.app/Contents/Resources/codex app-server --listen stdio://",
			sig:     "serve",
			want:    false,
		},
		{
			name:    "serve binary matches serve",
			cmdline: "node /Users/me/self/app/node_modules/.bin/serve -l 3000",
			sig:     "serve",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandMatchesSignature(tt.cmdline, tt.sig)
			if got != tt.want {
				t.Fatalf("commandMatchesSignature(%q, %q) = %v, want %v", tt.cmdline, tt.sig, got, tt.want)
			}
		})
	}
}

func TestTargetDirContainmentIsPathAware(t *testing.T) {
	s := New(&config.Config{TargetDirs: []string{"/Users/me/self"}})

	if !s.inTargetDir("/Users/me/self/app") {
		t.Fatal("expected child path to be inside target dir")
	}
	if !s.inTargetDir("/Users/me/self") {
		t.Fatal("expected exact target path to be inside target dir")
	}
	if s.inTargetDir("/Users/me/selfish/app") {
		t.Fatal("expected sibling prefix path not to be inside target dir")
	}
}

func TestDefaultDevSignaturesDoNotIncludeBroadRuntimes(t *testing.T) {
	broad := map[string]bool{
		"node": true, "python": true, "python3": true, "npm": true,
		"pnpm": true, "yarn": true, "bun": true, "deno": true,
		"tsx": true, "ts-node": true, "npm run": true, "npm exec": true, "npx": true,
	}

	for _, sig := range config.DefaultDevSignatures {
		if broad[sig] {
			t.Fatalf("default dev signature %q is too broad for safe cleanup", sig)
		}
	}
}
