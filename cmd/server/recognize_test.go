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

func TestRunRecognize_MissingDatabaseNotCreated(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	dir := t.TempDir()
	image := filepath.Join(dir, "shot.png")
	require.NoError(t, os.WriteFile(image, testhelpers.PNGImage(t, 8, 8), 0o600))
	path := filepath.Join(dir, "missing.db")
	t.Setenv("DATABASE_PATH", path)
	t.Setenv("LLM_OLLAMA_HOST", "http://127.0.0.1:1")

	var stdout, stderr bytes.Buffer
	err := recognizeCommand(&stderr)(context.Background(), []string{image}, nil, &stdout)

	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Empty(t, stdout.String())
	assert.NoFileExists(t, path)
}

func TestRunRecognize_InvalidHost(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "untouched.db"))
	t.Setenv("LLM_OLLAMA_HOST", "ftp://127.0.0.1:1")

	var stdout, stderr bytes.Buffer
	err := recognizeCommand(&stderr)(context.Background(), []string{"a.png"}, nil, &stdout)

	require.ErrorContains(t, err, "LLM_OLLAMA_HOST")
	assert.NotErrorIs(t, err, errRecognizeDisabled)
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
