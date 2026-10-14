#!/bin/sh
set -eu
profile="$(mktemp "${TMPDIR:-/tmp}/openrpc-coverage.XXXXXX")"
trap 'rm -f "$profile"' EXIT HUP INT TERM
packages="$(go run scripts/coverage-exception/check.go --targets)"
go test ./... -covermode=atomic -coverpkg="$packages" -coverprofile="$profile" -count=1 -timeout=3m
go run scripts/coverage-exception/check.go --profile "$profile"
