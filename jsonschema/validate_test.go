package jsonschema_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/faustbrian/go-openrpc/v2/jsonschema"
	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
)

func TestValidatorAppliesDraft7WithoutNumericCoercion(t *testing.T) {
	t.Parallel()

	schema := parseSchema(t, `{
		"$schema":"http://json-schema.org/draft-07/schema#",
		"type":"object",
		"required":["name","count"],
		"properties":{
			"name":{"type":"string","minLength":3},
			"count":{"type":"integer","minimum":9007199254740993}
		}
	}`)
	validator, err := jsonschema.Compile(schema, jsonschema.DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	valid := parseValue(t, `{"name":"valid","count":9007199254740993}`)
	if report := validator.Validate(context.Background(), valid); !report.Valid() {
		t.Fatalf("valid report = %#v", report.Issues())
	}
	invalid := parseValue(t, `{"name":"x","count":9007199254740992}`)
	report := validator.Validate(context.Background(), invalid)
	if report.Valid() || len(report.Issues()) != 2 {
		t.Fatalf("invalid report = %#v", report.Issues())
	}
	for _, issue := range report.Issues() {
		if issue.InstancePointer == "" || issue.SchemaPointer == "" || issue.Keyword == "" || issue.Message == "" {
			t.Fatalf("incomplete issue = %#v", issue)
		}
	}
	if report.Issues()[0].InstancePointer != "#/count" ||
		report.Issues()[1].InstancePointer != "#/name" {
		t.Fatalf("issues are not deterministic: %#v", report.Issues())
	}
	returned := report.Issues()
	returned[0].Keyword = "changed"
	if report.Issues()[0].Keyword == "changed" {
		t.Fatal("Issues exposed mutable report storage")
	}
}

func TestValidatorSupportsBooleanSchemas(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		schema string
		valid  bool
	}{
		{schema: "true", valid: true},
		{schema: "false", valid: false},
	} {
		validator, err := jsonschema.Compile(parseSchema(t, test.schema), jsonschema.DefaultValidationOptions())
		if err != nil {
			t.Fatal(err)
		}
		if got := validator.Validate(context.Background(), parseValue(t, `null`)).Valid(); got != test.valid {
			t.Errorf("schema %s valid = %t", test.schema, got)
		}
	}
}

func TestCompileRejectsInvalidOrNonDraft7Schemas(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		`{"maxLength":"not-an-integer"}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema"}`,
		`{"$ref":"https://example.com/external.json"}`,
	} {
		_, err := jsonschema.Compile(parseSchema(t, source), jsonschema.DefaultValidationOptions())
		if !errors.Is(err, jsonschema.ErrSchemaCompile) {
			t.Errorf("Compile(%s) error = %v", source, err)
		}
	}
}

func TestCompileEnforcesExactOptionBoundaries(t *testing.T) {
	t.Parallel()

	for _, mutate := range []func(*jsonschema.ValidationOptions){
		func(options *jsonschema.ValidationOptions) { options.MaxResources = 0 },
		func(options *jsonschema.ValidationOptions) { options.MaxSchemaBytes = 0 },
		func(options *jsonschema.ValidationOptions) { options.MaxInstanceBytes = 0 },
		func(options *jsonschema.ValidationOptions) { options.MaxValidationSteps = 0 },
		func(options *jsonschema.ValidationOptions) { options.MaxIssues = 0 },
		func(options *jsonschema.ValidationOptions) { options.RegexpTimeout = 0 },
		func(options *jsonschema.ValidationOptions) { options.RegexpTimeout = 10*time.Second + 1 },
	} {
		options := jsonschema.DefaultValidationOptions()
		mutate(&options)
		if _, err := jsonschema.Compile(parseSchema(t, `true`), options); !errors.Is(err, jsonschema.ErrValidationPolicy) {
			t.Fatalf("Compile options %#v error = %v", options, err)
		}
	}
	options := jsonschema.DefaultValidationOptions()
	options.RegexpTimeout = 10 * time.Second
	if _, err := jsonschema.Compile(parseSchema(t, `true`), options); err != nil {
		t.Fatalf("exact timeout boundary error = %v", err)
	}
}

type validationCancelContext struct {
	context.Context
	checks    atomic.Int64
	cancelAt  int64
	done      chan struct{}
	closeOnce sync.Once
}

func (ctx *validationCancelContext) Done() <-chan struct{} { return ctx.done }

