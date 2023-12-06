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
	"testing"

	"github.com/go-openapi/analysis/internal/antest"
	"github.com/stretchr/testify/require"
)

const (
	widgetFile     = "fixtures/widget-crud.yml"
	fooFile        = "fixtures/foo-crud.yml"
	barFile        = "fixtures/bar-crud.yml"
	noPathsFile    = "fixtures/no-paths.yml"
	emptyPathsFile = "fixtures/empty-paths.json"
	securityFile   = "fixtures/securitydef.yml"
	otherMixin     = "fixtures/other-mixin.yml"
	emptyProps     = "fixtures/empty-props.yml"
	swaggerProps   = "fixtures/swagger-props.yml"
)

func TestMixin_All(t *testing.T) {
	t.Parallel()

	primary := antest.LoadOrFail(t, widgetFile)
	mixin1 := antest.LoadOrFail(t, fooFile)
	mixin2 := antest.LoadOrFail(t, barFile)
	mixin3 := antest.LoadOrFail(t, noPathsFile)
	mixin4 := antest.LoadOrFail(t, securityFile)
	mixin5 := antest.LoadOrFail(t, otherMixin)

	collisions := Mixin(primary, mixin1, mixin2, mixin3, mixin4, mixin5)

	require.Lenf(t, collisions, 19, "TestMixin: Expected 19 collisions, got %v\n%v", len(collisions), collisions)
	require.Lenf(t, primary.Paths.Paths, 7, "TestMixin: Expected 7 paths in merged, got %v\n", len(primary.Paths.Paths))
	require.Lenf(t, primary.Definitions, 8, "TestMixin: Expected 8 definitions in merged, got %v\n", len(primary.Definitions))
	require.Lenf(t, primary.Parameters, 4, "TestMixin: Expected 4 top level parameters in merged, got %v\n", len(primary.Parameters))
	require.Lenf(t, primary.Responses, 2, "TestMixin: Expected 2 top level responses in merged, got %v\n", len(primary.Responses))
	require.Lenf(t, primary.SecurityDefinitions, 5, "TestMixin: Expected 5 top level SecurityDefinitions in merged, got %v\n", len(primary.SecurityDefinitions))
	require.Lenf(t, primary.Security, 3, "TestMixin: Expected 3 top level Security requirements in merged, got %v\n", len(primary.Security))
	require.Lenf(t, primary.Tags, 3, "TestMixin: Expected 3 top level tags merged, got %v\n", len(primary.Security))
	require.Lenf(t, primary.Schemes, 2, "TestMixin: Expected 2 top level schemes merged, got %v\n", len(primary.Security))
	require.Lenf(t, primary.Consumes, 2, "TestMixin: Expected 2 top level Consumers merged, got %v\n", len(primary.Security))
	require.Lenf(t, primary.Produces, 2, "TestMixin: Expected 2 top level Producers merged, got %v\n", len(primary.Security))
}

func TestMixin_EmptyPath(t *testing.T) {
	t.Parallel()

	// test that adding paths to a primary with no paths works (was NPE)
	primary := antest.LoadOrFail(t, widgetFile)
	emptyPaths := antest.LoadOrFail(t, emptyPathsFile)

	collisions := Mixin(emptyPaths, primary)

	require.Emptyf(t, collisions, "TestMixin: Expected 0 collisions, got %v\n%v", len(collisions), collisions)
}

func TestMixin_FromNilPath(t *testing.T) {
	t.Parallel()

	primary := antest.LoadOrFail(t, otherMixin)
	mixin1 := antest.LoadOrFail(t, widgetFile)

	collisions := Mixin(primary, mixin1)

	require.Lenf(t, collisions, 1, "TestMixin: Expected 1 collisions, got %v\n%v", len(collisions), collisions)
	require.Lenf(t, primary.Paths.Paths, 3, "TestMixin: Expected 3 paths in merged, got %v\n", len(primary.Paths.Paths))
}

func TestMixin_SwaggerProps(t *testing.T) {
	t.Parallel()

	primary := antest.LoadOrFail(t, emptyProps)
	mixin := antest.LoadOrFail(t, swaggerProps)

	collisions := Mixin(primary, mixin)

	require.Lenf(t, collisions, 1, "TestMixin: Expected 1 collisions, got %v\n%v", len(collisions), collisions)
}
