// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/analysis/internal/flatten/operations"
	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	rex    = regexp.MustCompile(`"\$ref":\s*"(.+)"`)
	oairex = regexp.MustCompile(`oiagen`)
)

type refFixture struct {
	Key      string
	Ref      spec.Ref
	Location string
	Expected interface{}
}

func makeRefFixtures() []refFixture {
	return []refFixture{
		{Key: "#/parameters/someParam/schema", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/paths/~1some~1where~1{id}/parameters/1/schema", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/paths/~1some~1where~1{id}/get/parameters/2/schema", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/responses/someResponse/schema", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/paths/~1some~1where~1{id}/get/responses/default/schema", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/paths/~1some~1where~1{id}/get/responses/200/schema", Ref: spec.MustCreateRef("#/definitions/tag")},
		{Key: "#/definitions/namedAgain", Ref: spec.MustCreateRef("#/definitions/named")},
		{Key: "#/definitions/datedTag/allOf/1", Ref: spec.MustCreateRef("#/definitions/tag")},
		{Key: "#/definitions/datedRecords/items/1", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/definitions/datedTaggedRecords/items/1", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/definitions/datedTaggedRecords/additionalItems", Ref: spec.MustCreateRef("#/definitions/tag")},
		{Key: "#/definitions/otherRecords/items", Ref: spec.MustCreateRef("#/definitions/record")},
		{Key: "#/definitions/tags/additionalProperties", Ref: spec.MustCreateRef("#/definitions/tag")},
		{Key: "#/definitions/namedThing/properties/name", Ref: spec.MustCreateRef("#/definitions/named")},
	}
}

func TestFlatten_ImportExternalReferences(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	// this fixture is the same as external_definitions.yml, but no more
	// checks if invalid construct is supported (i.e. $ref in parameters items)
	bp := filepath.Join(".", "fixtures", "external_definitions_valid.yml")
	sp := antest.LoadOrFail(t, bp)

	opts := &FlattenOpts{
		Spec:     New(sp),
		BasePath: bp,
	}

	// NOTE(fredbi): now we no more expand, but merely resolve and iterate until there is no more ext ref
	// so calling importExternalReferences is not idempotent
	_, erx := importExternalReferences(opts)
	require.NoError(t, erx)

	require.Len(t, sp.Definitions, 11)
	require.Contains(t, sp.Definitions, "tag")
	require.Contains(t, sp.Definitions, "named")
	require.Contains(t, sp.Definitions, "record")

	for idx, toPin := range makeRefFixtures() {
		i := idx
		v := toPin
		sp := sp // the pointer passed to Get(node) must be pinned

		t.Run(fmt.Sprintf("import check ref [%d]: %q", i, v.Key), func(t *testing.T) {
			t.Parallel()

			ptr, err := jsonpointer.New(v.Key[1:])
			require.NoErrorf(t, err, "error on jsonpointer.New(%q)", v.Key[1:])

			vv, _, err := ptr.Get(sp)
			require.NoErrorf(t, err, "error on ptr.Get(p for key=%s)", v.Key[1:])

			switch tv := vv.(type) {
			case *spec.Schema:
				require.Equal(t, v.Ref.String(), tv.Ref.String(), "for %s", v.Key)

			case spec.Schema:
				require.Equal(t, v.Ref.String(), tv.Ref.String(), "for %s", v.Key)

			case *spec.SchemaOrBool:
				require.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "for %s", v.Key)

			case *spec.SchemaOrArray:
				require.Equal(t, v.Ref.String(), tv.Schema.Ref.String(), "for %s", v.Key)

			default:
				require.Fail(t, "unknown type", "got %T", vv)
			}
		})
	}

	// check the complete result for clarity
	jazon := antest.AsJSON(t, sp)

	expected, err := ioutil.ReadFile(filepath.Join("fixtures", "expected", "external-references-1.json"))
	require.NoError(t, err)

	assert.JSONEq(t, string(expected), jazon)

	// iterate again: this time all external schema $ref's should be reinlined
	opts.Spec.reload()

	_, err = importExternalReferences(&FlattenOpts{
		Spec:     New(sp),
		BasePath: bp,
	})
	require.NoError(t, err)

	opts.Spec.reload()
	for _, ref := range opts.Spec.references.schemas {
		require.True(t, ref.HasFragmentOnly)
	}

	// now try complete flatten
	sp = antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: true}))

	jazon = antest.AsJSON(t, an.spec)

	expected, err = ioutil.ReadFile(filepath.Join("fixtures", "expected", "external-references-2.json"))
	require.NoError(t, err)

	assert.JSONEq(t, string(expected), jazon)
}

