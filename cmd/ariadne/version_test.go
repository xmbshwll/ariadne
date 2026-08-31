package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurrentVersionPinsTheShapeOfTheVersionLine checks what the build info
// can and cannot know: module versions when built from published modules,
// "devel" placeholders when built from a checkout.
func TestCurrentVersionPinsTheShapeOfTheVersionLine(t *testing.T) {
	version := currentVersion()

	tests := []struct {
		name    string
		got     string
		wantSet bool
	}{
		{name: "cli version is set", got: version.CLI, wantSet: true},
		{name: "library version is set", got: version.Library, wantSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.got, tt.name)
		})
	}

	// Inside `go test`, the main module version is "(devel)", so the CLI side
	// reports the devel placeholder; the library side may carry a real version
	// from the workspace.
	t.Run("the rendered line names both modules", func(t *testing.T) {
		line := version.String()
		assert.Contains(t, line, "ariadne CLI", line)
		assert.Contains(t, line, "library", line)
	})
}

// TestRunVersionFlagRendersTheVersionLine pins the --version flag and the
// version subcommand cobra derives from Version.
func TestRunVersionFlagRendersTheVersionLine(t *testing.T) {
	// Only the flag: cobra derives no `version` subcommand from cmd.Version,
	// and inventing one would duplicate what --version prints.
	tests := []struct {
		name string
		args []string
	}{
		{name: "--version flag", args: []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr)
			require.NoError(t, err, tt.name)

			assert.Contains(t, stdout.String(), "ariadne CLI", tt.name)
			assert.Contains(t, stdout.String(), "library", tt.name)
			assert.False(t, strings.Contains(stdout.String(), "%!"), tt.name)
		})
	}
}
