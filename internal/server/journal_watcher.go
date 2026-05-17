package server

import (
	"os"
	"sync"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// journalPollInterval is how often the watcher checks file modification times.
const journalPollInterval = 10 * time.Second

// JournalWatcher monitors watched journal files for out-of-band edits.
// When a file changes outside the app (based on mtime, confirmed by SHA256),
// it sets JournalDirtyKey = "true" in the metadata table so that
// GET /api/config and GET /api/journal/status immediately reflect the change.
type JournalWatcher struct {
	db     *gorm.DB
	mu     sync.RWMutex
	files  []string             // cached file list from last sync / seed
	mtimes map[string]time.Time // last-known mtime per file
}

// NewJournalWatcher creates a JournalWatcher backed by the given database.
func NewJournalWatcher(db *gorm.DB) *JournalWatcher {
	return &JournalWatcher{db: db}
}

// RefreshFilesFromConfig discovers all journal files via the ledger CLI and
// resets the watched-file list together with their current modification times.
// Call this after every successful journal sync so the watcher tracks the
// most up-to-date set of included files.
func (w *JournalWatcher) RefreshFilesFromConfig() {
	journalPath := config.GetJournalPath()
	if journalPath == "" {
		return
	}
	files, err := ledger.Cli().Files(journalPath)
	if err != nil {
		files = []string{journalPath}
	}
	w.RefreshFiles(files)
}

// RefreshFiles updates the watched file list to files and snapshots their
// current modification times.  Subsequent check() calls will compare against
// these timestamps.
func (w *JournalWatcher) RefreshFiles(files []string) {
	mtimes := make(map[string]time.Time, len(files))
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			mtimes[f] = info.ModTime()
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.files = files
	w.mtimes = mtimes
}

// Start seeds the file list from the current config and launches the
// background polling goroutine.
func (w *JournalWatcher) Start() {
	w.RefreshFilesFromConfig()
	go w.run()
}

func (w *JournalWatcher) run() {
	ticker := time.NewTicker(journalPollInterval)
	defer ticker.Stop()
	for range ticker.C {
		w.check()
	}
}

// check is the hot-path poll.  It uses cheap mtime comparisons first and only
// reads file contents (SHA256) when at least one mtime has advanced.
func (w *JournalWatcher) check() {
	w.mu.RLock()
	files := w.files
	mtimes := w.mtimes
	w.mu.RUnlock()

	if len(files) == 0 {
		return
	}

	// --- pass 1: mtime (cheap: just os.Stat) ---
	anyChanged := false
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			// file gone / inaccessible → treat as changed
			anyChanged = true
			break
		}
		if mt, ok := mtimes[f]; !ok || info.ModTime().After(mt) {
			anyChanged = true
			break
		}
	}
	if !anyChanged {
		return
	}

	// --- pass 2: SHA256 to confirm real content change (avoids false positives
	//     from `touch` or filesystem timestamp rounding) ---
	currentHash, err := utils.SHA256Files(files)
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.watcher", "error": err}).
			Debug("Failed to hash journal files during watch check")
		return
	}

	lastHash, err := metadata.GetOrDefault(w.db, model.JournalHashKey, "")
	if err != nil {
		return
	}

	// Refresh mtimes regardless of whether content changed so we don't
	// re-enter pass 2 on every poll after a no-op touch.
	newMtimes := make(map[string]time.Time, len(files))
	for _, f := range files {
		if info, err2 := os.Stat(f); err2 == nil {
			newMtimes[f] = info.ModTime()
		}
	}
	w.mu.Lock()
	w.mtimes = newMtimes
	w.mu.Unlock()

	if currentHash == lastHash {
		// mtime moved but bytes are identical (e.g. no-op save / touch)
		return
	}

	// Content has changed outside the app — mark the journal as dirty.
	if err := metadata.Set(w.db, model.JournalDirtyKey, "true"); err != nil {
		log.WithFields(log.Fields{"stage": "journal.watcher", "error": err}).
			Warn("Failed to persist journal dirty flag")
		return
	}
	log.WithFields(log.Fields{"stage": "journal.watcher"}).
		Info("Journal files changed outside the app; marked dirty")
}
