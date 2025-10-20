package usecase

import (
	"strings"
	"sync"
	"time"

	"mesaYaWs/internal/realtime/domain"
)

const (
	cacheKindList  = "list"
	cacheKindItem  = "item"
	cacheDelimiter = ":"
)

type snapshotCache struct {
	mu      sync.RWMutex
	entries map[string]map[string]*snapshotCacheEntry
}

type snapshotCacheEntry struct {
	sectionID   string
	scope       string
	kind        string
	key         string
	listOptions domain.PagedQuery
	resourceID  string
	token       string
	snapshot    *domain.SectionSnapshot
	fetchedAt   time.Time
}

func newSnapshotCache() *snapshotCache {
	return &snapshotCache{entries: make(map[string]map[string]*snapshotCacheEntry)}
}

func (c *snapshotCache) set(sectionID, scope, kind string, options domain.PagedQuery, resourceID, token string, snapshot *domain.SectionSnapshot) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries[sectionID] == nil {
		c.entries[sectionID] = make(map[string]*snapshotCacheEntry)
	}
	key := cacheEntryKey(scope, kind, options, resourceID)
	c.entries[sectionID][key] = &snapshotCacheEntry{
		sectionID:   sectionID,
		scope:       strings.TrimSpace(scope),
		kind:        kind,
		key:         key,
		listOptions: options,
		resourceID:  strings.TrimSpace(resourceID),
		token:       token,
		snapshot:    snapshot,
		fetchedAt:   time.Now().UTC(),
	}
}

func (c *snapshotCache) get(sectionID, scope, kind string, options domain.PagedQuery, resourceID string) (*snapshotCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sec := c.entries[strings.TrimSpace(sectionID)]
	if sec == nil {
		return nil, false
	}
	key := cacheEntryKey(scope, kind, options, resourceID)
	entry, ok := sec[key]
	if !ok {
		return nil, false
	}
	return entry.clone(), true
}

func (c *snapshotCache) delete(sectionID, scope, kind string, options domain.PagedQuery, resourceID string) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if sec := c.entries[sectionID]; sec != nil {
		key := cacheEntryKey(scope, kind, options, resourceID)
		delete(sec, key)
		if len(sec) == 0 {
			delete(c.entries, sectionID)
		}
	}
}

func (c *snapshotCache) entriesForSection(sectionID string) []*snapshotCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sec := c.entries[strings.TrimSpace(sectionID)]
	if len(sec) == 0 {
		return nil
	}
	results := make([]*snapshotCacheEntry, 0, len(sec))
	for _, entry := range sec {
		results = append(results, entry.clone())
	}
	return results
}

func (c *snapshotCache) sectionIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.entries) == 0 {
		return nil
	}
	results := make([]string, 0, len(c.entries))
	for sectionID := range c.entries {
		trimmed := strings.TrimSpace(sectionID)
		if trimmed == "" {
			continue
		}
		results = append(results, trimmed)
	}
	return results
}

func (e *snapshotCacheEntry) clone() *snapshotCacheEntry {
	if e == nil {
		return nil
	}
	cloned := *e
	return &cloned
}

func cacheEntryKey(scope, kind string, options domain.PagedQuery, resourceID string) string {
	trimmedScope := strings.ToLower(strings.TrimSpace(scope))
	switch strings.ToLower(kind) {
	case cacheKindItem:
		return trimmedScope + cacheDelimiter + cacheKindItem + cacheDelimiter + strings.TrimSpace(resourceID)
	default:
		return trimmedScope + cacheDelimiter + cacheKindList + cacheDelimiter + options.CanonicalKey()
	}
}
