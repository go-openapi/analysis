package analysis

import (
	"path/filepath"
	"testing"

	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

func TestSaveDefinition(t *testing.T) {
	sp := &spec.Swagger{}
	saveSchema(sp, "theName", spec.StringProperty())
	assert.Contains(t, sp.Definitions, "theName")
}

func TestNameFromRef(t *testing.T) {
	values := []struct{ Source, Expected string }{
		{"#/definitions/errorModel", "errorModel"},
		{"http://somewhere.com/definitions/errorModel", "errorModel"},
		{"http://somewhere.com/definitions/errorModel.json", "errorModel"},
		{"/definitions/errorModel", "errorModel"},
		{"/definitions/errorModel.json", "errorModel"},
		{"http://somewhere.com", "somewhereCom"},
		{"#", ""},
	}

	for _, v := range values {
		assert.Equal(t, v.Expected, nameFromRef(spec.MustCreateRef(v.Source)))
	}
}

func TestDefinitionName(t *testing.T) {
	values := []struct {
		Source, Expected string
		Definitions      spec.Definitions
	}{
		{"#/definitions/errorModel", "errorModel", map[string]spec.Schema(nil)},
		{"http://somewhere.com/definitions/errorModel", "errorModel", map[string]spec.Schema(nil)},
		{"#/definitions/errorModel", "errorModel", map[string]spec.Schema{"apples": *spec.StringProperty()}},
		{"#/definitions/errorModel", "errorModelOAIGen", map[string]spec.Schema{"errorModel": *spec.StringProperty()}},
		{"#/definitions/errorModel", "errorModelOAIGen1", map[string]spec.Schema{"errorModel": *spec.StringProperty(), "errorModelOAIGen": *spec.StringProperty()}},
		{"#", "oaiGen", nil},
	}

	for _, v := range values {
		assert.Equal(t, v.Expected, uniqifyName(v.Definitions, nameFromRef(spec.MustCreateRef(v.Source))))
	}
}

func TestUpdateRef(t *testing.T) {
	bp := filepath.Join("fixtures", "external_definitions.yml")
	sp, err := loadSpec(bp)
	if assert.NoError(t, err) {

		values := []struct {
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

		for _, v := range values {
			err := updateRef(sp, v.Key, v.Ref)
			if assert.NoError(t, err) {
				ptr, err := jsonpointer.New(v.Key[1:])
				if assert.NoError(t, err) {
					vv, _, err := ptr.Get(sp)

					if assert.NoError(t, err) {
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
			}
		}
	}
}

func TestImportExternalReferences(t *testing.T) {
	bp := filepath.Join(".", "fixtures", "external_definitions.yml")
	sp, err := loadSpec(bp)
	if assert.NoError(t, err) {

		values := []struct {
			Key string
			Ref spec.Ref
		}{
			{"#/parameters/someParam/schema", spec.MustCreateRef("#/definitions/record")},
			{"#/paths/~1some~1where~1{id}/parameters/1/schema", spec.MustCreateRef("#/definitions/record")},
			{"#/paths/~1some~1where~1{id}/get/parameters/2/schema", spec.MustCreateRef("#/definitions/record")},
			{"#/responses/someResponse/schema", spec.MustCreateRef("#/definitions/record")},
			{"#/paths/~1some~1where~1{id}/get/responses/default/schema", spec.MustCreateRef("#/definitions/record")},
			{"#/paths/~1some~1where~1{id}/get/responses/200/schema", spec.MustCreateRef("#/definitions/tag")},
			{"#/definitions/namedAgain", spec.MustCreateRef("#/definitions/named")},
			{"#/definitions/datedTag/allOf/1", spec.MustCreateRef("#/definitions/tag")},
			{"#/definitions/datedRecords/items/1", spec.MustCreateRef("#/definitions/record")},
			{"#/definitions/datedTaggedRecords/items/1", spec.MustCreateRef("#/definitions/record")},
			{"#/definitions/datedTaggedRecords/additionalItems", spec.MustCreateRef("#/definitions/tag")},
			{"#/definitions/otherRecords/items", spec.MustCreateRef("#/definitions/record")},
			{"#/definitions/tags/additionalProperties", spec.MustCreateRef("#/definitions/tag")},
			{"#/definitions/namedThing/properties/name", spec.MustCreateRef("#/definitions/named")},
		}
		for _, v := range values {
			// technically not necessary to run for each value, but if things go right
			// this is idempotent, so having it repeat shouldn't matter
			// this validates that behavior
			err := importExternalReferences(&FlattenOpts{
				Spec:     New(sp),
				BasePath: bp,
			})

			if assert.NoError(t, err) {

				ptr, err := jsonpointer.New(v.Key[1:])
				if assert.NoError(t, err) {
					vv, _, err := ptr.Get(sp)

					if assert.NoError(t, err) {
						switch tv := vv.(type) {
						case *spec.Schema:
							assert.Equal(t, v.Ref.String(), tv.Ref.String(), "for %s", v.Key)
						case spec.Schema:
							assert.Equal(t, v.Ref.String(), tv.Ref.String(), "for %s", v.Key)
						case *spec.SchemaOrBool:
							assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "for %s", v.Key)
						case *spec.SchemaOrArray:
							assert.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "for %s", v.Key)
						default:
							assert.Fail(t, "unknown type", "got %T", vv)
						}
					}
				}
			}
		}
		assert.Len(t, sp.Definitions, 11)
		assert.Contains(t, sp.Definitions, "tag")
		assert.Contains(t, sp.Definitions, "named")
		assert.Contains(t, sp.Definitions, "record")
	}
}

// func TestNameInlinedSchemas(t *testing.T) {
// 	bp := filepath.Join(".", "fixtures", "inline_schemas.yml")
// 	sp, err := loadSpec(bp)
// 	if assert.NoError(t, err) {
// 		nameInlinedSchemas(&FlattenOpts{
// 			Spec:     New(sp),
// 			BasePath: bp,
// 		})
// 	}
// }

func TestRewriteSchemaRef(t *testing.T) {
	bp := filepath.Join("fixtures", "inline_schemas.yml")
	sp, err := loadSpec(bp)
	if assert.NoError(t, err) {

		values := []struct {
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

		for i, v := range values {
			err := rewriteSchemaToRef(sp, v.Key, v.Ref)
			if assert.NoError(t, err) {
				ptr, err := jsonpointer.New(v.Key[1:])
				if assert.NoError(t, err) {
					vv, _, err := ptr.Get(sp)

					if assert.NoError(t, err) {
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
			}
		}
	}
}
