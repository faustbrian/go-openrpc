package reference_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/faustbrian/go-openrpc/v2/reference"
)

type cancelableFS struct{}

func (cancelableFS) Open(string) (fs.File, error) {
	time.Sleep(300 * time.Millisecond)
	return nil, fs.ErrNotExist
}

func (cancelableFS) ReadFileContext(ctx context.Context, _ string, _ int) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type contextMapFS struct{ fstest.MapFS }

func (filesystem contextMapFS) ReadFileContext(ctx context.Context, name string, maxBytes int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(filesystem.MapFS, name)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, reference.ErrStoreLimit
	}
	return data, nil
}

type oversizedFS struct{ fstest.MapFS }

func (filesystem oversizedFS) ReadFileContext(context.Context, string, int) ([]byte, error) {
	return []byte("too large"), nil
}

type sharedBufferFS struct {
	fstest.MapFS
	data []byte
}

type typedNilContextFS struct{ fstest.MapFS }

func (*typedNilContextFS) ReadFileContext(context.Context, string, int) ([]byte, error) {
	panic("typed-nil filesystem called")
}

type cancelAfterReadContext struct {
	context.Context
	checks int
}

func (ctx *cancelAfterReadContext) Err() error {
	ctx.checks++
	if ctx.checks >= 2 {
		return context.Canceled
	}
	return nil
}

func (filesystem *sharedBufferFS) ReadFileContext(context.Context, string, int) ([]byte, error) {
	return filesystem.data, nil
}

func TestMemoryStoreOwnsDocumentsAndEnforcesBounds(t *testing.T) {
	t.Parallel()

	documents := map[string][]byte{"https://example.com/schema.json": []byte(`{"type":"string"}`)}
	store, err := reference.NewMemoryStore(documents)
	if err != nil {
		t.Fatal(err)
	}
	documents["https://example.com/schema.json"][0] = '['
	loaded, err := store.Load(context.Background(), "https://example.com/schema.json", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if loaded[0] != '{' {
		t.Fatal("NewMemoryStore retained caller-owned bytes")
	}
	loaded[0] = '['
	again, err := store.Load(context.Background(), "https://example.com/schema.json", 1024)
	if err != nil || again[0] != '{' {
		t.Fatal("Load exposed mutable store bytes")
	}
	if _, err := store.Load(context.Background(), "https://example.com/schema.json", 1); !errors.Is(err, reference.ErrStoreLimit) {
		t.Fatalf("limit error = %v", err)
	}
}

func TestFSStoreMapsOnlyURIsBelowExplicitBase(t *testing.T) {
	t.Parallel()

	store, err := reference.NewFSStore(contextMapFS{MapFS: fstest.MapFS{
		"schemas/value.json": {Data: []byte(`{"value":true}`)},
	}}, "https://example.com/assets/")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), "https://example.com/assets/schemas/value.json", 1024)
	if err != nil || string(loaded) != `{"value":true}` {
		t.Fatalf("Load = %s, error = %v", loaded, err)
	}
	for _, uri := range []string{
		"https://[::1",
		"http://example.com/assets/schemas/value.json",
		"https://other.example/assets/schemas/value.json",
		"https://example.com/private.json",
		"https://user@example.com/assets/schemas/value.json",
		"https://example.com/assets/schemas/value.json?version=1",
		"https://example.com/assets/schemas/value.json#value",
		"https://example.com/assets/%2e%2e/private.json",
		"https://example.com/assets/schemas%2fvalue.json",
		"https://example.com/assets/schemas/../schemas/value.json",
	} {
		if _, err := store.Load(context.Background(), uri, 1024); !errors.Is(err, reference.ErrStoreURI) {
			t.Errorf("Load(%q) error = %v", uri, err)
		}
	}
	if _, err := store.Load(context.Background(), "https://example.com/assets/schemas/value.json", 1); !errors.Is(err, reference.ErrStoreLimit) {
		t.Fatalf("limit error = %v", err)
	}
}

func TestStoresHonorCancellationAndInvalidConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := reference.NewMemoryStore(map[string][]byte{"relative": []byte(`{}`)}); !errors.Is(err, reference.ErrStoreURI) {
		t.Fatalf("NewMemoryStore error = %v", err)
	}
	if _, err := reference.NewFSStore(nil, "relative/"); !errors.Is(err, reference.ErrStorePolicy) {
		t.Fatalf("NewFSStore error = %v", err)
	}
	var typedNil *typedNilContextFS
	if _, err := reference.NewFSStore(typedNil, "https://example.com/assets/"); !errors.Is(err, reference.ErrStorePolicy) {
		t.Fatalf("NewFSStore typed-nil error = %v", err)
	}
	store, err := reference.NewMemoryStore(map[string][]byte{"https://example.com/a": []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Load(ctx, "https://example.com/a", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestFSStoreUsesContextAwareBoundedReads(t *testing.T) {
	t.Parallel()

	store, err := reference.NewFSStore(cancelableFS{}, "https://example.com/assets/")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := store.Load(ctx, "https://example.com/assets/schema.json", 1_024); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Load error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("canceled filesystem read returned after %s", elapsed)
	}
}

func TestFSStoreRejectsNoncompliantFilesystemResults(t *testing.T) {
	t.Parallel()

	store, err := reference.NewFSStore(oversizedFS{}, "https://example.com/assets/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), "https://example.com/assets/schema.json", 1); !errors.Is(err, reference.ErrStoreLimit) {
		t.Fatalf("noncompliant filesystem error = %v", err)
	}
	shared := &sharedBufferFS{MapFS: fstest.MapFS{}, data: []byte("safe")}
	store, err = reference.NewFSStore(shared, "https://example.com/assets/")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Load(context.Background(), "https://example.com/assets/schema.json", 4)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 'X'
	second, err := store.Load(context.Background(), "https://example.com/assets/schema.json", 4)
	if err != nil || string(second) != "safe" {
		t.Fatalf("filesystem store exposed adapter bytes: %q, %v", second, err)
	}
	ctx := &cancelAfterReadContext{Context: context.Background()}
	if _, err := store.Load(ctx, "https://example.com/assets/schema.json", 4); !errors.Is(err, context.Canceled) {
		t.Fatalf("post-read cancellation error = %v", err)
	}
}

func TestStoreFuncAdaptsExplicitLoadContract(t *testing.T) {
	t.Parallel()

	store := reference.StoreFunc(func(_ context.Context, uri string, maximum int) ([]byte, error) {
		if uri != "https://example.com/schema.json" || maximum != 128 {
			t.Fatalf("Load(%q, %d)", uri, maximum)
		}
		return []byte(`{"type":"string"}`), nil
	})
	loaded, err := store.Load(context.Background(), "https://example.com/schema.json", 128)
	if err != nil || string(loaded) != `{"type":"string"}` {
		t.Fatalf("Load() = %s, %v", loaded, err)
	}
}