func makeFlattenFixtures() []refFixture {
	return []refFixture{
		{
			Key:      "#/responses/notFound/schema",
			Location: "#/responses/notFound/schema",
			Ref:      spec.MustCreateRef("#/definitions/error"),
			Expected: nil,
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/parameters/0",
			Location: "#/paths/~1some~1where~1{id}/parameters/0/name",
			Ref:      spec.Ref{},
			Expected: "id",
		},
		{
			Key:      "#/paths/~1other~1place",
			Location: "#/paths/~1other~1place/get/operationId",
			Ref:      spec.Ref{},
			Expected: "modelOp",
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/get/parameters/0",
			Location: "#/paths/~1some~1where~1{id}/get/parameters/0/name",
			Ref:      spec.Ref{},
			Expected: "limit",
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/get/parameters/1",
			Location: "#/paths/~1some~1where~1{id}/get/parameters/1/name",
			Ref:      spec.Ref{},
			Expected: "some",
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/get/parameters/2",
			Location: "#/paths/~1some~1where~1{id}/get/parameters/2/name",
			Ref:      spec.Ref{},
			Expected: "other",
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/get/parameters/3",
			Location: "#/paths/~1some~1where~1{id}/get/parameters/3/schema",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBody"),
			Expected: "",
		},
		{
			Key:      "#/paths/~1some~1where~1{id}/get/responses/200",
			Location: "#/paths/~1some~1where~1{id}/get/responses/200/schema",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdOKBody"),
			Expected: "",
		},
		{
			Key:      "#/definitions/namedAgain",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/named"),
			Expected: "",
		},
		{
			Key:      "#/definitions/namedThing/properties/name",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/named"),
			Expected: "",
		},
		{
			Key:      "#/definitions/namedThing/properties/namedAgain",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/namedAgain"),
			Expected: "",
		},
		{
			Key:      "#/definitions/datedRecords/items/1",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/record"),
			Expected: "",
		},
		{
			Key:      "#/definitions/otherRecords/items",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/record"),
			Expected: "",
		},
		{
			Key:      "#/definitions/tags/additionalProperties",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/tag"),
			Expected: "",
		},
		{
			Key:      "#/definitions/datedTag/allOf/1",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/tag"),
			Expected: "",
		},
		{
			Key:      "#/definitions/nestedThingRecord/items/1",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/nestedThingRecordItems1"),
			Expected: "",
		},
		{
			Key:      "#/definitions/nestedThingRecord/items/2",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/nestedThingRecordItems2"),
			Expected: "",
		},
		{
			Key:      "#/definitions/nestedThing/properties/record",
			Location: "",
			Ref:      spec.MustCreateRef("#/definitions/nestedThingRecord"),
			Expected: "",
		},
		{
			Key:      "#/definitions/named",
			Location: "#/definitions/named/type",
			Ref:      spec.Ref{},
			Expected: spec.StringOrArray{"string"},
		},
		{
			Key:      "#/definitions/error",
			Location: "#/definitions/error/properties/id/type",
			Ref:      spec.Ref{},
			Expected: spec.StringOrArray{"integer"},
		},
		{
			Key:      "#/definitions/record",
			Location: "#/definitions/record/properties/createdAt/format",
			Ref:      spec.Ref{},
			Expected: "date-time",
		},
		{
			Key:      "#/definitions/getSomeWhereIdOKBody",
			Location: "#/definitions/getSomeWhereIdOKBody/properties/record",
			Ref:      spec.MustCreateRef("#/definitions/nestedThing"),
			Expected: nil,
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBody",
			Location: "#/definitions/getSomeWhereIdParamsBody/properties/record",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecord"),
			Expected: nil,
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBodyRecord",
			Location: "#/definitions/getSomeWhereIdParamsBodyRecord/items/1",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems1"),
			Expected: nil,
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBodyRecord",
			Location: "#/definitions/getSomeWhereIdParamsBodyRecord/items/2",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems2"),
			Expected: nil,
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBodyRecordItems2",
			Location: "#/definitions/getSomeWhereIdParamsBodyRecordItems2/allOf/0/format",
			Ref:      spec.Ref{},
			Expected: "date",
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBodyRecordItems2Name",
			Location: "#/definitions/getSomeWhereIdParamsBodyRecordItems2Name/properties/createdAt/format",
			Ref:      spec.Ref{},
			Expected: "date-time",
		},
		{
			Key:      "#/definitions/getSomeWhereIdParamsBodyRecordItems2",
			Location: "#/definitions/getSomeWhereIdParamsBodyRecordItems2/properties/name",
			Ref:      spec.MustCreateRef("#/definitions/getSomeWhereIdParamsBodyRecordItems2Name"),
			Expected: "date",
		},
	}
}

