# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected module's docs.
2. Run the standalone Go baseline below.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

Repository-wide checks are provided by the checksum-pinned
`go-library-tools` release configured in [`.golib.yaml`](.golib.yaml). Keep
generic verification behavior in that shared tool; repository-specific checks
belong under `verification/` and must be documented here and in the relevant
technical guide.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable entries in the
[specification decision register](docs/specification-decisions.md), and complete
the Specification Decisions section of the pull request template. An
unresolved interpretation or stale source pin is release-blocking; peer
behavior cannot silently select policy.

Explicit full and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not fork shared verification workflows, add permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Use the declared Go 1.27.2 producer; the module minimum remains Go 1.27.0.
Test independently of sibling checkouts:

```bash
GOWORK=off go test ./...
```

The module declares no services or test/build tags. This command runs
deterministic fuzz seeds and saved corpus regressions, not active exploration.
Existing builder, discovery, observer, reference and HTTP-store TestMain
checks also retain goroutine cleanup assertions in ordinary tests.
With the checksum-pinned public Tools v2.4.0 verifier selected by `.golib.yaml`:

```bash
make inventory
make check
make ci
```

Both Make routes run bounded local checks without building the verifier release
from source. One selected `builder` race verifies successful concurrent method
registration, sorted snapshot contents and duplicate refusal, once within two
minutes. Shared setup still checks out immutable tooling, but no sibling
repository or goal ledger is needed. Base-free checks run seeds/corpora;
changed-source CI selects bounded active smoke for causally affected targets.
Short active exploration selects one target; optional local mutation selects
one package:

```bash
GOMAXPROCS=2 GOWORK=off go test -run '^$' -fuzz '^FuzzDecodeOpenRPCDocument$' -fuzztime=2s -parallel=1 -timeout=2m ./parse
golib mutation --module . --package ./parse
```

`make ci` adds repository and offline specification/decision validation.
`make specification-check-online` selects upstream authority checks explicitly;
`make check-full` includes them and retains all 12 native two-second fuzz targets,
full coverage/race/mutation/API/security, conformance and matrix-drift checks,
the pinned JSON-RPC interoperability consumer, and 100-millisecond benchmarks
with their original bounds. The interoperability consumer uses a temporary
module with the candidate root and published JSON-RPC dependency; it does not
prove that the candidate is published. Online authority and public dependency
resolution require network access.

PR, main and scheduled shared CI remain bounded. Select full qualification
explicitly with `make check-full` or `release_dry_run: true`. Required accepts
only shared-CI success. The separate manual native-security diagnostic still
uses an immutable source-built `dev` verifier, refuses a mixed release
rehearsal and retains its exact-producer, clean-tree and cleanup assertions; it
is development evidence, not release evidence. Report unavailable prerequisites
or failures; partial results are not release-ready.

## Selected Monthly Assurance

The owner-level [selected-assurance caller](.github/workflows/assurance.yml) is
configured for main only, on day 25 of each month at 05:41 UTC. It selects one
of the 12 current declared root-module fuzz targets using the shared UTC-week
rotation, with 30 seconds of active exploration. The selected risks are:
finite document/JSON parsing, input ownership, canonical round trips, Draft 7
validation, internal references, expressions, composition, semantic diff and
static discovery snapshots.
These targets use in-process inputs with external reference loading disabled;
they do not qualify live HTTP stores, upstream authorities or the pinned
published JSON-RPC interoperability consumer. Those full/manual routes remain.

Base-free ordinary checks execute deterministic seeds, not active exploration;
committed-range CI selects affected active smoke. Manual assurance requires an
exact anchored selector from the root fuzz operation in `.golib.yaml`, for
example `^FuzzDecodeOpenRPCDocument$`.
Empty, unknown and malformed selections refuse. The shared job bounds setup
and execution to ten minutes, active workers and CPU to two, owned memory to
2 GiB, native output to 1 MiB and private-cache lifetime. Failure, cancellation,
exhausted budgets and cleanup refusal remain attributable failures. This lane
does not run mutation or replace Required, original full fuzz budgets,
explicit `make check-full`, or relevant release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.

## Source Documentation

Install the checksum-pinned public verifier declared in `.golib.yaml`.
`make source-docs-check` validates objective Go comment, marker, link and
generated-producer structure; `make check`, `make check-full` and `make ci`
require it before their existing checks. Shared CI runs the same strict
contract and includes its result in Required. This is structural enforcement,
not a substitute for source-truth review or behavioral verification.
