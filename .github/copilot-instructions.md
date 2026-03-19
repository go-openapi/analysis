# Copilot Instructions

## Project Overview

Go library for analyzing, diffing, flattening, merging (mixin), and fixing
[Swagger 2.0](https://swagger.io/specification/v2/) specifications. Built on top of
`go-openapi/spec`, it is a central utility in the go-swagger ecosystem for code generation
and validation tooling.

### Key packages

| File | Contents |
|------|----------|
| `analyzer.go` | Core `Spec` type — analyzed spec index, operation lookups, ref/pattern/enum collection |
| `schema.go` | Schema classification: `AnalyzedSchema` (IsKnownType, IsArray, IsMap, IsTuple, IsBaseType, IsEnum, ...) |
| `flatten.go` | `Flatten(FlattenOpts)` — multi-step pipeline: expand, normalize, name inline schemas, strip OAIGen, remove unused |
| `mixin.go` | `Mixin(primary, mixins...)` — merges specs, returns collision warnings |
| `fixer.go` | `FixEmptyResponseDescriptions()` — patches empty response descriptions |
| `diff/` | `Compare(spec1, spec2)` — compares two specs and reports structural and compatibility changes |

Internal packages live under `internal/` (debug, antest, flatten/{normalize,operations,replace,schutils,sortref}).

### Key API

- `New(*spec.Swagger) *Spec` — build an analyzed index from a parsed spec
- `Flatten(FlattenOpts) error` — flatten/expand a spec
- `Mixin(primary, mixins...) []string` — merge multiple specs
- `diff.Compare(spec1, spec2 *spec.Swagger) (SpecDifferences, error)` — compare two specs
- `Schema(SchemaOpts) (*AnalyzedSchema, error)` — classify a schema

### Dependencies

- `github.com/go-openapi/spec` — Swagger 2.0 document model
- `github.com/go-openapi/jsonpointer` — JSON pointer navigation
- `github.com/go-openapi/strfmt` — string format types
- `github.com/go-openapi/swag` — JSON utilities, spec loading, name mangling
- `github.com/go-openapi/testify/v2` — test-only assertions

## Conventions

Coding conventions are found beneath `.github/copilot`

### Summary

- All `.go` files must have SPDX license headers (Apache-2.0).
- Commits require DCO sign-off (`git commit -s`).
- Linting: `golangci-lint run` — config in `.golangci.yml` (posture: `default: all` with explicit disables).
- Every `//nolint` directive **must** have an inline comment explaining why.
- Tests: `go test work ./...` (mono-repo). CI runs on `{ubuntu, macos, windows} x {stable, oldstable}` with `-race`.
- Test framework: `github.com/go-openapi/testify/v2` (not `stretchr/testify`; `testifylint` does not work).

See `.github/copilot/` (symlinked to `.claude/rules/`) for detailed rules on Go conventions, linting, testing, and contributions.
