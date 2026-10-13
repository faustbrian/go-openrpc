package discovery_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/discovery"
)

// TestCachePanickingLeaderReleasesLiveFollower preserves a trusted provider's
// panic identity while requiring the cache-owned completion channel to release.
func TestCachePanickingLeaderReleasesLiveFollower(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	panicValue := &struct{ name string }{"trusted provider"}
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			panic(panicValue)
		}
		return testDocument(t, "fresh"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}
	leaderDone := make(chan any, 1)
	waiterDone := make(chan error, 1)
	leaderJoined, waiterJoined := false, false
	waiterContext, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		closeSignal(release)
		cancel()
		if !leaderJoined {
			select {
			case <-leaderDone:
			case <-time.After(2 * time.Second):
				t.Error("leader cleanup did not finish")
			}
		}
		if !waiterJoined {
			_ = awaitError(t, waiterDone, "follower cancellation cleanup")
		}
	})
	go func() { defer func() { leaderDone <- recover() }(); _, _ = cache.Discover(context.Background()) }()
	awaitSignal(t, entered, "panicking leader entry")
	observed := &doneObservedContext{Context: waiterContext, observed: make(chan struct{})}
	go func() { _, err := cache.Discover(observed); waiterDone <- err }()
	awaitSignal(t, observed.observed, "follower joined loading entry")
	closeSignal(release)
	select {
	case recovered := <-leaderDone:
		leaderJoined = true
		if recovered != panicValue {
			t.Fatal("cache swallowed or changed the trusted provider panic")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("leader did not unwind")
	}
	if err := awaitError(t, waiterDone, "follower released by panicking leader"); err != nil {
		waiterJoined = true
		t.Fatal("live follower failed instead of refreshing")
	}
	waiterJoined = true
	if calls.Load() != 2 {
		t.Fatal("released follower did not perform exactly one fresh discovery")
	}
	if _, err := cache.Discover(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatal("fresh result was not reusable")
	}
}

// TestCachePanickingLeaderReleasesPartitionCapacity checks that an abandoned
// flight cannot exhaust the finite partition budget after the caller recovers.
func TestCachePanickingLeaderReleasesPartitionCapacity(t *testing.T) {
	panicValue := &struct{ name string }{"trusted provider"}
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			panic(panicValue)
		}
		return testDocument(t, "fresh"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	options := discovery.DefaultCacheOptions(func(ctx context.Context) (string, error) { return ctx.Value(cacheIdentityKey{}).(string), nil })
	options.MaxPartitions = 1
	cache, err := discovery.NewPartitionedCache(service, options)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() != panicValue {
				t.Fatal("trusted provider panic identity changed")
			}
		}()
		_, _ = cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "old"))
	}()
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), cacheIdentityKey{}, "new"), 2*time.Second)
	defer cancel()
	if _, err := cache.Discover(ctx); err != nil {
		t.Fatal("panicking leader retained partition capacity")
	}
}

// TestCachePanickingInvalidatedLeaderKeepsReplacement ensures panic cleanup
// cannot delete a newer successful entry sharing the old partition key.
func TestCachePanickingInvalidatedLeaderKeepsReplacement(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	panicValue := &struct{ name string }{"trusted provider"}
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			panic(panicValue)
		}
		return testDocument(t, "replacement"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}
	leaderDone := make(chan any, 1)
	joined := false
	t.Cleanup(func() {
		closeSignal(release)
		if !joined {
			select {
			case <-leaderDone:
			case <-time.After(2 * time.Second):
				t.Error("invalidated leader cleanup did not finish")
			}
		}
	})
	go func() { defer func() { leaderDone <- recover() }(); _, _ = cache.Discover(context.Background()) }()
	awaitSignal(t, entered, "old leader entry")
	if err := cache.InvalidatePartition("public"); err != nil {
		t.Fatal(err)
	}
	fresh, err := cache.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	closeSignal(release)
	select {
	case recovered := <-leaderDone:
		joined = true
		if recovered != panicValue {
			t.Fatal("invalidated leader panic identity changed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("invalidated leader did not unwind")
	}
	cached, err := cache.Discover(context.Background())
	if err != nil || cached.Revision() != fresh.Revision() || calls.Load() != 2 {
		t.Fatal("old leader panic cleanup removed a replacement snapshot")
	}
}
