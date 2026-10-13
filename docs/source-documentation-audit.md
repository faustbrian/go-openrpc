# Source documentation audit

## Initial scope and bounded plan

Tier A comments and directly related current contract prose from
`91716c9fbfe873225336dc0bc97bde527bd2cb8e`. This worker owns source documentation;
the coordinator owns publication and exact-source CI. Preserve the original
checkout, executable behavior, dependencies, fixtures, authority pins and
decision/history records.

Reproduce inventory with `git ls-files "*.go"`, `modules.json`, `packages.json`
and `golib source-docs check --json`. All 116 tracked Go files (44 implementation,
72 assurance) belong to `github.com/faustbrian/go-openrpc/v2`. Sixteen source
directories include 14 public packages, internal specification evidence and its
private matrix command. The inventory contains 300 test functions, 12 fuzz
targets, nine benchmarks and one executable Example; counts are scope, not runtime proof.
No build/platform variants or generated Go were found; two command security
annotations must remain unchanged. Generated specification evidence and pinned
external corpora have their own producers, not permission for an update campaign.

Outcome: truthful OpenRPC 1.3/1.4 model, parser, references, validation, composition,
discovery and adapter contracts, including optional presence and copied/borrowed
values. Direct consumers are document authors/parsers, schema validators, explicit
reference loaders, discovery handlers and compatibility/composition callers.
Named risks: lossless numbers versus canonical bytes, depth/size/evaluation
budgets, duplicate/unknown members, reference identity and cycles, optional HTTP
SSRF/loading, synchronous callbacks and cancellation, cache/registry ownership,
partial diagnostics and fail-closed compatibility. Missing comments are separate
from questions requiring source/specification investigation.

Smallest checks: all-file Go1.27.2 formatting and AST/token/literal/directive
preservation, all package renders, configured API producer discovery, selected
native docs and existing Examples where applicable, structural/source/local-link
and offline specification checks, pinned full/added/staged secrets, one ordinary
complete final-diff review and explicit-path Intent commit. API mode is apidiff;
no prose snapshot is configured. No broad runtime, race, fuzz, network, upstream
authority, benchmark, release or interoperability campaign. All Go/native checks
await an explicit serialized slot and use three owned disposable caches.

## Complete tracked inventory and initial gaps

The initial structural checker reports 305 missing declaration/package comments.
The rows below record the full source boundary before implementation edits.

