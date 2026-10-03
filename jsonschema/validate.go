package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/faustbrian/go-openrpc/v2/jsonvalue"
	validator "github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	// ErrValidationPolicy reports invalid compilation or diagnostic bounds.
	ErrValidationPolicy = errors.New("jsonschema: invalid validation policy")
	// ErrSchemaCompile reports a schema that is not valid Draft 7 or contains
	// a reference absent from the explicit resource set.
	ErrSchemaCompile = errors.New("jsonschema: Draft 7 compilation failed")
	// ErrInvalidInstance reports an invalid zero instance value.
	ErrInvalidInstance = errors.New("jsonschema: invalid instance")
	// ErrValidationResourceLimit reports that bounded validation could not
	// complete without exceeding its configured work limit.
	ErrValidationResourceLimit = errors.New("jsonschema: validation resource limit exceeded")
)

const defaultBaseURI = "https://openrpc.invalid/schema.json"

// ValidationOptions configures Draft 7 compilation and bounded reporting.
// Resources are the only external schemas available during compilation; no
// filesystem or network loader is installed.
type ValidationOptions struct {
	BaseURI            string
	Resources          map[string]Schema
	MaxResources       int
	MaxSchemaBytes     int
	MaxInstanceBytes   int
	MaxValidationSteps int
	MaxIssues          int
	RegexpTimeout      time.Duration
}

// DefaultValidationOptions returns strict Draft 7 compilation with finite
// diagnostic output and no external resources.
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		BaseURI: defaultBaseURI, MaxResources: 1_024,
		MaxSchemaBytes: 64 << 20, MaxInstanceBytes: 16 << 20,
		MaxValidationSteps: 1_000_000, MaxIssues: 1_000,
		RegexpTimeout: 100 * time.Millisecond,
	}
}

// Validator is an immutable compiled Draft 7 schema safe for concurrent use.
type Validator struct {
	compiled           *validator.Schema
	maxIssues          int
	maxInstanceBytes   int
	maxValidationSteps int
	schemaNodes        int
	control            *validationControl
}

// WithMaxIssues returns a validator sharing the immutable compiled schema with
// a replacement diagnostic bound.
func (compiled Validator) WithMaxIssues(maxIssues int) (Validator, error) {
	if compiled.compiled == nil || maxIssues <= 0 {
		return Validator{}, ErrValidationPolicy
	}
	return Validator{
		compiled: compiled.compiled, maxIssues: maxIssues,
		maxInstanceBytes:   compiled.maxInstanceBytes,
		maxValidationSteps: compiled.maxValidationSteps, control: compiled.control,
		schemaNodes: compiled.schemaNodes,
	}, nil
}

// Compile compiles one Draft 7 schema using only explicitly supplied
// resources. It never performs network or filesystem access.
func Compile(schema Schema, options ValidationOptions) (Validator, error) {
	if options.MaxResources <= 0 || options.MaxSchemaBytes <= 0 ||
		options.MaxInstanceBytes <= 0 || options.MaxValidationSteps <= 0 ||
		options.MaxIssues <= 0 ||
		options.RegexpTimeout <= 0 || options.RegexpTimeout > 10*time.Second {
		return Validator{}, ErrValidationPolicy
	}
	if schema.ByteLen() > options.MaxSchemaBytes {
		return Validator{}, ErrValidationPolicy
	}
	if !absoluteURI(options.BaseURI) {
		return Validator{}, ErrValidationPolicy
	}
	if len(options.Resources) > options.MaxResources {
		return Validator{}, ErrValidationPolicy
	}
	schemaBytes := schema.Bytes()
	if !declaresDraft7(schemaBytes) {
		return Validator{}, ErrSchemaCompile
	}
	// Schema guarantees syntactically valid object or boolean JSON.
	document, _ := validator.UnmarshalJSON(bytes.NewReader(schemaBytes))
	compiler := validator.NewCompiler()
	compiler.DefaultDraft(validator.Draft7)
	compiler.AssertFormat()
	control := newValidationControl(options.RegexpTimeout)
	compiler.UseRegexpEngine(ecmaRegexpEngine(control))

	resourceNames := make([]string, 0, len(options.Resources))
	for resourceURI := range options.Resources {
		resourceNames = append(resourceNames, resourceURI)
	}
	sort.Strings(resourceNames)
	totalSchemaBytes := len(schemaBytes)
	for _, resourceURI := range resourceNames {
		if !absoluteURI(resourceURI) {
			return Validator{}, ErrValidationPolicy
		}
		resource := options.Resources[resourceURI]
		if resource.ByteLen() > options.MaxSchemaBytes-totalSchemaBytes {
			return Validator{}, ErrValidationPolicy
		}
		resourceBytes := resource.Bytes()
		totalSchemaBytes += len(resourceBytes)
		if !declaresDraft7(resourceBytes) {
			return Validator{}, ErrSchemaCompile
		}
		// Resource schemas share the same syntax invariant, and resource names
		// are unique absolute map keys.
		decoded, _ := validator.UnmarshalJSON(bytes.NewReader(resourceBytes))
		_ = compiler.AddResource(resourceURI, decoded)
	}
	if err := compiler.AddResource(options.BaseURI, document); err != nil {
		return Validator{}, ErrSchemaCompile
	}
	compiled, err := compiler.Compile(options.BaseURI)
	if err != nil || compiled.DraftVersion != 7 {
		return Validator{}, ErrSchemaCompile
	}
	schemaNodes := attachValidationCheckpoints(compiled, control)
	return Validator{
		compiled: compiled, maxIssues: options.MaxIssues,
		maxInstanceBytes:   options.MaxInstanceBytes,
		maxValidationSteps: options.MaxValidationSteps, control: control,
		schemaNodes: schemaNodes,
	}, nil
}

