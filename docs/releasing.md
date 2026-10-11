# Releasing

1. Confirm the worktree is clean on `main` and the release commit is reviewed.
2. Run `make check-full` with the declared Go 1.27.2 producer; the public
   module minimum remains Go 1.27.0. Ordinary and scheduled shared CI use the
   bounded route, not this full qualification. Select the hosted rehearsal
   explicitly with `release_dry_run: true`.
3. Review `CHANGELOG.md`, supported versions, conformance matrices, dependency
   licenses, vulnerability output, API compatibility, and benchmark evidence.
4. Confirm every workflow uses immutable action commits and least-privilege
   permissions.
5. After explicit release authorization, deliberately create and push a signed
   semantic-version tag from the qualified `main` commit. The CI workflow
   validates and rehearses releases; it does not publish tags or releases.
6. Build source artifacts from the tag, generate checksums and provenance, and
   publish release notes from the reviewed changelog.
7. Verify the module proxy resolves the tag and a clean consumer can build it.

Do not release while coverage, mutation, conformance, vulnerability, docs, API,
or integration gates are waived. A failed optional NilAway advisory must remain
visible even when it is not blocking.
