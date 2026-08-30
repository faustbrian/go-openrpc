package validate

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc"
	"github.com/faustbrian/go-openrpc/jsonvalue"
	peer "github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPublishedMetaSchemaDiffersFromNormativeServerURLPolicy(t *testing.T) {
	t.Parallel()

	decodePeerSchema := func(input []byte) any {
		t.Helper()
		decoder := json.NewDecoder(bytes.NewReader(input))
		decoder.UseNumber()
		var document any
		if err := decoder.Decode(&document); err != nil {
			t.Fatal(err)
		}
		rewriteMetaDialect(document)
		return document
	}

	compiler := peer.NewCompiler()
	compiler.DefaultDraft(peer.Draft7)
	compiler.AssertFormat()
	if err := compiler.AddResource(
		"https://meta.json-schema.tools/",
		decodePeerSchema(openrpc.JSONSchemaToolsMetaSchema()),
	); err != nil {
		t.Fatal(err)
	}
	const schemaURL = "https://openrpc.invalid/published-schema.json"
	if err := compiler.AddResource(schemaURL, decodePeerSchema(openrpc.MetaSchema())); err != nil {
		t.Fatal(err)
	}
	published, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}

	document := []byte(`{
		"openrpc":"1.4.1",
		"info":{"title":"Servers","version":"1"},
		"servers":[{"url":"../rpc"}],
		"methods":[]
	}`)
	var peerDocument any
	if err := json.Unmarshal(document, &peerDocument); err != nil {
		t.Fatal(err)
	}
	if err := published.Validate(peerDocument); err == nil {
		t.Fatal("published meta-schema peer accepted the normative relative Server URL")
	}

	value, err := jsonvalue.Parse(document, jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	report := MetaSchema(context.Background(), value, 100)
	if report.Err() != nil || !report.Valid() {
		t.Fatalf("owned normative policy report = %#v, error = %v", report.Issues(), report.Err())
	}
}
