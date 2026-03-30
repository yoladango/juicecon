package weather

import (
	"fmt"
	"sync"
	"time"
)

const defaultCacheTTL = 10 * time.Minute

// cacheEntry holds a cached observation and its expiration time.
type cacheEntry struct {
	obs       *Observation
	expiresAt time.Time
}

// observationCache is a concurrent-safe in-memory cache for weather observations.
type observationCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	now     func() time.Time // injectable for testing
}

// newObservationCache creates a cache with the given TTL.
func newObservationCache(ttl time.Duration) *observationCache {
	return &observationCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

// cacheKey builds a cache key from lat/lon, rounded to 2 decimal places
// so nearby requests share the same cached observation.
func cacheKey(lat, lon float64) string {
	return fmt.Sprintf("%.2f,%.2f", lat, lon)
}

// get returns a cached observation if present and not expired. The second
// return value indicates whether the cache hit was valid.
func (c *observationCache) get(key string) (*Observation, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if c.now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.obs, true
}

// set stores an observation in the cache with the configured TTL.
func (c *observationCache) set(key string, obs *Observation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		obs:       obs,
		expiresAt: c.now().Add(c.ttl),
	}
}

// CacheStats holds cache metrics.
type CacheStats struct {
	Entries int `json:"entries"`
}

// stats returns the current number of non-expired entries in the cache.
func (c *observationCache) stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := c.now()
	count := 0
	for _, entry := range c.entries {
		if now.Before(entry.expiresAt) || now.Equal(entry.expiresAt) {
			count++
		}
	}
	return CacheStats{Entries: count}
}
