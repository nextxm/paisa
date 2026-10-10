package server

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBackupZip_ContainsExpectedFilesAndExcludesDB(t *testing.T) {
	tmpDir := t.TempDir()

	// Prepare mock journal file
	journalFile := filepath.Join(tmpDir, "main.ledger")
	require.NoError(t, os.WriteFile(journalFile, []byte("2026-05-01 * Sample\n  Assets:Cash  100\n  Income:Salary\n"), 0644))

	// Prepare mock paisa.yaml config file with password
	configFile := filepath.Join(tmpDir, "paisa.yaml")
	configYAML := `
db_path: paisa.db
journal_path: main.ledger
default_currency: INR
user_accounts:
  - username: admin
    password: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
`
	require.NoError(t, os.WriteFile(configFile, []byte(configYAML), 0644))

	// Prepare mock database file (should NOT be included in backup)
	dbFile := filepath.Join(tmpDir, "paisa.db")
	require.NoError(t, os.WriteFile(dbFile, []byte("SQLITE_DUMMY_DATA"), 0644))

	// Prepare mock sheet file
	sheetFile := filepath.Join(tmpDir, "budget.sheet")
	require.NoError(t, os.WriteFile(sheetFile, []byte("Sheet Data"), 0644))

	// Load config pointing to tmpDir
	require.NoError(t, config.LoadConfig([]byte(configYAML), configFile))

	// 1. Test Backup without secrets (default)
	zipData, filename, err := GenerateBackupZip(false)
	require.NoError(t, err)
	assert.Contains(t, filename, "paisa-backup-")
	assert.NotEmpty(t, zipData)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	entryNames := make([]string, 0, len(reader.File))
	var configContent string

	for _, file := range reader.File {
		entryNames = append(entryNames, file.Name)
		if file.Name == "config/paisa.yaml" {
			rc, err := file.Open()
			require.NoError(t, err)
			b, err := io.ReadAll(rc)
			require.NoError(t, err)
			_ = rc.Close()
			configContent = string(b)
		}
	}

	// Verify journal and config are included
	assert.Contains(t, entryNames, "journals/main.ledger")
	assert.Contains(t, entryNames, "config/paisa.yaml")

	// Verify database file is EXCLUDED
	assert.NotContains(t, entryNames, "paisa.db")
	assert.NotContains(t, entryNames, "journals/paisa.db")

	// Verify secrets are redacted
	assert.Contains(t, configContent, "[REDACTED]")
	assert.NotContains(t, configContent, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	// 2. Test Backup with secrets (includeSecrets = true)
	zipDataSecrets, _, err := GenerateBackupZip(true)
	require.NoError(t, err)

	readerSecrets, err := zip.NewReader(bytes.NewReader(zipDataSecrets), int64(len(zipDataSecrets)))
	require.NoError(t, err)

	for _, file := range readerSecrets.File {
		if file.Name == "config/paisa.yaml" {
			rc, err := file.Open()
			require.NoError(t, err)
			b, err := io.ReadAll(rc)
			require.NoError(t, err)
			_ = rc.Close()
			assert.Contains(t, string(b), "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		}
	}
}
