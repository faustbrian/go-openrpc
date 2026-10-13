package validate_test

import (
	"context"
	"testing"

	openrpc "github.com/faustbrian/go-openrpc/v2"
	"github.com/faustbrian/go-openrpc/v2/parse"
	"github.com/faustbrian/go-openrpc/v2/reference"
	"github.com/faustbrian/go-openrpc/v2/validate"
)

// TestEquivalentErrorCodeLexemesHaveOneIdentity applies DEC-004 inline and after
// resolution without changing Integer's public retained-lexeme representation.
func TestEquivalentErrorCodeLexemesHaveOneIdentity(t *testing.T) {
	for _, codes := range []struct {
		first, second string
		duplicate     bool
	}{
		{"0", "-0", true}, {"-0", "0", true},
		{"9007199254740992", "9007199254740993", false},
	} {
		for _, resolved := range []bool{false, true} {
			name := "inline"
			if resolved {
				name = "resolved"
			}
			t.Run(codes.first+"/"+codes.second+"/"+name, func(t *testing.T) {
				errorsJSON := `[{"code":` + codes.first + `,"message":"First."},{"code":` + codes.second + `,"message":"Second."}]`
				components := ""
				if resolved {
					components = `,"components":{"errors":{"First":{"code":` + codes.first + `,"message":"First."},"Second":{"code":` + codes.second + `,"message":"Second."}}}`
					errorsJSON = `[{"$ref":"#/components/errors/First"},{"$ref":"#/components/errors/Second"}]`
				}
				input := []byte(`{"openrpc":"1.4.1","info":{"title":"Identity","version":"1"},"methods":[{"name":"read","params":[],"errors":` + errorsJSON + `}]` + components + `}`)
				parsed, err := parse.Decode(input, parse.DefaultOptions())
				if err != nil {
					t.Fatal(err)
				}
				report := validate.Document(context.Background(), parsed.Document(), validate.DefaultOptions())
				if resolved {
					resolver, err := reference.NewResolver(nil, reference.DefaultResolvePolicy())
					if err != nil {
						t.Fatal(err)
					}
					report = validate.ResolvedDocument(context.Background(), parsed.Document(), "https://example.test/openrpc.json", resolver, validate.DefaultResolvedOptions())
				}
				duplicates := 0
				for _, diagnostic := range report.Diagnostics() {
					if diagnostic.Code == validate.CodeDuplicateErrorCode {
						duplicates++
						if diagnostic.Pointer != "#/methods/0/errors/1/code" {
							t.Fatal("duplicate identity diagnostic location changed")
						}
					}
				}
				if report.Truncated() || (duplicates == 1) != codes.duplicate || duplicates > 1 || len(report.Diagnostics()) != duplicates {
					t.Fatal("equivalent integer codes did not use exact numeric identity")
				}
				if !codes.duplicate && !report.Valid() {
					t.Fatal("distinct exact integer codes were rejected")
				}
				integer, err := openrpc.ParseInteger(codes.first)
				if err != nil || integer.String() != codes.first {
					t.Fatal("duplicate detection changed public integer lexemes")
				}
			})
		}
	}
}