| Go file | Initial gaps | Source boundary |
| --- | --- | --- |
| `benchmarks_test.go` | 0 | BenchmarkParseCompleteDocument; BenchmarkValidateCompleteDocument; additional cases/helpers |
| `builder/document.go` | 0 | implementation contracts and private invariants |
| `builder/document_test.go` | 0 | TestDocumentBuilderIsImmutableAndDeterministic; TestMethodRegistryIsConcurrentAndReturnsSortedSnapshots; additional cases/helpers |
| `builder/leak_test.go` | 0 | TestMain |
| `builder/registry.go` | 0 | implementation contracts and private invariants |
| `compose/coverage_test.go` | 0 | TestFilterCopiesOptionalMetadataAndSanitizesFailures; TestMergeCoversReferencesMetadataAndEveryRegistryConflict; additional cases/helpers |
| `compose/filter.go` | 3 | implementation contracts and private invariants |
| `compose/filter_test.go` | 0 | TestFilterMethodsPreservesDocumentAndAllowsEmptyVisibility; TestFilterMethodsHandlesReferencesByExplicitPolicy; additional cases/helpers |
| `compose/fuzz_test.go` | 0 | FuzzCompositionIsDeterministicAndPanicFree |
| `compose/merge.go` | 11 | implementation contracts and private invariants |
| `compose/merge_test.go` | 0 | TestMergeCombinesMethodsDeterministically; TestMergeAppliesExplicitConflictPolicy; additional cases/helpers |
| `compose/overlay.go` | 4 | implementation contracts and private invariants |
| `compose/overlay_test.go` | 0 | TestApplyOverlaysUsesOrderedRFC7396Semantics; TestApplyOverlaysRejectsInvalidResultsAndBounds; additional cases/helpers |
| `compose/rename.go` | 11 | implementation contracts and private invariants |
| `compose/rename_test.go` | 0 | TestRenameComponentsRewritesEveryMatchingReference; TestRenameComponentsRejectsCollisionsAndInvalidOptions |
| `diff/coverage_test.go` | 0 | TestReportCompatibilityAndHelperFailures; TestCompareCoversExactBoundsAndLoopCancellation; additional cases/helpers |
| `diff/diff.go` | 36 | implementation contracts and private invariants |
| `diff/diff_test.go` | 0 | TestCompareClassifiesMethodAndParameterCompatibility; TestCompareHonorsByPositionOrdering; additional cases/helpers |
| `diff/fuzz_test.go` | 0 | FuzzSemanticDiffIsDeterministic |
| `diff/resolved.go` | 0 | implementation contracts and private invariants |
| `discovery/cache.go` | 4 | implementation contracts and private invariants |
| `discovery/cache_edges_test.go` | 0 | TestCacheEdgeRejectsValidKeyWhenDerivationFails; TestCacheEdgeFailedLeaderReleasesPartitionCapacity; additional cases/helpers |
| `discovery/cache_test.go` | 0 | TestCacheDeduplicatesConcurrentDiscoveryAndInvalidatesExplicitly; TestCacheWaiterCanCancelWithoutCancelingSharedDiscovery; additional cases/helpers |
| `discovery/discovery.go` | 4 | implementation contracts and private invariants |
| `discovery/discovery_test.go` | 0 | TestServiceProducesCanonicalDiscoverySnapshot; TestServiceAppliesContextVisibilityPolicy; additional cases/helpers |
| `discovery/fuzz_test.go` | 0 | FuzzDiscoverySnapshotsAreDeterministic |
| `discovery/leak_test.go` | 0 | TestMain |
| `document.go` | 18 | implementation contracts and private invariants |
| `document_test.go` | 0 | TestDocumentRequiresVersionInfoAndMethods; TestDocumentPreservesEmptyMethodsAndServerDefault; additional cases/helpers |
| `example_test.go` | 0 | Example |
| `expression/coverage_test.go` | 0 | TestParserAndEvaluatorCoverEveryBoundary; TestInternalExpressionInvariants; additional cases/helpers |
| `expression/expression.go` | 6 | implementation contracts and private invariants |
| `expression/expression_test.go` | 0 | TestTemplateEvaluatesJSONTemplateLanguageVectors; TestWholeExpressionPreservesReferencedJSONType; additional cases/helpers |
| `expression/link.go` | 0 | implementation contracts and private invariants |
| `expression/link_test.go` | 0 | TestEvaluateLinkParamsPreservesExpressionTypesAndConstants; TestEvaluateLinkParamsHandlesAbsenceAndMissingValues |
| `expression/server.go` | 0 | implementation contracts and private invariants |
| `expression/server_test.go` | 0 | TestEvaluateServerUsesDefaultsAndValidatedOverrides; TestEvaluateServerRejectsUnknownInvalidAndMissingVariables |
| `fields.go` | 2 | implementation contracts and private invariants |
| `fields_test.go` | 0 | TestNewExtensionsRequiresUniquePrefixedNames; TestFieldsAreDeterministicAndOwnershipSafe |
| `fuzz_test.go` | 0 | FuzzAcceptedDocumentsRoundTripDeterministically |
| `integer.go` | 0 | implementation contracts and private invariants |
| `internal/specification/cmd/specmatrix/main.go` | 0 | implementation contracts and private invariants |
| `internal/specification/cmd/specmatrix/main_test.go` | 0 | TestRunRegeneratesReviewedMatrices; TestRunReportsMissingInputs; additional cases/helpers |
| `internal/specification/conformance.go` | 1 | implementation contracts and private invariants |
| `internal/specification/coverage_test.go` | 0 | TestManifestRejectsEveryInvalidShape; TestManifestAcceptsMetaDialectAsOnlyPinnedInput; additional cases/helpers |
| `internal/specification/evidence_test.go` | 0 | TestConformanceEvidenceReferencesExecutableTests; TestConformanceEvidenceCoversEveryNormativeStatement; additional cases/helpers |
| `internal/specification/manifest.go` | 0 | implementation contracts and private invariants |
| `internal/specification/manifest_test.go` | 0 | TestVerifyPinnedInputsChecksEveryManifestDigest; TestVerifyPinnedInputsRejectsMissingMalformedAndChangedInputs |
| `internal/specification/matrix.go` | 0 | implementation contracts and private invariants |
| `internal/specification/matrix_test.go` | 0 | TestNormativeSentencesInventoriesStatements; TestNormativeSentencesDoesNotMistakePartialPreambles; additional cases/helpers |
| `jsonrpc/adapter.go` | 1 | implementation contracts and private invariants |
| `jsonrpc/adapter_test.go` | 0 | TestDiscoveryHandlerReturnsCanonicalRawDocument; TestDiscoveryHandlerValidatesParamsAndSanitizesFailures; additional cases/helpers |
| `jsonschema/diagnostics_test.go` | 0 | TestDiagnosticHelpersHandleEmptyInputs |
| `jsonschema/evaluation_budget_test.go` | 0 | TestOrdinaryValidationChargesEverySchemaEvaluation |
| `jsonschema/fuzz_test.go` | 0 | FuzzDraft7CompileAndValidateDeterministically |
| `jsonschema/schema.go` | 0 | implementation contracts and private invariants |
| `jsonschema/schema_test.go` | 0 | TestParsePreservesBooleanAndObjectSchemas; TestParseRejectsNonSchemaJSONKinds; additional cases/helpers |
| `jsonschema/validate.go` | 12 | implementation contracts and private invariants |
| `jsonschema/validate_internal_test.go` | 0 | TestECMARegexpAdapterPreservesPatternIdentity; TestValidationBoundaryRepanicsUnexpectedFailures; additional cases/helpers |
| `jsonschema/validate_test.go` | 0 | TestValidatorAppliesDraft7WithoutNumericCoercion; TestValidatorSupportsBooleanSchemas; additional cases/helpers |
| `jsonschema/validation_owner_cancellation_internal_test.go` | 0 | TestValidationOwnerRegexpDeadlineFailurePrecedesDiagnosticConversion; TestValidationOwnerDependencyCheckpointPreservesCancellationAndReuse |
| `jsonschema/validation_owner_deadline_internal_test.go` | 0 | TestValidationOwnerRegexpDeadlineTimeoutUsesInclusivePositiveRemainingTime |
| `jsonschema/validation_owner_edges_test.go` | 0 | TestValidationOwnerZeroSchemaPreservesPolicyErrorPrecedence; TestValidationOwnerInstanceByteLimitIsInclusive; additional cases/helpers |
| `jsonvalue/fuzz_test.go` | 0 | FuzzParseStrictJSON |
| `jsonvalue/value.go` | 3 | implementation contracts and private invariants |
| `jsonvalue/value_test.go` | 0 | TestParsePreservesExactJSONAndOwnsMemory; TestValueReportsByteLengthWithoutAllocating; additional cases/helpers |
| `metadata.go` | 35 | implementation contracts and private invariants |
| `metadata_test.go` | 0 | TestMetadataObjectsPreserveOptionalPresence; TestRequiredMetadataFieldsAreExplicit; additional cases/helpers |
| `method.go` | 58 | ExampleValue; ExampleReference; additional cases/helpers |
| `method_test.go` | 0 | TestReferenceAndContentDescriptorRequiredFields; TestExamplesPreserveNullAndNotificationResultAbsence; additional cases/helpers |
| `model_accessors_test.go` | 0 | TestCompleteModelAccessorsAndUnionCases |
| `observe/leak_test.go` | 0 | TestMain |
| `observe/observe.go` | 16 | implementation contracts and private invariants |
| `observe/observe_test.go` | 0 | TestOperationsReportOnlyBoundedMetadata; TestOperationsContainObserversAndClassifyFailures; additional cases/helpers |
| `parse/complete_test.go` | 0 | TestCompleteObjectFieldRoundTrip; TestObjectFieldRequirednessAndNullabilityMatrix |
| `parse/coverage_test.go` | 0 | TestDecodeRejectsMalformedReferencesInEveryUnion; TestDecodeRejectsMalformedUnionEntriesAndComponentValues; additional cases/helpers |
| `parse/fuzz_test.go` | 0 | FuzzDecodeOpenRPCDocument |
| `parse/parse.go` | 13 | implementation contracts and private invariants |
| `parse/parse_test.go` | 0 | TestDecodeMinimalDocument; TestDecodeUnknownFieldsUsesExplicitMode; additional cases/helpers |
| `reference/advanced_coverage_test.go` | 0 | TestResolverCoversInputErrorsCancellationAndLoadFailures; TestLoadCoversDirectPolicyBranches; additional cases/helpers |
| `reference/basic_coverage_test.go` | 0 | TestPointerCoversExactLimitsAndEveryTargetFailure; TestReferenceCoversZeroValuesControlsAndResolvedLimits; additional cases/helpers |
| `reference/bundle.go` | 0 | implementation contracts and private invariants |
| `reference/bundle_test.go` | 0 | TestBundlePreservesRootAndTransitiveResourcesByURI; TestBundleBoundsRootReferenceFanoutBeforeLoading |
| `reference/fuzz_test.go` | 0 | FuzzReferenceAndPointerParsing; FuzzResolverInternalReferences |
| `reference/httpstore/httpstore.go` | 16 | implementation contracts and private invariants |
| `reference/httpstore/httpstore_test.go` | 0 | TestStoreLoadsAllowedBoundedIdentityResponses; TestStoreRejectsLimitsCompressionStatusAndRedirects; additional cases/helpers |
| `reference/httpstore/internal_test.go` | 0 | TestAuthorizeAndDialRejectInvalidInternalInputs; TestDeniedAddressRejectsEveryNonPublicAddressClass |
| `reference/httpstore/leak_test.go` | 0 | TestMain |
| `reference/leak_test.go` | 0 | TestMain |
| `reference/pointer.go` | 3 | implementation contracts and private invariants |
| `reference/pointer_test.go` | 0 | TestPointerParsesEscapesAndEvaluatesRawTargets; TestPointerRejectsMalformedOrUnresolvablePaths; additional cases/helpers |
| `reference/reference.go` | 1 | implementation contracts and private invariants |
| `reference/reference_test.go` | 0 | TestReferenceClassifiesAndResolvesAgainstBase; TestReferenceExposesJSONPointerFragment; additional cases/helpers |
| `reference/resolver.go` | 11 | implementation contracts and private invariants |
| `reference/resolver_test.go` | 0 | TestResolverFollowsAliasesAndLoadsEachDocumentOnce; TestResolverResolveManySharesFetchedDocumentsAcrossReferences; additional cases/helpers |
| `reference/resources.go` | 0 | implementation contracts and private invariants |
| `reference/resources_test.go` | 0 | TestResourcesLoadsTransitiveSchemaGraphOnceAndHonorsIDs; TestResourcesBoundsTransitiveReferenceFanout; additional cases/helpers |
| `reference/store.go` | 1 | implementation contracts and private invariants |
| `reference/store_test.go` | 0 | TestMemoryStoreOwnsDocumentsAndEnforcesBounds; TestFSStoreMapsOnlyURIsBelowExplicitBase; additional cases/helpers |
| `reference/transform.go` | 7 | implementation contracts and private invariants |
| `reference/transform_test.go` | 0 | TestDereferenceExpandsInternalAndExternalTargets; TestDereferenceRejectsCyclesAndTransformLimits; additional cases/helpers |
| `serialization.go` | 0 | implementation contracts and private invariants |
| `serialization_coverage_test.go` | 0 | TestSerializationStandardFieldCollisions; TestSerializationUnionAndNestedFailures; additional cases/helpers |
| `serialization_test.go` | 0 | TestMarshalCanonicalIsStableAcrossObjectOrderAndWhitespace; TestMarshalCanonicalRejectsPreservedFieldCollision; additional cases/helpers |
| `specification.go` | 0 | implementation contracts and private invariants |
| `validate/coverage_internal_test.go` | 0 | TestMetaSchemaInternalFailurePaths; TestMetaSchemaHelpersCoverMalformedAndPartialShapes; additional cases/helpers |
| `validate/metaschema.go` | 0 | implementation contracts and private invariants |
| `validate/metaschema_differential_test.go` | 0 | TestPublishedMetaSchemaDiffersFromNormativeServerURLPolicy |
| `validate/metaschema_test.go` | 0 | TestPinnedOfficialExamplesRespectSupportedVersionLines; TestPinnedMetaSchemaAcceptsCurrentMinimalDocument; additional cases/helpers |
| `validate/resolved.go` | 3 | implementation contracts and private invariants |
| `validate/resolved_schema.go` | 0 | implementation contracts and private invariants |
| `validate/schema.go` | 0 | implementation contracts and private invariants |
| `validate/validate.go` | 25 | implementation contracts and private invariants |
| `validate/validate_test.go` | 0 | TestCollectReportsDeterministicSemanticDiagnostics; TestFailFastUsesTheSameFirstRule; additional cases/helpers |
| `version.go` | 0 | implementation contracts and private invariants |
| `version_test.go` | 0 | TestParseVersionAcceptsSupportedPatchLine; TestParseVersionRejectsUnsupportedOrMalformedVersions; additional cases/helpers |