func TestFlatten_CheckRef(t *testing.T) {
	bp := filepath.Join("fixtures", "flatten.yml")
	sp := antest.LoadOrFail(t, bp)

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp}))

	for idx, toPin := range makeFlattenFixtures() {
		i := idx
		v := toPin
		sp := sp

		t.Run(fmt.Sprintf("check ref after flatten %q", v.Key), func(t *testing.T) {
			pk := v.Key[1:]
			if v.Location != "" {
				pk = v.Location[1:]
			}

			ptr, err := jsonpointer.New(pk)
			require.NoError(t, err, "at %d for %s", i, v.Key)

			d, _, err := ptr.Get(sp)
			require.NoError(t, err)

			if v.Ref.String() == "" {
				assert.Equal(t, v.Expected, d)

				return
			}

			switch s := d.(type) {
			case *spec.Schema:
				assert.Equal(t, v.Ref.String(), s.Ref.String(), "at %d for %s", i, v.Key)

			case spec.Schema:
				assert.Equal(t, v.Ref.String(), s.Ref.String(), "at %d for %s", i, v.Key)

			case *spec.SchemaOrArray:
				var sRef spec.Ref
				if s != nil && s.Schema != nil {
					sRef = s.Schema.Ref
				}
				assert.Equal(t, v.Ref.String(), sRef.String(), "at %d for %s", i, v.Key)

			case *spec.SchemaOrBool:
				var sRef spec.Ref
				if s != nil && s.Schema != nil {
					sRef = s.Schema.Ref
				}
				assert.Equal(t, v.Ref.String(), sRef.String(), "at %d for %s", i, v.Key)

			default:
				assert.Fail(t, "unknown type", "got %T at %d for %s", d, i, v.Key)
			}
		})
	}
}

