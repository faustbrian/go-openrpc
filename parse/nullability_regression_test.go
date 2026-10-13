package parse_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/parse"
)

// TestDecodeScalarNullabilityUsesPinnedFieldKinds checks Decode itself, not the
// meta-schema/parser conjunction that can conceal null-to-zero coercion.
func TestDecodeScalarNullabilityUsesPinnedFieldKinds(t *testing.T) {
	input, err := os.ReadFile("testdata/complete-openrpc.json")
	if err != nil {
		t.Fatal(err)
	}
	matrix, err := os.ReadFile("../specification/conformance/object-fields.tsv")
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(bytes.NewReader(matrix))
	reader.Comma = '\t'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	for _, row := range rows[1:] {
		if row[4] != "false" || row[2] != "string" && row[2] != "boolean" {
			continue
		}
		cases++
		t.Run(row[0]+"/"+row[1], func(t *testing.T) {
			for _, invalid := range []any{nil, []any{}} {
				document := decodeJSON(t, input)
				objectAt(t, document, completeObjectPaths[row[0]])[row[1]] = invalid
				encoded, err := json.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				for _, mode := range []parse.UnknownFieldMode{parse.RejectUnknownFields, parse.PreserveUnknownFields} {
					options := parse.DefaultOptions()
					options.UnknownFields = mode
					_, err := parse.Decode(encoded, options)
					if !errors.Is(err, parse.ErrInvalidObject) {
						t.Fatal("typed field accepted null or did not report a kind error")
					}
				}
			}
		})
	}
	if cases == 0 {
		t.Fatal("pinned matrix selected no scalar-kind cases")
	}
}

// TestDecodeStringArrayRejectsNullMembers preserves array element kinds rather
// than allowing encoding/json to turn a null enum member into an empty string.
func TestDecodeStringArrayRejectsNullMembers(t *testing.T) {
	input := []byte(`{"openrpc":"1.4.1","info":{"title":"Nulls","version":"1"},"methods":[],"servers":[{"name":"api","url":"https://example.test/{region}","variables":{"region":{"default":"","enum":[null]}}}]}`)
	if _, err := parse.Decode(input, parse.DefaultOptions()); !errors.Is(err, parse.ErrInvalidObject) {
		t.Fatal("null string-array member was admitted")
	}
}

// TestDecodePermittedNullPayloadsAndScalarPresence keeps payload nulls, absent
// optional values and explicit empty/false values distinct on both feature lines.
func TestDecodePermittedNullPayloadsAndScalarPresence(t *testing.T) {
	for _, version := range []string{"1.3.0", "1.4.1"} {
		for _, explicit := range []bool{false, true} {
			fields := ""
			if explicit {
				fields = `,"description":"","deprecated":false`
			}
			input := []byte(`{"openrpc":"` + version + `","info":{"title":"Nulls","version":"1"},"methods":[{"name":"read","params":[]` + fields + `,"errors":[{"code":1,"message":"Failure.","data":null}]}],"components":{"examples":{"Null":{"name":"Null","value":null}}}}`)
			parsed, err := parse.Decode(input, parse.DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := openrpc.MarshalCanonical(parsed.Document())
			if err != nil {
				t.Fatal(err)
			}
			for _, fragment := range []string{`"data":null`, `"value":null`} {
				if !bytes.Contains(canonical, []byte(fragment)) {
					t.Fatal("allowed explicit null payload disappeared")
				}
			}
			if strings.Contains(string(canonical), `"deprecated":false`) != explicit || strings.Contains(string(canonical), `"description":""`) != explicit {
				t.Fatal("optional scalar presence changed")
			}
			if !bytes.Equal(parsed.PreservingJSON(), input) {
				t.Fatal("retained input changed")
			}
		}
	}
}
