package reference

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"path"
	"reflect"
	"strings"
)

// ContextReadFS supplies context-aware bounded file reads. Implementations must
// stop promptly when ctx is canceled, must not return more than maxBytes, and
// must prevent reads from escaping their intended filesystem root.
type ContextReadFS interface {
	fs.FS
	ReadFileContext(ctx context.Context, name string, maxBytes int) ([]byte, error)
}

var (
	// ErrStorePolicy reports invalid store configuration or byte bounds.
	ErrStorePolicy = errors.New("reference: invalid store policy")
	// ErrStoreURI reports a malformed, unknown, or out-of-scope document URI.
	ErrStoreURI = errors.New("reference: document URI outside store scope")
	// ErrStoreLimit reports a document larger than the supplied read bound.
	ErrStoreLimit = errors.New("reference: store byte limit exceeded")
	// ErrStoreRead reports a filesystem read failure without exposing paths or
	// underlying system details.
	ErrStoreRead = errors.New("reference: store read failed")
)

// MemoryStore is an immutable caller-supplied document store.
type MemoryStore struct {
	documents map[string][]byte
}

// NewMemoryStore validates absolute document URIs and owns every byte slice.
func NewMemoryStore(documents map[string][]byte) (*MemoryStore, error) {
	owned := make(map[string][]byte, len(documents))
	for documentURI, document := range documents {
		canonical, err := storeDocumentURI(documentURI)
		if err != nil {
			return nil, ErrStoreURI
		}
		if _, duplicate := owned[canonical]; duplicate {
			return nil, ErrStoreURI
		}
		owned[canonical] = append([]byte(nil), document...)
	}
	return &MemoryStore{documents: owned}, nil
}

// Load implements Store without I/O.
func (store *MemoryStore) Load(ctx context.Context, documentURI string, maxBytes int) ([]byte, error) {
	if store == nil || maxBytes <= 0 || ctx == nil {
		return nil, ErrStorePolicy
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := storeDocumentURI(documentURI)
	if err != nil {
		return nil, ErrStoreURI
	}
	document, exists := store.documents[canonical]
	if !exists {
		return nil, ErrStoreURI
	}
	if len(document) > maxBytes {
		return nil, ErrStoreLimit
	}
	return append([]byte(nil), document...), nil
}

// FSStore maps document URIs below one explicit absolute base into a supplied
// context-aware filesystem boundary.
type FSStore struct {
	filesystem ContextReadFS
	base       *url.URL
}

// NewFSStore constructs a URI traversal-safe store over a caller-owned
// context-aware filesystem boundary.
func NewFSStore(filesystem ContextReadFS, baseURI string) (*FSStore, error) {
	if nilContextReadFS(filesystem) {
		return nil, ErrStorePolicy
	}
	base, err := url.Parse(baseURI)
	if err != nil || !base.IsAbs() || base.Fragment != "" || base.RawQuery != "" ||
		base.User != nil || base.RawPath != "" || !strings.HasSuffix(base.Path, "/") {
		return nil, ErrStorePolicy
	}
	return &FSStore{filesystem: filesystem, base: base}, nil
}

func nilContextReadFS(filesystem ContextReadFS) bool {
	if filesystem == nil {
		return true
	}
	value := reflect.ValueOf(filesystem)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Load implements Store with a hard read limit and URI-scope enforcement.
func (store *FSStore) Load(ctx context.Context, documentURI string, maxBytes int) ([]byte, error) {
	if store == nil || store.filesystem == nil || store.base == nil || maxBytes <= 0 || ctx == nil {
		return nil, ErrStorePolicy
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := url.Parse(documentURI)
	if err != nil {
		return nil, ErrStoreURI
	}
	if target.Fragment != "" {
		return nil, ErrStoreURI
	}
	if target.RawQuery != "" {
		return nil, ErrStoreURI
	}
	if target.User != nil {
		return nil, ErrStoreURI
	}
	if target.RawPath != "" {
		return nil, ErrStoreURI
	}
	if !strings.EqualFold(target.Scheme, store.base.Scheme) {
		return nil, ErrStoreURI
	}
	if !strings.EqualFold(target.Host, store.base.Host) {
		return nil, ErrStoreURI
	}
	if !strings.HasPrefix(target.Path, store.base.Path) {
		return nil, ErrStoreURI
	}
	relative := strings.TrimPrefix(target.Path, store.base.Path)
	if relative == "" {
		return nil, ErrStoreURI
	}
	if path.Clean(relative) != relative {
		return nil, ErrStoreURI
	}
	if !fs.ValidPath(relative) {
		return nil, ErrStoreURI
	}
	data, err := store.filesystem.ReadFileContext(ctx, relative, maxBytes)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		if errors.Is(err, ErrStoreLimit) {
			return nil, ErrStoreLimit
		}
		return nil, ErrStoreRead
	}
	if len(data) > maxBytes {
		return nil, ErrStoreLimit
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), data...), nil
}

func storeDocumentURI(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil || !parsed.IsAbs() || parsed.Fragment != "" || !utf8ValidURI(input) {
		return "", ErrStoreURI
	}
	return parsed.String(), nil
}
