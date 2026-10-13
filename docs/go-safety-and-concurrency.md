# Go safety and concurrency

[Documentation index](README.md)

Parsed and constructed values own their input bytes, slices, maps, and fields.
Public collection and byte getters return copies. Concurrent read-only use is
therefore supported after construction.

`builder.MethodRegistry` and `discovery.Cache` synchronize their mutable state.
Resolver caches are per call, so authorization and fetched resources do not
leak across requests. Cancellation of one cache waiter does not cancel shared
discovery work owned by the refresh leader in the same explicit authorization
partition. Canceling that leader can fail its refresh. Different partition keys
never share a snapshot or in-flight load.

Compiled Draft 7 validators serialize validation calls because their aggregate
step budget and bounded ECMAScript regular expressions carry the active caller
context. A caller that cancels while waiting does not start work; cancellation
is checked during decoding, object-schema evaluation, regular expressions, and
diagnostic traversal without leaving a background goroutine.

No constructor starts a goroutine. No package installs a global resolver,
schema loader, registry, cache, or telemetry exporter. Callers own lifecycle,
shutdown, transport, and exporter behavior.

Run `GOWORK=off go test -race ./...` for full concurrency qualification, or
select the affected packages after a focused concurrency change. `make check-full`
retains the complete race campaign; ordinary Make selects the named registry
race. Existing TestMain checks in builder, discovery, observe, reference and
the HTTP reference store enforce goroutine cleanup during their ordinary tests,
including cancellation cases; there is no separate root `make leak` target.
Ownership claims also require mutation tests that modify caller inputs and
returned snapshots, not only race-detector silence.

Continue with [API guide](api.md).
