package parse_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/parse"
)

// TestDecodeRequiredIdentityFailuresRetainConstructorOwnership distinguishes
// a present string of the correct JSON kind from a valid required identity.
func TestDecodeRequiredIdentityFailuresRetainConstructorOwnership(t *testing.T) {
	input, err := os.ReadFile("testdata/complete-openrpc.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ Name, Object, Field, Pointer string }{
		{"info title", "#/definitions/infoObject", "title", "#/info"},
		{"info version", "#/definitions/infoObject", "version", "#/info"},
		{"method name", "#/definitions/methodObject", "name", "#/methods/0"},
		{"descriptor name", "#/definitions/contentDescriptorObject", "name", "#/methods/0/params/0"},
		{"reference target", "#/definitions/referenceObject", "$ref", "#/components/examplePairings/Read/params/0"},
		{"tag name", "#/definitions/tagObject", "name", "#/methods/0/tags/0"},
		{"pairing name", "#/definitions/examplePairingObject", "name", "#/methods/0/examples/0"},
		{"example name", "#/definitions/exampleObject", "name", "#/methods/0/examples/0/params/0"},
		{"external documentation URL", "#/definitions/externalDocumentationObject", "url", "#/externalDocs"},
		{"server URL", "#/definitions/serverObject", "url", "#/servers/0"},
	}
	for _, version := range []string{"1.3.0", "1.4.1"} {
		for _, mode := range []parse.UnknownFieldMode{parse.RejectUnknownFields, parse.PreserveUnknownFields} {
			options := parse.DefaultOptions()
			options.UnknownFields = mode
			control := decodeJSON(t, input).(map[string]any)
			control["openrpc"] = version
			encoded, err := json.Marshal(control)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := parse.Decode(encoded, options)
			if err != nil || parsed.Document().Version().String() != version {
				t.Fatalf("admitted fixture control: %v", err)
			}
			for _, test := range cases {
				t.Run(version+"/"+test.Name, func(t *testing.T) {
					document := decodeJSON(t, encoded)
					objectAt(t, document, completeObjectPaths[test.Object])[test.Field] = ""
					changed, err := json.Marshal(document)
					if err != nil {
						t.Fatal(err)
					}
					result, err := parse.Decode(changed, options)
					var owner *parse.Error
					var missing *openrpc.MissingRequiredFieldError
					if !errors.Is(err, openrpc.ErrMissingRequiredField) || !errors.As(err, &owner) || owner.Pointer != test.Pointer || !errors.As(err, &missing) || missing.Field != test.Field {
						t.Fatalf("required identity lost constructor cause/location: error=%v", err)
					}
					if len(result.PreservingJSON()) != 0 || result.Document().Version().String() != "" {
						t.Fatal("failed identity admission published a partial document/source")
					}
				})
			}
		}
	}
}
