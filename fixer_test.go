// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package analysis

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestFixer_EmptyResponseDescriptions(t *testing.T) {
	t.Parallel()

	bp := filepath.Join("fixtures", "fixer", "fixer.yaml")
	sp := antest.LoadOrFail(t, bp)

	FixEmptyResponseDescriptions(sp)

	for path, pathItem := range sp.Paths.Paths {
		require.NotNil(t, pathItem, "expected a fixture with all path items provided in: %s", path)

		if path == "/noDesc" {
			// scope for fixed descriptions
			assertAllVerbs(t, pathItem, true)
		} else {
			// scope for unchanged descriptions
			assertAllVerbs(t, pathItem, false)
		}
	}

	for r, toPin := range sp.Responses {
		resp := toPin
		assert.Truef(t, assertResponse(t, "/responses/"+r, &resp, true),
			"expected a fixed empty description in response %s", r)
	}
}

func assertAllVerbs(t testing.TB, pathItem spec.PathItem, isEmpty bool) {
	msg := "expected %s description for %s"
	var mode string
	if isEmpty {
		mode = "a fixed empty"
	} else {
		mode = "an unmodified"
	}

	assert.Truef(t, assertResponseInOperation(t, pathItem.Get, isEmpty), msg, mode, "GET")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Put, isEmpty), msg, mode, "PUT")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Post, isEmpty), msg, mode, "POST")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Delete, isEmpty), msg, mode, "DELETE")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Options, isEmpty), msg, mode, "OPTIONS")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Patch, isEmpty), msg, mode, "PATCH")
	assert.Truef(t, assertResponseInOperation(t, pathItem.Head, isEmpty), msg, mode, "HEAD")
}

func assertResponseInOperation(t testing.TB, op *spec.Operation, isEmpty bool) bool {
	require.NotNilf(t, op, "expected a fixture with all REST verbs set")

	if op.Responses == nil {
		return true
	}

	if op.Responses.Default != nil {
		return assert.Truef(t, assertResponse(t, "default", op.Responses.Default, isEmpty),
			"unexpected description in response %s for operation", "default")
	}

	for code, resp := range op.Responses.StatusCodeResponses {
		pin := resp

		return assert.Truef(t, assertResponse(t, strconv.Itoa(code), &pin, isEmpty),
			"unexpected description in response %d for operation", code)
	}

	return true
}

func assertResponse(t testing.TB, path string, resp *spec.Response, isEmpty bool) bool {
	var expected string

	if isEmpty {
		expected = "(empty)"
	} else {
		expected = "my description"
	}

	if resp.Ref.String() != "" {
		expected = ""
	}

	if !assert.Equalf(t, expected, resp.Description, "unexpected description for resp. %s", path) {
		return false
	}

	return true
}
