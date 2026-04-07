package validation

import (
	"fmt"
	"os"
	"strings"
)

// ResolveOutputDir returns an explicit output directory when provided, or a new
// temporary directory when path is empty.
func ResolveOutputDir(path string, pattern string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath != "" {
		if err := os.MkdirAll(trimmedPath, 0o755); err != nil {
			return "", fmt.Errorf("create output dir: %w", err)
		}
		return trimmedPath, nil
	}

	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp output dir: %w", err)
	}
	return dir, nil
}
