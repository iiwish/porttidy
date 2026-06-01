package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesDefaultDenylistWithUserDenylist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "porttidy.yaml")
	if err := os.WriteFile(path, []byte(`
target_dirs:
  - ~/work
denylist:
  - Custom App
`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, want := range []string{"Codex", "Cursor", "Google Chrome", "Custom App"} {
		if !contains(cfg.Denylist, want) {
			t.Fatalf("denylist missing %q: %#v", want, cfg.Denylist)
		}
	}
}

func TestLoadExpandsIgnoreDirsAndKeepsUserSignatures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "porttidy.yaml")
	if err := os.WriteFile(path, []byte(`
target_dirs:
  - ~/work
ignore_dirs:
  - ~/work/critical
user_signatures:
  - air
`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantIgnore := filepath.Join(os.Getenv("HOME"), "work", "critical")
	if len(cfg.IgnoreDirs) != 1 || cfg.IgnoreDirs[0] != wantIgnore {
		t.Fatalf("IgnoreDirs = %#v, want [%q]", cfg.IgnoreDirs, wantIgnore)
	}
	if len(cfg.UserSignatures) != 1 || cfg.UserSignatures[0] != "air" {
		t.Fatalf("UserSignatures = %#v, want [air]", cfg.UserSignatures)
	}
}

func TestMergeUniqueKeepsOrderAndRemovesDuplicates(t *testing.T) {
	got := mergeUnique([]string{"Code", "Codex"}, []string{"Codex", "Custom"})
	want := []string{"Code", "Codex", "Custom"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; full: %#v", i, got[i], want[i], got)
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
