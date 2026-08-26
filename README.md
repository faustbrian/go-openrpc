# openrpc

[![CI](https://github.com/faustbrian/go-openrpc/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-openrpc/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-openrpc/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-openrpc.svg)](https://pkg.go.dev/github.com/faustbrian/go-openrpc)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-openrpc?sort=semver)](https://github.com/faustbrian/go-openrpc/releases)
[![Go](https://img.shields.io/badge/go-1.26.6-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`openrpc` is a design-first Go implementation of OpenRPC 1.3.x and 1.4.x. It
models their shared document shape, preserves arbitrary Draft 7 schemas and
extension values, parses untrusted JSON under finite policies, emits canonical
JSON, and provides explicit validation, reference resolution, runtime
expressions, discovery, composition, compatibility diffing, and JSON-RPC
handler integration.

The core performs no implicit network or filesystem access. External reference
resolution requires a caller-supplied store and an allowlist policy. Earlier
or future OpenRPC feature lines are rejected until their semantics are
separately inventoried and tested.

## Quick start

```go
version, _ := openrpc.ParseVersion("1.4.1")
info, _ := openrpc.NewInfo(openrpc.InfoInput{
    Title: "Calculator",
    Version: "1.0.0",
})
add, _ := openrpc.NewMethod(openrpc.MethodInput{
    Name: "add",
    Params: []openrpc.ContentDescriptorOrReference{},
})

documentBuilder, _ := builder.NewDocument(version, info)
documentBuilder, _ = documentBuilder.WithMethod(add)
document, _ := documentBuilder.Build()
encoded, _ := openrpc.MarshalCanonical(document)
```

### Parse and validate

```go
options := parse.DefaultOptions()
options.UnknownFields = parse.RejectUnknownFields
parsed, err := parse.Decode(untrustedJSON, options)
if err != nil {
    return err
}

semantic := validate.Document(ctx, parsed.Document(), validate.DefaultOptions())
if !semantic.Valid() {
    return fmt.Errorf("invalid OpenRPC document: %v", semantic.Diagnostics())
}

raw, _ := jsonvalue.Parse(untrustedJSON, jsonvalue.DefaultPolicy())
structural := validate.MetaSchema(ctx, raw, 1000)
if !structural.Valid() {
    return fmt.Errorf("meta-schema failure: %v", structural.Issues())
}
```

`parse.Preserving` retains the exact accepted source for lossless re-emission;
canonical serialization sorts object keys and omits insignificant whitespace.

### Discovery

```go
service, _ := discovery.NewService(discovery.Static(document), visibilityPolicy)
snapshot, err := service.Discover(ctx)
if err != nil {
    return err
}

fmt.Println(snapshot.ETag())
fmt.Println(string(snapshot.Bytes()))
```

Wrap a service with `discovery.NewCache` for explicit concurrent miss
deduplication. Call `Invalidate` when the provider revision changes. No cache,
goroutine, or registry is process-global.

## Optional observability

The `observe` leaf package wraps parse, validate, resolve, bundle, diff, and
discovery operations without changing core APIs or installing an exporter.
Observers receive only finite phase and outcome labels, diagnostic or reference
counts, and duration. Events never contain documents, schemas, method names,
references, URLs, or error strings. Observer panics are contained.

```go
result, err := observe.Parse(ctx, input, parse.DefaultOptions(),
    observe.ObserverFunc(func(ctx context.Context, event observe.Event) {
        metrics.Record(event.Phase, event.Outcome, event.Duration)
    }),
)
```

## JSON-RPC integration

```go
registry := gojsonrpc.NewRegistry()
err := openrpcjsonrpc.RegisterDiscovery[gojsonrpc.Handler](registry, service)
if err != nil {
    return err
}

handler, _ := registry.Lookup("rpc.discover")
result, err := handler(ctx, requestParams)
```

The sibling `jsonrpc.Registry` exposes an explicit trusted system-method
path while continuing to reserve `rpc.*` from application registration. The
adapter does not fork JSON-RPC batch, notification, request, response, error,
or transport behavior.

## Explicit references

```go
store, _ := reference.NewMemoryStore(map[string][]byte{
    "https://schemas.example/value.json": schemaBytes,
})
policy := reference.DefaultResolvePolicy()
policy.AllowExternal = true
policy.AllowedSchemes = []string{"https"}
policy.AllowedHosts = []string{"schemas.example"}
resolver, _ := reference.NewResolver(store, policy)

target, err := resolver.Resolve(ctx, rootJSON, documentURI, rawReference)
```

`reference.NewFSStore` scopes an explicit `fs.FS`. The optional
`reference/httpstore` package adds DNS/IP checks, HTTPS-by-default behavior,
redirect and timeout limits, compression rejection, and streamed byte limits.

## Compatibility and support

- Supported OpenRPC feature lines: `1.3.x` and `1.4.x`.
- Authoritative pinned release: OpenRPC 1.4.1.
- JSON Schema dialect: Draft 7, including boolean schemas.
- Minimum Go version: see `.go-version` and `go.mod`.
- The official `1.3.0` metrics example is retained as accepted
  interoperability evidence. Examples on earlier feature lines remain explicit
  rejection fixtures.

See [security](docs/security.md), [architecture](docs/architecture.md),
[compatibility](docs/compatibility.md), the explicit
[specification decisions](docs/specification-decisions.md), and the generated
conformance evidence under `specification/conformance/`.

The [documentation index](docs/README.md) organizes adoption, reference,
operations, specification, and maintainer material.

## Local verification

```sh
make check
make check-all
```

`make coverage` enforces exact production statement coverage. `make check-all`
runs the complete repository contract, including mutation and conformance
checks.

## Related packages

- [JSON-RPC](https://github.com/faustbrian/go-jsonrpc) provides the runtime
  protocol implementation used by the discovery adapter.
- [JSON Schema](https://github.com/faustbrian/go-json-schema) is appropriate
  when applications need schema validation outside OpenRPC documents.
