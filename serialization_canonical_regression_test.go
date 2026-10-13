package openrpc_test

import (
	"bytes"
	"context"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/discovery"
	"github.com/faustbrian/go-openrpc/v2/parse"
)

// TestCanonicalOpaqueObjectsSortWithoutChangingRawValues locks DEC-009 at the
// canonical boundary, leaving exact source, JSON values and number lexemes alone.
func TestCanonicalOpaqueObjectsSortWithoutChangingRawValues(t *testing.T) {
	first := []byte(`{"openrpc":"1.4.1","info":{"title":"Opaque","version":"1"},"methods":[{"name":"read","params":[],"result":{"name":"out","schema":{"x-data":{"z":-0,"a":[{"z":9007199254740993,"a":1e+20}]}}}}],"x-data":{"z":-0,"a":[{"z":9007199254740993,"a":1e+20}]},"future":{"z":2,"a":1},"components":{"examples":{"Opaque":{"name":"Opaque","value":{"z":2,"a":1}}}}}`)
	second := []byte(`{"openrpc":"1.4.1","info":{"title":"Opaque","version":"1"},"methods":[{"name":"read","params":[],"result":{"name":"out","schema":{"x-data":{"a":[{"a":1e+20,"z":9007199254740993}],"z":-0}}}}],"x-data":{"a":[{"a":1e+20,"z":9007199254740993}],"z":-0},"future":{"a":1,"z":2},"components":{"examples":{"Opaque":{"name":"Opaque","value":{"a":1,"z":2}}}}}`)
	var canonical []byte
	var revision string
	for _, input := range [][]byte{first, second} {
		options := parse.DefaultOptions()
		options.UnknownFields = parse.PreserveUnknownFields
		parsed, err := parse.Decode(input, options)
		if err != nil {
			t.Fatal(err)
		}
		rawBefore, _ := parsed.Document().Extensions().Get("x-data")
		components, _ := parsed.Document().Components()
		examples, _ := components.Examples()
		exampleBefore := examples["Opaque"].Value().Bytes()
		output, err := openrpc.MarshalCanonical(parsed.Document())
		if err != nil {
			t.Fatal(err)
		}
		if canonical != nil && !bytes.Equal(output, canonical) {
			t.Fatal("opaque member ordering changed canonical bytes")
		}
		canonical = output
		for _, lexeme := range []string{"-0", "9007199254740993", "1e+20"} {
			if !bytes.Contains(output, []byte(lexeme)) {
				t.Fatal("canonicalization changed an exact number lexeme")
			}
		}
		rawAfter, _ := parsed.Document().Extensions().Get("x-data")
		if !bytes.Equal(rawBefore.Bytes(), rawAfter.Bytes()) || !bytes.Equal(exampleBefore, examples["Opaque"].Value().Bytes()) || !bytes.Equal(parsed.PreservingJSON(), input) {
			t.Fatal("canonicalization mutated retained raw bytes")
		}
		service, err := discovery.NewService(discovery.Static(parsed.Document()), nil)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := service.Discover(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if revision != "" && snapshot.Revision() != revision {
			t.Fatal("opaque order changed discovery content identity")
		}
		revision = snapshot.Revision()
		if !bytes.Equal(snapshot.Bytes(), output) || snapshot.ETag() != `"`+revision+`"` {
			t.Fatal("discovery did not publish canonical output")
		}
	}
}

// TestCanonicalRetainsArrayOrderAndNumberSpelling bounds the selected profile:
// neither array permutations nor mathematically equal number lexemes normalize.
func TestCanonicalRetainsArrayOrderAndNumberSpelling(t *testing.T) {
	var outputs [][]byte
	for _, value := range []string{`[3,1]`, `[1,3]`, `0`, `-0`, `1e+20`, `9007199254740993`} {
		input := []byte(`{"openrpc":"1.4.1","info":{"title":"Opaque","version":"1"},"methods":[],"x-value":` + value + `}`)
		parsed, err := parse.Decode(input, parse.DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		output, err := openrpc.MarshalCanonical(parsed.Document())
		if err != nil || !bytes.Contains(output, []byte(`"x-value":`+value)) {
			t.Fatal("canonical profile changed array order or number spelling")
		}
		outputs = append(outputs, output)
	}
	if bytes.Equal(outputs[0], outputs[1]) || bytes.Equal(outputs[2], outputs[3]) {
		t.Fatal("canonical profile normalized intentionally distinct lexical representations")
	}
}
