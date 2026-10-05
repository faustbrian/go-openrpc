# API guide

The Go package documentation is authoritative for signatures. This guide maps
responsibility and failure boundaries.

| Package | Owned contract |
| --- | --- |
| root | Immutable OpenRPC values, constructors, versions, canonical output |
| `builder` | Persistent document builder and concurrent method registry |
| `compose` | Merge, overlay, filter, and component rename operations |
| `diff` | Bounded semantic compatibility reports, including resolved mode |
| `discovery` | Provider, visibility filter, bounded snapshots, revision, and cache |
| `expression` | Bounded JSON Template Language and server-variable evaluation |
| `jsonrpc` | Discovery handler and typed system-registration seam |
| `jsonschema` | Lossless Draft 7 values and explicit-resource validation |
| `jsonvalue` | Strict immutable arbitrary JSON with duplicate detection |
| `observe` | Optional payload-free operation events |
| `parse` | Strict or preserving bounded OpenRPC JSON decoding |
| `reference` | URI references, JSON Pointer, stores, resolver, bundle, transform |
| `reference/httpstore` | Opt-in hardened HTTP document loading |
| `validate` | Structural, semantic, resolved, and schema diagnostics |

Constructors reject absent required values with stable error categories.
Optional scalar getters return `(value, present)`. Collection getters return
owned snapshots and usually also report explicit presence. Union getters return
the selected case and a boolean. Zero values are not silently serialized as
valid OpenRPC objects.

`Document.MethodCount` supports allocation-free policy checks before callers
request the owned method snapshot. Semantic validation applies its own method
limit so generated documents receive the same bounded treatment as parsed
documents.

Reports and errors never embed full documents or fetched bodies. Callers should
branch with `errors.Is`, inspect bounded diagnostic codes and pointers, and log
their own correlation identifiers rather than raw input.

`jsonschema.Validator.Validate` reports a bounded regular-expression timeout as
`ErrValidationResourceLimit`, not as an ordinary schema mismatch. Callers can
therefore distinguish an invalid instance from validation that did not finish
within `ValidationOptions.RegexpTimeout`; the default is 100 milliseconds and
the accepted maximum is 10 seconds. `MaxInstanceBytes` rejects oversized values
before decoding. `MaxValidationSteps` is shared by context-aware input decoding,
compiled-schema checkpoints, regular expressions, and diagnostic traversal. A
nearer context deadline is returned as `context.DeadlineExceeded`. Concurrent
calls on one validator are serialized and waiting callers can cancel without
starting validation work.

`discovery.NewPartitionedCache` requires explicit `CacheOptions`; its key function owns
authorization partitioning, while `MaxKeyBytes` and `MaxPartitions` bound
retained keys and snapshots. Empty, oversized, or failed keys and exhausted
capacity fail closed. A provider failure releases the empty partition entry.

`reference.NewFSStore` requires `ContextReadFS`. The supplied read receives the
caller context and remaining byte allowance; the store rechecks both before
returning data to the resolver.