## Investigation disposition

All 44 implementation files were traced for public fields/methods, private
normalization, budgets, provider seams and pinned-rule interpretation. All 72
assurance files were inventoried for assertions, fixtures, callback/channel
ownership, package TestMain cleanup, fuzz oracles and benchmark setup. The
305 initial structural omissions are repaired without private-symbol filler.

- Models copy mutable slices/maps and pointed optional scalars, while immutable
  JSON/schema/model values may share backing storage. Presence flags preserve
  absent versus explicit empty/false/null where allowed. Constructors check
  local requiredness, not complete nested/semantic validity or bounded input.
  A zero union is unselected; a zero JSON/schema value is not JSON null.
- Generic JSON admits UTF-8, unique names, one complete value and finite
  byte/container/token limits before taking owned bytes. Typed parsing preserves
  source input, rejects Reference Object siblings even in preserving mode and
  bounds every component registry plus their aggregate. It is not complete
  meta-schema/semantic validation; scalar-null coercions in encoding/json are
  not a universal nullability check. The object-field matrix tests the separate
  structural validator rather than every Decode admission boundary.
- Canonical serialization sorts modeled maps and compacts JSON, but raw schema/
  value payloads retain object-member order and exact number lexemes. It is not
  RFC 8785 or universal semantic normalization. Serialization has no context or
  output bound; discovery/transform/composition output checks follow allocation.
  Integer retains -0 rather than normalizing it; error-code duplicate checks use
  exact lexemes. Desired OPENRPC-DEC-004/009 policy is not rewritten to hide these
  representation/normalization limitations.
