package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadSampleURL returns rawURL when provided, otherwise it loads and trims a
// sample URL from path.
func LoadSampleURL(rawURL string, path string, sourceName string, requiredErr error, emptyErr error) (string, error) {
	trimmedRawURL := strings.TrimSpace(rawURL)
	if trimmedRawURL != "" {
		return trimmedRawURL, nil
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", requiredErr
	}

	content, err := os.ReadFile(filepath.Clean(trimmedPath))
	if err != nil {
		return "", fmt.Errorf("read %s sample url file: %w", sourceName, err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", fmt.Errorf("%w: %s", emptyErr, trimmedPath)
	}
	return value, nil
}