func TestFlatten_FullWithOAIGen(t *testing.T) {
	bp := filepath.Join("fixtures", "oaigen", "fixture-oaigen.yaml")
	sp := antest.LoadOrFail(t, bp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: New(sp), BasePath: bp, Verbose: true,
		Minimal: false, RemoveUnused: false,
	}))

	res := getInPath(t, sp, "/some/where", "/get/responses/204/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/uniqueName1"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/204/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/d"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/206/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/a"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/304/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/transitive11"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/205/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/b"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/200/schema")
	assert.JSONEqf(t, `{"type": "integer"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/default/schema")
	// pointer expanded
	assert.JSONEqf(t, `{"type": "integer"}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "a")
	assert.JSONEqf(t,
		`{"type": "object", "properties": { "a": { "$ref": "#/definitions/aAOAIGen" }}}`,
		res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "aA")
	assert.JSONEqf(t, `{"type": "string", "format": "date"}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "aAOAIGen")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "properties": {
		"b": {
		  "type": "integer"
		}
	  },
	  "x-go-gen-location": "models"
    }
	`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "bB")
	assert.JSONEqf(t, `{"type": "string", "format": "date-time"}`, res, "Expected a simple schema for response")

	_, ok := sp.Definitions["bItems"]
	assert.Falsef(t, ok, "Did not expect a definition for %s", "bItems")

	res = getDefinition(t, sp, "d")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "properties": {
	    "c": {
		  "type": "integer"
		}
      }
    }
   `, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "b")
	assert.JSONEqf(t, `
	{
	  "type": "array",
	  "items": {
	    "$ref": "#/definitions/d"
	  }
	}
	`, res, "Expected a ref in response")

	res = getDefinition(t, sp, "myBody")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "properties": {
	    "aA": {
		  "$ref": "#/definitions/aA"
		},
		"prop1": {
		  "type": "integer"
	    }
      }
	}
	`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "uniqueName2")
	assert.JSONEqf(t, `{"$ref": "#/definitions/notUniqueName2"}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "notUniqueName2")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "properties": {
		"prop6": {
		  "type": "integer"
		}
	  }
	}
	`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "uniqueName1")
	assert.JSONEqf(t, `{
		   "type": "object",
		   "properties": {
		    "prop5": {
		     "type": "integer"
		    }}}`, res, "Expected a simple schema for response")

	// allOf container: []spec.Schema
	res = getDefinition(t, sp, "getWithSliceContainerDefaultBody")
	assert.JSONEqf(t, `{
		"allOf": [
		    {
		     "$ref": "#/definitions/uniqueName3"
		    },
		    {
		     "$ref": "#/definitions/getWithSliceContainerDefaultBodyAllOf1"
		    }
		   ],
		   "x-go-gen-location": "operations"
		    }`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "getWithSliceContainerDefaultBodyAllOf1")
	assert.JSONEqf(t, `{
		"type": "object",
		   "properties": {
		    "prop8": {
		     "type": "string"
		    }
		   },
		   "x-go-gen-location": "models"
		    }`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "getWithTupleContainerDefaultBody")
	assert.JSONEqf(t, `{
		   "type": "array",
		   "items": [
		    {
		     "$ref": "#/definitions/uniqueName3"
		    },
		    {
		     "$ref": "#/definitions/getWithSliceContainerDefaultBodyAllOf1"
		    }
		   ],
		   "x-go-gen-location": "operations"
		    }`, res, "Expected a simple schema for response")

	// with container SchemaOrArray
	res = getDefinition(t, sp, "getWithTupleConflictDefaultBody")
	assert.JSONEqf(t, `{
		   "type": "array",
		   "items": [
		    {
		     "$ref": "#/definitions/uniqueName4"
		    },
		    {
		     "$ref": "#/definitions/getWithTupleConflictDefaultBodyItems1"
		    }
		   ],
		   "x-go-gen-location": "operations"
	}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "getWithTupleConflictDefaultBodyItems1")
	assert.JSONEqf(t, `{
		   "type": "object",
		   "properties": {
		    "prop10": {
		     "type": "string"
		    }
		   },
		   "x-go-gen-location": "models"
	}`, res, "Expected a simple schema for response")
}

func TestFlatten_MinimalWithOAIGen(t *testing.T) {
	var sp *spec.Swagger
	defer func() {
		if t.Failed() && sp != nil {
			t.Log(antest.AsJSON(t, sp))
		}
	}()

	bp := filepath.Join("fixtures", "oaigen", "fixture-oaigen.yaml")
	sp = antest.LoadOrFail(t, bp)

	var logCapture bytes.Buffer
	log.SetOutput(&logCapture)
	defer log.SetOutput(os.Stdout)

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))

	msg := logCapture.String()
	if !assert.NotContainsf(t, msg,
		"warning: duplicate flattened definition name resolved as aAOAIGen", "Expected log message") {
		t.Logf("Captured log: %s", msg)
	}
	if !assert.NotContainsf(t, msg,
		"warning: duplicate flattened definition name resolved as uniqueName2OAIGen", "Expected log message") {
		t.Logf("Captured log: %s", msg)
	}
	res := getInPath(t, sp, "/some/where", "/get/responses/204/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/uniqueName1"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/204/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/d"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/206/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/a"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/304/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/transitive11"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/get/responses/205/schema")
	assert.JSONEqf(t, `{"$ref": "#/definitions/b"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/200/schema")
	assert.JSONEqf(t, `{"type": "integer"}`, res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where", "/post/responses/default/schema")
	// This JSON pointer is expanded
	assert.JSONEqf(t, `{"type": "integer"}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "aA")
	assert.JSONEqf(t, `{"type": "string", "format": "date"}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "a")
	assert.JSONEqf(t, `{
		   "type": "object",
		   "properties": {
		    "a": {
		     "type": "object",
		     "properties": {
		      "b": {
		       "type": "integer"
		      }
		     }
		    }
		   }
		  }`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "bB")
	assert.JSONEqf(t, `{"type": "string", "format": "date-time"}`, res, "Expected a simple schema for response")

	_, ok := sp.Definitions["bItems"]
	assert.Falsef(t, ok, "Did not expect a definition for %s", "bItems")

	res = getDefinition(t, sp, "d")
	assert.JSONEqf(t, `{
		   "type": "object",
		   "properties": {
		    "c": {
		     "type": "integer"
		    }
		   }
	}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "b")
	assert.JSONEqf(t, `{
		   "type": "array",
		   "items": {
			   "$ref": "#/definitions/d"
		   }
	}`, res, "Expected a ref in response")

	res = getDefinition(t, sp, "myBody")
	assert.JSONEqf(t, `{
		   "type": "object",
		   "properties": {
		    "aA": {
		     "$ref": "#/definitions/aA"
		    },
		    "prop1": {
		     "type": "integer"
		    }
		   }
	}`, res, "Expected a simple schema for response")

	res = getDefinition(t, sp, "uniqueName2")
	assert.JSONEqf(t, `{"$ref": "#/definitions/notUniqueName2"}`, res, "Expected a simple schema for response")

	// with allOf container: []spec.Schema
	res = getInPath(t, sp, "/with/slice/container", "/get/responses/default/schema")
	assert.JSONEqf(t, `{
 			"allOf": [
		        {
		         "$ref": "#/definitions/uniqueName3"
		        },
				{
			     "$ref": "#/definitions/getWithSliceContainerDefaultBodyAllOf1"
				}
		       ]
	}`, res, "Expected a simple schema for response")

	// with tuple container
	res = getInPath(t, sp, "/with/tuple/container", "/get/responses/default/schema")
	assert.JSONEqf(t, `{
		       "type": "array",
		       "items": [
		        {
		         "$ref": "#/definitions/uniqueName3"
		        },
		        {
		         "$ref": "#/definitions/getWithSliceContainerDefaultBodyAllOf1"
		        }
		       ]
	}`, res, "Expected a simple schema for response")

	// with SchemaOrArray container
	res = getInPath(t, sp, "/with/tuple/conflict", "/get/responses/default/schema")
	assert.JSONEqf(t, `{
		       "type": "array",
		       "items": [
		        {
		         "$ref": "#/definitions/uniqueName4"
		        },
		        {
		         "type": "object",
		         "properties": {
		          "prop10": {
		           "type": "string"
		          }
		         }
		        }
		       ]
	}`, res, "Expected a simple schema for response")
}

func assertNoOAIGen(t *testing.T, bp string, sp *spec.Swagger) (success bool) {
	var logCapture bytes.Buffer
	log.SetOutput(&logCapture)
	defer log.SetOutput(os.Stdout)

	defer func() {
		success = !t.Failed()
	}()

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: false}))

	msg := logCapture.String()
	assert.NotContains(t, msg, "warning")

	for k := range sp.Definitions {
		require.NotContains(t, k, "OAIGen")
	}

	return
}

func TestFlatten_OAIGen(t *testing.T) {
	for _, fixture := range []string{
		filepath.Join("fixtures", "oaigen", "test3-swagger.yaml"),
		filepath.Join("fixtures", "oaigen", "test3-bis-swagger.yaml"),
		filepath.Join("fixtures", "oaigen", "test3-ter-swagger.yaml"),
	} {
		t.Run(fmt.Sprintf("flatten_oiagen_1260_%s", fixture), func(t *testing.T) {
			t.Parallel()

			bp := filepath.Join("fixtures", "oaigen", "test3-swagger.yaml")
			sp := antest.LoadOrFail(t, bp)

			require.Truef(t, assertNoOAIGen(t, bp, sp), "did not expect an OAIGen definition here")
		})
	}
}

func TestMoreNameInlinedSchemas(t *testing.T) {
	bp := filepath.Join("fixtures", "more_nested_inline_schemas.yml")
	sp := antest.LoadOrFail(t, bp)

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: false}))

	res := getInPath(t, sp, "/some/where/{id}", "/post/responses/200/schema")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "additionalProperties": {
		"type": "object",
		"additionalProperties": {
		  "type": "object", "additionalProperties": {
			"$ref": "#/definitions/postSomeWhereIdOKBodyAdditionalPropertiesAdditionalPropertiesAdditionalProperties"
		  }
	    }
      }
	}`,
		res, "Expected a simple schema for response")

	res = getInPath(t, sp, "/some/where/{id}", "/post/responses/204/schema")
	assert.JSONEqf(t, `
	{
	  "type": "object",
	  "additionalProperties": {
		 "type": "array",
		 "items": {
		   "type": "object",
		   "additionalProperties": {
		     "type": "array",
		     "items": {
		       "type": "object",
		       "additionalProperties": {
		         "type": "array",
		         "items": {
				   "$ref": "#/definitions/postSomeWhereIdNoContentBodyAdditionalPropertiesItemsAdditionalPropertiesItemsAdditionalPropertiesItems"
		         }
		       }
		     }
		   }
		 }
	   }
	 }
`, res, "Expected a simple schema for response")
}

func TestRemoveUnused(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "oaigen", "fixture-oaigen.yaml")
	sp := antest.LoadOrFail(t, bp)

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Verbose: false, Minimal: true, RemoveUnused: true}))

	assert.Nil(t, sp.Parameters)
	assert.Nil(t, sp.Responses)

	bp = filepath.Join("fixtures", "parameters", "fixture-parameters.yaml")
	sp = antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: false, Minimal: true, RemoveUnused: true}))

	assert.Nil(t, sp.Parameters)
	assert.Nil(t, sp.Responses)

	op, ok := an.OperationFor("GET", "/some/where")
	assert.True(t, ok)
	assert.Lenf(t, op.Parameters, 4, "Expected 4 parameters expanded for this operation")
	assert.Lenf(t, an.ParamsFor("GET", "/some/where"), 7, "Expected 7 parameters (with default) expanded for this operation")

	op, ok = an.OperationFor("PATCH", "/some/remote")
	assert.True(t, ok)
	assert.Lenf(t, op.Parameters, 1, "Expected 1 parameter expanded for this operation")
	assert.Lenf(t, an.ParamsFor("PATCH", "/some/remote"), 2, "Expected 2 parameters (with default) expanded for this operation")

	_, ok = sp.Definitions["unused"]
	assert.False(t, ok, "Did not expect to find #/definitions/unused")

	bp = filepath.Join("fixtures", "parameters", "fixture-parameters.yaml")
	sp = antest.LoadOrFail(t, bp)

	require.NoError(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: true}))

	assert.Nil(t, sp.Parameters)
	assert.Nil(t, sp.Responses)
	_, ok = sp.Definitions["unused"]
	assert.Falsef(t, ok, "Did not expect to find #/definitions/unused")
}

