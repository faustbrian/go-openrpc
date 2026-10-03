package jsonschema

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
	validator "github.com/santhosh-tekuri/jsonschema/v6"
)

type cancelOnSecondErrContext struct {
	context.Context
	checks   int
	cancelAt int
}

func (ctx *cancelOnSecondErrContext) Err() error {
	ctx.checks++
	if ctx.checks >= ctx.cancelAt {
		return context.Canceled
	}
	return nil
}

type observedErrContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (ctx *observedErrContext) Err() error {
	ctx.once.Do(func() { close(ctx.observed) })
	return ctx.Context.Err()
}

type expiredDeadlineContext struct{ context.Context }

func (expiredDeadlineContext) Deadline() (time.Time, bool) {
	return time.Now().Add(-time.Second), true
}

type cancelAfterValidation struct{ cancel context.CancelFunc }

func (extension cancelAfterValidation) Validate(*validator.ValidatorContext, any) {
	extension.cancel()
}

func TestECMARegexpAdapterPreservesPatternIdentity(t *testing.T) {
	t.Parallel()

	control := newValidationControl(time.Second)
	if err := control.begin(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	defer control.end()
	compiled, err := ecmaRegexpEngine(control)(`^\cc$`)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.String() != `^\cc$` || !compiled.MatchString("\x03") {
		t.Fatalf("compiled regexp = %q", compiled.String())
	}
	if _, err := ecmaRegexpEngine(newValidationControl(time.Second))(`[`); err == nil {
		t.Fatal("invalid pattern compiled")
	}
}

func TestValidationBoundaryRepanicsUnexpectedFailures(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered != "unexpected" {
			t.Fatalf("recovered panic = %#v", recovered)
		}
	}()
	_ = runValidation(func() error { panic("unexpected") })
}

