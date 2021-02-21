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
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer_All(t *testing.T) {
	t.Parallel()

	formatParam := spec.QueryParam("format").Typed("string", "")

	limitParam := spec.QueryParam("limit").Typed("integer", "int32")
	limitParam.Extensions = spec.Extensions(map[string]interface{}{})
	limitParam.Extensions.Add("go-name", "Limit")

	skipParam := spec.QueryParam("skip").Typed("integer", "int32")
	pi := spec.PathItem{}
	pi.Parameters = []spec.Parameter{*limitParam}

	op := &spec.Operation{}
	op.Consumes = []string{"application/x-yaml"}
	op.Produces = []string{"application/x-yaml"}
	op.Security = []map[string][]string{
		{"oauth2": {}},
		{"basic": nil},
	}

	op.ID = "someOperation"
	op.Parameters = []spec.Parameter{*skipParam}
	pi.Get = op

	pi2 := spec.PathItem{}
	pi2.Parameters = []spec.Parameter{*limitParam}
	op2 := &spec.Operation{}
	op2.ID = "anotherOperation"
	op2.Parameters = []spec.Parameter{*skipParam}
	pi2.Get = op2

	spec := makeFixturepec(pi, pi2, formatParam)
	analyzer := New(spec)

	assert.Len(t, analyzer.consumes, 2)
	assert.Len(t, analyzer.produces, 2)
	assert.Len(t, analyzer.operations, 1)
	assert.Equal(t, analyzer.operations["GET"]["/"], spec.Paths.Paths["/"].Get)

	expected := []string{"application/x-yaml"}
	sort.Strings(expected)
	consumes := analyzer.ConsumesFor(spec.Paths.Paths["/"].Get)
	sort.Strings(consumes)
	assert.Equal(t, expected, consumes)

	produces := analyzer.ProducesFor(spec.Paths.Paths["/"].Get)
	sort.Strings(produces)
	assert.Equal(t, expected, produces)

	expected = []string{"application/json"}
	sort.Strings(expected)
	consumes = analyzer.ConsumesFor(spec.Paths.Paths["/items"].Get)
	sort.Strings(consumes)
	assert.Equal(t, expected, consumes)

	produces = analyzer.ProducesFor(spec.Paths.Paths["/items"].Get)
	sort.Strings(produces)
	assert.Equal(t, expected, produces)

	expectedSchemes := [][]SecurityRequirement{
		{
			{Name: "oauth2", Scopes: []string{}},
			{Name: "basic", Scopes: nil},
		},
	}
	schemes := analyzer.SecurityRequirementsFor(spec.Paths.Paths["/"].Get)
	assert.Equal(t, schemeNames(expectedSchemes), schemeNames(schemes))

	securityDefinitions := analyzer.SecurityDefinitionsFor(spec.Paths.Paths["/"].Get)
	assert.Equal(t, *spec.SecurityDefinitions["basic"], securityDefinitions["basic"])
	assert.Equal(t, *spec.SecurityDefinitions["oauth2"], securityDefinitions["oauth2"])

	parameters := analyzer.ParamsFor("GET", "/")
	assert.Len(t, parameters, 2)

	operations := analyzer.OperationIDs()
	assert.Len(t, operations, 2)

	producers := analyzer.RequiredProduces()
	assert.Len(t, producers, 2)
	consumers := analyzer.RequiredConsumes()
	assert.Len(t, consumers, 2)
	authSchemes := analyzer.RequiredSecuritySchemes()
	assert.Len(t, authSchemes, 3)

	ops := analyzer.Operations()
	assert.Len(t, ops, 1)
	assert.Len(t, ops["GET"], 2)

	op, ok := analyzer.OperationFor("get", "/")
	assert.True(t, ok)
	assert.NotNil(t, op)

	op, ok = analyzer.OperationFor("delete", "/")
	assert.False(t, ok)
	assert.Nil(t, op)

	// check for duplicates in sec. requirements for operation
	pi.Get.Security = []map[string][]string{
		{"oauth2": {}},
		{"basic": nil},
		{"basic": nil},
	}

	spec = makeFixturepec(pi, pi2, formatParam)
	analyzer = New(spec)
	securityDefinitions = analyzer.SecurityDefinitionsFor(spec.Paths.Paths["/"].Get)
	assert.Len(t, securityDefinitions, 2)
	assert.Equal(t, *spec.SecurityDefinitions["basic"], securityDefinitions["basic"])
	assert.Equal(t, *spec.SecurityDefinitions["oauth2"], securityDefinitions["oauth2"])

	// check for empty (optional) in sec. requirements for operation
	pi.Get.Security = []map[string][]string{
		{"oauth2": {}},
		{"": nil},
		{"basic": nil},
	}

	spec = makeFixturepec(pi, pi2, formatParam)
	analyzer = New(spec)
	securityDefinitions = analyzer.SecurityDefinitionsFor(spec.Paths.Paths["/"].Get)
	assert.Len(t, securityDefinitions, 2)
	assert.Equal(t, *spec.SecurityDefinitions["basic"], securityDefinitions["basic"])
	assert.Equal(t, *spec.SecurityDefinitions["oauth2"], securityDefinitions["oauth2"])
}