- Draft 7 syntax and compilation are distinct. Only explicit resources load;
  compile has no context. Validator copies share one cancellable active-owner
  slot protecting mutable regexp timeout and checkpoint state. Wrapper graph
  edges preserve checkpoints before short-circuit results/recursive references.
  Admission/step counts are not RSS or exact wall-time guarantees. Unexpected
  dependency panics propagate; only known budget/cancellation signals convert.
- URI identity strips fragments for document caches and uses decoded pointer
  spelling for alias-cycle detection. Resources and bundling preserve loaded
  bytes and schema identifiers rather than flattening recursive schema refs.
  DereferenceSelected keeps explicit OpenRPC object locations separate from
  schema graph compilation. Stores/selectors are borrowed, callbacks unlocked
  and synchronous; cancellation cannot force a blocked provider to return.
- FSStore enforces URI/path spelling and copies/checks provider results; the
  ContextReadFS implementation owns actual root/symlink containment and prompt
  cancellation. HTTP loading closes responses, checks status/encoding/body limits
  and direct DNS/IP candidates, but clones ambient proxy/TLS settings and requires
  http.DefaultTransport to be a nonnil *http.Transport. A proxy is a separate
  trust/target-resolution boundary. There is no public idle-close operation.
  Load itself does not parse JSON; body-read cancellation may become ErrRequest.
