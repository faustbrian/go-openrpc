package openrpc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
)

// TestCanonicalAggregateDepthRefusalPreservesAdmittedRawValue exercises finite
// public constructor input, not a forged internal JSON value. The Go 1.27.2
// codec admits each 10000-deep standalone payload but rejects the extra
// document level during the first marshal phase under the default JSONv2 codec.
func TestCanonicalAggregateDepthRefusalPreservesAdmittedRawValue(t *testing.T) {
	for _, depth := range []int{9999, 10000} {
		raw := []byte(strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth))
		policy := jsonvalue.DefaultPolicy()
		policy.MaxDepth = 10000
		value, err := jsonvalue.Parse(raw, policy)
		if err != nil {
			t.Fatalf("finite standalone value was not admitted: depth=%d error=%v", depth, err)
		}
		standalone, err := json.Marshal(value)
		if err != nil || !bytes.Equal(standalone, raw) {
			t.Fatalf("individual raw marshal control failed: depth=%d error=%v", depth, err)
		}
		extension, err := openrpc.NewExtensions(openrpc.Field{Name: "x-depth", Value: value})
		if err != nil {
			t.Fatal(err)
		}
		info, err := openrpc.NewInfo(openrpc.InfoInput{Title: "Depth boundary", Version: "1"})
		if err != nil {
			t.Fatal(err)
		}
		version, err := openrpc.ParseVersion("1.4.1")
		if err != nil {
			t.Fatal(err)
		}
		document, err := openrpc.NewDocument(openrpc.DocumentInput{Version: version, Info: &info, Methods: []openrpc.MethodOrReference{}, Extensions: extension})
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := openrpc.MarshalCanonical(document)
		if depth == 10000 {
			var failure *json.SyntaxError
			var marshaler *json.MarshalerError
			if !errors.As(err, &failure) || !errors.As(err, &marshaler) || failure.Offset <= 0 || !strings.Contains(failure.Error(), "exceeded max depth") || canonical != nil {
				t.Fatalf("aggregate depth refusal lost codec cause or published partial output: error=%v", err)
			}
		} else {
			if err != nil {
				t.Fatalf("admitted aggregate control: %v", err)
			}
			if _, err := jsonvalue.Parse(canonical, policy); err != nil {
				t.Fatalf("canonical aggregate control is invalid: %v", err)
			}
		}
		retained, ok := document.Extensions().Get("x-depth")
		if !ok || !bytes.Equal(retained.Bytes(), raw) || !bytes.Equal(value.Bytes(), raw) {
			t.Fatal("canonical success/refusal changed owned raw input")
		}
	}
}
