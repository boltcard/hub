package web

import (
	"testing"
	"time"
)

// TestVersionCache_CachesWithinTTL verifies the underlying fetch runs at most
// once per TTL, so frequent About polls don't each hit Docker Hub.
func TestVersionCache_CachesWithinTTL(t *testing.T) {
	calls := 0
	c := newVersionCache(10*time.Minute, func() string {
		calls++
		return "0.1.0"
	})
	base := time.Unix(1_700_000_000, 0)
	cur := base
	c.now = func() time.Time { return cur }

	if got := c.get(); got != "0.1.0" {
		t.Fatalf("first get: expected 0.1.0, got %q", got)
	}
	// repeated gets within the TTL are served from cache
	cur = base.Add(5 * time.Minute)
	if got := c.get(); got != "0.1.0" {
		t.Fatalf("cached get: expected 0.1.0, got %q", got)
	}
	if calls != 1 {
		t.Fatalf("expected 1 fetch within TTL, got %d", calls)
	}

	// once the TTL elapses, the next get refreshes
	cur = base.Add(10 * time.Minute)
	c.get()
	if calls != 2 {
		t.Fatalf("expected a refetch at TTL, got %d fetches", calls)
	}
}

// TestVersionCache_KeepsLastGoodValueOnEmpty verifies a transient Docker Hub
// failure (empty result) doesn't blank the last known version.
func TestVersionCache_KeepsLastGoodValueOnEmpty(t *testing.T) {
	results := []string{"0.1.0", ""}
	i := 0
	c := newVersionCache(1*time.Minute, func() string {
		v := results[i]
		if i < len(results)-1 {
			i++
		}
		return v
	})
	base := time.Unix(1_700_000_000, 0)
	cur := base
	c.now = func() time.Time { return cur }

	if got := c.get(); got != "0.1.0" {
		t.Fatalf("first get: expected 0.1.0, got %q", got)
	}
	// past the TTL the refetch returns "" (error) — keep the last good value
	cur = base.Add(2 * time.Minute)
	if got := c.get(); got != "0.1.0" {
		t.Fatalf("expected last good value 0.1.0 retained on empty refetch, got %q", got)
	}
}
