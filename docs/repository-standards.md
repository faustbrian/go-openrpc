# Repository standards

- The verification producer is Go 1.27.2 in `.go-version` and the root catalog;
  the public module minimum remains Go 1.27.0 in `go.mod`.
- Dependencies and tools use reviewed versions; workflow actions use immutable
  commit SHAs.
- `make check` and `make ci` run bounded local verification; `make ci` adds
  offline specification validation. `make check-full` retains the explicit full
  race, fuzz, coverage, mutation, conformance, interoperability, API, docs,
  security and benchmark campaigns. Scheduled shared CI remains bounded;
  hosted release rehearsal requires `release_dry_run: true`.
- Production code may not use `unsafe`, cgo, or `go:linkname`.
- Core packages perform no implicit I/O and install no global mutable state.
- Conventional commits include a rationale body; release notes come from the
  changelog.
- Security reports remain private until coordinated disclosure.
- Licenses and provenance are reviewed whenever code, specification inputs, or
  dependencies change.

Repository settings should enable vulnerability alerts, dependency review,
secret scanning, signed releases, protected branches, required reviews, and
the blocking workflow jobs documented here.
