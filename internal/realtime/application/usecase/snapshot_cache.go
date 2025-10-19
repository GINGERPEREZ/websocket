package usecase

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"mesaYaWs/internal/realtime/domain"
)

type snapshotCache struct {
	mu      sync.RWMutex
	entries map[string]map[string]*snapshotCacheEntry
}

type snapshotCacheEntry struct {
	sectionID string
	queryKey  string
	query     url.Values
	token     string
	snapshot  *domain.SectionSnapshot
	fetchedAt time.Time
}

func newSnapshotCache() *snapshotCache {
	return &snapshotCache{entries: make(map[string]map[string]*snapshotCacheEntry)}
}

func (c *snapshotCache) get(sectionID string, query url.Values) (*snapshotCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sec := c.entries[strings.TrimSpace(sectionID)]
	if sec == nil {
		return nil, false
	}
	key := canonicalQuery(query)
	entry, ok := sec[key]
	if !ok {
		return nil, false
	}
	return entry.clone(), true
}

func (c *snapshotCache) set(sectionID string, query url.Values, token string, snapshot *domain.SectionSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return
	}
	if c.entries[sectionID] == nil {
		c.entries[sectionID] = make(map[string]*snapshotCacheEntry)
	}
	key := canonicalQuery(query)
	c.entries[sectionID][key] = &snapshotCacheEntry{
		sectionID: sectionID,
		queryKey:  key,
		query:     cloneValues(query),
		token:     token,
		snapshot:  snapshot,
		fetchedAt: time.Now().UTC(),
	}
}

func (c *snapshotCache) delete(sectionID string, query url.Values) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return
	}
	if sec := c.entries[sectionID]; sec != nil {
		key := canonicalQuery(query)
		delete(sec, key)
		if len(sec) == 0 {
			delete(c.entries, sectionID)
		}
	}
}

func (c *snapshotCache) entriesForSection(sectionID string) []*snapshotCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sectionID = strings.TrimSpace(sectionID)
	sec := c.entries[sectionID]
	if len(sec) == 0 {
		return nil
	}
	results := make([]*snapshotCacheEntry, 0, len(sec))
	for _, entry := range sec {
		results = append(results, entry.clone())
	}
	return results
}

func (e *snapshotCacheEntry) clone() *snapshotCacheEntry {
	if e == nil {
		return nil
	}
	cloned := *e
	cloned.query = cloneValues(e.query)
	return &cloned
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ik := strings.ToLower(strings.TrimSpace(keys[i]))
		jk := strings.ToLower(strings.TrimSpace(keys[j]))
		if ik == jk {
			return keys[i] < keys[j]
		}
		return ik < jk
	})

	var builder strings.Builder
	for idx, key := range keys {
		if idx > 0 {
			builder.WriteByte('&')
		}
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		builder.WriteString(url.QueryEscape(normalizedKey))
		builder.WriteByte('=')
		vals := append([]string(nil), values[key]...)
		for i := range vals {
			vals[i] = strings.TrimSpace(vals[i])
		}
		sort.Strings(vals)
		if len(vals) > 0 {
			builder.WriteString(url.QueryEscape(vals[0]))
		}
	}
	return builder.String()
}

func cloneValues(src url.Values) url.Values {
	if src == nil {
		return nil
	}
	dst := make(url.Values, len(src))
	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}
