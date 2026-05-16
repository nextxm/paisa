package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeJournalFile creates (or truncates) a ledger file with the given content.
func writeJournalFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestJournalWatcher_NoChangeDoesNotSetDirty(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	journalFile := filepath.Join(dir, "main.ledger")
	writeJournalFile(t, journalFile, "2024-01-01 * Opening\n  Assets:Bank  1000 USD\n  Equity:Opening\n")

	// Simulate a successful sync: store the hash and seed the watcher.
	w := NewJournalWatcher(db)
	w.RefreshFiles([]string{journalFile})

	h, err := utils.SHA256Files([]string{journalFile})
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, model.JournalHashKey, h))
	require.NoError(t, metadata.Set(db, model.JournalDirtyKey, "false"))

	// check() must not change the dirty flag when the file is unchanged.
	w.check()

	dirty, _ := metadata.GetOrDefault(db, model.JournalDirtyKey, "false")
	assert.Equal(t, "false", dirty)
}

func TestJournalWatcher_ExternalEditSetsDirty(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	journalFile := filepath.Join(dir, "main.ledger")
	writeJournalFile(t, journalFile, "2024-01-01 * Opening\n  Assets:Bank  1000 USD\n  Equity:Opening\n")

	// Simulate a successful sync: store the hash and seed the watcher.
	w := NewJournalWatcher(db)
	w.RefreshFiles([]string{journalFile})

	h, err := utils.SHA256Files([]string{journalFile})
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, model.JournalHashKey, h))
	require.NoError(t, metadata.Set(db, model.JournalDirtyKey, "false"))

	// Simulate an external edit: wait briefly so mtime advances, then rewrite.
	time.Sleep(5 * time.Millisecond)
	writeJournalFile(t, journalFile, "2024-01-01 * Opening\n  Assets:Bank  2000 USD\n  Equity:Opening\n")

	// check() must detect the change and set dirty = "true".
	w.check()

	dirty, _ := metadata.GetOrDefault(db, model.JournalDirtyKey, "false")
	assert.Equal(t, "true", dirty)
}

func TestJournalWatcher_TouchWithoutContentChangeDoesNotSetDirty(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	journalFile := filepath.Join(dir, "main.ledger")
	content := "2024-01-01 * Opening\n  Assets:Bank  1000 USD\n  Equity:Opening\n"
	writeJournalFile(t, journalFile, content)

	w := NewJournalWatcher(db)
	w.RefreshFiles([]string{journalFile})

	h, err := utils.SHA256Files([]string{journalFile})
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, model.JournalHashKey, h))
	require.NoError(t, metadata.Set(db, model.JournalDirtyKey, "false"))

	// Advance mtime without changing content.
	time.Sleep(5 * time.Millisecond)
	now := time.Now()
	require.NoError(t, os.Chtimes(journalFile, now, now))

	w.check()

	dirty, _ := metadata.GetOrDefault(db, model.JournalDirtyKey, "false")
	assert.Equal(t, "false", dirty)
}

func TestJournalWatcher_MissingFileDoesNotPanic(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	journalFile := filepath.Join(dir, "main.ledger")
	writeJournalFile(t, journalFile, "2024-01-01 * Opening\n  Assets:Bank  1000 USD\n  Equity:Opening\n")

	w := NewJournalWatcher(db)
	w.RefreshFiles([]string{journalFile})

	h, err := utils.SHA256Files([]string{journalFile})
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, model.JournalHashKey, h))
	require.NoError(t, metadata.Set(db, model.JournalDirtyKey, "false"))

	// Remove the file to simulate a deletion.
	require.NoError(t, os.Remove(journalFile))

	// check() must not panic when a watched file is missing.
	// The mtime check detects it as changed but SHA256 fails, so the
	// watcher returns early without setting dirty.
	assert.NotPanics(t, func() { w.check() })

	dirty, _ := metadata.GetOrDefault(db, model.JournalDirtyKey, "false")
	assert.Equal(t, "false", dirty)
}
