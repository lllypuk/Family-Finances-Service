package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/recognize"
	"family-budget-service/internal/testhelpers"
)

func TestRunRecognize_WithoutHost(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	path := filepath.Join(t.TempDir(), "untouched.db")
	t.Setenv("DATABASE_PATH", path)
	t.Setenv("LLM_OLLAMA_HOST", "")

	var stdout, stderr bytes.Buffer
	err := recognizeCommand(&stderr)(context.Background(), []string{"a.png"}, nil, &stdout)

	require.ErrorIs(t, err, errRecognizeDisabled)
	assert.Empty(t, stdout.String())
	assert.NoFileExists(t, path)
}

func TestExitCode_RecognizeDisabledIsUsage(t *testing.T) {
	assert.Equal(t, exitUsage, exitCode(fmt.Errorf("%s: %w", cmdRecognize, errRecognizeDisabled)))
	assert.Equal(t, 1, exitCode(errors.New("boom")))
}

func TestParseRecognizeArgs_FileCount(t *testing.T) {
	_, err := parseRecognizeArgs(nil)
	require.ErrorContains(t, err, "got 0")

	_, err = parseRecognizeArgs([]string{"1", "2", "3", "4", "5", "6"})
	require.ErrorContains(t, err, "got 6")

	paths, err := parseRecognizeArgs([]string{"a.png", "b.jpg"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.png", "b.jpg"}, paths)
}

func TestReadImages_RejectsNonImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("not an image"), 0o600))

	_, err := readImages([]string{path})

	require.ErrorIs(t, err, recognize.ErrNotImage)
}
