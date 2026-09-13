package discovery_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/discovery"
)

type cacheIdentityKey struct{}

type typedNilDiscoverer struct{}

type valueDiscoverer struct{ discovery.Discoverer }

func (*typedNilDiscoverer) Discover(context.Context) (discovery.Snapshot, error) {
	panic("typed-nil discoverer called")
}

func publicCacheOptions() discovery.CacheOptions {
	return discovery.DefaultCacheOptions(func(context.Context) (string, error) {
		return "public", nil
	})
}

func TestCacheDeduplicatesConcurrentDiscoveryAndInvalidatesExplicitly(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { closeSignal(release) })
	provider := discovery.ProviderFunc(func(ctx context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			return testDocument(t, "cached"), nil
		case <-ctx.Done():
			return openrpc.Document{}, ctx.Err()
		}
	})
	service, err := discovery.NewService(provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}

	const callers = 8
	results := make(chan discovery.Snapshot, callers)
	errors := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			snapshot, discoverErr := cache.Discover(context.Background())
			results <- snapshot
			errors <- discoverErr
		}()
	}
	awaitSignal(t, entered, "provider entry")
	closeSignal(release)
	groupDone := make(chan struct{})
	go func() {
		group.Wait()
		close(groupDone)
	}()
	awaitSignal(t, groupDone, "concurrent callers")
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var revision string
	for snapshot := range results {
		if revision == "" {
			revision = snapshot.Revision()
		}
		if snapshot.Revision() != revision {
			t.Fatalf("revision = %q, want %q", snapshot.Revision(), revision)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d", calls.Load())
	}

	cache.Invalidate()
	if _, err := cache.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("provider calls after invalidation = %d", calls.Load())
	}
}

func TestCacheWaiterCanCancelWithoutCancelingSharedDiscovery(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { closeSignal(release) })
	service, err := discovery.NewService(discovery.ProviderFunc(func(ctx context.Context) (openrpc.Document, error) {
		close(entered)
		select {
		case <-release:
			return testDocument(t, "cached"), nil
		case <-ctx.Done():
			return openrpc.Document{}, ctx.Err()
		}
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}
	leaderDone := make(chan error, 1)
	go func() {
		_, discoverErr := cache.Discover(context.Background())
		leaderDone <- discoverErr
	}()
	awaitSignal(t, entered, "leader provider entry")
	baseContext, cancel := context.WithCancel(context.Background())
	ctx := &checkedContext{Context: baseContext, checked: make(chan struct{})}
	waiterDone := make(chan error, 1)
	go func() {
		_, discoverErr := cache.Discover(ctx)
		waiterDone <- discoverErr
	}()
	awaitSignal(t, ctx.checked, "waiter context check")
	cancel()
	if err := awaitError(t, waiterDone, "canceled waiter"); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter error = %v", err)
	}
	closeSignal(release)
	if err := awaitError(t, leaderDone, "leader completion"); err != nil {
		t.Fatal(err)
	}
}

