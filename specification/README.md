# Authoritative specification inputs

OpenRPC 1.4.1 is the authoritative stable specification for this module. The
release is pinned to its immutable Git commit and each copied input is verified
by SHA-256 in `manifest.json`. The upstream changelog tracks published errata,
including the 1.4.1 version-pattern fix. The Apache 2.0 license is copied beside
the inputs.

The official example corpus is pinned independently to its immutable repository
commit and copied with the same checksum policy. Some examples predate the 1.4
release; they are interoperability fixtures, not evidence of 1.4 conformance by
themselves.

Run `scripts/sync-spec.sh` from this module directory to reproduce the local
copies. The command performs explicit network access; package parsing and
validation never invoke it.

The pinned 1.4 schema structurally accepts the `1.4.x` compatibility line. The
typed model, parser, semantic validator, discovery, and canonical serializer
also accept the inventoried `1.3.x` line. Earlier minor lines and future minor
or major lines remain rejected until their semantics are separately
inventoried and tested. JSON Schema values use Draft 7, as required by OpenRPC.

The published OpenRPC meta-schema references `https://meta.json-schema.tools/`
as its companion Draft 7 schema dialect. That response is pinned, normalized
with `jq -S`, checksummed in the manifest, and embedded for offline validation.
The validator rewrites the companion dialect declaration to the canonical
Draft 7 URI before compilation. It also removes the Server Object `url` format
assertion because that generic URI check contradicts the normative support for
relative URLs and server-variable templates; dedicated semantic validation
still validates those forms. No validation path fetches the live URL.

The normative and object-field matrices are generated from these pinned inputs
and then reviewed against prose requirements that JSON Schema cannot express.
The canonical
[`docs/specification-decisions.md`](../docs/specification-decisions.md)
records every material interpretation, consequence, and reconsideration
condition behind those matrices.

## Decision conformance matrix

Interoperability and maintained-peer comparisons are recorded only when they
were actually performed. `not assessed` is intentional evidence status, not a
passing interoperability claim.

| Authority boundary | Decisions | Executable evidence |
| --- | --- | --- |
| OpenRPC prose, meta-schema, and examples | OPENRPC-DEC-001, OPENRPC-DEC-002, OPENRPC-DEC-003, OPENRPC-DEC-004, OPENRPC-DEC-005, OPENRPC-DEC-008, OPENRPC-DEC-009, OPENRPC-DEC-010, OPENRPC-DEC-011 | `specification/conformance.json` |
| JSON Schema Draft 7 | OPENRPC-DEC-006 | `specification/conformance.json` |
| JSON-RPC 2.0 discovery binding | OPENRPC-DEC-007 | `specification/conformance.json` |

The structured register, source monitoring, conformance bindings, and immutable
decision digests live in `decisions.json`, `monitoring.json`,
`conformance.json`, and `decision-history.json` in this directory.

The practical [interoperability register](interoperability.tsv) records the
pinned official-example outcomes and the maintained
`santhosh-tekuri/jsonschema` comparison separately from normative policy.

## Upstream review history

### 2026-09-03

- JSON Schema specification `main` advanced from
  `499eba5749b0a22940e15660dafe50b74df05cb9` through
  `0932747f3f3128758f3166e0d3e23e0b8d1025ee`. The reviewed changes affect
  future v1 meta-test infrastructure and an illustrative trailing comma;
  immutable Draft 7 sources are unchanged. The range is behavior-neutral for
  the bound reference-resolution and embedded-schema dialect decisions, so
  their selected behavior remains unchanged. The releases monitor advances
  from response SHA-256
  `545aea6453cf680d77e832f76a8bc666124862fe82d409127f1903c985329c2e`
  to `812a513adefe7b4ef88ffc59a9e643cd447c1da5a79023424406354ce8081184`.