func TestAnalyzer_DefinitionAnalysis(t *testing.T) {
	t.Parallel()

	doc := antest.LoadOrFail(t, filepath.Join("fixtures", "definitions.yml"))

	analyzer := New(doc)
	definitions := analyzer.allSchemas
	require.NotNil(t, definitions)

	for _, key := range []string{
		"#/parameters/someParam/schema",
		"#/paths/~1some~1where~1{id}/parameters/1/schema",
		"#/paths/~1some~1where~1{id}/get/parameters/1/schema",

		// responses
		"#/responses/someResponse/schema",
		"#/paths/~1some~1where~1{id}/get/responses/default/schema",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema",

		// definitions
		"#/definitions/tag",
		"#/definitions/tag/properties/id",
		"#/definitions/tag/properties/value",
		"#/definitions/tag/definitions/category",
		"#/definitions/tag/definitions/category/properties/id",
		"#/definitions/tag/definitions/category/properties/value",
		"#/definitions/withAdditionalProps",
		"#/definitions/withAdditionalProps/additionalProperties",
		"#/definitions/withAdditionalItems",
		"#/definitions/withAdditionalItems/items/0",
		"#/definitions/withAdditionalItems/items/1",
		"#/definitions/withAdditionalItems/additionalItems",
		"#/definitions/withNot",
		"#/definitions/withNot/not",
		"#/definitions/withAnyOf",
		"#/definitions/withAnyOf/anyOf/0",
		"#/definitions/withAnyOf/anyOf/1",
		"#/definitions/withAllOf",
		"#/definitions/withAllOf/allOf/0",
		"#/definitions/withAllOf/allOf/1",
		"#/definitions/withOneOf/oneOf/0",
		"#/definitions/withOneOf/oneOf/1",
	} {
		t.Run(fmt.Sprintf("ref %q exists", key), func(t *testing.T) {
			t.Parallel()

			assertSchemaRefExists(t, definitions, key)
		})
	}

	allOfs := analyzer.allOfs
	assert.Len(t, allOfs, 1)
	assert.Contains(t, allOfs, "#/definitions/withAllOf")
}

func TestAnalyzer_ReferenceAnalysis(t *testing.T) {
	t.Parallel()

	doc := antest.LoadOrFail(t, filepath.Join("fixtures", "references.yml"))
	an := New(doc)

	definitions := an.references

	require.NotNil(t, definitions)
	require.NotNil(t, definitions.parameters)
	require.NotNil(t, definitions.responses)
	require.NotNil(t, definitions.pathItems)
	require.NotNil(t, definitions.schemas)
	require.NotNil(t, definitions.parameterItems)
	require.NotNil(t, definitions.headerItems)
	require.NotNil(t, definitions.allRefs)

	for _, toPin := range []struct {
		Input        map[string]spec.Ref
		ExpectedKeys []string
	}{
		{
			Input: definitions.parameters,
			ExpectedKeys: []string{
				"#/paths/~1some~1where~1{id}/parameters/0",
				"#/paths/~1some~1where~1{id}/get/parameters/0",
			},
		},
		{
			Input: definitions.pathItems,
			ExpectedKeys: []string{
				"#/paths/~1other~1place",
			},
		},
		{
			Input: definitions.responses,
			ExpectedKeys: []string{
				"#/paths/~1some~1where~1{id}/get/responses/404",
			},
		},
		{
			Input: definitions.schemas,
			ExpectedKeys: []string{
				"#/responses/notFound/schema",
				"#/paths/~1some~1where~1{id}/get/responses/200/schema",
				"#/definitions/tag/properties/audit",
			},
		},
		{
			// Supported non-swagger 2.0 constructs ($ref in simple schema items)
			Input: definitions.allRefs,
			ExpectedKeys: []string{
				"#/paths/~1some~1where~1{id}/get/parameters/1/items",
				"#/paths/~1some~1where~1{id}/get/parameters/2/items",
				"#/paths/~1some~1where~1{id}/get/responses/default/headers/x-array-header/items",
			},
		},
		{
			Input: definitions.parameterItems,
			ExpectedKeys: []string{
				"#/paths/~1some~1where~1{id}/get/parameters/1/items",
				"#/paths/~1some~1where~1{id}/get/parameters/2/items",
			},
		},
		{
			Input: definitions.headerItems,
			ExpectedKeys: []string{
				"#/paths/~1some~1where~1{id}/get/responses/default/headers/x-array-header/items",
			},
		},
	} {
		fixture := toPin
		for _, key := range fixture.ExpectedKeys {
			t.Run(fmt.Sprintf("ref %q exists", key), func(t *testing.T) {
				t.Parallel()
				assertRefExists(t, fixture.Input, key)
			})
		}
	}

	assert.Lenf(t, an.AllItemsReferences(), 3, "Expected 3 items references in this spec")
}

