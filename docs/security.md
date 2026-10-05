# Security model

Model revision: 2. Applies to the v2.0.0 source contract, published on
2026-10-03; it does not change the behavior of published v1.

All JSON inputs are untrusted. `jsonvalue` rejects invalid UTF-8, duplicate
object names, trailing data, excessive bytes, depth, and tokens before a value
enters the model. Parse, pointer, expression, validation, diff, resolver, and
composition policies add operation-specific finite limits. Errors and
diagnostics identify safe codes and pointers without including complete
documents, fetched bodies, credentials, or provider errors.

Core parsing and validation never access the network or filesystem. Reference
resolution is disabled externally by default. Enabling it requires all of:

1. an explicit `reference.Store`;
2. `AllowExternal`;
3. an allowed scheme; and
4. an allowed host.

Resolver work is bounded by depth, aggregate references, loaded documents,
fetched bytes, JSON bytes, JSON tokens, pointer length, and pointer tokens.
The aggregate reference limit applies before allocating direct input results
and across aliases, bundle roots, and transitive resource-graph fan-out.

Draft 7 compilation accepts only caller-supplied resources and limits both
resource count and aggregate schema bytes before compiler registration. The
defaults allow 1,024 resources, 64 MiB across the root and resources, a 16 MiB
validation instance, and 1,000,000 validation steps. The per-call step budget
covers context-aware instance decoding, reachable compiled-schema size,
schema-evaluation checkpoints, regular expressions, and diagnostic traversal.
Each compiled Draft 7 node is guarded before its original evaluation, including
boolean results and references; post-evaluation extension callbacks alone do
not enforce this boundary.
Validation serializes access to compiled ECMAScript regular expressions. A
nearer caller deadline replaces the per-expression timeout, canceled waiters do
not start validation, and cancellation is returned as its context error rather
than an ordinary mismatch.

The memory store owns its input. The `ContextReadFS` store rejects scope escape,
encoded paths, fragments, queries, credentials, and traversal. Filesystem reads
require a `ContextReadFS` implementation that owns prompt cancellation and the
remaining byte limit; returned data is rechecked. The optional
HTTP store requires exact hosts, uses HTTPS unless HTTP is explicitly enabled,
resolves and checks every address before dialing, blocks private and special
address classes by default, disables transparent decompression, rejects content
encodings, bounds redirects, headers, dial and request time, and streams through
the caller's remaining byte allowance.

Callers remain responsible for choosing trustworthy allowlisted hosts and
stores. Allowing private addresses or HTTP intentionally weakens the SSRF and
transport boundary and should be limited to controlled development networks.

Runtime expressions implement JSON value lookup only. They define no loops,
operators, callbacks, reflection, or code execution. Whole expressions preserve
JSON types; objects and arrays cannot be interpolated into surrounding text.

Discovery validates provider and filter output under explicit diagnostic and
canonical-byte limits. Oversized generated documents fail before a snapshot is
published or returned through the JSON-RPC adapter.

Partitioned discovery caches require a non-sensitive caller key and retain a
bounded number of partitions. The key must include every tenant, role, and
authorization dimension used by the visibility filter. Missing, empty, failed,
and capacity-exhausted partitions fail closed; keys have an independent byte
bound, failed loads release their partition entry, and snapshots and in-flight
work are never shared between distinct keys.

## Discovery authorization threat model

| Threat | Control |
| --- | --- |
| Cross-tenant or cross-role snapshot reuse | Every cache lookup derives an explicit caller-owned partition key before reading or starting work |
| Incomplete authorization identity | `NewPartitionedCache` requires a key function; deployment review must prove that it includes every filter input |
| Attacker-driven key size or cardinality | `MaxKeyBytes` rejects oversized keys; `MaxPartitions` rejects new keys at a finite bound; invalidation and failed refreshes release retained entries |
| Sensitive key disclosure | Keys are caller-defined, retained only in memory, and never included in package errors or observations |

## Accepted residual risks

| Risk and owner | Rationale | Mitigation | Review condition |
| --- | --- | --- | --- |
| A JSON Schema dependency keyword can run until its current bounded value or schema segment completes; `openrpc` maintainers | The dependency exposes no context-aware validation entry point, and detached validation goroutines would violate lifecycle ownership | Byte and step limits bound admitted data and aggregate evaluations; checkpoints cover decoding, object-schema evaluations, regular expressions, and diagnostics; deployments should lower limits and use deadlines | Revisit when the dependency exposes cooperative cancellation, when a supported keyword changes complexity, or when profiling shows one admitted segment exceeds the deployment latency budget |
| A caller-provided `ContextReadFS` can violate its cancellation or byte contract; application integrator | Go interfaces cannot force a custom implementation to stop, and wrapping a blocking read in a goroutine would leak lifecycle ownership | The v2 constructor requires the bounded interface; the adapter receives context and byte allowance; returned size and post-read cancellation are rechecked and bytes are copied | Revisit when adopting a new filesystem adapter, when the standard library gains context-aware file reads, or after any cancellation regression |
| A cache key can omit an authorization dimension; application integrator | Only the application knows every input used by its visibility filter | Construction requires an explicit key function; empty, oversized, failed, and capacity-exhausted keys fail closed; deployment review must trace every filter input into the key | Revisit whenever discovery filters, tenant identity, roles, or authorization policy change |

Semantic validation checks generated document method counts before copying or
traversing the method collection. Parser limits therefore are not the only
defense for code-first documents.

The production module does not use `unsafe`, cgo, `go:linkname`, background
network fetches, global mutable registries, or telemetry exporters.

## Deployment checklist

- Reduce parser and JSON limits when the deployment contract is smaller than
  the defaults.
- Keep unknown-field rejection enabled for untrusted current-version input.
- Run both meta-schema and semantic validation for externally supplied
  documents.
- Prefer embedded or in-memory reference stores. When HTTP resolution is
  required, keep HTTPS, exact host allowlists, private-address rejection,
  redirect limits, compression rejection, timeouts, and streamed byte limits.
- Treat conditional compatibility findings as requiring application review.
- Scope discovery filtering and caching to the caller's authorization model.
- Export only bounded `observe` events, never raw documents or provider errors.

## Reporting vulnerabilities

Do not open a public issue for a suspected vulnerability. Send a private report
to the repository owner with the affected version, a minimal reproducer, impact,
and any proposed mitigation. Avoid including credentials or production data.