func TestCacheWaiterUsesCompletedSharedDiscovery(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { closeSignal(release) })
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		calls.Add(1)
		close(entered)
		<-release
		return testDocument(t, "shared"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}

	leaderDone := make(chan error, 1)
	go func() {
		_, discoverErr := cache.Discover(context.Background())
		leaderDone <- discoverErr
	}()
	awaitSignal(t, entered, "leader provider entry")

	waiterContext := &doneObservedContext{
		Context:  context.Background(),
		observed: make(chan struct{}),
	}
	waiterDone := make(chan error, 1)
	go func() {
		_, discoverErr := cache.Discover(waiterContext)
		waiterDone <- discoverErr
	}()
	awaitSignal(t, waiterContext.observed, "waiter completion observation")
	closeSignal(release)

	if err := awaitError(t, leaderDone, "leader completion"); err != nil {
		t.Fatal(err)
	}
	if err := awaitError(t, waiterDone, "waiter completion"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

type checkedContext struct {
	context.Context
	checked chan struct{}
}

type doneObservedContext struct {
	context.Context
	observed chan struct{}
}

func (ctx *doneObservedContext) Done() <-chan struct{} {
	select {
	case <-ctx.observed:
	default:
		close(ctx.observed)
	}
	return ctx.Context.Done()
}

func (ctx *checkedContext) Err() error {
	select {
	case <-ctx.checked:
	default:
		close(ctx.checked)
	}
	return ctx.Context.Err()
}

func TestNewPartitionedCacheRequiresDiscovererAndOptions(t *testing.T) {
	t.Parallel()

	if _, err := discovery.NewPartitionedCache(nil, discovery.DefaultCacheOptions(
		func(context.Context) (string, error) { return "public", nil },
	)); !errors.Is(err, discovery.ErrInvalidOptions) {
		t.Fatalf("NewPartitionedCache nil discoverer error = %v", err)
	}
	var typedNil *typedNilDiscoverer
	if _, err := discovery.NewPartitionedCache(typedNil, publicCacheOptions()); !errors.Is(err, discovery.ErrInvalidOptions) {
		t.Fatalf("NewPartitionedCache typed-nil discoverer error = %v", err)
	}
	service, err := discovery.NewService(discovery.Static(testDocument(t, "public")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := discovery.NewPartitionedCache(valueDiscoverer{Discoverer: service}, publicCacheOptions()); err != nil {
		t.Fatalf("NewPartitionedCache value discoverer error = %v", err)
	}
	for _, options := range []discovery.CacheOptions{
		{},
		{Key: func(context.Context) (string, error) { return "public", nil }, MaxPartitions: 1},
		{Key: func(context.Context) (string, error) { return "public", nil }, MaxKeyBytes: 1},
	} {
		if _, err := discovery.NewPartitionedCache(service, options); !errors.Is(err, discovery.ErrCachePartition) {
			t.Fatalf("NewPartitionedCache options %#v error = %v", options, err)
		}
	}
}

func TestCachePartitionsFilteredSnapshotsByCallerKey(t *testing.T) {
	t.Parallel()

	service, err := discovery.NewService(
		discovery.Static(testDocument(t, "base")),
		discovery.FilterFunc(func(ctx context.Context, _ openrpc.Document) (openrpc.Document, error) {
			identity, _ := ctx.Value(cacheIdentityKey{}).(string)
			return testDocument(t, identity), nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, discovery.DefaultCacheOptions(
		func(ctx context.Context) (string, error) {
			identity, _ := ctx.Value(cacheIdentityKey{}).(string)
			return identity, nil
		},
	))
	if err != nil {
		t.Fatal(err)
	}

	first, err := cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "second"))
	if err != nil {
		t.Fatal(err)
	}
	firstMethod, _ := first.Document().Methods()[0].Method()
	secondMethod, _ := second.Document().Methods()[0].Method()
	if firstMethod.Name() != "first" {
		t.Fatalf("first partition returned %q", firstMethod.Name())
	}
	if secondMethod.Name() != "second" {
		t.Fatalf("second partition returned %q", secondMethod.Name())
	}
}

func TestCacheRejectsInvalidKeysAndBoundsPartitions(t *testing.T) {
	t.Parallel()

	service, err := discovery.NewService(discovery.Static(testDocument(t, "bounded")), nil)
	if err != nil {
		t.Fatal(err)
	}
	keyError := errors.New("private key detail")
	cache, err := discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key:           func(context.Context) (string, error) { return "", keyError },
		MaxKeyBytes:   64,
		MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.Background()); !errors.Is(err, discovery.ErrCachePartition) || errors.Is(err, keyError) {
		t.Fatalf("cache key error = %v", err)
	}

	cache, err = discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key: func(ctx context.Context) (string, error) {
			key, _ := ctx.Value(cacheIdentityKey{}).(string)
			return key, nil
		},
		MaxKeyBytes:   64,
		MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "first")); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "second")); !errors.Is(err, discovery.ErrCacheLimit) {
		t.Fatalf("partition limit error = %v", err)
	}
	if err := cache.InvalidatePartition("first"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.WithValue(context.Background(), cacheIdentityKey{}, "second")); err != nil {
		t.Fatalf("partition capacity was not released: %v", err)
	}
	if err := cache.InvalidatePartition(""); !errors.Is(err, discovery.ErrCachePartition) {
		t.Fatalf("empty invalidation key error = %v", err)
	}
}