type expectedPattern struct {
	Key     string
	Pattern string
}

func TestAnalyzer_PatternAnalysis(t *testing.T) {
	t.Parallel()

	doc := antest.LoadOrFail(t, filepath.Join("fixtures", "patterns.yml"))
	an := New(doc)
	pt := an.patterns

	require.NotNil(t, pt)
	require.NotNil(t, pt.parameters)
	require.NotNil(t, pt.headers)
	require.NotNil(t, pt.schemas)
	require.NotNil(t, pt.items)

	for _, toPin := range []struct {
		Input        map[string]string
		ExpectedKeys []expectedPattern
	}{
		{
			Input: pt.parameters,
			ExpectedKeys: []expectedPattern{
				{Key: "#/parameters/idParam", Pattern: "a[A-Za-Z0-9]+"},
				{Key: "#/paths/~1some~1where~1{id}/parameters/1", Pattern: "b[A-Za-z0-9]+"},
				{Key: "#/paths/~1some~1where~1{id}/get/parameters/0", Pattern: "[abc][0-9]+"},
			},
		},
		{
			Input: pt.headers,
			ExpectedKeys: []expectedPattern{
				{Key: "#/responses/notFound/headers/ContentLength", Pattern: "[0-9]+"},
				{Key: "#/paths/~1some~1where~1{id}/get/responses/200/headers/X-Request-Id", Pattern: "d[A-Za-z0-9]+"},
			},
		},
		{
			Input: pt.schemas,
			ExpectedKeys: []expectedPattern{
				{Key: "#/paths/~1other~1place/post/parameters/0/schema/properties/value", Pattern: "e[A-Za-z0-9]+"},
				{Key: "#/paths/~1other~1place/post/responses/200/schema/properties/data", Pattern: "[0-9]+[abd]"},
				{Key: "#/definitions/named", Pattern: "f[A-Za-z0-9]+"},
				{Key: "#/definitions/tag/properties/value", Pattern: "g[A-Za-z0-9]+"},
			},
		},
		{
			Input: pt.items,
			ExpectedKeys: []expectedPattern{
				{Key: "#/paths/~1some~1where~1{id}/get/parameters/1/items", Pattern: "c[A-Za-z0-9]+"},
				{Key: "#/paths/~1other~1place/post/responses/default/headers/Via/items", Pattern: "[A-Za-z]+"},
			},
		},
	} {
		fixture := toPin
		for _, toPinExpected := range fixture.ExpectedKeys {
			expected := toPinExpected
			t.Run(fmt.Sprintf("pattern at %q exists", expected.Key), func(t *testing.T) {
				t.Parallel()
				assertPattern(t, fixture.Input, expected.Key, expected.Pattern)
			})
		}
	}

	// patternProperties (beyond Swagger 2.0)
	_, ok := an.spec.Definitions["withPatternProperties"]
	assert.True(t, ok)

	_, ok = an.allSchemas["#/definitions/withPatternProperties/patternProperties/^prop[0-9]+$"]
	assert.True(t, ok)
}

func TestAnalyzer_ParamsAsMap(t *testing.T) {
	t.Parallel()

	s := prepareTestParamsValid()
	require.NotNil(t, s)

	m := make(map[string]spec.Parameter)
	pi, ok := s.spec.Paths.Paths["/items"]
	require.True(t, ok)

	s.paramsAsMap(pi.Parameters, m, nil)
	assert.Len(t, m, 1)

	p, ok := m["query#Limit"]
	require.True(t, ok)

	assert.Equal(t, p.Name, "limit")

	// An invalid spec, but passes this step (errors are figured out at a higher level)
	s = prepareTestParamsInvalid(t, "fixture-1289-param.yaml")
	require.NotNil(t, s)

	m = make(map[string]spec.Parameter)
	pi, ok = s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	s.paramsAsMap(pi.Parameters, m, nil)
	assert.Len(t, m, 1)

	p, ok = m["body#DespicableMe"]
	require.True(t, ok)

	assert.Equal(t, p.Name, "despicableMe")
}

