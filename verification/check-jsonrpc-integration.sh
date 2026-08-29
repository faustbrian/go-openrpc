#!/usr/bin/env bash
set -euo pipefail

openrpc_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
integration_dir=$(mktemp -d "${TMPDIR:-/tmp}/openrpc-jsonrpc.XXXXXX")
cleanup() {
    find "${integration_dir}" -depth -delete
}
trap cleanup EXIT HUP INT TERM

cat > "${integration_dir}/go.mod" <<EOF
module integration.test/openrpcjsonrpc

go 1.26.6

require (
    github.com/faustbrian/go-jsonrpc v1.0.0
    github.com/faustbrian/go-openrpc v0.0.0
)

replace github.com/faustbrian/go-openrpc => ${openrpc_root}
EOF

cat > "${integration_dir}/integration_test.go" <<'EOF'
package integration_test

import (
    "context"
    "encoding/json"
    "testing"

    gojsonrpc "github.com/faustbrian/go-jsonrpc"
    "github.com/faustbrian/go-openrpc/discovery"
    openrpcjsonrpc "github.com/faustbrian/go-openrpc/jsonrpc"
    openrpcparse "github.com/faustbrian/go-openrpc/parse"
)

func TestRegisterDiscoveryWithGoJSONRPCRegistry(t *testing.T) {
    parsed, err := openrpcparse.Decode([]byte(`{
        "openrpc":"1.4.1",
        "info":{"title":"Integration","version":"1"},
        "methods":[]
    }`), openrpcparse.DefaultOptions())
    if err != nil {
        t.Fatal(err)
    }
    service, err := discovery.NewService(discovery.Static(parsed.Document()), nil)
    if err != nil {
        t.Fatal(err)
    }
    registry := gojsonrpc.NewRegistry()
    if err := openrpcjsonrpc.RegisterDiscovery[gojsonrpc.Handler](registry, service); err != nil {
        t.Fatal(err)
    }
    handler, found := registry.Lookup(discovery.MethodName)
    if !found {
        t.Fatal("rpc.discover was not registered")
    }
    result, err := handler(context.Background(), json.RawMessage(`[]`))
    if err != nil {
        t.Fatal(err)
    }
    raw, ok := result.(json.RawMessage)
    if !ok || !json.Valid(raw) {
        t.Fatalf("result = %#v", result)
    }
}
EOF

(
    cd "${integration_dir}"
    GOWORK=off go mod tidy
    GOWORK=off go test ./... -count=1
)