func TestOperationIDs(t *testing.T) {
	bp := filepath.Join("fixtures", "operations", "fixture-operations.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: false, Minimal: false, RemoveUnused: false}))

	res := operations.GatherOperations(New(sp), []string{"getSomeWhere", "getSomeWhereElse"})
	_, ok := res["getSomeWhere"]
	assert.Truef(t, ok, "Expected to find operation")
	_, ok = res["getSomeWhereElse"]
	assert.Truef(t, ok, "Expected to find operation")
	_, ok = res["postSomeWhere"]
	assert.Falsef(t, ok, "Did not expect to find operation")

	op, ok := an.OperationFor("GET", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("getSomeWhereElse"), 2)

	op, ok = an.OperationFor("POST", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("postSomeWhereElse"), 1)

	op, ok = an.OperationFor("PUT", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("putSomeWhereElse"), 1)

	op, ok = an.OperationFor("PATCH", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("patchSomeWhereElse"), 1)

	op, ok = an.OperationFor("DELETE", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("deleteSomeWhereElse"), 1)

	op, ok = an.OperationFor("HEAD", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("headSomeWhereElse"), 1)

	op, ok = an.OperationFor("OPTIONS", "/some/where/else")
	assert.True(t, ok)
	assert.NotNil(t, op)
	assert.Len(t, an.ParametersFor("optionsSomeWhereElse"), 1)

	assert.Len(t, an.ParametersFor("outOfThisWorld"), 0)
}

func TestFlatten_Pointers(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "pointers", "fixture-pointers.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))

	// re-analyse and check all $ref's point to #/definitions
	bn := New(sp)
	for _, r := range bn.AllRefs() {
		assert.True(t, path.Dir(r.String()) == definitionsPath)
	}
}

// unit test guards in flatten not easily testable with actual specs
func TestFlatten_ErrorHandling(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	const wantedFailure = "Expected a failure"
	bp := filepath.Join("fixtures", "errors", "fixture-unexpandable.yaml")

	// invalid spec expansion
	sp := antest.LoadOrFail(t, bp)
	require.Errorf(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Expand: true}), wantedFailure)

	// reload original spec
	sp = antest.LoadOrFail(t, bp)
	require.Errorf(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Expand: false}), wantedFailure)

	bp = filepath.Join("fixtures", "errors", "fixture-unexpandable-2.yaml")
	sp = antest.LoadOrFail(t, bp)
	require.Errorf(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Expand: false}), wantedFailure)

	// reload original spec
	sp = antest.LoadOrFail(t, bp)
	require.Errorf(t, Flatten(FlattenOpts{Spec: New(sp), BasePath: bp, Minimal: true, Expand: false}), wantedFailure)
}