func TestAnalyzer_ParamsAsMapWithCallback(t *testing.T) {
	t.Parallel()

	s := prepareTestParamsInvalid(t, "fixture-342.yaml")
	require.NotNil(t, s)

	// No bail out callback
	m := make(map[string]spec.Parameter)
	e := []string{}
	pi, ok := s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	s.paramsAsMap(pi.Parameters, m, func(param spec.Parameter, err error) bool {
		// pt.Logf("ERROR on %+v : %v", param, err)
		e = append(e, err.Error())

		return true // Continue
	})

	assert.Contains(t, e, `resolved reference is not a parameter: "#/definitions/sample_info/properties/sid"`)
	assert.Contains(t, e, `invalid reference: "#/definitions/sample_info/properties/sids"`)

	// bail out callback
	m = make(map[string]spec.Parameter)
	e = []string{}
	pi, ok = s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	s.paramsAsMap(pi.Parameters, m, func(param spec.Parameter, err error) bool {
		// pt.Logf("ERROR on %+v : %v", param, err)
		e = append(e, err.Error())

		return false // Bail
	})

	// We got one then bail
	assert.Len(t, e, 1)

	// Bail after ref failure: exercising another path
	s = prepareTestParamsInvalid(t, "fixture-342-2.yaml")
	require.NotNil(t, s)

	// bail callback
	m = make(map[string]spec.Parameter)
	e = []string{}
	pi, ok = s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	s.paramsAsMap(pi.Parameters, m, func(param spec.Parameter, err error) bool {
		e = append(e, err.Error())

		return false // Bail
	})
	// We got one then bail
	assert.Len(t, e, 1)

	// Bail after ref failure: exercising another path
	s = prepareTestParamsInvalid(t, "fixture-342-3.yaml")
	require.NotNil(t, s)

	// bail callback
	m = make(map[string]spec.Parameter)
	e = []string{}
	pi, ok = s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	s.paramsAsMap(pi.Parameters, m, func(param spec.Parameter, err error) bool {
		e = append(e, err.Error())

		return false // Bail
	})
	// We got one then bail
	assert.Len(t, e, 1)
}

func TestAnalyzer_ParamsAsMapPanic(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{
		"fixture-342.yaml",
		"fixture-342-2.yaml",
		"fixture-342-3.yaml",
	} {
		t.Run(fmt.Sprintf("panic_%s", fixture), func(t *testing.T) {
			t.Parallel()

			s := prepareTestParamsInvalid(t, fixture)
			require.NotNil(t, s)

			panickerParamsAsMap := func() {
				m := make(map[string]spec.Parameter)
				if pi, ok := s.spec.Paths.Paths["/fixture"]; ok {
					pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
					s.paramsAsMap(pi.Parameters, m, nil)
				}
			}
			assert.Panics(t, panickerParamsAsMap)
		})
	}
}

func TestAnalyzer_SafeParamsFor(t *testing.T) {
	t.Parallel()

	s := prepareTestParamsInvalid(t, "fixture-342.yaml")
	require.NotNil(t, s)

	e := []string{}
	pi, ok := s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters

	errFunc := func(param spec.Parameter, err error) bool {
		e = append(e, err.Error())

		return true // Continue
	}

	for range s.SafeParamsFor("Get", "/fixture", errFunc) {
		require.Fail(t, "There should be no safe parameter in this testcase")
	}

	assert.Contains(t, e, `resolved reference is not a parameter: "#/definitions/sample_info/properties/sid"`)
	assert.Contains(t, e, `invalid reference: "#/definitions/sample_info/properties/sids"`)
}

func TestAnalyzer_ParamsFor(t *testing.T) {
	t.Parallel()

	// Valid example
	s := prepareTestParamsValid()
	require.NotNil(t, s)

	params := s.ParamsFor("Get", "/items")
	assert.True(t, len(params) > 0)

	panickerParamsFor := func() {
		s := prepareTestParamsInvalid(t, "fixture-342.yaml")
		pi, ok := s.spec.Paths.Paths["/fixture"]
		if ok {
			pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
			s.ParamsFor("Get", "/fixture")
		}
	}

	// Invalid example
	assert.Panics(t, panickerParamsFor)
}

