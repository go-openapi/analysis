# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go library for analyzing, flattening, merging (mixin), and fixing
[Swagger 2.0](https://swagger.io/specification/v2/) specifications. Built on top of
`go-openapi/spec`, it is a central utility in the go-swagger ecosystem for code generation
and validation tooling.

Mono-repo with `go.work` tying together modules: root (`github.com/go-openapi/analysis`) and `internal/testintegration`.

See [docs/MAINTAINERS.md](../docs/MAINTAINERS.md) for CI/CD, release process, and repo structure details.

### Package layout

| File | Contents |
|------|----------|
| `doc.go` | Package-level godoc |
| `analyzer.go` | Core `Spec` type (analyzed spec index), `New()`, security/consumer/producer queries, operation lookups, ref/pattern/enum collection |
| `schema.go` | Schema classification: `Schema()`, `AnalyzedSchema` (IsKnownType, IsArray, IsMap, IsTuple, IsBaseType, IsEnum, …) |
| `flatten.go` | `Flatten(FlattenOpts)` — multi-step pipeline: expand → normalize → name inline schemas → strip OAIGen → remove unused |
| `flatten_options.go` | `FlattenOpts` configuration struct |
| `flatten_name.go` | `InlineSchemaNamer`, `GenLocation()` — derives stable definition names from JSON Pointer paths |
| `mixin.go` | `Mixin(primary, mixins...)` — merges paths, definitions, parameters, responses, security, tags; returns collision warnings |
| `fixer.go` | `FixEmptyResponseDescriptions()` — patches empty response descriptions from Go JSON unmarshalling quirks |
| `errors.go` | Sentinel errors (`ErrAnalysis`, `ErrNoSchema`) and error factory functions |
| `debug.go` | Debug logger wired to `SWAGGER_DEBUG` env var |

### Internal packages (`internal/`)

| Package | Contents |
|---------|----------|
| `internal/debug` | `GetLogger()` — conditional debug logging |
| `internal/antest` | Test helpers: `LoadSpec()`, `LoadOrFail()`, `AsJSON()`, `LongTestsEnabled()` |
| `internal/flatten/normalize` | `RebaseRef()`, `Path()` — canonical ref rebasing |
| `internal/flatten/operations` | `OpRef`, `GatherOperations()`, `AllOpRefsByRef()` |
| `internal/flatten/replace` | Low-level ref rewriting: `RewriteSchemaToRef()`, `UpdateRef()`, `DeepestRef()` |
| `internal/flatten/schutils` | `Save()`, `Clone()` — schema storage and deep-copy helpers |
| `internal/flatten/sortref` | `SplitKey` (parsed JSON pointer), `DepthFirst()`, `TopmostFirst()`, `ReverseIndex()` |

### Key API

- `New(*spec.Swagger) *Spec` — build an analyzed index from a parsed spec
- `Spec` methods — `Operations()`, `AllDefinitions()`, `AllRefs()`, `ConsumesFor()`, `ProducesFor()`, `ParametersFor()`, `SecurityRequirementsFor()`, pattern/enum enumeration
- `Schema(SchemaOpts) (*AnalyzedSchema, error)` — classify a schema (array, map, tuple, base type, etc.)
- `Flatten(FlattenOpts) error` — flatten/expand a spec (inline schemas → definitions, remote refs → local)
- `Mixin(primary, mixins...) []string` — merge multiple specs, returning collision warnings
- `FixEmptyResponseDescriptions(*spec.Swagger)` — patch empty response descriptions

### Dependencies

- `github.com/go-openapi/spec` — Swagger 2.0 document model
- `github.com/go-openapi/jsonpointer` — JSON pointer navigation (ref rewriting)
- `github.com/go-openapi/strfmt` — string format types
- `github.com/go-openapi/swag/jsonutils` — JSON utilities
- `github.com/go-openapi/swag/loading` — remote spec loading
- `github.com/go-openapi/swag/mangling` — name mangling for Go identifiers
- `github.com/go-openapi/testify/v2` — test-only assertions

### Notable design decisions

- **Flatten is a multi-step pipeline** (expand → normalize refs → name inline schemas → strip
  OAIGen artifacts → remove unused). It distinguishes "expand" mode (fully dereference) from
  "minimal" flatten (only pull in remote refs).
- **OAIGen naming convention** — when flattening creates a definition from an anonymous schema
  and no better name can be derived, it uses an `OAIGen` prefix as a sentinel. A post-pass
  attempts to resolve these into better names.
- **Mixin returns collision strings, not errors** — duplicate paths/definitions are soft failures
  reported as warning strings, letting callers decide severity.
- **`FixEmptyResponseDescriptions` compensates for Go JSON marshalling** — unmarshalling can
  produce `Response` objects with empty descriptions; this fixer patches them.
