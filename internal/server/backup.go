package server

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

// GenerateBackupZip creates a zip archive containing the journal files,
// configuration file (paisa.yaml), and sheet files.
// It excludes the database cache file and sanitizes password hashes unless includeSecrets is true.
func GenerateBackupZip(includeSecrets bool) ([]byte, string, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	configDir := config.GetConfigDir()
	journalPath := config.GetJournalPath()
	journalDir := filepath.Dir(journalPath)

	// Helper to safely write a file into the zip archive.
	addFileToZip := func(zipPath string, content []byte, modTime time.Time) error {
		header := &zip.FileHeader{
			Name:     zipPath,
			Method:   zip.Deflate,
			Modified: modTime,
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = writer.Write(content)
		return err
	}

	// Helper to check if a target file is within an allowed root directory.
	isPathSafe := func(target string) bool {
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return false
		}
		configAbs, _ := filepath.Abs(configDir)
		journalAbs, _ := filepath.Abs(journalDir)

		targetClean := strings.ToLower(filepath.Clean(targetAbs))
		configClean := strings.ToLower(filepath.Clean(configAbs))
		journalClean := strings.ToLower(filepath.Clean(journalAbs))

		return strings.HasPrefix(targetClean, configClean) || strings.HasPrefix(targetClean, journalClean)
	}

	// 1. Add Journal Files
	journalFiles, err := ledger.Cli().Files(journalPath)
	if err != nil || len(journalFiles) == 0 {
		journalFiles = []string{journalPath}
	}

	// Include AddJournalPath if configured
	if addJournal := config.GetAddJournalPath(); addJournal != "" {
		journalFiles = append(journalFiles, addJournal)
	}

	addedFiles := make(map[string]bool)
	for _, file := range journalFiles {
		if file == "" || addedFiles[file] {
			continue
		}
		addedFiles[file] = true

		if !isPathSafe(file) {
			log.WithField("path", file).Warn("Skipping file outside allowed directory during backup")
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(journalDir, file)
		if err != nil || strings.HasPrefix(relPath, "..") {
			relPath = filepath.Base(file)
		}
		zipRelPath := filepath.ToSlash(relPath)
		err = addFileToZip("journals/"+zipRelPath, content, info.ModTime())
		if err != nil {
			return nil, "", fmt.Errorf("failed to add journal file %s: %w", file, err)
		}
	}

	// 2. Add Configuration File (paisa.yaml)
	configPath := config.GetConfigPath()
	if configPath != "" && isPathSafe(configPath) {
		if content, err := os.ReadFile(configPath); err == nil {
			info, _ := os.Stat(configPath)
			modTime := time.Now()
			if info != nil {
				modTime = info.ModTime()
			}

			finalConfigContent := content
			if !includeSecrets {
				// Sanitize password hashes
				var cfg config.Config
				if err := yaml.Unmarshal(content, &cfg); err == nil {
					for i := range cfg.UserAccounts {
						cfg.UserAccounts[i].Password = "[REDACTED]"
					}
					if sanitizedYAML, err := yaml.Marshal(cfg); err == nil {
						finalConfigContent = sanitizedYAML
					}
				}
			}

			_ = addFileToZip("config/paisa.yaml", finalConfigContent, modTime)
		}
	}

	// 3. Add Sheet Files
	sheetDir := config.GetSheetDir()
	if sheetDir != "" && isPathSafe(sheetDir) {
		entries, err := os.ReadDir(sheetDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				fullPath := filepath.Join(sheetDir, entry.Name())
				if !isPathSafe(fullPath) {
					continue
				}
				if strings.HasSuffix(entry.Name(), ".sheet") || strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".csv") {
					content, err := os.ReadFile(fullPath)
					if err != nil {
						continue
					}
					info, _ := entry.Info()
					modTime := time.Now()
					if info != nil {
						modTime = info.ModTime()
					}
					_ = addFileToZip("sheets/"+entry.Name(), content, modTime)
				}
			}
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to finalize backup zip: %w", err)
	}

	filename := fmt.Sprintf("paisa-backup-%s.zip", time.Now().Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}

// HandleExportBackup handles the GET /api/backup/export endpoint.
func HandleExportBackup(c *gin.Context) {
	includeSecrets := c.Query("include_secrets") == "true"

	log.WithFields(log.Fields{
		"event":           "backup_export",
		"include_secrets": includeSecrets,
		"ip":              c.ClientIP(),
	}).Info("Exporting system backup archive")

	zipData, filename, err := GenerateBackupZip(includeSecrets)
	if err != nil {
		RespondError(c, 500, ErrCodeInternalError, err.Error())
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(200, "application/zip", zipData)
}