type ecmaRegexp struct {
	compiled *regexp2.Regexp
	control  *validationControl
}

type regexpTimeout struct{}

type validationCanceled struct{ err error }

type validationLimit struct{}

type validationControl struct {
	slot           chan struct{}
	mu             sync.RWMutex
	ctx            context.Context
	regexpTimeout  time.Duration
	remainingSteps int
}

func newValidationControl(regexpTimeout time.Duration) *validationControl {
	control := &validationControl{slot: make(chan struct{}, 1), regexpTimeout: regexpTimeout}
	control.slot <- struct{}{}
	return control
}

func (control *validationControl) begin(ctx context.Context, maxSteps int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-control.slot:
	}
	control.mu.Lock()
	control.ctx = ctx
	control.remainingSteps = maxSteps
	control.mu.Unlock()
	return nil
}

func (control *validationControl) checkpoint() {
	control.mu.Lock()
	ctx := control.ctx
	exhausted := control.remainingSteps <= 0
	if !exhausted {
		control.remainingSteps--
	}
	control.mu.Unlock()
	if ctx == nil {
		panic(validationLimit{})
	}
	if err := ctx.Err(); err != nil {
		panic(validationCanceled{err: err})
	}
	if exhausted {
		panic(validationLimit{})
	}
}

func (control *validationControl) end() {
	control.mu.Lock()
	control.ctx = nil
	control.mu.Unlock()
	control.slot <- struct{}{}
}

func (control *validationControl) operation() (context.Context, time.Duration) {
	control.mu.RLock()
	ctx := control.ctx
	timeout := control.regexpTimeout
	control.mu.RUnlock()
	return ctx, timeout
}

func (expression ecmaRegexp) MatchString(input string) bool {
	expression.control.checkpoint()
	ctx, timeout := expression.control.operation()
	if err := ctx.Err(); err != nil {
		panic(validationCanceled{err: err})
	}
	if deadline, ok := ctx.Deadline(); ok {
		deadlineTimeout, err := regexpDeadlineTimeout(time.Until(deadline), timeout)
		if err != nil {
			panic(validationCanceled{err: err})
		}
		timeout = deadlineTimeout
	}
	expression.compiled.MatchTimeout = timeout
	matched, err := expression.compiled.MatchString(input)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			panic(validationCanceled{err: contextErr})
		}
		panic(regexpTimeout{})
	}
	return matched
}

func regexpDeadlineTimeout(remaining, configured time.Duration) (time.Duration, error) {
	if remaining <= 0 {
		return 0, context.DeadlineExceeded
	}
	return min(remaining, configured), nil
}

type validationCheckpoint struct{ control *validationControl }

func (checkpoint validationCheckpoint) Validate(*validator.ValidatorContext, any) {
	checkpoint.control.checkpoint()
}