- Builders are immutable copies; the concurrent method registry owns its lock
  and initializes its zero value lazily. Discovery services borrow providers/
  filters and do not recover their panics. Cache keys must include every tenant/
  authorization input; cached success has no expiry. Leaders own refresh context
  and completion-channel closure, while waiters can cancel independently. Entry
  identity prevents invalidated leaders from caching old results. A panicking
  leader never closes its completion channel; existing waiters need their own
  cancellation even if future calls start a new invalidated partition.
- Composition owns explicit collision policy and deterministic output, not a
  standardized OpenRPC merge. Output is all-or-error without mutating inputs;
  intermediate work may precede final output admission. Sorted sequential renames
  are not simultaneous permutations; matching $ref strings in opaque values are
  rewritten by spelling, not by base URI. Diff is a package compatibility model,
  not schema implication proof or a validator for duplicate/invalid inputs.
  Returned change bounds truncate after collection; logical paths use identities.
- Messages omit values, but pointers may contain caller-controlled names. Public
  error structs can print manually supplied causes. Observe contains callback
  panics only during wrapper dispatch; direct ObserverFunc calls do not. Parse
  observes context only before its synchronous Decode; observer callbacks can
  block and outcomes identify wrapper execution rather than independent validity.

## Assurance, producers and unresolved constraints