func TestAnalyzer_SafeParametersFor(t *testing.T) {
	t.Parallel()

	s := prepareTestParamsInvalid(t, "fixture-342.yaml")
	require.NotNil(t, s)

	e := []string{}
	pi, ok := s.spec.Paths.Paths["/fixture"]
	require.True(t, ok)

	errFunc := func(param spec.Parameter, err error) bool {
		e = append(e, err.Error())

		return true // Continue
	}

	pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
	for range s.SafeParametersFor("fixtureOp", errFunc) {
		require.Fail(t, "There should be no safe parameter in this testcase")
	}

	assert.Contains(t, e, `resolved reference is not a parameter: "#/definitions/sample_info/properties/sid"`)
	assert.Contains(t, e, `invalid reference: "#/definitions/sample_info/properties/sids"`)
}

func TestAnalyzer_ParametersFor(t *testing.T) {
	t.Parallel()

	// Valid example
	s := prepareTestParamsValid()
	params := s.ParamsFor("Get", "/items")
	assert.True(t, len(params) > 0)

	panickerParametersFor := func() {
		s := prepareTestParamsInvalid(t, "fixture-342.yaml")
		if s == nil {
			return
		}

		pi, ok := s.spec.Paths.Paths["/fixture"]
		if ok {
			pi.Parameters = pi.PathItemProps.Get.OperationProps.Parameters
			// func (s *Spec) ParametersFor(operationID string) []spec.Parameter {
			s.ParametersFor("fixtureOp")
		}
	}

	// Invalid example
	assert.Panics(t, panickerParametersFor)
}

func TestAnalyzer_SecurityDefinitionsFor(t *testing.T) {
	t.Parallel()

	spec := prepareTestParamsAuth()
	pi1 := spec.spec.Paths.Paths["/"].Get
	pi2 := spec.spec.Paths.Paths["/items"].Get

	defs1 := spec.SecurityDefinitionsFor(pi1)
	require.Contains(t, defs1, "oauth2")
	require.Contains(t, defs1, "basic")
	require.NotContains(t, defs1, "apiKey")

	defs2 := spec.SecurityDefinitionsFor(pi2)
	require.Contains(t, defs2, "oauth2")
	require.Contains(t, defs2, "basic")
	require.Contains(t, defs2, "apiKey")
}

func TestAnalyzer_SecurityRequirements(t *testing.T) {
	t.Parallel()

	spec := prepareTestParamsAuth()
	pi1 := spec.spec.Paths.Paths["/"].Get
	pi2 := spec.spec.Paths.Paths["/items"].Get
	scopes := []string{"the-scope"}

	reqs1 := spec.SecurityRequirementsFor(pi1)
	require.Len(t, reqs1, 2)
	require.Len(t, reqs1[0], 1)
	require.Equal(t, reqs1[0][0].Name, "oauth2")
	require.Equal(t, reqs1[0][0].Scopes, scopes)
	require.Len(t, reqs1[1], 1)
	require.Equal(t, reqs1[1][0].Name, "basic")
	require.Empty(t, reqs1[1][0].Scopes)

	reqs2 := spec.SecurityRequirementsFor(pi2)
	require.Len(t, reqs2, 3)
	require.Len(t, reqs2[0], 1)
	require.Equal(t, reqs2[0][0].Name, "oauth2")
	require.Equal(t, reqs2[0][0].Scopes, scopes)
	require.Len(t, reqs2[1], 1)
	require.Empty(t, reqs2[1][0].Name)
	require.Empty(t, reqs2[1][0].Scopes)
	require.Len(t, reqs2[2], 2)
	require.Contains(t, reqs2[2], SecurityRequirement{Name: "basic", Scopes: []string{}})
	require.Empty(t, reqs2[2][0].Scopes)
	require.Contains(t, reqs2[2], SecurityRequirement{Name: "apiKey", Scopes: []string{}})
	require.Empty(t, reqs2[2][1].Scopes)
}

