package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iiwish/porttidy/pkg/model"
)

func TestSafetyLabel(t *testing.T) {
	tests := []struct {
		name string
		p    model.Process
		want string
	}{
		{
			name: "safe cleanup",
			p:    model.Process{CleanupDecision: model.CleanupAuto, SafetyLevel: model.SafetySafeToCleanup},
			want: "auto",
		},
		{
			name: "needs confirmation",
			p:    model.Process{CleanupDecision: model.CleanupAsk, SafetyLevel: model.SafetyNeedsConfirm},
			want: "ask",
		},
		{
			name: "blocked",
			p:    model.Process{SafetyLevel: model.SafetyBlocked},
			want: "blocked",
		},
		{
			name: "legacy safe fallback",
			p:    model.Process{CanForceCleanup: true},
			want: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafetyLabel(tt.p)
			if got != tt.want {
				t.Fatalf("SafetyLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessReason(t *testing.T) {
	p := model.Process{
		MatchReason:  "matched specific dev-server signature: vite",
		OrphanReason: "ppid=1",
	}

	got := ProcessReason(p)
	want := "vite; ppid=1"
	if got != want {
		t.Fatalf("ProcessReason() = %q, want %q", got, want)
	}
}

func TestShortenText(t *testing.T) {
	got := ShortenText("abcdef", 4)
	want := "abc…"
	if got != want {
		t.Fatalf("ShortenText() = %q, want %q", got, want)
	}
}

func TestPrintTableIncludesSafetyAndReason(t *testing.T) {
	result := &model.ScanResult{
		Projects: []model.Project{
			{
				Name: "app",
				Path: "/Users/me/self/app",
				Processes: []model.Process{
					{
						PID:             1001,
						CWD:             "/Users/me/self/app",
						Cmdline:         "node /Users/me/self/app/node_modules/.bin/vite --port 5173",
						Ports:           []int{5173},
						SafetyLevel:     model.SafetySafeToCleanup,
						CleanupDecision: model.CleanupAuto,
						CanForceCleanup: true,
						MatchReason:     "matched specific dev-server signature: vite",
						OrphanReason:    "ppid=1",
					},
					{
						PID:             1002,
						CWD:             "/Users/me/self/app",
						Cmdline:         "node /Users/me/self/app/node_modules/.bin/vite --port 5174",
						Ports:           []int{5174},
						SafetyLevel:     model.SafetyNeedsConfirm,
						CleanupDecision: model.CleanupAsk,
						MatchReason:     "matched specific dev-server signature: vite",
					},
				},
			},
		},
		Summary: model.Summary{Total: 2, Orphans: 1, ProjectsCount: 1, PortsOccupied: []int{5173, 5174}},
	}

	got := captureStdout(t, func() {
		PrintTable(result)
	})

	for _, want := range []string{"State", "Reason", "auto", "ask", "vite; ppid=1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PrintTable output did not contain %q:\n%s", want, got)
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