func TestFlatten_PointersLoop(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "pointers", "fixture-pointers-loop.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.Error(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))
}

func TestFlatten_Bitbucket(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "bugs", "bitbucket.json")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))

	// reload original spec
	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: false}))

	// reload original spec
	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Expand: true, RemoveUnused: false}))

	// reload original spec
	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Expand: true, RemoveUnused: true}))

	assert.Len(t, sp.Definitions, 2) // only 2 remaining refs after expansion: circular $ref
	_, ok := sp.Definitions["base_commit"]
	assert.True(t, ok)
	_, ok = sp.Definitions["repository"]
	assert.True(t, ok)
}

func TestFlatten_Issue_1602(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	// $ref as schema to #/responses or #/parameters

	// minimal repro test case
	bp := filepath.Join("fixtures", "bugs", "1602", "fixture-1602-1.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      true,
		Expand:       false,
		RemoveUnused: false,
	}))

	// reload spec
	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: false,
		Minimal:      false,
		Expand:       false,
		RemoveUnused: false,
	}))

	// reload spec
	// with  prior expansion, a pseudo schema is produced
	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: false,
		Minimal:      false,
		Expand:       true,
		RemoveUnused: false,
	}))
}

func TestFlatten_Issue_1602_All(t *testing.T) {
	for _, fixture := range []string{
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-full.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-1.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-2.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-3.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-4.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-5.yaml"),
		filepath.Join("fixtures", "bugs", "1602", "fixture-1602-6.yaml"),
	} {
		t.Run(fmt.Sprintf("issue_1602_all_%s", fixture), func(t *testing.T) {
			t.Parallel()
			sp := antest.LoadOrFail(t, fixture)

			an := New(sp)
			require.NoError(t, Flatten(FlattenOpts{
				Spec: an, BasePath: fixture, Verbose: false, Minimal: true, Expand: false,
				RemoveUnused: false,
			}))
		})
	}
}

func TestFlatten_Issue_1614(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	// $ref as schema to #/responses or #/parameters
	// test warnings

	bp := filepath.Join("fixtures", "bugs", "1614", "gitea.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, Expand: false,
		RemoveUnused: false}))

	// check that responses subject to warning have been expanded
	jazon := antest.AsJSON(t, sp)
	assert.NotContains(t, jazon, `#/responses/forbidden`)
	assert.NotContains(t, jazon, `#/responses/empty`)
}