func TestAnalyzer_SecurityRequirementsDefinitions(t *testing.T) {
	t.Parallel()

	spec := prepareTestParamsAuth()
	pi1 := spec.spec.Paths.Paths["/"].Get
	pi2 := spec.spec.Paths.Paths["/items"].Get

	reqs1 := spec.SecurityRequirementsFor(pi1)
	defs11 := spec.SecurityDefinitionsForRequirements(reqs1[0])
	require.Contains(t, defs11, "oauth2")
	defs12 := spec.SecurityDefinitionsForRequirements(reqs1[1])
	require.Contains(t, defs12, "basic")
	require.NotContains(t, defs12, "apiKey")

	reqs2 := spec.SecurityRequirementsFor(pi2)
	defs21 := spec.SecurityDefinitionsForRequirements(reqs2[0])
	require.Len(t, defs21, 1)
	require.Contains(t, defs21, "oauth2")
	require.NotContains(t, defs21, "basic")
	require.NotContains(t, defs21, "apiKey")
	defs22 := spec.SecurityDefinitionsForRequirements(reqs2[1])
	require.NotNil(t, defs22)
	require.Empty(t, defs22)
	defs23 := spec.SecurityDefinitionsForRequirements(reqs2[2])
	require.Len(t, defs23, 2)
	require.NotContains(t, defs23, "oauth2")
	require.Contains(t, defs23, "basic")
	require.Contains(t, defs23, "apiKey")
}

func TestAnalyzer_MoreParamAnalysis(t *testing.T) {
	t.Parallel()

	bp := filepath.Join("fixtures", "parameters", "fixture-parameters.yaml")
	sp := antest.LoadOrFail(t, bp)

	an := New(sp)

	res := an.AllPatterns()
	assert.Lenf(t, res, 6, "Expected 6 patterns in this spec")

	res = an.SchemaPatterns()
	assert.Lenf(t, res, 1, "Expected 1 schema pattern in this spec")

	res = an.HeaderPatterns()
	assert.Lenf(t, res, 2, "Expected 2 header pattern in this spec")

	res = an.ItemsPatterns()
	assert.Lenf(t, res, 2, "Expected 2 items pattern in this spec")

	res = an.ParameterPatterns()
	assert.Lenf(t, res, 1, "Expected 1 simple param pattern in this spec")

	refs := an.AllRefs()
	assert.Lenf(t, refs, 10, "Expected 10 reference usage in this spec")

	references := an.AllReferences()
	assert.Lenf(t, references, 14, "Expected 14 reference usage in this spec")

	references = an.AllItemsReferences()
	assert.Lenf(t, references, 0, "Expected 0 items reference in this spec")

	references = an.AllPathItemReferences()
	assert.Lenf(t, references, 1, "Expected 1 pathItem reference in this spec")

	references = an.AllResponseReferences()
	assert.Lenf(t, references, 3, "Expected 3 response references in this spec")

	references = an.AllParameterReferences()
	assert.Lenf(t, references, 6, "Expected 6 parameter references in this spec")

	schemaRefs := an.AllDefinitions()
	assert.Lenf(t, schemaRefs, 14, "Expected 14 schema definitions in this spec")
	schemaRefs = an.SchemasWithAllOf()
	assert.Lenf(t, schemaRefs, 1, "Expected 1 schema with AllOf definition in this spec")

	method, path, op, found := an.OperationForName("postSomeWhere")
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/some/where", path)
	require.NotNil(t, op)
	require.True(t, found)

	sec := an.SecurityRequirementsFor(op)
	assert.Nil(t, sec)
	secScheme := an.SecurityDefinitionsFor(op)
	assert.Nil(t, secScheme)

	bag := an.ParametersFor("postSomeWhere")
	assert.Lenf(t, bag, 6, "Expected 6 parameters for this operation")

	method, path, op, found = an.OperationForName("notFound")
	assert.Equal(t, "", method)
	assert.Equal(t, "", path)
	assert.Nil(t, op)
	assert.False(t, found)

	// does not take ops under pathItem $ref
	ops := an.OperationMethodPaths()
	assert.Lenf(t, ops, 3, "Expected 3 ops")
	ops = an.OperationIDs()
	assert.Lenf(t, ops, 3, "Expected 3 ops")
	assert.Contains(t, ops, "postSomeWhere")
	assert.Contains(t, ops, "GET /some/where/else")
	assert.Contains(t, ops, "GET /some/where")
}

