# Changelog

All notable changes will be documented here. The format follows Keep a
Changelog principles, and releases use semantic versioning.

## Unreleased

### Changed

- Record attributable official-corpus and maintained-validator comparisons for
  specification decisions.
- Make every OpenRPC, Draft 7, and JSON-RPC interpretation discoverable through
  a structured decision register with pinned authority monitoring, executable
  conformance bindings, and immutable decision history. See the
  [specification decision register](docs/specification-decisions.md).
  Current decision digests:
  - OPENRPC-DEC-001 sha256:005dd1f56d5dc911ebdae3d7ab500c1bc6a06a5b342dc56b9afdcd7164901e61
  - OPENRPC-DEC-002 sha256:ba974caef310f07a68f587b7617742d013d3ac264fb887a7c88041ad751b24f1
  - OPENRPC-DEC-003 sha256:e80273092fa3a728d3ae9b105121054f6b7434753052cdff7323670722dea1fe
  - OPENRPC-DEC-004 sha256:ebbe99d02c8c58c5e2f4eed73abae7db3bf34e277b60adbfbb7d1920921a0860
  - OPENRPC-DEC-005 sha256:15179e069b7678d93ed7c6ec8f653d81bc0e1f921c8e67983c8f547f54bd2740
  - OPENRPC-DEC-006 sha256:f8f8e0b1ad9f0f16ee7715b8636308393def9db8042a2dc432f50eed51345f3c
  - OPENRPC-DEC-007 sha256:6c89045cf9e5782af2f0abc52d8d7b92c57c74ee90e1839ba9ce14aa32d29f45
  - OPENRPC-DEC-008 sha256:d13693bd67216a75c18f58eff0322b4ff7782371f79791fc0b7eccdd9d75d527
  - OPENRPC-DEC-009 sha256:30798813f1d0f951cf3fc0d702cd0e1b77c0b28b0bfcc24ea21f459b38d01ee7
  - OPENRPC-DEC-010 sha256:fb9d73438f4495c52c513a50eaba20faf4c18fd425a402966c122e012b51fa46
  - OPENRPC-DEC-011 sha256:a7fb6226a384fb39d107265c4954579ff8f4ed2a139e94153eed4f58080194b5
- Adopt the checksum-pinned `go-library-tools` v1.2.0 release and immutable
  `1f9629e5f27418600460b55a50a5b2fc81697fab` workflow for repository
  verification, while keeping OpenRPC conformance and JSON-RPC interoperability
  checks as source-owned operations.
- Replace repository-local generic tooling with the standalone repository
  contract and retain mutation evidence under `.verification`.

### Documentation

- Replace archived monorepo and AI-generated documentation entry points with
  a standalone, human-oriented documentation structure.

## 1.0.0 - 2026-08-25

### Changed

- Verify JSON-RPC interoperability through the configured module proxy instead
  of requiring a sibling repository checkout.

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Regenerate the complete standalone documentation bundle from current source
  documents.

- Correct stale package, standalone, and authoritative-source links in public
  documentation.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  monorepo-canonical targets and current release guidance.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-openrpc` identity while preserving its documented API and behavior.
- Upgrade `golang.org/x/text` to v0.41.0 so the dependency graph no longer
  contains GO-2026-5970.
- Link authoritative specification inputs directly to the canonical decision
  register.
- Report bounded Draft 7 regular-expression timeouts as resource-limit errors
  instead of ordinary pattern mismatches, and give checksum-pinned OpenRPC
  meta-schema patterns the maximum bounded validation window.
- Accept OpenRPC `1.3.x` documents through the typed model, parser, semantic
  validator, discovery, and canonical serializer alongside `1.4.x`; the pinned
  `1.4.1` meta-schema remains the structural validator for that feature line,
  and semantic diffs classify feature-line changes as conditional.
- Make every allowed reference path shape length-exact before indexing, keeping
  malformed and truncated paths fail-closed.
- Execute API compatibility tooling against the isolated module graph so owned
  dependency source changes cannot conflict with release checksums.
- Regenerated the complete documentation bundle from the current public
  guides and API documentation.
- The integration target now invokes the checked-in JSON-RPC interoperability
  script, so local and CI integration checks execute the documented contract.

### Added

- An auditable OpenRPC specification decision register covering artifact
  precedence, references, examples, Draft 7 schemas, discovery, extensions,
  serialization, composition, and compatibility diffing.
- Attributable repository execution for the normative conformance matrix.
- Complete lossless OpenRPC 1.4.1 document model and strict parser.
- Draft 7 schema validation with explicit external resources.
- Bounded reference resolution, dereferencing, and lossless resource bundles.
- Runtime expression and server-variable evaluation.
- Discovery, composition, builders, resolved semantic diff, and JSON-RPC
  discovery registration contracts.
- Configurable discovery validation and canonical output budgets for static,
  generated, and filtered documents.
- Allocation-free document method counts and bounded semantic validation for
  generated documents that bypass parser collection limits.
- Normative and object-field conformance matrices with executable evidence.
- Payload-free optional observability, fuzz targets, and allocation benchmarks.
- Blocking goroutine leak gates for registries, discovery caching, observer
  hooks, resolution, HTTP loading, and cancellation paths.
- End-to-end `rpc.discover` registration with the sibling `jsonrpc`
  registry, verified through an isolated cross-module integration gate.
- Fail-closed semantic compatibility decisions for conditional and truncated
  diff reports.
- Supported-version, meta-schema, ecosystem interoperability, resource-budget,
  resolver-threat, and reproducible benchmark evidence reports.
- Deterministic semantic validation on every document accepted by the parser
  fuzz target.

### Security

- External access is disabled by default and HTTP resolution enforces explicit
  scheme, host, IP, redirect, compression, timeout, and byte policies.
- Resolver inputs, alias chains, bundle roots, and transitive resource graphs
  share an explicit aggregate reference-count budget.
- Draft 7 compilation bounds both the number of explicit schema resources and
  their aggregate encoded bytes.

### Notes

- Pinned upstream examples retain their declared feature lines. The `1.3.0`
  metrics document is accepted by the typed parser; earlier examples remain
  explicit rejections rather than being relabeled as 1.4.1.
