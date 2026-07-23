package web

import (
	"sync"
	"time"
)

// latestVersionTTL is how long a Docker Hub version check is cached. The About
// page polls /about frequently so the "Update available" button appears without
// a manual refresh; caching keeps those polls cheap and avoids hammering the
// Docker Hub registry (and its anonymous rate limits).
const latestVersionTTL = 10 * time.Minute

// versionCache memoises the result of a version-fetch function for a TTL. A
// failed fetch (empty string) keeps the last known-good value rather than
// blanking it, and still counts as a refresh so a persistent outage doesn't
// hammer the upstream.
type versionCache struct {
	mu        sync.Mutex
	value     string
	fetchedAt time.Time
	ttl       time.Duration
	fetch     func() string
	now       func() time.Time
}

func newVersionCache(ttl time.Duration, fetch func() string) *versionCache {
	return &versionCache{ttl: ttl, fetch: fetch, now: time.Now}
}

func (c *versionCache) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.fetchedAt.IsZero() || c.now().Sub(c.fetchedAt) >= c.ttl {
		v := c.fetch()
		c.fetchedAt = c.now()
		if v != "" {
			c.value = v
		}
	}
	return c.value
}

// latestVersionCache is the process-wide cache used by the About endpoint.
var latestVersionCache = newVersionCache(latestVersionTTL, CheckLatestVersion)
