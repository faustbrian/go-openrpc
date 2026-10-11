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

Do not add package-local workflows, permanent replacements, machine-specific
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
With the checksum-pinned public Tools v2.1.1 verifier selected by `.golib.yaml`:

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

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