func (ctx *validationCancelContext) Err() error {
	if ctx.checks.Add(1) < ctx.cancelAt {
		return nil
	}
	ctx.closeOnce.Do(func() { close(ctx.done) })
	return context.Canceled
}

func TestValidatorCancellationInterruptsAggregateSchemaWork(t *testing.T) {
	t.Parallel()

	options := jsonschema.DefaultValidationOptions()
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"allOf":[{"minimum":1},{"minimum":2},{"minimum":3},{"minimum":4},{"minimum":5},{"minimum":6},{"minimum":7},{"minimum":8}]}`),
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &validationCancelContext{Context: context.Background(), cancelAt: 5, done: make(chan struct{})}
	if report := validator.Validate(ctx, parseValue(t, `0`)); !errors.Is(report.Err(), context.Canceled) {
		t.Fatalf("aggregate cancellation error = %v", report.Err())
	}
}

func TestValidatorBoundsAggregateSchemaEvaluations(t *testing.T) {
	t.Parallel()

	options := jsonschema.DefaultValidationOptions()
	options.MaxValidationSteps = 16
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"items":{"allOf":[{"minimum":0},{"minimum":1}]}}`),
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report := validator.Validate(context.Background(), parseValue(t, `[0,1,2,3,4]`)); !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) {
		t.Fatalf("aggregate work error = %v", report.Err())
	}
}

func TestValidatorBoundsInstanceTraversalBeforeDependencyValidation(t *testing.T) {
	t.Parallel()

	options := jsonschema.DefaultValidationOptions()
	options.MaxValidationSteps = 4
	validator, err := jsonschema.Compile(parseSchema(t, `true`), options)
	if err != nil {
		t.Fatal(err)
	}
	if report := validator.Validate(context.Background(), parseValue(t, `[0,1,2,3]`)); !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) {
		t.Fatalf("instance traversal error = %v", report.Err())
	}
}

func TestValidatorReportsRegexpTimeoutAsResourceLimit(t *testing.T) {
	t.Parallel()

	options := jsonschema.DefaultValidationOptions()
	options.RegexpTimeout = time.Millisecond
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"pattern":"^(a+)+$"}`), options,
	)
	if err != nil {
		t.Fatal(err)
	}
	instance := parseValue(t, `"`+strings.Repeat("a", 4_096)+`!"`)
	report := validator.Validate(context.Background(), instance)
	if !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) {
		t.Fatalf("regexp timeout error = %v, issues = %#v", report.Err(), report.Issues())
	}
	if report.Valid() || len(report.Issues()) != 0 {
		t.Fatalf("regexp timeout report = %#v", report)
	}
}

func TestValidatorCancellationBoundsRegexpValidation(t *testing.T) {
	options := jsonschema.DefaultValidationOptions()
	options.RegexpTimeout = time.Second
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"pattern":"^(a+)+$"}`), options,
	)
	if err != nil {
		t.Fatal(err)
	}
	instance := parseValue(t, `"`+strings.Repeat("a", 4_096)+`!"`)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	report := validator.Validate(ctx, instance)
	if !errors.Is(report.Err(), context.DeadlineExceeded) {
		t.Fatalf("validation error = %v, want deadline exceeded", report.Err())
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("canceled validation returned after %s", elapsed)
	}
}

func TestValidatorRejectsInstancesAboveDefaultByteBudget(t *testing.T) {
	t.Parallel()

	validator, err := jsonschema.Compile(
		parseSchema(t, `true`), jsonschema.DefaultValidationOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	policy := jsonvalue.Policy{MaxBytes: (16 << 20) + 2, MaxDepth: 2, MaxTokens: 2}
	instance, err := jsonvalue.Parse([]byte(`"`+strings.Repeat("a", 16<<20)+`"`), policy)
	if err != nil {
		t.Fatal(err)
	}
	if report := validator.Validate(context.Background(), instance); !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) {
		t.Fatalf("oversized instance error = %v", report.Err())
	}
}