func TestValidationControlCancellationAndLimitBranches(t *testing.T) {
	control := newValidationControl(time.Second)
	if err := control.begin(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := control.begin(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting begin error = %v", err)
	}
	control.end()
	if err := runValidation(func() error {
		control.checkpoint()
		return nil
	}); !errors.Is(err, ErrValidationResourceLimit) {
		t.Fatalf("inactive checkpoint error = %v", err)
	}

	for _, ctx := range []context.Context{
		&cancelOnSecondErrContext{Context: context.Background(), cancelAt: 2},
		expiredDeadlineContext{Context: context.Background()},
	} {
		control = newValidationControl(time.Second)
		if err := control.begin(ctx, 10); err != nil {
			t.Fatal(err)
		}
		expression, err := ecmaRegexpEngine(control)("a")
		if err != nil {
			t.Fatal(err)
		}
		validationErr := runValidation(func() error {
			expression.MatchString("a")
			return nil
		})
		control.end()
		if !errors.Is(validationErr, context.Canceled) && !errors.Is(validationErr, context.DeadlineExceeded) {
			t.Fatalf("regexp context error = %v", validationErr)
		}
	}
}

func TestAttachValidationCheckpointsTraversesEverySchemaEdge(t *testing.T) {
	control := newValidationControl(time.Second)
	children := make([]*validator.Schema, 0, 28)
	next := func() *validator.Schema {
		child := &validator.Schema{}
		children = append(children, child)
		return child
	}
	root := &validator.Schema{
		Ref: next(), RecursiveRef: next(),
		DynamicRef: &validator.DynamicRef{Ref: next()},
		Not:        next(), AllOf: []*validator.Schema{next()},
		AnyOf: []*validator.Schema{next()}, OneOf: []*validator.Schema{next()},
		If: next(), Then: next(), Else: next(), PropertyNames: next(),
		Properties:            map[string]*validator.Schema{"a": next()},
		PatternProperties:     map[validator.Regexp]*validator.Schema{ecmaRegexp{}: next()},
		AdditionalProperties:  next(),
		Dependencies:          map[string]any{"schema": next(), "names": []string{"a"}},
		DependentSchemas:      map[string]*validator.Schema{"a": next()},
		UnevaluatedProperties: next(), Contains: next(), Items: []*validator.Schema{next()},
		AdditionalItems: next(), PrefixItems: []*validator.Schema{next()},
		Items2020: next(), UnevaluatedItems: next(), ContentSchema: next(),
	}
	root.Ref.Ref = root
	if got, want := attachValidationCheckpoints(root, control), len(children)+1; got != want {
		t.Fatalf("schema node count = %d, want %d", got, want)
	}
	for index, schema := range append(children, root) {
		if len(schema.AllOf) != 2 || len(schema.AllOf[0].Extensions) != 1 {
			t.Fatalf("schema %d lacks its pre-evaluation checkpoint", index)
		}
	}
}

func TestValidationDecoderRejectsEveryMalformedBoundary(t *testing.T) {
	for _, input := range []string{
		`true false`, `{"`, `{"a":`, `{"a":1`, `{"a":1]`, `[true,`, `[true`, `[true}`,
	} {
		control := newValidationControl(time.Second)
		if err := control.begin(context.Background(), 100); err != nil {
			t.Fatal(err)
		}
		_, err := decodeValidationValue(control, []byte(input))
		control.end()
		if err == nil {
			t.Fatalf("decodeValidationValue(%q) succeeded", input)
		}
	}
}

func TestValidatorCoversQueuedCancellationAndPostValidationLimits(t *testing.T) {
	schema, err := Parse([]byte(`true`), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultValidationOptions()
	options.BaseURI = "relative"
	if _, err := Compile(schema, options); !errors.Is(err, ErrValidationPolicy) {
		t.Fatalf("relative base error = %v", err)
	}

	options = DefaultValidationOptions()
	compiled, err := Compile(schema, options)
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.control.begin(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	observed := &observedErrContext{Context: ctx, observed: make(chan struct{})}
	done := make(chan Report, 1)
	value := mustInternalValue(t, `true`)
	go func() { done <- compiled.Validate(observed, value) }()
	<-observed.observed
	cancel()
	report := <-done
	compiled.control.end()
	if !errors.Is(report.Err(), context.Canceled) {
		t.Fatalf("queued cancellation error = %v", report.Err())
	}

	compiled.schemaNodes = 2
	compiled.maxValidationSteps = 1
	if report = compiled.Validate(context.Background(), mustInternalValue(t, `true`)); !errors.Is(report.Err(), ErrValidationResourceLimit) {
		t.Fatalf("schema node bound error = %v", report.Err())
	}

	compiled.schemaNodes = 1
	compiled.maxValidationSteps = 10
	finalContext, cancelFinal := context.WithCancel(context.Background())
	defer cancelFinal()
	// Outer extensions run after the guarded original schema completes. This
	// cancels at completion rather than depending on checkpoint call counts.
	compiled.compiled.Extensions = []validator.SchemaExt{cancelAfterValidation{cancel: cancelFinal}}
	if report = compiled.Validate(finalContext, mustInternalValue(t, `true`)); !errors.Is(report.Err(), context.Canceled) {
		t.Fatalf("post-validation cancellation error = %v", report.Err())
	}
	if report = compiled.Validate(context.Background(), mustInternalValue(t, `true`)); !report.Valid() {
		t.Fatalf("validator was not reusable after final cancellation: %v", report.Err())
	}

	falseSchema, err := Parse([]byte(`false`), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	options = DefaultValidationOptions()
	options.MaxValidationSteps = 3
	compiled, err = Compile(falseSchema, options)
	if err != nil {
		t.Fatal(err)
	}
	if report = compiled.Validate(context.Background(), mustInternalValue(t, `true`)); !errors.Is(report.Err(), ErrValidationResourceLimit) {
		t.Fatalf("diagnostic traversal bound error = %v", report.Err())
	}
}

func mustInternalValue(t *testing.T, input string) jsonvalue.Value {
	t.Helper()
	value, err := jsonvalue.Parse([]byte(input), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return value
}