func TestFlatten_Issue_1621(t *testing.T) {
	// repeated remote refs

	// minimal repro test case
	bp := filepath.Join("fixtures", "bugs", "1621", "fixture-1621.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      true,
		Expand:       false,
		RemoveUnused: false,
	}))

	sch1 := sp.Paths.Paths["/v4/users/"].Get.Responses.StatusCodeResponses[200].Schema
	jazon := antest.AsJSON(t, sch1)
	assert.JSONEq(t, `{"type": "array", "items": {"$ref": "#/definitions/v4UserListItem" }}`, jazon)

	sch2 := sp.Paths.Paths["/v4/user/"].Get.Responses.StatusCodeResponses[200].Schema
	jazon = antest.AsJSON(t, sch2)
	assert.JSONEq(t, `{"$ref": "#/definitions/v4UserListItem"}`, jazon)

	sch3 := sp.Paths.Paths["/v4/users/{email}/"].Get.Responses.StatusCodeResponses[200].Schema
	jazon = antest.AsJSON(t, sch3)
	assert.JSONEq(t, `{"$ref": "#/definitions/v4UserListItem"}`, jazon)
}

func TestFlatten_Issue_1796(t *testing.T) {
	// remote cyclic ref
	bp := filepath.Join("fixtures", "bugs", "1796", "queryIssue.json")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal: true, Expand: false,
		RemoveUnused: false,
	}))

	// assert all $ref match  "$ref": "#/definitions/something"
	for _, ref := range an.AllReferences() {
		assert.True(t, strings.HasPrefix(ref, "#/definitions"))
	}
}

func TestFlatten_Issue_1767(t *testing.T) {
	// remote cyclic ref again
	bp := filepath.Join("fixtures", "bugs", "1767", "fixture-1767.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal: true, Expand: false,
		RemoveUnused: false,
	}))

	// assert all $ref match  "$ref": "#/definitions/something"
	for _, ref := range an.AllReferences() {
		assert.True(t, strings.HasPrefix(ref, "#/definitions"))
	}
}

func TestFlatten_Issue_1774(t *testing.T) {
	// remote cyclic ref again
	bp := filepath.Join("fixtures", "bugs", "1774", "def_api.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      false,
		Expand:       false,
		RemoveUnused: false,
	}))

	// assert all $ref match  "$ref": "#/definitions/something"
	for _, ref := range an.AllReferences() {
		assert.True(t, strings.HasPrefix(ref, "#/definitions"))
	}
}

func TestFlatten_1429(t *testing.T) {
	// nested / remote $ref in response / param schemas
	// issue go-swagger/go-swagger#1429
	bp := filepath.Join("fixtures", "bugs", "1429", "swagger.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      true,
		RemoveUnused: false,
	}))
}

func TestFlatten_1851(t *testing.T) {
	// nested / remote $ref in response / param schemas
	// issue go-swagger/go-swagger#1851
	bp := filepath.Join("fixtures", "bugs", "1851", "fixture-1851.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      true,
		RemoveUnused: false,
	}))

	serverDefinition, ok := an.spec.Definitions["server"]
	assert.True(t, ok)

	serverStatusDefinition, ok := an.spec.Definitions["serverStatus"]
	assert.True(t, ok)

	serverStatusProperty, ok := serverDefinition.Properties["Status"]
	assert.True(t, ok)

	jazon := antest.AsJSON(t, serverStatusProperty)
	assert.JSONEq(t, `{"$ref": "#/definitions/serverStatus"}`, jazon)

	jazon = antest.AsJSON(t, serverStatusDefinition)
	assert.JSONEq(t, `{"type": "string", "enum": [ "OK", "Not OK" ]}`, jazon)

	// additional test case: this one used to work
	bp = filepath.Join("fixtures", "bugs", "1851", "fixture-1851-2.yaml")
	sp = antest.LoadOrFail(t, bp)

	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))

	serverDefinition, ok = an.spec.Definitions["Server"]
	assert.True(t, ok)

	serverStatusDefinition, ok = an.spec.Definitions["ServerStatus"]
	assert.True(t, ok)

	serverStatusProperty, ok = serverDefinition.Properties["Status"]
	assert.True(t, ok)

	jazon = antest.AsJSON(t, serverStatusProperty)
	assert.JSONEq(t, `{"$ref": "#/definitions/ServerStatus"}`, jazon)

	jazon = antest.AsJSON(t, serverStatusDefinition)
	assert.JSONEq(t, `{"type": "string", "enum": [ "OK", "Not OK" ]}`, jazon)
}

