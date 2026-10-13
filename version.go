// Package openrpc provides an ownership-safe OpenRPC document model and the
// core contracts shared by parsing, validation, resolution, and discovery.
// Start with ParseVersion and NewDocument or the parse/builder packages.
// OpenRPC 1.3.x/1.4.x share the modeled shape; pinned 1.4.1 authority and
// package-owned policies are recorded in docs/specification-decisions.md.
// Constructors copy mutable collections and optional values but do not perform
// complete structural/semantic validation. Immutable values may share storage;
// accessors returning slices/maps/bytes copy them. Zero union values select no
// case; wrapping a case does not validate its underlying value. Absent fields are
// distinct from explicit empty/false/null values where the API permits them.
// Core modeling owns no transport, reference fetching, workers or cgo; adapters
// and explicit validation/resolution have their own resource/lifetime contracts.
package openrpc

import (
	"errors"
	"regexp"
)

// ErrUnsupportedVersion reports a malformed, unsupported, or future OpenRPC
// specification version. Callers must not infer semantics for rejected values.
var ErrUnsupportedVersion = errors.New("openrpc: unsupported specification version")

var supportedVersionPattern = regexp.MustCompile(
	`^1\.(3|4)\.(0|[1-9][0-9]*)$`,
)

// Version is a validated OpenRPC specification version.
type Version struct {
	value string
}

// ParseVersion validates that value belongs to an explicitly supported OpenRPC
// feature line. Patch releases share their major.minor feature set as required
// by the specification's versioning rules.
func ParseVersion(value string) (Version, error) {
	if !supportedVersionPattern.MatchString(value) {
		return Version{}, ErrUnsupportedVersion
	}
	return Version{value: value}, nil
}

// String returns the exact validated semantic version.
func (version Version) String() string {
	return version.value
}

// FeatureSet returns the major.minor OpenRPC feature line.
func (version Version) FeatureSet() string {
	if version.value == "" {
		return ""
	}
	return version.value[:3]
}

// SupportedVersions returns the supported OpenRPC compatibility lines.
func SupportedVersions() []string {
	return []string{"1.3.x", "1.4.x"}
}
