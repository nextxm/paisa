package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCountingJournalHashCache(files map[string][]string) (*journalHashCache, *int, *int) {
	listCalls, hashCalls := 0, 0
	c := newJournalHashCache(
		func(journalPath string) ([]string, error) {
			listCalls++
			return files[journalPath], nil
		},
		func(paths []string) (string, error) {
			hashCalls++
			return utils.SHA256Files(paths)
		},
	)
	return c, &listCalls, &hashCalls
}

func TestJournalHashCacheReusesHashWhenFilesUnchanged(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ledger")
	includePath := filepath.Join(dir, "include.ledger")
	require.NoError(t, os.WriteFile(mainPath, []byte("include include.ledger\n"), 0644))
	require.NoError(t, os.WriteFile(includePath, []byte("2026-01-01 * A\n"), 0644))

	c, listCalls, hashCalls := newCountingJournalHashCache(map[string][]string{
		mainPath: {mainPath, includePath},
	})

	first := c.CurrentHash(mainPath)
	require.NotEmpty(t, first)
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, c.CurrentHash(mainPath))
	}
	assert.Equal(t, 1, *listCalls, "ledger file listing should run once while files are unchanged")
	assert.Equal(t, 1, *hashCalls, "files should be hashed once while unchanged")

	// Modify an included file (different size and mtime).
	require.NoError(t, os.WriteFile(includePath, []byte("2026-01-01 * A\n2026-01-02 * B\n"), 0644))
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(includePath, future, future))

	second := c.CurrentHash(mainPath)
	assert.NotEqual(t, first, second, "modification should be detected")
	assert.Equal(t, 2, *listCalls)
	assert.Equal(t, 2, *hashCalls)

	assert.Equal(t, second, c.CurrentHash(mainPath))
	assert.Equal(t, 2, *hashCalls, "unchanged files after refresh should hit the cache")
}

func TestJournalHashCacheDetectsSameSizeModTimeChange(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ledger")
	require.NoError(t, os.WriteFile(mainPath, []byte("aaaa"), 0644))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(mainPath, past, past))

	c, _, hashCalls := newCountingJournalHashCache(map[string][]string{mainPath: {mainPath}})
	first := c.CurrentHash(mainPath)

	require.NoError(t, os.WriteFile(mainPath, []byte("bbbb"), 0644))
	now := time.Now()
	require.NoError(t, os.Chtimes(mainPath, now, now))

	assert.NotEqual(t, first, c.CurrentHash(mainPath))
	assert.Equal(t, 2, *hashCalls)
}

func TestJournalHashCacheInvalidationAndPathChange(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.ledger")
	pathB := filepath.Join(dir, "b.ledger")
	require.NoError(t, os.WriteFile(pathA, []byte("a"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("b"), 0644))

	c, listCalls, hashCalls := newCountingJournalHashCache(map[string][]string{
		pathA: {pathA},
		pathB: {pathB},
	})

	hashA := c.CurrentHash(pathA)
	hashB := c.CurrentHash(pathB)
	assert.NotEqual(t, hashA, hashB, "changing the journal path must not reuse the old hash")
	assert.Equal(t, 2, *listCalls)

	c.CurrentHash(pathB)
	assert.Equal(t, 2, *hashCalls)

	c.Invalidate()
	assert.Equal(t, hashB, c.CurrentHash(pathB))
	assert.Equal(t, 3, *listCalls, "explicit invalidation should force a refresh")
	assert.Equal(t, 3, *hashCalls)
}

func TestJournalHashCacheDetectsNewFileFromGlobInclude(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ledger")
	require.NoError(t, os.WriteFile(mainPath, []byte("include *.journal\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.journal"), []byte("a"), 0644))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(dir, past, past))

	listCalls := 0
	c := newJournalHashCache(
		func(journalPath string) ([]string, error) {
			listCalls++
			matches, err := filepath.Glob(filepath.Join(dir, "*.journal"))
			return append([]string{journalPath}, matches...), err
		},
		utils.SHA256Files,
	)

	first := c.CurrentHash(mainPath)
	assert.Equal(t, first, c.CurrentHash(mainPath))
	assert.Equal(t, 1, listCalls)

	// A new file matching the glob; main.ledger and a.journal are untouched.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.journal"), []byte("b"), 0644))
	now := time.Now()
	require.NoError(t, os.Chtimes(dir, now, now))

	assert.NotEqual(t, first, c.CurrentHash(mainPath), "new glob-included file should be detected")
	assert.Equal(t, 2, listCalls)
}

func TestJournalHashCacheDeletedFileForcesRefresh(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ledger")
	includePath := filepath.Join(dir, "include.ledger")
	require.NoError(t, os.WriteFile(mainPath, []byte("main"), 0644))
	require.NoError(t, os.WriteFile(includePath, []byte("inc"), 0644))

	files := map[string][]string{mainPath: {mainPath, includePath}}
	c, listCalls, _ := newCountingJournalHashCache(files)
	require.NotEmpty(t, c.CurrentHash(mainPath))

	require.NoError(t, os.Remove(includePath))
	files[mainPath] = []string{mainPath}
	assert.NotEmpty(t, c.CurrentHash(mainPath))
	assert.Equal(t, 2, *listCalls)
}