func TestFlatten_RemoteAbsolute(t *testing.T) {
	for _, fixture := range []string{
		// this one has simple remote ref pattern
		filepath.Join("fixtures", "bugs", "remote-absolute", "swagger-mini.json"),
		// this has no remote ref
		filepath.Join("fixtures", "bugs", "remote-absolute", "swagger.json"),
		// this one has local ref, no naming conflict (same as previous but with external ref imported)
		filepath.Join("fixtures", "bugs", "remote-absolute", "swagger-with-local-ref.json"),
		// this one has remote ref, no naming conflict (same as previous but with external ref imported)
		filepath.Join("fixtures", "bugs", "remote-absolute", "swagger-with-remote-only-ref.json"),
	} {
		t.Run(fmt.Sprintf("remote_absolute_%s", fixture), func(t *testing.T) {
			t.Parallel()

			an := testFlattenWithDefaults(t, fixture)
			checkRefs(t, an.spec, true)
		})
	}

	// This one has both remote and local ref with naming conflict.
	// This creates some "oiagen" definitions to address naming conflict,
	// which are removed by the oaigen pruning process (reinlined / merged with parents).
	an := testFlattenWithDefaults(t, filepath.Join("fixtures", "bugs", "remote-absolute", "swagger-with-ref.json"))
	checkRefs(t, an.spec, false)
}

func TestFlatten_2092(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "bugs", "2092", "swagger.yaml")
	rexOAIGen := regexp.MustCompile(`(?i)("\$ref":\s*")(.?oaigen.?)"`)

	// #2092 exhibits a stability issue: repeat 100 times the process to make sure it is stable
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))
	firstJSONMinimal := antest.AsJSON(t, an.spec)

	// verify we don't have dangling oaigen refs
	require.Falsef(t, rexOAIGen.MatchString(firstJSONMinimal), "unmatched regexp for: %s", firstJSONMinimal)

	sp = antest.LoadOrFail(t, bp)
	an = New(sp)
	require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: false}))
	firstJSONFull := antest.AsJSON(t, an.spec)

	// verify we don't have dangling oaigen refs
	require.Falsef(t, rexOAIGen.MatchString(firstJSONFull), "unmatched regexp for: %s", firstJSONFull)

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("issue_2092_%d", i), func(t *testing.T) {
			t.Parallel()

			// verify that we produce a stable result
			sp := antest.LoadOrFail(t, bp)
			an := New(sp)

			require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: false}))

			jazon := antest.AsJSON(t, an.spec)
			assert.JSONEq(t, firstJSONMinimal, jazon)

			require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: true, RemoveUnused: true}))

			sp = antest.LoadOrFail(t, bp)
			an = New(sp)

			require.NoError(t, Flatten(FlattenOpts{Spec: an, BasePath: bp, Verbose: true, Minimal: false, RemoveUnused: false}))

			jazon = antest.AsJSON(t, an.spec)
			assert.JSONEq(t, firstJSONFull, jazon)
		})
	}
}

func TestFlatten_2113(t *testing.T) {
	// flatten $ref under path
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	bp := filepath.Join("fixtures", "bugs", "2113", "base.yaml")
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Expand:       true,
		RemoveUnused: false,
	}))

	sp = antest.LoadOrFail(t, bp)
	an = New(sp)

	require.NoError(t, Flatten(FlattenOpts{
		Spec: an, BasePath: bp, Verbose: true,
		Minimal:      true,
		RemoveUnused: false,
	}))

	jazon := antest.AsJSON(t, sp)

	expected, err := ioutil.ReadFile(filepath.Join("fixtures", "expected", "issue-2113.json"))
	require.NoError(t, err)

	require.JSONEq(t, string(expected), jazon)
}

func getDefinition(t testing.TB, sp *spec.Swagger, key string) string {
	d, ok := sp.Definitions[key]
	require.Truef(t, ok, "Expected definition for %s", key)
	res, _ := json.Marshal(d)

	return string(res)
}

func getInPath(t testing.TB, sp *spec.Swagger, path, key string) string {
	ptr, erp := jsonpointer.New(key)
	require.NoError(t, erp, "at %s no key", key)

	d, _, erg := ptr.Get(sp.Paths.Paths[path])
	require.NoError(t, erg, "at %s no value for %s", path, key)

	res, _ := json.Marshal(d)

	return string(res)
}

func checkRefs(t testing.TB, spec *spec.Swagger, expectNoConflict bool) {
	// all $ref resolve locally
	jazon := antest.AsJSON(t, spec)
	m := rex.FindAllStringSubmatch(jazon, -1)
	require.NotNil(t, m)

	for _, matched := range m {
		subMatch := matched[1]
		assert.True(t, strings.HasPrefix(subMatch, "#/definitions/"),
			"expected $ref to be inlined, got: %s", matched[0])
	}

	if expectNoConflict {
		// no naming conflict
		m := oairex.FindAllStringSubmatch(jazon, -1)
		assert.Empty(t, m)
	}
}

func testFlattenWithDefaults(t *testing.T, bp string) *Spec {
	sp := antest.LoadOrFail(t, bp)
	an := New(sp)
	require.NoError(t, Flatten(FlattenOpts{
		Spec:         an,
		BasePath:     bp,
		Verbose:      true,
		Minimal:      true,
		RemoveUnused: false,
	}))

	return an
}