func TestAnalyzer_EdgeCases(t *testing.T) {
	t.Parallel()

	// check return values are consistent in some nil/empty edge cases
	sp := Spec{}
	res1 := sp.AllPaths()
	assert.Nil(t, res1)

	res2 := sp.OperationIDs()
	assert.Nil(t, res2)

	res3 := sp.OperationMethodPaths()
	assert.Nil(t, res3)

	res4 := sp.structMapKeys(nil)
	assert.Nil(t, res4)

	res5 := sp.structMapKeys(make(map[string]struct{}, 10))
	assert.Nil(t, res5)

	// check AllRefs() skips empty $refs
	sp.references.allRefs = make(map[string]spec.Ref, 3)
	for i := 0; i < 3; i++ {
		sp.references.allRefs["ref"+strconv.Itoa(i)] = spec.Ref{}
	}
	assert.Len(t, sp.references.allRefs, 3)
	res6 := sp.AllRefs()
	assert.Len(t, res6, 0)

	// check AllRefs() skips duplicate $refs
	sp.references.allRefs["refToOne"] = spec.MustCreateRef("#/ref1")
	sp.references.allRefs["refToOneAgain"] = spec.MustCreateRef("#/ref1")
	res7 := sp.AllRefs()
	assert.NotNil(t, res7)
	assert.Len(t, res7, 1)
}

func TestAnalyzer_EnumAnalysis(t *testing.T) {
	t.Parallel()

	doc := antest.LoadOrFail(t, filepath.Join("fixtures", "enums.yml"))

	an := New(doc)
	en := an.enums

	// parameters
	assertEnum(t, en.parameters, "#/parameters/idParam", []interface{}{"aA", "b9", "c3"})
	assertEnum(t, en.parameters, "#/paths/~1some~1where~1{id}/parameters/1", []interface{}{"bA", "ba", "b9"})
	assertEnum(t, en.parameters, "#/paths/~1some~1where~1{id}/get/parameters/0", []interface{}{"a0", "b1", "c2"})

	// responses
	assertEnum(t, en.headers, "#/responses/notFound/headers/ContentLength", []interface{}{"1234", "123"})
	assertEnum(t, en.headers,
		"#/paths/~1some~1where~1{id}/get/responses/200/headers/X-Request-Id", []interface{}{"dA", "d9"})

	// definitions
	assertEnum(t, en.schemas,
		"#/paths/~1other~1place/post/parameters/0/schema/properties/value", []interface{}{"eA", "e9"})
	assertEnum(t, en.schemas, "#/paths/~1other~1place/post/responses/200/schema/properties/data",
		[]interface{}{"123a", "123b", "123d"})
	assertEnum(t, en.schemas, "#/definitions/named", []interface{}{"fA", "f9"})
	assertEnum(t, en.schemas, "#/definitions/tag/properties/value", []interface{}{"gA", "ga", "g9"})
	assertEnum(t, en.schemas, "#/definitions/record",
		[]interface{}{`{"createdAt": "2018-08-31"}`, `{"createdAt": "2018-09-30"}`})

	// array enum
	assertEnum(t, en.parameters, "#/paths/~1some~1where~1{id}/get/parameters/1",
		[]interface{}{[]interface{}{"cA", "cz", "c9"}, []interface{}{"cA", "cz"}, []interface{}{"cz", "c9"}})

	// items
	assertEnum(t, en.items, "#/paths/~1some~1where~1{id}/get/parameters/1/items", []interface{}{"cA", "cz", "c9"})
	assertEnum(t, en.items, "#/paths/~1other~1place/post/responses/default/headers/Via/items",
		[]interface{}{"AA", "Ab"})

	res := an.AllEnums()
	assert.Lenf(t, res, 14, "Expected 14 enums in this spec, but got %d", len(res))

	res = an.ParameterEnums()
	assert.Lenf(t, res, 4, "Expected 4 enums in this spec, but got %d", len(res))

	res = an.SchemaEnums()
	assert.Lenf(t, res, 6, "Expected 6 schema enums in this spec, but got %d", len(res))

	res = an.HeaderEnums()
	assert.Lenf(t, res, 2, "Expected 2 header enums in this spec, but got %d", len(res))

	res = an.ItemsEnums()
	assert.Lenf(t, res, 2, "Expected 2 items enums in this spec, but got %d", len(res))
}

/* helpers for the Analyzer test suite */

func schemeNames(schemes [][]SecurityRequirement) []string {
	var names []string
	for _, scheme := range schemes {
		for _, v := range scheme {
			names = append(names, v.Name)
		}
	}
	sort.Strings(names)

	return names
}

func makeFixturepec(pi, pi2 spec.PathItem, formatParam *spec.Parameter) *spec.Swagger {
	return &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Consumes: []string{"application/json"},
			Produces: []string{"application/json"},
			Security: []map[string][]string{
				{"apikey": nil},
			},
			SecurityDefinitions: map[string]*spec.SecurityScheme{
				"basic":  spec.BasicAuth(),
				"apiKey": spec.APIKeyAuth("api_key", "query"),
				"oauth2": spec.OAuth2AccessToken("http://authorize.com", "http://token.com"),
			},
			Parameters: map[string]spec.Parameter{"format": *formatParam},
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/":      pi,
					"/items": pi2,
				},
			},
		},
	}
}