func attachValidationCheckpoints(root *validator.Schema, control *validationControl) int {
	seen := make(map[*validator.Schema]struct{})
	var visit func(*validator.Schema)
	visit = func(schema *validator.Schema) {
		if schema == nil {
			return
		}
		if _, exists := seen[schema]; exists {
			return
		}
		seen[schema] = struct{}{}
		visit(schema.Ref)
		visit(schema.RecursiveRef)
		if schema.DynamicRef != nil {
			visit(schema.DynamicRef.Ref)
		}
		visit(schema.Not)
		for _, child := range schema.AllOf {
			visit(child)
		}
		for _, child := range schema.AnyOf {
			visit(child)
		}
		for _, child := range schema.OneOf {
			visit(child)
		}
		visit(schema.If)
		visit(schema.Then)
		visit(schema.Else)
		visit(schema.PropertyNames)
		for _, child := range schema.Properties {
			visit(child)
		}
		for _, child := range schema.PatternProperties {
			visit(child)
		}
		if child, ok := schema.AdditionalProperties.(*validator.Schema); ok {
			visit(child)
		}
		for _, dependency := range schema.Dependencies {
			if child, ok := dependency.(*validator.Schema); ok {
				visit(child)
			}
		}
		for _, child := range schema.DependentSchemas {
			visit(child)
		}
		visit(schema.UnevaluatedProperties)
		visit(schema.Contains)
		if child, ok := schema.Items.(*validator.Schema); ok {
			visit(child)
		}
		if children, ok := schema.Items.([]*validator.Schema); ok {
			for _, child := range children {
				visit(child)
			}
		}
		if child, ok := schema.AdditionalItems.(*validator.Schema); ok {
			visit(child)
		}
		for _, child := range schema.PrefixItems {
			visit(child)
		}
		visit(schema.Items2020)
		visit(schema.UnevaluatedItems)
		visit(schema.ContentSchema)

		// Draft 7 validation can return before its extensions run. Preserve the
		// original node as the second allOf member so the first member checks
		// the caller's budget before any keyword or reference is evaluated.
		// Keep the original pointer as the wrapper: every existing graph edge,
		// including recursive references, must pass through the same guard.
		body := *schema
		checkpoint := &validator.Schema{
			DraftVersion: body.DraftVersion,
			Location:     body.Location,
			Extensions:   []validator.SchemaExt{validationCheckpoint{control: control}},
		}
		*schema = validator.Schema{
			DraftVersion: body.DraftVersion,
			Location:     body.Location,
			AllOf:        []*validator.Schema{checkpoint, &body},
		}
	}
	visit(root)
	return len(seen)
}

func (expression ecmaRegexp) String() string { return expression.compiled.String() }

func ecmaRegexpEngine(control *validationControl) validator.RegexpEngine {
	return func(pattern string) (validator.Regexp, error) {
		compiled, err := regexp2.Compile(pattern, regexp2.ECMAScript)
		if err != nil {
			return nil, err
		}
		compiled.MatchTimeout = control.regexpTimeout
		return ecmaRegexp{compiled: compiled, control: control}, nil
	}
}

// Issue is one safe validation failure. Messages contain no instance value.
type Issue struct {
	InstancePointer string
	SchemaPointer   string
	Keyword         string
	Message         string
}

// Report is one immutable bounded validation result.
type Report struct {
	issues    []Issue
	truncated bool
	err       error
}

// Issues returns an owned diagnostic snapshot.
func (report Report) Issues() []Issue { return append([]Issue(nil), report.issues...) }

// Truncated reports that more failures existed than the configured bound.
func (report Report) Truncated() bool { return report.truncated }

// Err reports cancellation, invalid policy, or an invalid instance.
func (report Report) Err() error { return report.err }

// Valid reports successful validation and no execution error.
func (report Report) Valid() bool { return report.err == nil && len(report.issues) == 0 }

// Validate checks one immutable JSON value and converts dependency diagnostics
// into stable, payload-free package-owned issues.
func (compiled Validator) Validate(ctx context.Context, instance jsonvalue.Value) Report {
	if compiled.compiled == nil || compiled.maxIssues <= 0 ||
		compiled.control == nil || ctx == nil {
		return Report{err: ErrValidationPolicy}
	}
	if err := ctx.Err(); err != nil {
		return Report{err: err}
	}
	if instance.ByteLen() > compiled.maxInstanceBytes {
		return Report{err: ErrValidationResourceLimit}
	}
	instanceBytes := instance.Bytes()
	if err := compiled.control.begin(ctx, compiled.maxValidationSteps); err != nil {
		return Report{err: err}
	}
	defer compiled.control.end()
	if compiled.schemaNodes > compiled.maxValidationSteps {
		return Report{err: ErrValidationResourceLimit}
	}
	var value any
	err := runValidation(func() error {
		var decodeErr error
		value, decodeErr = decodeValidationValue(compiled.control, instanceBytes)
		return decodeErr
	})
	if errors.Is(err, ErrValidationResourceLimit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return Report{err: err}
	}
	if err != nil {
		return Report{err: ErrInvalidInstance}
	}
	err = runValidation(func() error {
		return compiled.compiled.Validate(value)
	})
	if err == nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return Report{err: contextErr}
		}
		return Report{}
	}
	if errors.Is(err, ErrValidationResourceLimit) {
		return Report{err: err}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return Report{err: err}
	}
	// A compiled schema returns only nil or *ValidationError for a decoded
	// instance; decoding failures were handled above.
	var validationError *validator.ValidationError
	_ = errors.As(err, &validationError)
	issues := make([]Issue, 0, min(compiled.maxIssues, 8))
	total := 0
	err = runValidation(func() error {
		sortValidationErrors(validationError, compiled.control)
		total = collectIssues(validationError, compiled.maxIssues+1, &issues)
		return nil
	})
	if err != nil {
		return Report{err: err}
	}
	truncated := total > compiled.maxIssues
	if truncated {
		issues = issues[:compiled.maxIssues]
	}
	return Report{issues: issues, truncated: truncated}
}

