package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/analysis/internal/flatten/operations"
	"github.com/go-openapi/analysis/internal/flatten/sortref"
	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestName_FromRef(t *testing.T) {
	t.Parallel()

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

func TestName_Definition(t *testing.T) {
	values := []struct {
		Source, Expected string
		Definitions      spec.Definitions
	}{
		{"#/definitions/errorModel", "errorModel", map[string]spec.Schema(nil)},
		{"http://somewhere.com/definitions/errorModel", "errorModel", map[string]spec.Schema(nil)},
		{"#/definitions/errorModel", "errorModel", map[string]spec.Schema{"apples": *spec.StringProperty()}},
		{"#/definitions/errorModel", "errorModelOAIGen", map[string]spec.Schema{"errorModel": *spec.StringProperty()}},
		{"#/definitions/errorModel", "errorModelOAIGen1",
			map[string]spec.Schema{"errorModel": *spec.StringProperty(), "errorModelOAIGen": *spec.StringProperty()}},
		{"#", "oaiGen", nil},
	}

	for _, v := range values {
		u, _ := uniqifyName(v.Definitions, nameFromRef(spec.MustCreateRef(v.Source)))
		assert.Equal(t, v.Expected, u)
	}
}

func TestName_SplitKey(t *testing.T) {
	type KeyFlag uint64

	const (
		isOperation KeyFlag = 1 << iota
		isDefinition
		isSharedOperationParam
		isOperationParam
		isOperationResponse
		isDefaultResponse
		isStatusCodeResponse
	)

	values := []struct {
		Key         string
		Flags       KeyFlag
		PathItemRef spec.Ref
		PathRef     spec.Ref
		Name        string
	}{
		{
			"#/paths/~1some~1where~1{id}/parameters/1/schema",
			isOperation | isSharedOperationParam,
			spec.Ref{},
			spec.MustCreateRef("#/paths/~1some~1where~1{id}"),
			"",
		},
		{
			"#/paths/~1some~1where~1{id}/get/parameters/2/schema",
			isOperation | isOperationParam,
			spec.MustCreateRef("#/paths/~1some~1where~1{id}/GET"),
			spec.MustCreateRef("#/paths/~1some~1where~1{id}"),
			"",
		},
		{
			"#/paths/~1some~1where~1{id}/get/responses/default/schema",
			isOperation | isOperationResponse | isDefaultResponse,
			spec.MustCreateRef("#/paths/~1some~1where~1{id}/GET"),
			spec.MustCreateRef("#/paths/~1some~1where~1{id}"),
			"Default",
		},
		{
			"#/paths/~1some~1where~1{id}/get/responses/200/schema",
			isOperation | isOperationResponse | isStatusCodeResponse,
			spec.MustCreateRef("#/paths/~1some~1where~1{id}/GET"),
			spec.MustCreateRef("#/paths/~1some~1where~1{id}"),
			"OK",
		},
		{
			"#/definitions/namedAgain",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"namedAgain",
		},
		{
			"#/definitions/datedRecords/items/1",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"datedRecords",
		},
		{
			"#/definitions/datedRecords/items/1",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"datedRecords",
		},
		{
			"#/definitions/datedTaggedRecords/items/1",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"datedTaggedRecords",
		},
		{
			"#/definitions/datedTaggedRecords/additionalItems",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"datedTaggedRecords",
		},
		{
			"#/definitions/otherRecords/items",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"otherRecords",
		},
		{
			"#/definitions/tags/additionalProperties",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"tags",
		},
		{
			"#/definitions/namedThing/properties/name",
			isDefinition,
			spec.Ref{},
			spec.Ref{},
			"namedThing",
		},
	}

	for i, v := range values {
		parts := sortref.KeyParts(v.Key)
		pref := parts.PathRef()
		piref := parts.PathItemRef()
		assert.Equal(t, v.PathRef.String(), pref.String(), "pathRef: %s at %d", v.Key, i)
		assert.Equal(t, v.PathItemRef.String(), piref.String(), "pathItemRef: %s at %d", v.Key, i)

		if v.Flags&isOperation != 0 {
			assert.True(t, parts.IsOperation(), "isOperation: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsOperation(), "isOperation: %s at %d", v.Key, i)
		}

		if v.Flags&isDefinition != 0 {
			assert.True(t, parts.IsDefinition(), "isDefinition: %s at %d", v.Key, i)
			assert.Equal(t, v.Name, parts.DefinitionName(), "definition name: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsDefinition(), "isDefinition: %s at %d", v.Key, i)
			if v.Name != "" {
				assert.Equal(t, v.Name, parts.ResponseName(), "response name: %s at %d", v.Key, i)
			}
		}

		if v.Flags&isOperationParam != 0 {
			assert.True(t, parts.IsOperationParam(), "isOperationParam: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsOperationParam(), "isOperationParam: %s at %d", v.Key, i)
		}

		if v.Flags&isSharedOperationParam != 0 {
			assert.True(t, parts.IsSharedOperationParam(), "isSharedOperationParam: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsSharedOperationParam(), "isSharedOperationParam: %s at %d", v.Key, i)
		}

		if v.Flags&isOperationResponse != 0 {
			assert.True(t, parts.IsOperationResponse(), "isOperationResponse: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsOperationResponse(), "isOperationResponse: %s at %d", v.Key, i)
		}

		if v.Flags&isDefaultResponse != 0 {
			assert.True(t, parts.IsDefaultResponse(), "isDefaultResponse: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsDefaultResponse(), "isDefaultResponse: %s at %d", v.Key, i)
		}

		if v.Flags&isStatusCodeResponse != 0 {
			assert.True(t, parts.IsStatusCodeResponse(), "isStatusCodeResponse: %s at %d", v.Key, i)
		} else {
			assert.False(t, parts.IsStatusCodeResponse(), "isStatusCodeResponse: %s at %d", v.Key, i)
		}
	}
}

func TestName_NamesFromKey(t *testing.T) {
	bp := filepath.Join("fixtures", "inline_schemas.yml")
	sp := antest.LoadOrFail(t, bp)

	values := []struct {
		Key   string
		Names []string
	}{
		{"#/paths/~1some~1where~1{id}/parameters/1/schema",
			[]string{"GetSomeWhereID params body", "PostSomeWhereID params body"}},
		{"#/paths/~1some~1where~1{id}/get/parameters/2/schema", []string{"GetSomeWhereID params body"}},
		{"#/paths/~1some~1where~1{id}/get/responses/default/schema", []string{"GetSomeWhereID Default body"}},
		{"#/paths/~1some~1where~1{id}/get/responses/200/schema", []string{"GetSomeWhereID OK body"}},
		{"#/definitions/namedAgain", []string{"namedAgain"}},
		{"#/definitions/datedTag/allOf/1", []string{"datedTag allOf 1"}},
		{"#/definitions/datedRecords/items/1", []string{"datedRecords tuple 1"}},
		{"#/definitions/datedTaggedRecords/items/1", []string{"datedTaggedRecords tuple 1"}},
		{"#/definitions/datedTaggedRecords/additionalItems", []string{"datedTaggedRecords tuple additionalItems"}},
		{"#/definitions/otherRecords/items", []string{"otherRecords items"}},
		{"#/definitions/tags/additionalProperties", []string{"tags additionalProperties"}},
		{"#/definitions/namedThing/properties/name", []string{"namedThing name"}},
	}

	for i, v := range values {
		ptr, err := jsonpointer.New(definitionPtr(v.Key)[1:])
		require.NoError(t, err)

		vv, _, err := ptr.Get(sp)
		require.NoError(t, err)

		switch tv := vv.(type) {
		case *spec.Schema:
			aschema, err := Schema(SchemaOpts{Schema: tv, Root: sp, BasePath: bp})
			if assert.NoError(t, err) {
				names := namesFromKey(sortref.KeyParts(v.Key), aschema, operations.AllOpRefsByRef(New(sp), nil))
				assert.Equal(t, v.Names, names, "for %s at %d", v.Key, i)
			}
		case spec.Schema:
			aschema, err := Schema(SchemaOpts{Schema: &tv, Root: sp, BasePath: bp})
			if assert.NoError(t, err) {
				names := namesFromKey(sortref.KeyParts(v.Key), aschema, operations.AllOpRefsByRef(New(sp), nil))
				assert.Equal(t, v.Names, names, "for %s at %d", v.Key, i)
			}
		default:
			assert.Fail(t, "unknown type", "got %T", vv)
		}
	}
}

func TestName_BuildNameWithReservedKeyWord(t *testing.T) {
	s := sortref.SplitKey([]string{"definitions", "fullview", "properties", "properties"})
	startIdx := 2
	segments := []string{"fullview"}
	newName := s.BuildName(segments, startIdx, partAdder(nil))
	assert.Equal(t, "fullview properties", newName)

	s = sortref.SplitKey([]string{"definitions", "fullview",
		"properties", "properties", "properties", "properties", "properties", "properties"})
	newName = s.BuildName(segments, startIdx, partAdder(nil))
	assert.Equal(t, "fullview properties properties properties", newName)
}

func TestName_InlinedSchemas(t *testing.T) {
	values := []struct {
		Key      string
		Location string
		Ref      spec.Ref
	}{
		{"#/paths/~1some~1where~1{id}/get/parameters/2/schema/properties/record/items/2/properties/name",
			"#/definitions/getSomeWhereIdParamsBodyRecordItems2/properties/name",
			spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems2Name"),
		},
		{"#/paths/~1some~1where~1{id}/get/parameters/2/schema/properties/record/items/1",
			"#/definitions/getSomeWhereIdParamsBodyRecord/items/1",
			spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems1"),
		},

		{"#/paths/~1some~1where~1{id}/get/parameters/2/schema/properties/record/items/2",
			"#/definitions/getSomeWhereIdParamsBodyRecord/items/2",
			spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems2"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/record/items/2/properties/name",
			"#/definitions/getSomeWhereIdOKBodyRecordItems2/properties/name",
			spec.MustCreateRef("#/definitions/getSomeWhereIdOKBodyRecordItems2Name"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/record/items/1",
			"#/definitions/getSomeWhereIdOKBodyRecord/items/1",
			spec.MustCreateRef("#/definitions/getSomeWhereIdOKBodyRecordItems1"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/record/items/2",
			"#/definitions/getSomeWhereIdOKBodyRecord/items/2",
			spec.MustCreateRef("#/definitions/getSomeWhereIdOKBodyRecordItems2"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/record",
			"#/definitions/getSomeWhereIdOKBody/properties/record",
			spec.MustCreateRef("#/definitions/getSomeWhereIdOKBodyRecord"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/200/schema",
			"#/paths/~1some~1where~1{id}/get/responses/200/schema",
			spec.MustCreateRef("#/definitions/getSomeWhereIdOKBody"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/record/items/2/properties/name",
			"#/definitions/getSomeWhereIdDefaultBodyRecordItems2/properties/name",
			spec.MustCreateRef("#/definitions/getSomeWhereIdDefaultBodyRecordItems2Name"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/record/items/1",
			"#/definitions/getSomeWhereIdDefaultBodyRecord/items/1",
			spec.MustCreateRef("#/definitions/getSomeWhereIdDefaultBodyRecordItems1"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/record/items/2",
			"#/definitions/getSomeWhereIdDefaultBodyRecord/items/2",
			spec.MustCreateRef("#/definitions/getSomeWhereIdDefaultBodyRecordItems2"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/record",
			"#/definitions/getSomeWhereIdDefaultBody/properties/record",
			spec.MustCreateRef("#/definitions/getSomeWhereIdDefaultBodyRecord"),
		},

		{"#/paths/~1some~1where~1{id}/get/responses/default/schema",
			"#/paths/~1some~1where~1{id}/get/responses/default/schema",
			spec.MustCreateRef("#/definitions/getSomeWhereIdDefaultBody"),
		},
		// maps:
		// {"#/definitions/nestedThing/properties/record/items/2/allOf/1/additionalProperties",
		// "#/definitions/nestedThingRecordItems2AllOf1/additionalProperties",
		// spec.MustCreateRef("#/definitions/nestedThingRecordItems2AllOf1AdditionalProperties"),
		// },

		// {"#/definitions/nestedThing/properties/record/items/2/allOf/1",
		// "#/definitions/nestedThingRecordItems2/allOf/1",
		// spec.MustCreateRef("#/definitions/nestedThingRecordItems2AllOf1"),
		// },
		{"#/definitions/nestedThing/properties/record/items/2/properties/name",
			"#/definitions/nestedThingRecordItems2/properties/name",
			spec.MustCreateRef("#/definitions/nestedThingRecordItems2Name"),
		},

		{"#/definitions/nestedThing/properties/record/items/1",
			"#/definitions/nestedThingRecord/items/1",
			spec.MustCreateRef("#/definitions/nestedThingRecordItems1"),
		},

		{"#/definitions/nestedThing/properties/record/items/2",
			"#/definitions/nestedThingRecord/items/2",
			spec.MustCreateRef("#/definitions/nestedThingRecordItems2"),
		},

		{"#/definitions/datedRecords/items/1",
			"#/definitions/datedRecords/items/1",
			spec.MustCreateRef("#/definitions/datedRecordsItems1"),
		},

		{"#/definitions/datedTaggedRecords/items/1",
			"#/definitions/datedTaggedRecords/items/1",
			spec.MustCreateRef("#/definitions/datedTaggedRecordsItems1"),
		},

		{"#/definitions/namedThing/properties/name",
			"#/definitions/namedThing/properties/name",
			spec.MustCreateRef("#/definitions/namedThingName"),
		},

		{"#/definitions/nestedThing/properties/record",
			"#/definitions/nestedThing/properties/record",
			spec.MustCreateRef("#/definitions/nestedThingRecord"),
		},

		{"#/definitions/records/items/0",
			"#/definitions/records/items/0",
			spec.MustCreateRef("#/definitions/recordsItems0"),
		},

		{"#/definitions/datedTaggedRecords/additionalItems",
			"#/definitions/datedTaggedRecords/additionalItems",
			spec.MustCreateRef("#/definitions/datedTaggedRecordsItemsAdditionalItems"),
		},

		{"#/definitions/otherRecords/items",
			"#/definitions/otherRecords/items",
			spec.MustCreateRef("#/definitions/otherRecordsItems"),
		},

		{"#/definitions/tags/additionalProperties",
			"#/definitions/tags/additionalProperties",
			spec.MustCreateRef("#/definitions/tagsAdditionalProperties"),
		},
	}

	bp := filepath.Join("fixtures", "nested_inline_schemas.yml")
	sp := antest.LoadOrFail(t, bp)

	require.NoError(t, spec.ExpandSpec(sp, &spec.ExpandOptions{
		RelativeBase: bp,
		SkipSchemas:  true,
	}))

	require.NoError(t, nameInlinedSchemas(&FlattenOpts{
		Spec:     New(sp),
		BasePath: bp,
	}))

	for i, v := range values {
		ptr, err := jsonpointer.New(v.Location[1:])
		require.NoErrorf(t, err, "at %d for %s", i, v.Key)

		vv, _, err := ptr.Get(sp)
		require.NoErrorf(t, err, "at %d for %s", i, v.Key)

		switch tv := vv.(type) {
		case *spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String(), "at %d for %s", i, v.Key)
		case spec.Schema:
			assert.Equal(t, v.Ref.String(), tv.Ref.String(), "at %d for %s", i, v.Key)
		case *spec.SchemaOrBool:
			var sRef spec.Ref
			if tv != nil && tv.Schema != nil {
				sRef = tv.Schema.Ref
			}
			assert.Equal(t, v.Ref.String(), sRef.String(), "at %d for %s", i, v.Key)
		case *spec.SchemaOrArray:
			var sRef spec.Ref
			if tv != nil && tv.Schema != nil {
				sRef = tv.Schema.Ref
			}
			assert.Equal(t, v.Ref.String(), sRef.String(), "at %d for %s", i, v.Key)
		default:
			assert.Fail(t, "unknown type", "got %T", vv)
		}
	}

	for k, rr := range New(sp).allSchemas {
		if strings.HasPrefix(k, "#/responses") || strings.HasPrefix(k, "#/parameters") {
			continue
		}
		if rr.Schema == nil || rr.Schema.Ref.String() != "" || rr.TopLevel {
			continue
		}
		asch, err := Schema(SchemaOpts{Schema: rr.Schema, Root: sp, BasePath: bp})
		require.NoErrorf(t, err, "for key: %s", k)

		if !asch.IsSimpleSchema && !asch.IsArray && !asch.IsMap {
			assert.Fail(t, "not a top level schema", "for key: %s", k)
		}
	}
}

func TestFlattenSchema_UnitGuards(t *testing.T) {
	t.Parallel()

	parts := sortref.KeyParts("#/nowhere/arbitrary/pointer")
	res := GenLocation(parts)
	assert.Equal(t, "", res)
}

func definitionPtr(key string) string {
	if !strings.HasPrefix(key, "#/definitions") {
		return key
	}

	return strings.Join(strings.Split(key, "/")[:3], "/")
}
