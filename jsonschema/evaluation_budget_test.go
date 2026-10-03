package jsonschema

import (
	"context"
	"errors"
	"testing"

	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
)

func TestOrdinaryValidationChargesEverySchemaEvaluation(t *testing.T) {
	for _, test := range []struct {
		name   string
		schema string
		steps  int
	}{
		{name: "boolean", schema: `true`, steps: 3},
		{name: "reference", schema: `{"definitions":{"value":true},"$ref":"#/definitions/value"}`, steps: 4},
		{name: "allOf", schema: `{"allOf":[true]}`, steps: 4},
		{name: "anyOf", schema: `{"anyOf":[true]}`, steps: 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema, err := Parse([]byte(test.schema), jsonvalue.DefaultPolicy())
			if err != nil {
				t.Fatal(err)
			}
			instance, err := jsonvalue.Parse([]byte(`0`), jsonvalue.DefaultPolicy())
			if err != nil {
				t.Fatal(err)
			}
			for _, steps := range []int{test.steps - 1, test.steps} {
				options := DefaultValidationOptions()
				options.MaxValidationSteps = steps
				compiled, err := Compile(schema, options)
				if err != nil {
					t.Fatal(err)
				}
				report := compiled.Validate(context.Background(), instance)
				if steps < test.steps {
					if !errors.Is(report.Err(), ErrValidationResourceLimit) {
						t.Fatalf("budget %d: error = %v, want resource limit", steps, report.Err())
					}
				} else if !report.Valid() {
					t.Fatalf("budget %d: ordinary valid value rejected: %v", steps, report.Err())
				}
			}
		})
	}
}
