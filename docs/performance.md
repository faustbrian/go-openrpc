# Performance

[Documentation index](README.md)

Run allocation benchmarks with:

```sh
GOWORK=off go test -run '^$' -bench . -benchmem -benchtime=100ms -count=1 -timeout=20m ./...
```

`make check-full` retains this configured full benchmark campaign; ordinary
Make and scheduled shared CI do not replay it.

The suite measures complete-document parse, semantic validation, canonical
serialization, reference resolve, bundle collection, semantic diff, discovery,
large-schema compilation, and hostile bounded rejection. Use multiple runs and
`benchstat` before claiming a regression or improvement.

The checked [benchmark baseline](../benchmarks/baseline.txt) records one
reproducible evidence run. It is a comparison input, not a portable latency
claim; release comparisons must use identical hardware, Go, and benchmark
settings.

Resource policies are part of performance correctness. Lower limits for known
small documents. Reuse immutable parsed documents and compiled validators where
their policy and resource set are identical. Discovery caching is opt-in and
must be invalidated explicitly.

Schema compilation accounts for the root and every explicit resource before
registration. Lower `MaxResources` and `MaxSchemaBytes` when applications use a
small fixed schema graph.

Do not optimize by dropping duplicate-key detection, exact numeric lexemes,
ownership copies, diagnostics bounds, reference authorization, cancellation,
or schema semantics. Record baseline hardware, Go version, command, and
allocations when proposing a performance budget.

Continue with [API guide](api.md).
