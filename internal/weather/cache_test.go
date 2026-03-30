package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheKey_RoundsCoordinates(t *testing.T) {
	// Coordinates that differ only beyond 2 decimal places should share a key.
	// 41.8781 -> "41.88", 41.8812 -> "41.88"
	// -87.6298 -> "-87.63", -87.6271 -> "-87.63"
	k1 := cacheKey(41.8781, -87.6298)
	k2 := cacheKey(41.8812, -87.6271)
	if k1 != k2 {
		t.Errorf("expected same key for nearby coords, got %q and %q", k1, k2)
	}

	// Sufficiently different coordinates should produce different keys.
	k3 := cacheKey(42.00, -87.63)
	if k1 == k3 {
		t.Errorf("expected different keys, both got %q", k1)
	}
}

func TestCache_HitReturnsCachedValue(t *testing.T) {
	c := newObservationCache(5 * time.Minute)

	obs := &Observation{
		DewpointF: 72.0,
		Station:   "KORD",
		City:      "Chicago",
	}

	c.set("test-key", obs)

	got, ok := c.get("test-key")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got.Station != obs.Station {
		t.Errorf("Station = %q, want %q", got.Station, obs.Station)
	}
}

func TestCache_MissOnEmptyCache(t *testing.T) {
	c := newObservationCache(5 * time.Minute)

	_, ok := c.get("nonexistent")
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}
}

func TestCache_ExpiredEntryTriggersMiss(t *testing.T) {
	c := newObservationCache(5 * time.Minute)

	// Use a controllable clock.
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	obs := &Observation{Station: "KORD"}
	c.set("test-key", obs)

	// Immediately should hit.
	if _, ok := c.get("test-key"); !ok {
		t.Fatal("expected cache hit immediately after set")
	}

	// Advance time past TTL.
	now = now.Add(6 * time.Minute)

	if _, ok := c.get("test-key"); ok {
		t.Fatal("expected cache miss after TTL expired")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := newObservationCache(5 * time.Minute)
	var wg sync.WaitGroup

	// Hammer the cache from many goroutines.
	for i := 0; i < 100; i++ {
		wg.Add(2)
		key := fmt.Sprintf("key-%d", i%10)
		go func(k string) {
			defer wg.Done()
			c.set(k, &Observation{Station: k})
		}(key)
		go func(k string) {
			defer wg.Done()
			c.get(k) // just must not panic or race
		}(key)
	}

	wg.Wait()
}

// countingDoer wraps a urlRewriteDoer and counts how many HTTP requests
// pass through it.
type countingDoer struct {
	inner HTTPDoer
	count atomic.Int64
}

func (d *countingDoer) Do(req *http.Request) (*http.Response, error) {
	d.count.Add(1)
	return d.inner.Do(req)
}

// newTestServer creates a test HTTP server that serves valid NWS-like responses
// for the happy path.
func newTestServer(dewC, tempC float64) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		stationsURL := fmt.Sprintf("http://%s/stations-cache-test", r.Host)
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, pointsJSON(stationsURL, "Chicago", "IL"))
	})
	mux.HandleFunc("/stations-cache-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, stationsJSON("KORD"))
	})
	mux.HandleFunc("/stations/KORD/observations/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, observationJSON(&dewC, &tempC, "2026-03-30T12:00:00Z"))
	})
	return httptest.NewServer(mux)
}

func TestGetObservation_CacheHitSkipsHTTP(t *testing.T) {
	ts := newTestServer(20.0, 25.0)
	defer ts.Close()

	doer := &countingDoer{
		inner: &urlRewriteDoer{
			target:     ts.URL,
			underlying: ts.Client(),
		},
	}
	client := &Client{
		httpClient: doer,
		cache:      newObservationCache(10 * time.Minute),
	}

	ctx := context.Background()

	// First call: cache miss, should make HTTP requests.
	obs1, err := client.GetObservation(ctx, 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	firstCount := doer.count.Load()
	if firstCount == 0 {
		t.Fatal("expected HTTP requests on first call, got 0")
	}

	// Second call: same coords, should be served from cache.
	obs2, err := client.GetObservation(ctx, 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	secondCount := doer.count.Load()
	if secondCount != firstCount {
		t.Errorf("expected no additional HTTP calls on cache hit; had %d, now %d", firstCount, secondCount)
	}

	// Should be the same observation.
	if obs1.Station != obs2.Station {
		t.Errorf("cached Station = %q, original = %q", obs2.Station, obs1.Station)
	}
}

func TestGetObservation_ExpiredCacheTriggersRefetch(t *testing.T) {
	ts := newTestServer(20.0, 25.0)
	defer ts.Close()

	doer := &countingDoer{
		inner: &urlRewriteDoer{
			target:     ts.URL,
			underlying: ts.Client(),
		},
	}

	cache := newObservationCache(5 * time.Minute)
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	client := &Client{
		httpClient: doer,
		cache:      cache,
	}

	ctx := context.Background()

	// First call: cache miss.
	_, err := client.GetObservation(ctx, 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	afterFirst := doer.count.Load()

	// Advance time past TTL.
	now = now.Add(6 * time.Minute)

	// Third call: cache should be expired, so new HTTP requests.
	_, err = client.GetObservation(ctx, 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("post-expiry call error: %v", err)
	}
	afterExpiry := doer.count.Load()
	if afterExpiry <= afterFirst {
		t.Errorf("expected new HTTP calls after cache expiry; had %d, now %d", afterFirst, afterExpiry)
	}
}

func TestGetObservation_ConcurrentFetchesAreSafe(t *testing.T) {
	ts := newTestServer(20.0, 25.0)
	defer ts.Close()

	doer := &countingDoer{
		inner: &urlRewriteDoer{
			target:     ts.URL,
			underlying: ts.Client(),
		},
	}
	client := &Client{
		httpClient: doer,
		cache:      newObservationCache(10 * time.Minute),
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	// Fire off many concurrent requests. The race detector will catch
	// unsafe access if we run with -race.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.GetObservation(ctx, 41.8781, -87.6298)
		}()
	}

	wg.Wait()
}
