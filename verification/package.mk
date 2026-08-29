SHELL := /usr/bin/env bash

GO ?= go

.PHONY: conformance interoperability

conformance:
	$(GO) test ./internal/specification ./validate -count=1
	$(GO) run ./internal/specification/cmd/specmatrix
	git diff --exit-code -- specification/conformance

interoperability:
	./verification/check-jsonrpc-integration.sh