func TestValidationByteLimitsRejectBeforeCopying(t *testing.T) {
	instance := parseValue(t, `"oversized"`)
	options := jsonschema.DefaultValidationOptions()
	options.MaxInstanceBytes = instance.ByteLen() - 1
	validator, err := jsonschema.Compile(parseSchema(t, `true`), options)
	if err != nil {
		t.Fatal(err)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if report := validator.Validate(context.Background(), instance); !errors.Is(report.Err(), jsonschema.ErrValidationResourceLimit) {
			t.Fatalf("oversized instance error = %v", report.Err())
		}
	}); allocations != 0 {
		t.Fatalf("oversized validation allocations = %f", allocations)
	}

	schema := parseSchema(t, `{"type":"string"}`)
	options = jsonschema.DefaultValidationOptions()
	options.MaxSchemaBytes = schema.ByteLen() - 1
	if allocations := testing.AllocsPerRun(100, func() {
		if _, compileErr := jsonschema.Compile(schema, options); !errors.Is(compileErr, jsonschema.ErrValidationPolicy) {
			t.Fatalf("oversized schema error = %v", compileErr)
		}
	}); allocations != 0 {
		t.Fatalf("oversized compilation allocations = %f", allocations)
	}
}

func TestCompileUsesOnlyExplicitExternalResources(t *testing.T) {
	t.Parallel()

	options := jsonschema.DefaultValidationOptions()
	options.Resources = map[string]jsonschema.Schema{
		"https://example.com/positive.json": parseSchema(t, `{"type":"integer","minimum":1}`),
	}
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"$ref":"https://example.com/positive.json"}`),
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	if validator.Validate(context.Background(), parseValue(t, `0`)).Valid() {
		t.Fatal("explicit external schema was not applied")
	}
	options.Resources["https://example.com/positive.json"] = parseSchema(t, `true`)
	if validator.Validate(context.Background(), parseValue(t, `0`)).Valid() {
		t.Fatal("compiler retained caller-owned resource map")
	}
}

func TestCompileBoundsExplicitSchemaResources(t *testing.T) {
	t.Parallel()

	root := parseSchema(t, `true`)
	resource := parseSchema(t, `{"type":"string"}`)
	rootOptions := jsonschema.DefaultValidationOptions()
	rootOptions.MaxSchemaBytes = len(root.Bytes())
	if _, err := jsonschema.Compile(root, rootOptions); err != nil {
		t.Fatalf("exact root schema byte limit failed: %v", err)
	}
	rootOptions.MaxSchemaBytes--
	if _, err := jsonschema.Compile(root, rootOptions); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("root schema byte error = %v", err)
	}

	options := jsonschema.DefaultValidationOptions()
	options.MaxResources = 1
	options.Resources = map[string]jsonschema.Schema{
		"https://example.com/one.json": resource,
	}
	options.MaxSchemaBytes = len(root.Bytes()) + len(resource.Bytes())
	if _, err := jsonschema.Compile(root, options); err != nil {
		t.Fatalf("exact resource limits failed: %v", err)
	}

	options.Resources["https://example.com/two.json"] = resource
	if _, err := jsonschema.Compile(root, options); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("resource count error = %v", err)
	}
	delete(options.Resources, "https://example.com/two.json")
	options.MaxSchemaBytes--
	if _, err := jsonschema.Compile(root, options); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("aggregate schema byte error = %v", err)
	}

	options.MaxResources = 2
	options.Resources["https://example.com/two.json"] = resource
	options.MaxSchemaBytes = len(root.Bytes()) + 2*len(resource.Bytes()) - 1
	if _, err := jsonschema.Compile(root, options); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("multi-resource aggregate schema byte error = %v", err)
	}
}

func TestCompileRejectsInvalidResourcePolicies(t *testing.T) {
	t.Parallel()

	tests := []jsonschema.ValidationOptions{
		func() jsonschema.ValidationOptions {
			options := jsonschema.DefaultValidationOptions()
			options.Resources = map[string]jsonschema.Schema{"relative.json": parseSchema(t, `true`)}
			return options
		}(),
		func() jsonschema.ValidationOptions {
			options := jsonschema.DefaultValidationOptions()
			options.Resources = map[string]jsonschema.Schema{
				"https://example.com/future.json": parseSchema(t, `{"$schema":"https://json-schema.org/draft/2020-12/schema"}`),
			}
			return options
		}(),
	}
	for _, options := range tests {
		if _, err := jsonschema.Compile(parseSchema(t, `true`), options); err == nil {
			t.Fatalf("Compile options %#v succeeded", options)
		}
	}
	duplicate := jsonschema.DefaultValidationOptions()
	duplicate.BaseURI = "https://example.com/schema.json"
	duplicate.Resources = map[string]jsonschema.Schema{duplicate.BaseURI: parseSchema(t, `true`)}
	if _, err := jsonschema.Compile(parseSchema(t, `true`), duplicate); !errors.Is(err, jsonschema.ErrSchemaCompile) {
		t.Fatalf("duplicate base error = %v", err)
	}
}

func TestValidatorUsesBoundedECMAScriptRegularExpressions(t *testing.T) {
	t.Parallel()

	schema := parseSchema(t, `{
		"$schema":"http://json-schema.org/draft-07/schema#",
		"type":"string",
		"pattern":"^\\cc$"
	}`)
	compiled, err := jsonschema.Compile(schema, jsonschema.DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	instance, err := jsonvalue.Parse([]byte(`"\u0003"`), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if report := compiled.Validate(context.Background(), instance); !report.Valid() {
		t.Fatalf("issues = %#v, error = %v", report.Issues(), report.Err())
	}
}

func TestValidationBoundsAndCancellation(t *testing.T) {
	t.Parallel()

	if _, err := jsonschema.Compile(parseSchema(t, `true`), jsonschema.ValidationOptions{}); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("policy error = %v", err)
	}
	options := jsonschema.DefaultValidationOptions()
	options.MaxIssues = 1
	validator, err := jsonschema.Compile(
		parseSchema(t, `{"type":"array","items":{"type":"string"}}`),
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	report := validator.Validate(context.Background(), parseValue(t, `[1,2,3]`))
	if len(report.Issues()) != 1 || !report.Truncated() {
		t.Fatalf("bounded report = %#v, truncated = %t", report.Issues(), report.Truncated())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report = validator.Validate(ctx, parseValue(t, `[]`))
	if !errors.Is(report.Err(), context.Canceled) {
		t.Fatalf("canceled report error = %v", report.Err())
	}
	bounded, err := validator.WithMaxIssues(3)
	if err != nil {
		t.Fatal(err)
	}
	if bounded.Validate(context.Background(), parseValue(t, `[1,2,3]`)).Truncated() == true {
		t.Fatal("WithMaxIssues did not apply the replacement bound")
	}
	if _, err := validator.WithMaxIssues(0); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("WithMaxIssues error = %v", err)
	}
}

func TestValidateRejectsInvalidExecutionInputs(t *testing.T) {
	t.Parallel()

	var zero jsonschema.Validator
	if report := zero.Validate(context.Background(), parseValue(t, `null`)); !errors.Is(report.Err(), jsonschema.ErrValidationPolicy) {
		t.Fatalf("zero validator error = %v", report.Err())
	}
	if _, err := zero.WithMaxIssues(1); !errors.Is(err, jsonschema.ErrValidationPolicy) {
		t.Fatalf("zero WithMaxIssues error = %v", err)
	}
	compiled, err := jsonschema.Compile(parseSchema(t, `true`), jsonschema.DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	var invalidContext context.Context
	if report := compiled.Validate(invalidContext, parseValue(t, `null`)); !errors.Is(report.Err(), jsonschema.ErrValidationPolicy) {
		t.Fatalf("nil context error = %v", report.Err())
	}
	if report := compiled.Validate(context.Background(), jsonvalue.Value{}); !errors.Is(report.Err(), jsonschema.ErrInvalidInstance) {
		t.Fatalf("zero instance error = %v", report.Err())
	}
	ctx := &secondCheckCanceledContext{}
	if report := compiled.Validate(ctx, parseValue(t, `null`)); !errors.Is(report.Err(), context.Canceled) {
		t.Fatalf("post-validation context error = %v", report.Err())
	}
}

type secondCheckCanceledContext struct{ checks int }

func (ctx *secondCheckCanceledContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (ctx *secondCheckCanceledContext) Done() <-chan struct{}       { return nil }
func (ctx *secondCheckCanceledContext) Value(any) any               { return nil }
func (ctx *secondCheckCanceledContext) Err() error {
	ctx.checks++
	if ctx.checks > 1 {
		return context.Canceled
	}
	return nil
}

func parseSchema(t *testing.T, input string) jsonschema.Schema {
	t.Helper()
	schema, err := jsonschema.Parse([]byte(input), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func parseValue(t *testing.T, input string) jsonvalue.Value {
	t.Helper()
	value, err := jsonvalue.Parse([]byte(input), jsonvalue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestValidatorDependencyEqualityCorrections(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, schema, instance, keyword string
		valid                           bool
	}{
		{"const mixed types", `{"const":"1"}`, `1`, "const", false},
		{"const same string", `{"const":"1"}`, `"1"`, "", true},
		{"nested enum mixed types", `{"enum":[{"x":"1"}]}`, `{"x":1}`, "enum", false},
		{"nested enum same types", `{"enum":[{"x":"1"}]}`, `{"x":"1"}`, "", true},
		{"numeric equivalence", `{"const":1}`, `1.0`, "", true},
		{"exact integer match", `{"const":9007199254740993}`, `9007199254740993`, "", true},
		{"exact integer mismatch", `{"const":9007199254740993}`, `9007199254740992`, "const", false},
		{"unique mixed types", `{"uniqueItems":true}`, `["1",1]`, "", true},
		{"duplicate strings", `{"uniqueItems":true}`, `["1","1"]`, "uniqueItems", false},
		{"duplicate numeric values", `{"uniqueItems":true}`, `[1,1.0]`, "uniqueItems", false},
		{"unique exact integers", `{"uniqueItems":true}`, `[9007199254740992,9007199254740993]`, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			validator, err := jsonschema.Compile(parseSchema(t, test.schema), jsonschema.DefaultValidationOptions())
			if err != nil {
				t.Fatal(err)
			}
			report := validator.Validate(context.Background(), parseValue(t, test.instance))
			if report.Err() != nil || report.Valid() != test.valid {
				t.Fatalf("valid = %t, err = %v, issues = %#v", report.Valid(), report.Err(), report.Issues())
			}
			if test.valid {
				if len(report.Issues()) != 0 {
					t.Fatalf("valid issues = %#v", report.Issues())
				}
				return
			}
			issues := report.Issues()
			if len(issues) != 1 || issues[0].Keyword != test.keyword {
				t.Fatalf("issues = %#v", issues)
			}
		})
	}
}

func TestValidatorDependencyQuotedEmailCorrection(t *testing.T) {
	t.Parallel()
	validator, err := jsonschema.Compile(parseSchema(t, `{"format":"email"}`), jsonschema.DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, instance string
		valid          bool
	}{
		{"ordinary", `"user@example.com"`, true},
		{"closed quoted", `"\"user\"@example.com"`, true},
		{"unclosed quoted", `"\"user@example.com"`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := validator.Validate(context.Background(), parseValue(t, test.instance))
			if report.Err() != nil || report.Valid() != test.valid {
				t.Fatalf("valid = %t, err = %v, issues = %#v", report.Valid(), report.Err(), report.Issues())
			}
			if !test.valid {
				issues := report.Issues()
				if len(issues) != 1 || issues[0].Keyword != "format" || issues[0].Message != "value does not satisfy the schema keyword" {
					t.Fatalf("issues = %#v", issues)
				}
			}
		})
	}
}

func TestValidatorDependencyAdditionalItemsOffsets(t *testing.T) {
	t.Parallel()
	validator, err := jsonschema.Compile(parseSchema(t, `{"items":[{"type":"integer"}],"additionalItems":{"type":"boolean"}}`), jsonschema.DefaultValidationOptions())
	if err != nil {
		t.Fatal(err)
	}
	value := parseValue(t, `[1,"secret",false,7]`)
	report := validator.Validate(context.Background(), value)
	issues := report.Issues()
	if report.Err() != nil || report.Valid() || report.Truncated() || len(issues) != 2 {
		t.Fatalf("report = %#v, err = %v", issues, report.Err())
	}
	for index, pointer := range []string{"#/1", "#/3"} {
		issue := issues[index]
		if issue.InstancePointer != pointer || issue.SchemaPointer != "#/type" || issue.Keyword != "type" || issue.Message != "value does not satisfy the schema keyword" {
			t.Fatalf("issue = %#v, want pointer %s", issue, pointer)
		}
	}
	issues[0].InstancePointer = "changed"
	if report.Issues()[0].InstancePointer != "#/1" {
		t.Fatal("Issues exposes report storage")
	}
	bounded, err := validator.WithMaxIssues(1)
	if err != nil {
		t.Fatal(err)
	}
	capped := bounded.Validate(context.Background(), value)
	if capped.Err() != nil || capped.Valid() || !capped.Truncated() || len(capped.Issues()) != 1 || capped.Issues()[0].InstancePointer != "#/1" {
		t.Fatalf("capped = %#v, err = %v", capped.Issues(), capped.Err())
	}
}
