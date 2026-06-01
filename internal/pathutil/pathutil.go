package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Contains reports whether child is equal to or inside parent after expanding
// home and resolving symlinks where possible.
func Contains(parent, child string) bool {
	parent = ExpandHome(parent)
	child = ExpandHome(child)

	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		parentAbs = parent
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		childAbs = child
	}

	parentClean := filepath.Clean(parentAbs)
	childClean := filepath.Clean(childAbs)
	if resolved, err := filepath.EvalSymlinks(parentClean); err == nil {
		parentClean = resolved
	}
	if resolved, err := filepath.EvalSymlinks(childClean); err == nil {
		childClean = resolved
	}

	if childClean == parentClean {
		return true
	}

	rel, err := filepath.Rel(parentClean, childClean)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, "../")
}

// AnyContains reports whether child is inside any parent.
func AnyContains(parents []string, child string) bool {
	for _, parent := range parents {
		if Contains(parent, child) {
			return true
		}
	}
	return false
}

// ExpandHome expands leading ~ to the current user's home directory.
func ExpandHome(path string) string {
	if path == "~" {
		return os.Getenv("HOME")
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return path
}
