package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainsIsPathAware(t *testing.T) {
	if !Contains("/Users/me/self", "/Users/me/self/app") {
		t.Fatal("expected child path to be contained")
	}
	if !Contains("/Users/me/self", "/Users/me/self") {
		t.Fatal("expected exact path to be contained")
	}
	if Contains("/Users/me/self", "/Users/me/selfish/app") {
		t.Fatal("expected sibling prefix path not to be contained")
	}
}

func TestContainsResolvesSymlinks(t *testing.T) {
	tmp := t.TempDir()
	realDir := filepath.Join(tmp, "real")
	linkDir := filepath.Join(tmp, "link")
	child := filepath.Join(realDir, "child")

	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if !Contains(linkDir, child) {
		t.Fatalf("expected symlink parent %s to contain real child %s", linkDir, child)
	}
}
