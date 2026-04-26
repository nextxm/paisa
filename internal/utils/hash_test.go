package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSHA256File_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello paisa")
	err := os.WriteFile(path, content, 0600)
	assert.NoError(t, err)

	got, err := SHA256File(path)
	assert.NoError(t, err)

	// Cross-check against the string-based helper.
	want := Sha256(string(content))
	assert.Equal(t, want, got)
}

func TestSHA256File_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	err := os.WriteFile(path, []byte{}, 0600)
	assert.NoError(t, err)

	got, err := SHA256File(path)
	assert.NoError(t, err)
	assert.Equal(t, Sha256(""), got)
}

func TestSHA256File_MissingFile(t *testing.T) {
	_, err := SHA256File("/nonexistent/path/to/file.ledger")
	assert.Error(t, err)
}
