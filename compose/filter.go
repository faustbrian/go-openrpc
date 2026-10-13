// Package compose provides explicit, deterministic OpenRPC filtering, merging,
// and overlay operations. Start with FilterMethods, Merge or ApplyOverlays and
// explicit options. These are package policies (OPENRPC-DEC-010), not an OpenRPC
// merge protocol. Inputs remain immutable; returned results are all-or-error.
// Context checks are cooperative and cannot preempt caller predicates or one
// serialization/recursive rewrite. No implicit resolution, workers or cgo is owned.
package compose

import (
	"context"
	"errors"

	openrpc "github.com/faustbrian/go-openrpc/v2"
)

var (
	// ErrInvalidFilter reports invalid filter options or a nil policy.
	ErrInvalidFilter = errors.New("compose: invalid method filter")
	// ErrFilterLimit reports a method count beyond policy.
	ErrFilterLimit = errors.New("compose: method filter limit exceeded")
	// ErrFilterPolicy reports a sanitized caller-policy failure.
	ErrFilterPolicy = errors.New("compose: method filter policy failed")
)

// MethodPredicate decides whether one inline method remains visible.
type MethodPredicate interface {
	// Visible decides inline-method visibility synchronously; implementations own
	// cancellation, concurrency and bounded return.
	Visible(context.Context, openrpc.Method) (bool, error)
}

// MethodPredicateFunc adapts a function to MethodPredicate.
type MethodPredicateFunc func(context.Context, openrpc.Method) (bool, error)

// Visible implements MethodPredicate.
func (function MethodPredicateFunc) Visible(ctx context.Context, method openrpc.Method) (bool, error) {
	return function(ctx, method)
}

// FilterOptions controls bounded handling of unresolved method references.
type FilterOptions struct {
	// MaxMethods caps admitted input methods before visibility callbacks.
	MaxMethods int
	// KeepReferences retains unresolved methods without invoking the inline predicate;
	// false removes them.
	KeepReferences bool
}

// DefaultFilterOptions keeps references because their target visibility cannot
// be inferred without explicit resolution.
func DefaultFilterOptions() FilterOptions {
	return FilterOptions{MaxMethods: 10_000, KeepReferences: true}
}

// FilterMethods applies context-aware visibility to inline methods and returns
// an ownership-safe document. An empty method list remains valid. Predicates
// run synchronously in input order; failures are sanitized and panics propagate.
// It filters methods only, not unreachable components or deployment authority.
func FilterMethods(
	ctx context.Context,
	document openrpc.Document,
	predicate MethodPredicate,
	options FilterOptions,
) (openrpc.Document, error) {
	if ctx == nil || predicate == nil || options.MaxMethods <= 0 {
		return openrpc.Document{}, ErrInvalidFilter
	}
	if err := ctx.Err(); err != nil {
		return openrpc.Document{}, err
	}
	methods := document.Methods()
	if len(methods) > options.MaxMethods {
		return openrpc.Document{}, ErrFilterLimit
	}
	filtered := make([]openrpc.MethodOrReference, 0, len(methods))
	for _, union := range methods {
		if err := ctx.Err(); err != nil {
			return openrpc.Document{}, err
		}
		method, inline := union.Method()
		if !inline {
			if options.KeepReferences {
				filtered = append(filtered, union)
			}
		} else {
			visible, err := predicate.Visible(ctx, method)
			if err != nil {
				if contextErr := ctx.Err(); contextErr != nil {
					return openrpc.Document{}, contextErr
				}
				return openrpc.Document{}, ErrFilterPolicy
			}
			if visible {
				filtered = append(filtered, union)
			}
		}
	}
	return copyDocumentWithMethods(document, filtered)
}

func copyDocumentWithMethods(document openrpc.Document, methods []openrpc.MethodOrReference) (openrpc.Document, error) {
	schemaURI, explicitSchema := document.SchemaURI()
	var schema *string
	if explicitSchema {
		schema = &schemaURI
	}
	servers, hasServers := document.Servers()
	components, hasComponents := document.Components()
	var componentInput *openrpc.Components
	if hasComponents {
		componentInput = &components
	}
	externalDocs, hasExternalDocs := document.ExternalDocs()
	var docs *openrpc.ExternalDocumentation
	if hasExternalDocs {
		docs = &externalDocs
	}
	info := document.Info()
	methodCopy := make([]openrpc.MethodOrReference, len(methods))
	copy(methodCopy, methods)
	filtered, err := openrpc.NewDocument(openrpc.DocumentInput{
		Version:       document.Version(),
		SchemaURI:     schema,
		Info:          &info,
		ExternalDocs:  docs,
		Servers:       servers,
		HasServers:    hasServers,
		Methods:       methodCopy,
		Components:    componentInput,
		Extensions:    document.Extensions(),
		UnknownFields: document.UnknownFields(),
	})
	if err != nil {
		return openrpc.Document{}, ErrInvalidFilter
	}
	return filtered, nil
}
