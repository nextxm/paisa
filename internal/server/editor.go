package server

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"os"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type LedgerFile struct {
	Name      string   `json:"name"`
	Content   string   `json:"content"`
	Versions  []string `json:"versions"`
	Operation string   `json:"operation"`
}

func GetFiles(db *gorm.DB) gin.H {
	var accounts []string
	var payees []string
	var commodities []string
	db.Model(&posting.Posting{}).Distinct().Pluck("Account", &accounts)
	db.Model(&posting.Posting{}).Distinct().Pluck("Payee", &payees)
	db.Model(&posting.Posting{}).Distinct().Pluck("Commodity", &commodities)

	path := config.GetJournalPath()

	files := []*LedgerFile{}
	dir := filepath.Dir(path)
	paths, _ := doublestar.FilepathGlob(dir + "/**/*" + filepath.Ext(path))

	for _, path = range paths {
		file, err := readLedgerFileWithVersions(dir, path)
		if err != nil {
			log.Warn(err)
			continue
		}
		files = append(files, file)
	}

	return gin.H{"files": files, "accounts": accounts, "payees": payees, "commodities": commodities}
}

func GetFile(file LedgerFile) (gin.H, error) {
	path := config.GetJournalPath()
	dir := filepath.Dir(path)
	ledgerFile, err := readLedgerFile(dir, filepath.Join(dir, file.Name))
	if err != nil {
		return nil, err
	}
	return gin.H{"file": ledgerFile}, nil
}

func DeleteBackups(file LedgerFile) (gin.H, error) {
	path := config.GetJournalPath()
	dir := filepath.Dir(path)

	if !config.GetConfig().Readonly {
		versions, _ := filepath.Glob(filepath.Join(dir, file.Name+".backup.*"))
		for _, version := range versions {
			err := os.Remove(version)
			if err != nil {
				return nil, fmt.Errorf("failed to remove backup %s: %w", version, err)
			}
		}
	}

	ledgerFile, err := readLedgerFileWithVersions(dir, filepath.Join(dir, file.Name))
	if err != nil {
		return nil, err
	}
	return gin.H{"file": ledgerFile}, nil
}

func SaveFile(db *gorm.DB, file LedgerFile) gin.H {
	errors, _, err := validateFile(file)
	if err != nil {
		return gin.H{"errors": errors, "saved": false, "message": "Validation failed"}
	}

	path := config.GetJournalPath()
	dir := filepath.Dir(path)

	filePath, err := utils.BuildSubPath(dir, file.Name)
	if err != nil {
		log.Warn(err)
		return gin.H{"errors": errors, "saved": false, "message": "Invalid file name"}
	}

	backupPath := filePath + ".backup." + time.Now().Format("2006-01-02-15-04-05.000")

	err = os.MkdirAll(filepath.Dir(filePath), 0700)
	if err != nil {
		log.Warn(err)
		return gin.H{"errors": errors, "saved": false, "message": "Failed to create directory"}
	}

	fileStat, err := os.Stat(filePath)
	if err != nil && file.Operation != "overwrite" && file.Operation != "create" {
		log.Warn(err)
		return gin.H{"errors": errors, "saved": false, "message": "File does not exist"}
	}

	var perm os.FileMode = 0644
	if err == nil {
		if file.Operation == "create" {
			return gin.H{"errors": errors, "saved": false, "message": "File already exists"}
		}

		perm = fileStat.Mode().Perm()
		existingContent, err := os.ReadFile(filePath)
		if err != nil {
			log.Warn(err)
			return gin.H{"errors": errors, "saved": false, "message": "Failed to read file"}
		}

		err = os.WriteFile(backupPath, existingContent, perm)
		if err != nil {
			log.Warn(err)
			return gin.H{"errors": errors, "saved": false, "message": "Failed to create backup"}
		}
	}

	err = os.WriteFile(filePath, []byte(file.Content), perm)
	if err != nil {
		log.Warn(err)
		return gin.H{"errors": errors, "saved": false, "message": "Failed to write file"}
	}

	Sync(db, SyncRequest{Journal: true})

	savedFile, err := readLedgerFileWithVersions(dir, filePath)
	if err != nil {
		log.Warn(err)
		return gin.H{"errors": errors, "saved": true, "message": "Failed to read saved file"}
	}
	return gin.H{"errors": errors, "saved": true, "file": savedFile}
}

func ValidateFile(file LedgerFile) (gin.H, error) {
	errors, output, err := validateFile(file)
	if err != nil {
		return nil, err
	}
	return gin.H{"errors": errors, "output": output}, nil
}

func validateFile(file LedgerFile) ([]ledger.LedgerFileError, string, error) {
	dir := filepath.Dir(filepath.Dir(config.GetJournalPath()) + "/" + file.Name)

	tmpfile, err := os.CreateTemp(dir, "paisa-tmp-")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %w", err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(file.Content)); err != nil {
		return nil, "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpfile.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return ledger.Cli().ValidateFile(tmpfile.Name())
}

func readLedgerFile(dir string, path string) (*LedgerFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	name, err := filepath.Rel(dir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to compute relative path for %s: %w", path, err)
	}

	return &LedgerFile{
		Name:    name,
		Content: string(content),
	}, nil
}

func readLedgerFileWithVersions(dir string, path string) (*LedgerFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	versions, _ := filepath.Glob(filepath.Join(filepath.Dir(path), filepath.Base(path)+".backup.*"))
	versionPaths := lo.FilterMap(versions, func(vPath string, _ int) (string, bool) {
		name, err := filepath.Rel(dir, vPath)
		if err != nil {
			log.Warn(fmt.Errorf("failed to compute relative path for %s: %w", vPath, err))
			return "", false
		}
		return name, true
	})
	sort.Sort(sort.Reverse(sort.StringSlice(versionPaths)))

	name, err := filepath.Rel(dir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to compute relative path for %s: %w", path, err)
	}

	return &LedgerFile{
		Name:     name,
		Content:  string(content),
		Versions: versionPaths,
	}, nil
}