func assertEnum(t testing.TB, data map[string][]interface{}, key string, enum []interface{}) {
	require.Contains(t, data, key)
	assert.Equal(t, enum, data[key])
}

func assertRefExists(t testing.TB, data map[string]spec.Ref, key string) bool {
	_, ok := data[key]

	return assert.Truef(t, ok, "expected %q to exist in the ref bag", key)
}

func assertSchemaRefExists(t testing.TB, data map[string]SchemaRef, key string) bool {
	_, ok := data[key]

	return assert.Truef(t, ok, "expected %q to exist in schema ref bag", key)
}

func assertPattern(t testing.TB, data map[string]string, key, pattern string) bool {
	if assert.Contains(t, data, key) {
		return assert.Equal(t, pattern, data[key])
	}

	return false
}

func prepareTestParamsAuth() *Spec {
	formatParam := spec.QueryParam("format").Typed("string", "")

	limitParam := spec.QueryParam("limit").Typed("integer", "int32")
	limitParam.Extensions = spec.Extensions(map[string]interface{}{})
	limitParam.Extensions.Add("go-name", "Limit")

	skipParam := spec.QueryParam("skip").Typed("integer", "int32")
	pi := spec.PathItem{}
	pi.Parameters = []spec.Parameter{*limitParam}

	op := &spec.Operation{}
	op.Consumes = []string{"application/x-yaml"}
	op.Produces = []string{"application/x-yaml"}
	op.Security = []map[string][]string{
		{"oauth2": {"the-scope"}},
		{"basic": nil},
	}
	op.ID = "someOperation"
	op.Parameters = []spec.Parameter{*skipParam}
	pi.Get = op

	pi2 := spec.PathItem{}
	pi2.Parameters = []spec.Parameter{*limitParam}
	op2 := &spec.Operation{}
	op2.ID = "anotherOperation"
	op2.Security = []map[string][]string{
		{"oauth2": {"the-scope"}},
		{},
		{
			"basic":  {},
			"apiKey": {},
		},
	}
	op2.Parameters = []spec.Parameter{*skipParam}
	pi2.Get = op2

	oauth := spec.OAuth2AccessToken("http://authorize.com", "http://token.com")
	oauth.AddScope("the-scope", "the scope gives access to ...")
	spec := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Consumes: []string{"application/json"},
			Produces: []string{"application/json"},
			Security: []map[string][]string{
				{"apikey": nil},
			},
			SecurityDefinitions: map[string]*spec.SecurityScheme{
				"basic":  spec.BasicAuth(),
				"apiKey": spec.APIKeyAuth("api_key", "query"),
				"oauth2": oauth,
			},
			Parameters: map[string]spec.Parameter{"format": *formatParam},
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/":      pi,
					"/items": pi2,
				},
			},
		},
	}
	analyzer := New(spec)

	return analyzer
}

func prepareTestParamsValid() *Spec {
	formatParam := spec.QueryParam("format").Typed("string", "")

	limitParam := spec.QueryParam("limit").Typed("integer", "int32")
	limitParam.Extensions = spec.Extensions(map[string]interface{}{})
	limitParam.Extensions.Add("go-name", "Limit")

	skipParam := spec.QueryParam("skip").Typed("integer", "int32")
	pi := spec.PathItem{}
	pi.Parameters = []spec.Parameter{*limitParam}

	op := &spec.Operation{}
	op.Consumes = []string{"application/x-yaml"}
	op.Produces = []string{"application/x-yaml"}
	op.Security = []map[string][]string{
		{"oauth2": {}},
		{"basic": nil},
	}
	op.ID = "someOperation"
	op.Parameters = []spec.Parameter{*skipParam}
	pi.Get = op

	pi2 := spec.PathItem{}
	pi2.Parameters = []spec.Parameter{*limitParam}
	op2 := &spec.Operation{}
	op2.ID = "anotherOperation"
	op2.Parameters = []spec.Parameter{*skipParam}
	pi2.Get = op2

	spec := makeFixturepec(pi, pi2, formatParam)
	analyzer := New(spec)

	return analyzer
}

func prepareTestParamsInvalid(t testing.TB, fixture string) *Spec {
	bp := filepath.Join("fixtures", fixture)
	spec := antest.LoadOrFail(t, bp)

	analyzer := New(spec)

	return analyzer
}
