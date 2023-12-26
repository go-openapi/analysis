package replace

import (
	"path/filepath"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var refFixtures = []struct {
	Key string
	Ref spec.Ref
}{
	{"#/parameters/someParam/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/paths/~1some~1where~1{id}/parameters/1/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/paths/~1some~1where~1{id}/get/parameters/2/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/responses/someResponse/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/paths/~1some~1where~1{id}/get/responses/default/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/paths/~1some~1where~1{id}/get/responses/200/schema", spec.MustCreateRef("#/definitions/record")},
	{"#/definitions/namedAgain", spec.MustCreateRef("#/definitions/named")},
	{"#/definitions/datedTag/allOf/1", spec.MustCreateRef("#/definitions/tag")},
	{"#/definitions/datedRecords/items/1", spec.MustCreateRef("#/definitions/record")},
	{"#/definitions/datedTaggedRecords/items/1", spec.MustCreateRef("#/definitions/record")},
	{"#/definitions/datedTaggedRecords/additionalItems", spec.MustCreateRef("#/definitions/tag")},
	{"#/definitions/otherRecords/items", spec.MustCreateRef("#/definitions/record")},
	{"#/definitions/tags/additionalProperties", spec.MustCreateRef("#/definitions/tag")},
	{"#/definitions/namedThing/properties/name", spec.MustCreateRef("#/definitions/named")},
}

func TestUpdateRef(t *testing.T) {
	t.Parallel()

	bp := filepath.Join("..", "..", "..", "fixtures", "external_definitions.yml")
	sp := antest.LoadOrFail(t, bp)

	for _, v := range refFixtures {
		err := UpdateRef(sp, v.Key, v.Ref)
		require.NoError(t, err)

		ptr, err := jsonpointer.New(v.Key[1:])
		require.NoError(t, err)

		vv, _, err := ptr.Get(sp)
		require.NoError(t, err)

		switch tv := vv.(type) {
		case *spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String())
		case spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String())
		case *spec.SchemaOrBool:
			assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String())
		case *spec.SchemaOrArray:
			assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String())
		default:
			assert.Fail(t, "unknown type", "got %T", vv)
		}
	}
}

func TestRewriteSchemaRef(t *testing.T) {
	t.Parallel()

	bp := filepath.Join("..", "..", "..", "fixtures", "inline_schemas.yml")
	sp := antest.LoadOrFail(t, bp)

	for i, v := range refFixtures {
		err := RewriteSchemaToRef(sp, v.Key, v.Ref)
		require.NoError(t, err)

		ptr, err := jsonpointer.New(v.Key[1:])
		require.NoError(t, err)

		vv, _, err := ptr.Get(sp)
		require.NoError(t, err)

		switch tv := vv.(type) {
		case *spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String(), "at %d for %s", i, v.Key)
		case spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String(), "at %d for %s", i, v.Key)
		case *spec.SchemaOrBool:
			assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "at %d for %s", i, v.Key)
		case *spec.SchemaOrArray:
			assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "at %d for %s", i, v.Key)
		default:
			assert.Fail(t, "unknown type", "got %T", vv)
		}
	}
}

func TestReplace_ErrorHandling(t *testing.T) {
	t.Parallel()

	const wantedFailure = "expected a failure"
	bp := filepath.Join("..", "..", "..", "fixtures", "errors", "fixture-unexpandable-2.yaml")
	sp := antest.LoadOrFail(t, bp)

	t.Run("with invalid $ref", func(t *testing.T) {
		const ref = "#/invalidPointer/key"

		require.Errorf(t, RewriteSchemaToRef(sp, ref, spec.Ref{}), wantedFailure)
		require.Errorf(t, rewriteParentRef(sp, ref, spec.Ref{}), wantedFailure)
		require.Errorf(t, UpdateRef(sp, ref, spec.Ref{}), wantedFailure)
		require.Errorf(t, UpdateRefWithSchema(sp, ref, &spec.Schema{}), wantedFailure)
		_, _, err := getPointerFromKey(sp, ref)
		require.Errorf(t, err, wantedFailure)

		_, _, _, err = getParentFromKey(sp, ref)
		require.Errorf(t, err, wantedFailure)
	})

	t.Run("with invalid jsonpointer formatting", func(t *testing.T) {
		const pointer = "-->#/invalidJsonPointer"

		_, _, err := getPointerFromKey(sp, pointer)
		require.Errorf(t, err, wantedFailure)

		_, _, _, err = getParentFromKey(sp, pointer)
		require.Errorf(t, err, wantedFailure)
	})

	t.Run("with invalid target", func(t *testing.T) {
		require.Errorf(t, RewriteSchemaToRef(sp, "#/parameters/someWhere", spec.Ref{}), wantedFailure)
	})

	t.Run("with invalid response target", func(t *testing.T) {
		const ref = "#/paths/~1wrong/get/responses/200"
		require.Errorf(t, RewriteSchemaToRef(sp, ref, spec.Ref{}), wantedFailure)
	})

	t.Run("with invalid response code", func(t *testing.T) {
		const ref = "#/paths/~1wrongcode/get/responses/two-hundred"
		require.Errorf(t, rewriteParentRef(sp, ref, spec.Ref{}), wantedFailure)
	})

	t.Run("with panic case", func(t *testing.T) {
		t.Run("should not call with types other than *Schema or *Swagger", func(t *testing.T) {
			require.Panics(t, func() {
				_, _, _ = getPointerFromKey("oops", "#/key")
			})

			require.Panics(t, func() {
				_, _, _, _ = getParentFromKey("oops", "#/key")
			})

			require.Panics(t, func() {
				_ = UpdateRef("oops", "#/key", spec.Ref{})
			})
		})
	})
}