func decodeValidationValue(control *validationControl, input []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	value, err := decodeValidationToken(control, decoder)
	if err != nil {
		return nil, err
	}
	control.checkpoint()
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidInstance
	}
	return value, nil
}

func decodeValidationToken(control *validationControl, decoder *json.Decoder) (any, error) {
	control.checkpoint()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return token, nil
	}
	if delimiter == '{' {
		object := make(map[string]any)
		for decoder.More() {
			control.checkpoint()
			name, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			value, err := decodeValidationToken(control, decoder)
			if err != nil {
				return nil, err
			}
			object[name.(string)] = value
		}
		control.checkpoint()
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	}
	// At a value position encoding/json only returns an opening object or
	// array delimiter.
	array := make([]any, 0)
	for decoder.More() {
		value, err := decodeValidationToken(control, decoder)
		if err != nil {
			return nil, err
		}
		array = append(array, value)
	}
	control.checkpoint()
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return array, nil
}

func runValidation(validate func() error) (err error) {
	defer func() {
		switch recovered := recover().(type) {
		case nil:
		case regexpTimeout:
			err = ErrValidationResourceLimit
		case validationCanceled:
			err = recovered.err
		case validationLimit:
			err = ErrValidationResourceLimit
		default:
			panic(recovered)
		}
	}()
	return validate()
}

func sortValidationErrors(current *validator.ValidationError, control *validationControl) {
	control.checkpoint()
	for _, cause := range current.Causes {
		sortValidationErrors(cause, control)
	}
	sort.SliceStable(current.Causes, func(left int, right int) bool {
		return strings.Compare(
			validationErrorKey(current.Causes[left]),
			validationErrorKey(current.Causes[right]),
		) == -1
	})
}

func validationErrorKey(current *validator.ValidationError) string {
	for current != nil && len(current.Causes) != 0 {
		current = current.Causes[0]
	}
	if current == nil {
		return ""
	}
	return jsonPointer(current.InstanceLocation) + "\x00" +
		jsonPointer(current.ErrorKind.KeywordPath())
}

func collectIssues(current *validator.ValidationError, limit int, issues *[]Issue) int {
	if current == nil {
		return 0
	}
	if len(current.Causes) != 0 {
		total := 0
		for _, cause := range current.Causes {
			if len(*issues) >= limit {
				return limit
			}
			total = total + collectIssues(cause, limit, issues)
		}
		return total
	}
	keywordPath := current.ErrorKind.KeywordPath()
	keyword := "false"
	if len(keywordPath) != 0 {
		keyword = keywordPath[len(keywordPath)-1]
	}
	*issues = append(*issues, Issue{
		InstancePointer: jsonPointer(current.InstanceLocation),
		SchemaPointer:   jsonPointer(keywordPath),
		Keyword:         keyword,
		Message:         "value does not satisfy the schema keyword",
	})
	return 1
}

func jsonPointer(segments []string) string {
	if len(segments) == 0 {
		return "#"
	}
	escaped := make([]string, len(segments))
	for index, segment := range segments {
		segment = strings.ReplaceAll(segment, "~", "~0")
		escaped[index] = strings.ReplaceAll(segment, "/", "~1")
	}
	return "#/" + strings.Join(escaped, "/")
}

func declaresDraft7(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("true")) || bytes.Equal(trimmed, []byte("false")) {
		return true
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return false
	}
	raw, exists := object["$schema"]
	if !exists {
		return true
	}
	var dialect string
	if err := json.Unmarshal(raw, &dialect); err != nil {
		return false
	}
	switch dialect {
	case "http://json-schema.org/draft-07/schema#",
		"http://json-schema.org/draft-07/schema",
		"https://json-schema.org/draft-07/schema#",
		"https://json-schema.org/draft-07/schema":
		return true
	default:
		return false
	}
}

func absoluteURI(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && parsed.IsAbs() && parsed.Fragment == ""
}
