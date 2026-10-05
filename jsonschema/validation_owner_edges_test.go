package jsonschema_test

import (
	"context"
	"errors"
	"testing"

	"github.com/faustbrian/go-openrpc/v2/jsonschema"
)

func TestValidationOwnerZeroSchemaPreservesPolicyErrorPrecedence(t *testing.T) {
	var zero jsonschema.Schema
	options := jsonschema.DefaultValidationOptions()
	if _, err := jsonschema.Compile(zero, options); !errors.Is(err, jsonschema.ErrSchemaCompile) {
		t.Fatalf("zero schema with valid policy = %v, want schema compile error", err)
	}
	options.MaxSchemaBytes = 0
	if _, err := jsonschema.Compile(zero, options); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("zero schema with zero byte policy = %v, want policy error", err)
	}
}

func TestValidationOwnerInstanceByteLimitIsInclusive(t *testing.T) {
	instance := parseValue(t, `12`)
	for _, bound := range []int{instance.ByteLen(), instance.ByteLen() - 1} {
		options := jsonschema.DefaultValidationOptions()
		options.MaxInstanceBytes = bound
		compiled, err := jsonschema.Compile(parseSchema(t, `true`), options)
		if err != nil {
			t.Fatal(err)
		}
		compiled, err = compiled.WithMaxIssues(1)
		if err != nil {
			t.Fatal(err)
		}
		report := compiled.Validate(context.Background(), instance)
		if bound == instance.ByteLen() {
			if !report.Valid() {
				t.Fatalf("inclusive byte limit = %v", report.Err())
			}
		} else if !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) || len(report.Issues()) != 0 {
			t.Fatalf("below instance byte limit = %v, issues=%v", report.Err(), report.Issues())
		}
	}
}

func TestValidationOwnerShortCircuitGraphBoundaryIsInclusive(t *testing.T) {
	schema := parseSchema(t, `{"anyOf":[true,false,false]}`)
	instance := parseValue(t, `0`)
	for _, steps := range []int{4, 3} {
		options := jsonschema.DefaultValidationOptions()
		options.MaxValidationSteps = steps
		compiled, err := jsonschema.Compile(schema, options)
		if err != nil {
			t.Fatal(err)
		}
		report := compiled.Validate(context.Background(), instance)
		if steps == 4 {
			if !report.Valid() {
				t.Fatalf("four-node short-circuit graph at four steps = %v", report.Err())
			}
		} else if !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) || len(report.Issues()) != 0 {
			t.Fatalf("graph below inclusive boundary = %v, issues=%v", report.Err(), report.Issues())
		}
	}
}
