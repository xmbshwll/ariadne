package validation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTestSampleURLRequired = errors.New("sample url required")
	errTestSampleURLEmpty    = errors.New("sample url empty")
	errTestLoadFailed        = errors.New("load failed")
	errTestCollectFailed     = errors.New("collect failed")
	errTestWriteFailed       = errors.New("write failed")
)

func TestLoadSampleURL(t *testing.T) {
	t.Parallel()

	t.Run("returns explicit url", func(t *testing.T) {
		t.Parallel()

		url, err := LoadSampleURL(" https://example.test/album/1 ", filepath.Join(t.TempDir(), "missing.txt"), "spotify", errTestSampleURLRequired, errTestSampleURLEmpty)
		require.NoError(t, err)
		assert.Equal(t, "https://example.test/album/1", url)
	})

	t.Run("reads file when raw url missing", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "sample.txt")
		require.NoError(t, os.WriteFile(path, []byte("\nhttps://example.test/album/2\n"), 0o600))

		url, err := LoadSampleURL("", path, "spotify", errTestSampleURLRequired, errTestSampleURLEmpty)
		require.NoError(t, err)
		assert.Equal(t, "https://example.test/album/2", url)
	})

	t.Run("errors when url and path missing", func(t *testing.T) {
		t.Parallel()

		_, err := LoadSampleURL("", "", "spotify", errTestSampleURLRequired, errTestSampleURLEmpty)
		require.Error(t, err)
		assert.ErrorIs(t, err, errTestSampleURLRequired)
	})

	t.Run("errors when file empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "sample.txt")
		require.NoError(t, os.WriteFile(path, []byte(" \n\t "), 0o600))

		_, err := LoadSampleURL("", path, "spotify", errTestSampleURLRequired, errTestSampleURLEmpty)
		require.Error(t, err)
		assert.ErrorIs(t, err, errTestSampleURLEmpty)
		assert.Contains(t, err.Error(), path)
	})
}

func TestResolveOutputDir(t *testing.T) {
	t.Parallel()

	t.Run("uses provided dir", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "artifacts")
		resolved, err := ResolveOutputDir(path, "unused-")
		require.NoError(t, err)
		assert.Equal(t, path, resolved)
		info, statErr := os.Stat(resolved)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})

	t.Run("creates temp dir when missing", func(t *testing.T) {
		t.Parallel()

		resolved, err := ResolveOutputDir("", "ariadne-validation-")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(resolved) })
		assert.Contains(t, filepath.Base(resolved), "ariadne-validation-")
		info, statErr := os.Stat(resolved)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "summary.json")
	err := WriteJSON(path, map[string]string{"z": "last", "a": "first"})
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"a\": \"first\",\n  \"z\": \"last\"\n}\n", string(content))
}

func TestWritePrettyJSON(t *testing.T) {
	t.Parallel()

	t.Run("pretty prints raw json", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "payload.json")
		err := WritePrettyJSON(path, []byte(`{"name":"fixture","nested":{"count":2}}`))
		require.NoError(t, err)

		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "{\n  \"name\": \"fixture\",\n  \"nested\": {\n    \"count\": 2\n  }\n}\n", string(content))
	})

	t.Run("returns decode error for invalid raw json", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "payload.json")
		err := WritePrettyJSON(path, []byte(`{"name":`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode raw json")
		assert.Contains(t, err.Error(), path)
	})
}

type testRunInputs struct {
	outputDir      string
	successMessage string
}

func (i testRunInputs) OutputDir() string {
	return i.outputDir
}

func (i testRunInputs) SuccessMessage() string {
	return i.successMessage
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("writes artifacts and success message", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		artifactDir := t.TempDir()
		loadCalled := false
		collectCalled := false
		writeCalled := false

		err := Run(RunConfig[testRunInputs, string]{
			Args:    []string{"--flag"},
			Stdout:  &stdout,
			Timeout: 100 * time.Millisecond,
			Load: func(args []string) (testRunInputs, error) {
				loadCalled = true
				assert.Equal(t, []string{"--flag"}, args)
				return testRunInputs{outputDir: artifactDir, successMessage: "done"}, nil
			},
			Collect: func(ctx context.Context, inputs testRunInputs) (string, error) {
				collectCalled = true
				deadline, ok := ctx.Deadline()
				assert.True(t, ok)
				assert.WithinDuration(t, time.Now().Add(100*time.Millisecond), deadline, 75*time.Millisecond)
				assert.Equal(t, artifactDir, inputs.OutputDir())
				return "artifact", nil
			},
			Write: func(outputDir string, artifact string) error {
				writeCalled = true
				assert.Equal(t, artifactDir, outputDir)
				assert.Equal(t, "artifact", artifact)
				return nil
			},
		})
		require.NoError(t, err)
		assert.True(t, loadCalled)
		assert.True(t, collectCalled)
		assert.True(t, writeCalled)
		assert.Equal(t, "done\n", stdout.String())
	})

	t.Run("uses background context when timeout non positive", func(t *testing.T) {
		t.Parallel()

		err := Run(RunConfig[testRunInputs, string]{
			Timeout: 0,
			Load: func([]string) (testRunInputs, error) {
				return testRunInputs{outputDir: t.TempDir(), successMessage: "done"}, nil
			},
			Collect: func(ctx context.Context, _ testRunInputs) (string, error) {
				_, ok := ctx.Deadline()
				assert.False(t, ok)
				return "artifact", nil
			},
			Write: func(string, string) error {
				return nil
			},
		})
		require.NoError(t, err)
	})

	t.Run("returns load error", func(t *testing.T) {
		t.Parallel()

		collectCalled := false
		writeCalled := false

		err := Run(RunConfig[testRunInputs, string]{
			Stdout:  nil,
			Timeout: time.Second,
			Load: func([]string) (testRunInputs, error) {
				return testRunInputs{}, errTestLoadFailed
			},
			Collect: func(context.Context, testRunInputs) (string, error) {
				collectCalled = true
				return "", nil
			},
			Write: func(string, string) error {
				writeCalled = true
				return nil
			},
		})
		require.ErrorIs(t, err, errTestLoadFailed)
		assert.False(t, collectCalled)
		assert.False(t, writeCalled)
	})

	t.Run("returns collect error", func(t *testing.T) {
		t.Parallel()

		writeCalled := false

		err := Run(RunConfig[testRunInputs, string]{
			Timeout: time.Second,
			Load: func([]string) (testRunInputs, error) {
				return testRunInputs{outputDir: t.TempDir(), successMessage: "done"}, nil
			},
			Collect: func(context.Context, testRunInputs) (string, error) {
				return "", errTestCollectFailed
			},
			Write: func(string, string) error {
				writeCalled = true
				return nil
			},
		})
		require.ErrorIs(t, err, errTestCollectFailed)
		assert.False(t, writeCalled)
	})

	t.Run("returns write error", func(t *testing.T) {
		t.Parallel()

		err := Run(RunConfig[testRunInputs, string]{
			Timeout: time.Second,
			Load: func([]string) (testRunInputs, error) {
				return testRunInputs{outputDir: t.TempDir(), successMessage: "done"}, nil
			},
			Collect: func(context.Context, testRunInputs) (string, error) {
				return "artifact", nil
			},
			Write: func(string, string) error {
				return errTestWriteFailed
			},
		})
		require.ErrorIs(t, err, errTestWriteFailed)
	})
}
