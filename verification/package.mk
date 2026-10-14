SHELL := /usr/bin/env bash

GO ?= go

.PHONY: conformance interoperability

conformance:
	$(GO) test ./internal/specification ./validate -count=1
	$(GO) run ./internal/specification/cmd/specmatrix
	git diff --exit-code -- specification/conformance

interoperability:
	./verification/check-jsonrpc-integration.sh

.PHONY: coverage-exact

coverage-exact:
	$(GO) test scripts/coverage-exception/check.go scripts/coverage-exception/check_test.go -count=1
	./scripts/check-coverage.sh
