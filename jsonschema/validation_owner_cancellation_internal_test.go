package jsonschema

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
	validator "github.com/santhosh-tekuri/jsonschema/v6"
)

type validationOwnerBeforeCheckpoint struct{ stop func() }

func (extension validationOwnerBeforeCheckpoint) Validate(*validator.ValidatorContext, any) {
	extension.stop()
}

type validationOwnerContext struct{ context.Context }

type validationOwnerExpiredRegexpBudget struct{}

func (validationOwnerExpiredRegexpBudget) Validate(*validator.ValidatorContext, any) {
	_, err := regexpDeadlineTimeout(0, time.Second)
	panic(validationCanceled{err: err})
}

func TestValidationOwnerRegexpDeadlineFailurePrecedesDiagnosticConversion(t *testing.T) {
	schema, err := Parse([]byte(`true`), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(schema, DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	guard := compiled.compiled.AllOf[0]
	original := guard.Extensions
	guard.Extensions = append([]validator.SchemaExt{validationOwnerExpiredRegexpBudget{}}, original...)
	ctx := context.Background()
	value := mustInternalValue(t, `0`)
	report := compiled.Validate(ctx, value)
	guard.Extensions = original
	if !errors.Is(report.Err(), context.DeadlineExceeded) || report.Valid() || len(report.Issues()) != 0 {
		t.Fatalf("owned deadline report = %v, valid=%t, issues=%v", report.Err(), report.Valid(), report.Issues())
	}
	if ctx.Err() != nil {
		t.Fatal("owned deadline failure canceled caller context")
	}
	if report = compiled.Validate(ctx, value); !report.Valid() {
		t.Fatalf("validator reuse after owned deadline failure = %v", report.Err())
	}
}

func TestValidationOwnerDependencyCheckpointPreservesCancellationAndReuse(t *testing.T) {
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(want.Error(), func(t *testing.T) {
			schema, err := Parse([]byte(`true`), jsonvalue.DefaultPolicy())
			if err != nil {
				t.Fatal(err)
			}
			compiled, err := Compile(schema, DefaultValidationOptions())
			if err != nil {
				t.Fatal(err)
			}
			stopped, cancel := context.WithCancel(context.Background())
			if want == context.DeadlineExceeded {
				cancel()
				stopped, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			} else {
				cancel()
			}
			defer cancel()
			ctx := &validationOwnerContext{Context: context.Background()}
			guard := compiled.compiled.AllOf[0]
			original := guard.Extensions
			guard.Extensions = append([]validator.SchemaExt{validationOwnerBeforeCheckpoint{stop: func() {
				ctx.Context = stopped
			}}}, original...)
			value := mustInternalValue(t, `0`)
			report := compiled.Validate(ctx, value)
			guard.Extensions = original
			if !errors.Is(report.Err(), want) || report.Valid() || len(report.Issues()) != 0 {
				t.Fatalf("dependency checkpoint cancellation = %v, valid=%t, issues=%v", report.Err(), report.Valid(), report.Issues())
			}
			if report = compiled.Validate(context.Background(), value); !report.Valid() {
				t.Fatalf("validator reuse after dependency cancellation = %v", report.Err())
			}
		})
	}
}
