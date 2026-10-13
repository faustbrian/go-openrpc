// Package specification audits pinned OpenRPC inputs and generates local
// normative/object-field evidence matrices. These are maintenance artifacts,
// not runtime validation or a conformance claim by themselves. See
// specification/manifest.json and internal/specification/cmd/specmatrix.
// VerifyPinnedInputs reads an explicit filesystem; GenerateMatrices and evidence
// joins operate on caller inputs without network loading, workers or cgo.
package specification

// nonBehavioralRequirements records normative inventory rows whose condition
// is outside this module's deliberately transport-neutral API surface.
var nonBehavioralRequirements = map[string]struct{}{
	"ORPC-1.4-0008": {},
	"ORPC-1.4-0009": {},
	"ORPC-1.4-0010": {},
	"ORPC-1.4-0011": {},
	"ORPC-1.4-0015": {},
	"ORPC-1.4-0016": {},
	"ORPC-1.4-0018": {},
	"ORPC-1.4-0028": {},
}
