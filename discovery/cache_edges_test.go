package discovery_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/discovery"
)

func TestCacheEdgeRejectsValidKeyWhenDerivationFails(t *testing.T) {
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		calls.Add(1)
		return testDocument(t, "ready"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	keyError := errors.New("key derivation failed")
	cache, err := discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key:         func(context.Context) (string, error) { return "main", keyError },
		MaxKeyBytes: 4, MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := cache.Discover(context.Background())
	if !errors.Is(err, discovery.ErrCachePartition) || errors.Is(err, keyError) {
		t.Fatalf("failed key derivation = %v, want private ErrCachePartition", err)
	}
	if calls.Load() != 0 || snapshot.Revision() != "" || len(snapshot.Bytes()) != 0 {
		t.Fatalf("failed key derivation invoked discovery or returned a snapshot: calls=%d revision=%q", calls.Load(), snapshot.Revision())
	}
}

func TestCacheEdgeFailedLeaderReleasesPartitionCapacity(t *testing.T) {
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			return openrpc.Document{}, errors.New("refresh failed")
		}
		return testDocument(t, "ready"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	key := "old"
	cache, err := discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key:         func(context.Context) (string, error) { return key, nil },
		MaxKeyBytes: 4, MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := cache.Discover(context.Background())
	if !errors.Is(err, discovery.ErrProvider) || failed.Revision() != "" {
		t.Fatalf("failed leader snapshot=%q error=%v", failed.Revision(), err)
	}
	key = "new"
	snapshot, err := cache.Discover(context.Background())
	if err != nil || snapshot.Revision() == "" {
		t.Fatalf("replacement partition snapshot=%q error=%v", snapshot.Revision(), err)
	}
	cached, err := cache.Discover(context.Background())
	if err != nil || cached.Revision() != snapshot.Revision() || calls.Load() != 2 {
		t.Fatalf("replacement partition was not cached: error=%v calls=%d", err, calls.Load())
	}
}

func TestCacheEdgeInvalidatedFailedLeaderDoesNotRemoveReplacement(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "absent"
		if replacement {
			name = "replacement"
		}
		t.Run(name, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int64
			service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
				if calls.Add(1) == 1 {
					close(entered)
					<-release
					return openrpc.Document{}, errors.New("old refresh failed")
				}
				return testDocument(t, "fresh"), nil
			}), nil)
			if err != nil {
				t.Fatal(err)
			}
			key := "public"
			options := discovery.DefaultCacheOptions(func(context.Context) (string, error) { return key, nil })
			options.MaxPartitions = 1
			cache, err := discovery.NewPartitionedCache(service, options)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			joined := false
			t.Cleanup(func() {
				closeSignal(release)
				if !joined {
					_ = awaitError(t, done, "old leader cleanup")
				}
			})
			go func() {
				_, err := cache.Discover(context.Background())
				done <- err
			}()
			awaitSignal(t, entered, "old leader entry")
			if err := cache.InvalidatePartition("public"); err != nil {
				t.Fatal(err)
			}
			var fresh discovery.Snapshot
			if replacement {
				fresh, err = cache.Discover(context.Background())
				if err != nil || fresh.Revision() == "" {
					t.Fatalf("replacement refresh snapshot=%q error=%v", fresh.Revision(), err)
				}
			}
			closeSignal(release)
			err = awaitError(t, done, "old leader failure")
			joined = true
			if !errors.Is(err, discovery.ErrProvider) {
				t.Fatalf("invalidated leader error=%v, want ErrProvider", err)
			}
			if !replacement {
				// A distinct partition at capacity one proves the old failure did
				// not resurrect an entry, rather than merely allowing a retry.
				key = "next"
			}
			current, err := cache.Discover(context.Background())
			if err != nil || current.Revision() == "" {
				t.Fatalf("current partition snapshot=%q error=%v", current.Revision(), err)
			}
			if replacement && current.Revision() != fresh.Revision() {
				t.Fatal("old failure changed the replacement snapshot")
			}
			if calls.Load() != 2 {
				t.Fatalf("old failure removed or resurrected the current partition: calls=%d, want 2", calls.Load())
			}
		})
	}
}

func TestCacheEdgeExactBoundInvalidationDoesNotAllocateUnknownPartition(t *testing.T) {
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		calls.Add(1)
		return testDocument(t, "ready"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	key := "keep"
	cache, err := discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key:         func(context.Context) (string, error) { return key, nil },
		MaxKeyBytes: 4, MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	retained, err := cache.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.InvalidatePartition("miss"); err != nil {
		t.Fatalf("exact-bound unknown invalidation = %v", err)
	}
	cached, err := cache.Discover(context.Background())
	if err != nil || cached.Revision() != retained.Revision() || calls.Load() != 1 {
		t.Fatalf("unknown invalidation disturbed retained snapshot: error=%v calls=%d", err, calls.Load())
	}
	if err := cache.InvalidatePartition("keep"); err != nil {
		t.Fatalf("exact-bound retained invalidation = %v", err)
	}
	key = "next"
	if snapshot, err := cache.Discover(context.Background()); err != nil || snapshot.Revision() == "" || calls.Load() != 2 {
		t.Fatalf("unknown invalidation retained partition capacity: revision=%q error=%v calls=%d", snapshot.Revision(), err, calls.Load())
	}
}
