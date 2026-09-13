package discovery

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

var (
	// ErrCachePartition reports absent or invalid caller cache partitioning.
	ErrCachePartition = errors.New("discovery: invalid cache partition")
	// ErrCacheLimit reports that the configured partition bound was reached.
	ErrCacheLimit = errors.New("discovery: cache partition limit exceeded")
)

// Discoverer produces discovery snapshots.
type Discoverer interface {
	Discover(context.Context) (Snapshot, error)
}

// CacheKeyFunc derives a stable, non-sensitive partition key from the caller
// context. Callers must include every authorization and tenant dimension that
// can change the discovered snapshot.
type CacheKeyFunc func(context.Context) (string, error)

// CacheOptions configures explicit cache partitioning and bounds retained
// snapshots.
type CacheOptions struct {
	Key           CacheKeyFunc
	MaxKeyBytes   int
	MaxPartitions int
}

// DefaultCacheOptions returns finite cache bounds for an explicit key function.
func DefaultCacheOptions(key CacheKeyFunc) CacheOptions {
	return CacheOptions{Key: key, MaxKeyBytes: 1_024, MaxPartitions: 1_024}
}

// Cache is an explicitly owned, concurrency-safe discovery cache.
// NewPartitionedCache deduplicates concurrent misses within each partition;
// the caller that starts a refresh owns its context, and canceling a waiter
// does not cancel that shared refresh.
type Cache struct {
	discoverer Discoverer
	options    CacheOptions

	mu      sync.Mutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	snapshot Snapshot
	valid    bool
	loading  bool
	done     chan struct{}
}

// NewPartitionedCache wraps a discoverer with bounded authorization-aware
// snapshot retention and concurrent miss deduplication.
func NewPartitionedCache(discoverer Discoverer, options CacheOptions) (*Cache, error) {
	if nilDiscoverer(discoverer) {
		return nil, ErrInvalidOptions
	}
	if options.Key == nil || options.MaxKeyBytes <= 0 || options.MaxPartitions <= 0 {
		return nil, ErrCachePartition
	}
	return &Cache{
		discoverer: discoverer,
		options:    options,
		entries:    make(map[string]*cacheEntry),
	}, nil
}

func nilDiscoverer(discoverer Discoverer) bool {
	if discoverer == nil {
		return true
	}
	value := reflect.ValueOf(discoverer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Discover returns the cached snapshot or synchronously performs one
// deduplicated refresh.
func (cache *Cache) Discover(ctx context.Context) (Snapshot, error) {
	if cache == nil || cache.discoverer == nil || ctx == nil {
		return Snapshot{}, ErrInvalidOptions
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	key, err := cache.options.Key(ctx)
	if err != nil || key == "" || len(key) > cache.options.MaxKeyBytes {
		if contextErr := ctx.Err(); contextErr != nil {
			return Snapshot{}, contextErr
		}
		return Snapshot{}, ErrCachePartition
	}
	for {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		cache.mu.Lock()
		entry, exists := cache.entries[key]
		if !exists {
			if len(cache.entries) >= cache.options.MaxPartitions {
				cache.mu.Unlock()
				return Snapshot{}, ErrCacheLimit
			}
			entry = &cacheEntry{}
			cache.entries[key] = entry
		}
		if entry.valid {
			snapshot := entry.snapshot
			cache.mu.Unlock()
			return snapshot, nil
		}
		if entry.loading {
			done := entry.done
			cache.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return Snapshot{}, ctx.Err()
			}
		}

		entry.loading = true
		entry.done = make(chan struct{})
		done := entry.done
		cache.mu.Unlock()

		snapshot, err := cache.discoverer.Discover(ctx)
		cache.mu.Lock()
		current, currentExists := cache.entries[key]
		if err == nil && currentExists && current == entry {
			entry.snapshot = snapshot
			entry.valid = true
		} else if err != nil && currentExists && current == entry {
			delete(cache.entries, key)
		}
		entry.loading = false
		close(done)
		cache.mu.Unlock()
		return snapshot, err
	}
}

// Invalidate makes the next discovery call refresh the snapshot. If a refresh
// is already running, its result is returned to its leader but is not cached.
func (cache *Cache) Invalidate() {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	cache.entries = make(map[string]*cacheEntry)
	cache.mu.Unlock()
}

// InvalidatePartition makes the next discovery for key refresh its snapshot.
// Unknown keys are already uncached and therefore succeed without allocation.
func (cache *Cache) InvalidatePartition(key string) error {
	if cache == nil || key == "" || len(key) > cache.options.MaxKeyBytes {
		return ErrCachePartition
	}
	cache.mu.Lock()
	delete(cache.entries, key)
	cache.mu.Unlock()
	return nil
}