All twelve fuzz targets now state admission and actual assertions. Several
pointer/reference/expression targets discard outcomes and protect panic safety,
not successful resolution or rejection; schema fuzz compares issue slices rather
than every report field or an independent oracle. Parse fuzz checks MetaSchema.Err,
not MetaSchema.Valid. Round-trip, composition, diff and discovery targets retain
their specific determinism/ownership predicates. Benchmark comments distinguish
fixture setup, compilation, memory-only resolution and hostile depth refusal;
no new timing result is produced. One executable Example is distinct from public
Example-named constructors.

No generated Go or build/platform variants were found. The matrix command uses
pinned local prose/schema plus reviewed evidence maps and writes two outputs
sequentially, not atomically. The pinned source-sync producer was inspected, not
executed. Embedded authority bytes, generated/frozen evidence, apidiff baseline,
dependencies, eleven decisions and history prefixes remain unchanged.

Missing comments are resolved. Numerical-normalization/canonical-output scope,
parser-versus-meta-schema nullability, borrowed-provider behavior, ambient proxy
policy and post-allocation output checks remain explicit source constraints,
not newly proven desired guarantees. No broad runtime, upstream, HTTP, race,
fuzz, benchmark, conformance or interoperability campaign is scoped.

## Selected local verification

Go 1.27.2 all-file formatting, parsed AST/executable token/literal identity and
exact directive preservation passed for all 116 tracked Go files. All comment
groups and sixteen package/command documents render. Only parser-optional
implicit semicolons before closing delimiters are eligible for normalization;
the observed delta is zero. The source checker reports zero structural findings.

Native root documentation checks structure, spelling and links, without a
configured example operation. The selected existing root Example passed via
`go test ./... -run '^Example' -count=1 -timeout=2m`; other packages have no
selected test cases. Offline native specification validation accepts all eleven
unchanged decisions/histories. API mode is apidiff, so no prose snapshot or
frozen public API baseline is regenerated. Two finite offline observations
confirm that opaque object order changes canonical bytes and that 0/-0 integer
spellings are admitted and remain distinct. No external HTTP server or broad
runtime/assurance campaign was used.

## Objective Enforcement

The checksum-pinned public Tools v2.4.0 verifier now supplies
`make source-docs-check` as a prerequisite of the existing local and CI Make
routes. Ordinary shared CI selects strict source-documentation checking and
requires its result. Native prerequisites, operation budgets, security and
release controls are retained. Hosted exact-head evidence is recorded by the
coordination ledger; configuration alone does not establish green delivery.
