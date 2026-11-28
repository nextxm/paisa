package prediction

import (
	"fmt"
	"math"
	"regexp"
	"sync"

	"strings"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type index struct {
	Docs   map[string]map[string]int64 `json:"docs"`
	Tokens map[string]map[string]int64 `json:"tokens"`
}

type tfidfCache struct {
	mu     sync.RWMutex
	loadMu sync.Mutex
	loaded bool
	vector map[string]map[string]float64
	index  index
}

var cache tfidfCache

func ensureVectorCacheLoaded(db *gorm.DB) {
	cache.mu.RLock()
	loaded := cache.loaded
	cache.mu.RUnlock()
	if loaded {
		return
	}

	cache.loadMu.Lock()
	defer cache.loadMu.Unlock()

	cache.mu.RLock()
	loaded = cache.loaded
	cache.mu.RUnlock()
	if loaded {
		return
	}

	postings := query.Init(db).All()
	idx := buldIndex(postings)
	vector := make(map[string]map[string]float64, len(idx.Docs))
	for account := range idx.Docs {
		vector[account] = tfidf(account, idx)
	}

	cache.mu.Lock()
	cache.index = idx
	cache.vector = vector
	cache.loaded = true
	cache.mu.Unlock()
}

func ClearCache() {
	cache.mu.Lock()
	cache.loaded = false
	cache.vector = nil
	cache.index = index{}
	cache.mu.Unlock()
}

func buldIndex(postings []posting.Posting) index {
	idx := index{
		Docs:   make(map[string]map[string]int64),
		Tokens: make(map[string]map[string]int64),
	}
	for _, p := range postings {
		if idx.Docs[p.Account] == nil {
			idx.Docs[p.Account] = make(map[string]int64)
		}
		for _, token := range tokenize(strings.Join([]string{strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", p.Amount.InexactFloat64()), "0"), "."), p.Payee}, " ")) {
			if idx.Tokens[token] == nil {
				idx.Tokens[token] = make(map[string]int64)
			}
			idx.Tokens[token][p.Account]++
			idx.Docs[p.Account][token]++
		}
	}
	return idx
}

func tfidf(account string, idx index) map[string]float64 {
	tfidf := make(map[string]float64)
	for token, freq := range idx.Docs[account] {
		tf := float64(freq) / float64(len(idx.Docs[account]))
		idf := math.Log(float64(len(idx.Docs))/(1+float64(len(idx.Tokens[token])))) + 1
		tfidf[token] = tf * idf
	}
	return tfidf
}

func tokenize(s string) []string {
	tokens := regexp.MustCompile("[ .()/:]+").Split(s, -1)
	tokens = lo.Map(tokens, func(s string, _ int) string {
		return strings.ToLower(s)
	})
	return lo.Filter(tokens, func(s string, _ int) bool {
		return strings.TrimSpace(s) != ""
	})
}

func GetTfIdf(db *gorm.DB) gin.H {
	ensureVectorCacheLoaded(db)
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return gin.H{"tf_idf": cache.vector, "index": cache.index}
}
