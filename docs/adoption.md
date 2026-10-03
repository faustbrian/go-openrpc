# Adoption guide

## V2 migration

Version v2.0.0 is published at `github.com/faustbrian/go-openrpc/v2`.
Update module requirements and imports when adopting v2. Owned ecosystem
consumer adoption is verified separately at these boundaries:

- `go-jsonrpc` coordination metadata;
- `go-openapi` coordination metadata; and
- the `go-library-tools` release compatibility consumer.

Those repositories remain unchanged here. Their owners must update imports
or metadata deliberately and verify the affected integration
boundary. Existing v1 consumers continue to receive the published v1 behavior;
the cache and filesystem hardening described below requires the v2 release.

## Static documents

Start with `parse.Decode` in strict unknown-field mode. Preserve the returned
source only when exact re-emission matters; use `MarshalCanonical` for hashes,
reviews, and generated artifacts. Run both meta-schema and semantic validation.

## Existing JSON-RPC services

Inventory runtime method names, parameter structure, notification-only methods,
custom errors, and authorization visibility. Construct those concepts
explicitly. Do not infer protocol behavior from a structurally valid document.

Expose discovery through the transport-neutral `discovery.Service`. Keep batch,
notification, error envelope, request ID, and transport behavior in the
existing JSON-RPC implementation.

## Dynamic visibility

Supply a `discovery.Filter` that returns a new owned document for the current
authorization context. An empty methods array is valid. Construct
`discovery.Cache` with `NewPartitionedCache`, `DefaultCacheOptions`, and a
non-sensitive key that includes every tenant, role, and authorization dimension
capable of changing the snapshot. Missing, empty, failing, or
capacity-exhausted partitions fail closed.

Use `discovery.NewServiceWithOptions` for generated or tenant-dependent
documents and set `MaxOutputBytes` to the largest discovery response the
transport contract permits. The default constructor applies a finite 64 MiB
ceiling and bounded semantic diagnostics.

## Schema reuse and references

Keep core parsing offline. Configure a `reference.Resolver` only at the call
site that needs external resources. Prefer `MemoryStore`. Filesystem-backed
resolution must supply `reference.ContextReadFS`, whose bounded read honors the
operation context. For HTTP, allow exact schemes and hosts and retain the
default private-address, redirect, compression, timeout, and byte protections.

## CI compatibility diffing

Parse a released baseline and candidate, resolve both with the same explicit
resource policy, and call `diff.CompareResolved`. Treat `Breaking` as blocking
and review `Conditional` changes rather than silently accepting them.
`Report.Compatible` returns false until those findings are resolved.

## Extensions

Create specification extensions with `NewExtensions`; exact names must start
with lowercase `x-`. Preserve unknown standard fields only during an explicit
forward-compatibility workflow and do not reinterpret them as known fields.