func TestCacheReturnsCancellationThatOccursDuringKeyDerivation(t *testing.T) {
	t.Parallel()

	service, err := discovery.NewService(discovery.Static(testDocument(t, "bounded")), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []discovery.CacheKeyFunc{
		func(ctx context.Context) (string, error) {
			ctx.Value(cacheIdentityKey{}).(context.CancelFunc)()
			return "", errors.New("private key detail")
		},
		func(ctx context.Context) (string, error) {
			ctx.Value(cacheIdentityKey{}).(context.CancelFunc)()
			return "public", nil
		},
	} {
		cache, cacheErr := discovery.NewPartitionedCache(service, discovery.CacheOptions{
			Key: key, MaxKeyBytes: 64, MaxPartitions: 1,
		})
		if cacheErr != nil {
			t.Fatal(cacheErr)
		}
		baseContext, cancel := context.WithCancel(context.Background())
		ctx := context.WithValue(baseContext, cacheIdentityKey{}, context.CancelFunc(cancel))
		if _, discoverErr := cache.Discover(ctx); !errors.Is(discoverErr, context.Canceled) {
			t.Fatalf("canceled key derivation error = %v", discoverErr)
		}
	}
}

func TestCacheBoundsPartitionKeysAndReleasesFailedEntries(t *testing.T) {
	t.Parallel()

	key := "first"
	calls := 0
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		calls++
		if calls == 1 {
			return openrpc.Document{}, errors.New("provider failed")
		}
		return testDocument(t, key), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, discovery.CacheOptions{
		Key:           func(context.Context) (string, error) { return key, nil },
		MaxKeyBytes:   5,
		MaxPartitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.Background()); err == nil {
		t.Fatal("failed provider discovery succeeded")
	}
	key = "other"
	if _, err := cache.Discover(context.Background()); err != nil {
		t.Fatalf("failed partition retained capacity: %v", err)
	}
	key = "oversized"
	if _, err := cache.Discover(context.Background()); !errors.Is(err, discovery.ErrCachePartition) {
		t.Fatalf("oversized partition key error = %v", err)
	}
	if err := cache.InvalidatePartition("oversized"); !errors.Is(err, discovery.ErrCachePartition) {
		t.Fatalf("oversized invalidation key error = %v", err)
	}
}

func TestCacheRejectsInvalidStateAndRetriesFailures(t *testing.T) {
	t.Parallel()

	var nilCache *discovery.Cache
	if _, err := nilCache.Discover(context.Background()); !errors.Is(err, discovery.ErrInvalidOptions) {
		t.Fatalf("nil cache error = %v", err)
	}
	nilCache.Invalidate()
	zeroCache := &discovery.Cache{}
	if _, err := zeroCache.Discover(context.Background()); !errors.Is(err, discovery.ErrInvalidOptions) {
		t.Fatalf("zero cache error = %v", err)
	}

	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		if calls.Add(1) == 1 {
			return openrpc.Document{}, errors.New("transient provider detail")
		}
		return testDocument(t, "retried"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.Background()); err == nil {
		t.Fatal("failed refresh succeeded")
	}
	if _, err := cache.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.Discover(canceledContext); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context error = %v", err)
	}
	var invalidContext context.Context
	if _, err := cache.Discover(invalidContext); !errors.Is(err, discovery.ErrInvalidOptions) {
		t.Fatalf("nil context error = %v", err)
	}
}

func TestCacheDoesNotPublishRefreshInvalidatedInFlight(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { closeSignal(release) })
	var calls atomic.Int64
	service, err := discovery.NewService(discovery.ProviderFunc(func(context.Context) (openrpc.Document, error) {
		call := calls.Add(1)
		if call == 1 {
			close(entered)
			<-release
		}
		return testDocument(t, "generation"), nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := discovery.NewPartitionedCache(service, publicCacheOptions())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, discoverErr := cache.Discover(context.Background())
		done <- discoverErr
	}()
	awaitSignal(t, entered, "invalidated provider entry")
	cache.Invalidate()
	closeSignal(release)
	if err := awaitError(t, done, "invalidated refresh completion"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want 2", calls.Load())
	}
}

func awaitSignal(t *testing.T, signal <-chan struct{}, operation string) {
	t.Helper()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-signal:
	case <-timer.C:
		t.Fatalf("timed out waiting for %s", operation)
	}
}

func awaitError(t *testing.T, result <-chan error, operation string) error {
	t.Helper()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		t.Fatalf("timed out waiting for %s", operation)
		return nil
	}
}

func closeSignal(signal chan struct{}) {
	select {
	case <-signal:
	default:
		close(signal)
	}
}
