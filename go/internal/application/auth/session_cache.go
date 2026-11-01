package auth

import (
	"crypto/sha256"
	"sync"
	"time"
)

const (
	// defaultSessionCacheTTL bounds how long a positive "session exists"
	// answer is reused before the database is asked again. It is also the
	// worst-case delay before a revocation made on another replica takes
	// effect here; revocations made on this replica take effect immediately.
	defaultSessionCacheTTL = 10 * time.Second

	// defaultSessionCacheMax caps the number of cached sessions so a flood of
	// distinct valid tokens cannot grow memory without bound.
	defaultSessionCacheMax = 10000
)

type sessionCacheEntry struct {
	userID  string
	expires time.Time
}

// sessionCache is a small in-process cache of access tokens whose session row
// was recently confirmed to exist. Only positive answers are cached: a token
// whose session is gone is always re-checked against the database, so it can
// never be resurrected by the cache.
//
// Keys are SHA-256 digests of the token so the cache holds fixed-size keys and
// no bearer credentials.
type sessionCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	now     func() time.Time
	gen     uint64
	entries map[[sha256.Size]byte]sessionCacheEntry
}

func newSessionCache(ttl time.Duration, maxEntries int) *sessionCache {
	return &sessionCache{
		ttl:     ttl,
		max:     maxEntries,
		now:     time.Now,
		entries: make(map[[sha256.Size]byte]sessionCacheEntry),
	}
}

func sessionCacheKey(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}

// lookup reports whether token has a fresh cached session. On a miss it
// returns the current revocation generation, which must be passed to store so
// that a revocation racing with the database lookup is not overwritten by a
// stale positive answer.
func (c *sessionCache) lookup(token string) (hit bool, gen uint64) {
	key := sessionCacheKey(token)
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		if c.now().Before(e.expires) {
			return true, c.gen
		}
		delete(c.entries, key)
	}
	return false, c.gen
}

// store caches a confirmed session until min(now+ttl, tokenExpiry). It is a
// no-op when any revocation happened since gen was obtained from lookup.
func (c *sessionCache) store(token, userID string, tokenExpiry time.Time, gen uint64) {
	if c.ttl <= 0 {
		return
	}
	now := c.now()
	expires := now.Add(c.ttl)
	if !tokenExpiry.IsZero() && tokenExpiry.Before(expires) {
		expires = tokenExpiry
	}
	if !now.Before(expires) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if gen != c.gen {
		return
	}
	if len(c.entries) >= c.max {
		c.sweepLocked(now)
		if len(c.entries) >= c.max {
			// Still full of live entries: start over rather than tracking LRU
			// order. The cost is a burst of database lookups, not correctness.
			c.entries = make(map[[sha256.Size]byte]sessionCacheEntry)
		}
	}
	c.entries[sessionCacheKey(token)] = sessionCacheEntry{userID: userID, expires: expires}
}

func (c *sessionCache) sweepLocked(now time.Time) {
	for k, e := range c.entries {
		if !now.Before(e.expires) {
			delete(c.entries, k)
		}
	}
}

// evictToken drops a single token, e.g. on logout.
func (c *sessionCache) evictToken(token string) {
	key := sessionCacheKey(token)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	delete(c.entries, key)
}

// evictUser drops every cached token of a user, e.g. when all of the user's
// sessions are revoked.
func (c *sessionCache) evictUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	for k, e := range c.entries {
		if e.userID == userID {
			delete(c.entries, k)
		}
	}
}

func (c *sessionCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
