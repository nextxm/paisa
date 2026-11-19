package server

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/utils"
	"gorm.io/gorm"
)

// journalFileStamp captures the os.Stat fingerprint of a journal file at the
// time its contents were last hashed.
type journalFileStamp struct {
	path    string
	modTime time.Time
	size    int64
}

// journalHashCache memoizes the combined SHA-256 hash of the journal and all
// of its included files. The expensive work (spawning the ledger CLI to
// resolve includes and reading every file) is only redone when the journal
// path changes, any cached file's ModTime/Size changes, or the cache is
// explicitly invalidated (e.g. after an editor save, import or sync).
type journalHashCache struct {
	mu sync.Mutex

	listFiles func(journalPath string) ([]string, error)
	hashFiles func(paths []string) (string, error)

	valid       bool
	journalPath string
	stamps      []journalFileStamp
	hash        string
}

func newJournalHashCache(
	listFiles func(string) ([]string, error),
	hashFiles func([]string) (string, error),
) *journalHashCache {
	return &journalHashCache{listFiles: listFiles, hashFiles: hashFiles}
}

var defaultJournalHashCache = newJournalHashCache(
	func(journalPath string) ([]string, error) { return ledger.Cli().Files(journalPath) },
	utils.SHA256Files,
)

// Invalidate drops any cached state so the next CurrentHash call recomputes.
func (c *journalHashCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.valid = false
	c.stamps = nil
	c.hash = ""
}

// CurrentHash returns the combined hash of the journal at journalPath and its
// included files. An empty string is returned when hashing fails, matching
// the previous uncached behavior.
func (c *journalHashCache) CurrentHash(journalPath string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.valid && c.journalPath == journalPath && stampsUnchanged(c.stamps) {
		return c.hash
	}

	c.valid = false
	files, listErr := c.listFiles(journalPath)
	if listErr != nil {
		files = []string{journalPath}
	}

	// Stat before hashing so that a write racing with the hash will produce a
	// newer stamp on the next check and trigger a recompute.
	stamps, statErr := statJournalFiles(journalPath, files)
	hash, hashErr := c.hashFiles(files)
	if hashErr != nil {
		return ""
	}

	// Only cache when the include list was resolved reliably; otherwise keep
	// the old (uncached) behavior so a transient CLI failure is retried.
	if listErr == nil && statErr == nil {
		c.valid = true
		c.journalPath = journalPath
		c.stamps = stamps
		c.hash = hash
	}
	return hash
}

// statJournalFiles stamps every resolved journal file plus each directory
// containing one of them (and the journal's own directory). Directory mtimes
// change when entries are created, deleted or renamed, which lets the cache
// notice new files picked up by glob includes (e.g. `include *.ledger`) or
// previously-missing includes, even though no already-listed file changed.
func statJournalFiles(journalPath string, files []string) ([]journalFileStamp, error) {
	paths := make([]string, 0, len(files)*2+1)
	paths = append(paths, files...)
	seenDirs := make(map[string]bool)
	for _, f := range append([]string{journalPath}, files...) {
		dir := filepath.Dir(f)
		if !seenDirs[dir] {
			seenDirs[dir] = true
			paths = append(paths, dir)
		}
	}
	return statFiles(paths)
}

func statFiles(files []string) ([]journalFileStamp, error) {
	stamps := make([]journalFileStamp, 0, len(files))
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		stamps = append(stamps, journalFileStamp{path: f, modTime: info.ModTime(), size: info.Size()})
	}
	return stamps, nil
}

func stampsUnchanged(stamps []journalFileStamp) bool {
	for _, s := range stamps {
		info, err := os.Stat(s.path)
		if err != nil || !info.ModTime().Equal(s.modTime) || info.Size() != s.size {
			return false
		}
	}
	return true
}

// invalidateJournalDirtyCache should be called whenever paisa itself writes
// to journal files or changes the journal path.
func invalidateJournalDirtyCache() {
	defaultJournalHashCache.Invalidate()
}

// isJournalDirty reports whether the on-disk journal differs from the hash
// recorded at the last successful sync.
func isJournalDirty(db *gorm.DB) bool {
	currentHash := defaultJournalHashCache.CurrentHash(config.GetJournalPath())
	lastHash, _ := metadata.GetOrDefault(db, model.JournalHashKey, "")
	return currentHash != lastHash
}
