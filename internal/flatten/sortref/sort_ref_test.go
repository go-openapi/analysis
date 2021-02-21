package sortref

import (
	"testing"

	_ "github.com/go-openapi/analysis/internal/antest"
	"github.com/stretchr/testify/assert"
)

func TestSortRef_DepthFirstSort(t *testing.T) {
	values := []string{
		"#/definitions/datedTag/allOf/0",
		"#/definitions/pneumonoultramicroscopicsilicovolcanoconiosisAntidisestablishmentarianism",
		"#/definitions/namedThing",
		"#/definitions/datedTag/properties/id",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema",
		"#/definitions/tags/additionalProperties/properties/id",
		"#/parameters/someParam/schema",
		"#/definitions/records/items/0/properties/createdAt",
		"#/definitions/datedTaggedRecords",
		"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/createdAt",
		"#/definitions/namedAgain",
		"#/definitions/tags",
		"#/paths/~1some~1where~1{id}/get/responses/404/schema",
		"#/definitions/datedRecords/items/1",
		"#/definitions/records/items/0",
		"#/definitions/datedTaggedRecords/items/0",
		"#/definitions/datedTag/allOf/1",
		"#/definitions/otherRecords/items/properties/createdAt",
		"#/responses/someResponse/schema/properties/createdAt",
		"#/definitions/namedAgain/properties/id",
		"#/definitions/datedTag",
		"#/paths/~1some~1where~1{id}/parameters/1/schema",
		"#/parameters/someParam/schema/properties/createdAt",
		"#/paths/~1some~1where~1{id}/get/parameters/2/schema/properties/createdAt",
		"#/definitions/otherRecords",
		"#/definitions/datedTaggedRecords/items/1",
		"#/definitions/datedTaggedRecords/items/1/properties/createdAt",
		"#/definitions/otherRecords/items",
		"#/definitions/datedRecords/items/0",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/id",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/value",
		"#/definitions/records",
		"#/definitions/namedThing/properties/name/properties/id",
		"#/definitions/datedTaggedRecords/additionalItems/properties/id",
		"#/definitions/datedTaggedRecords/additionalItems/properties/value",
		"#/definitions/datedRecords",
		"#/definitions/datedTag/properties/value",
		"#/definitions/pneumonoultramicroscopicsilicovolcanoconiosisAntidisestablishmentarianism/properties/floccinaucinihilipilificationCreatedAt",
		"#/definitions/datedRecords/items/1/properties/createdAt",
		"#/definitions/tags/additionalProperties",
		"#/paths/~1some~1where~1{id}/parameters/1/schema/properties/createdAt",
		"#/definitions/namedThing/properties/name",
		"#/paths/~1some~1where~1{id}/get/responses/default/schema",
		"#/definitions/tags/additionalProperties/properties/value",
		"#/responses/someResponse/schema",
		"#/definitions/datedTaggedRecords/additionalItems",
		"#/paths/~1some~1where~1{id}/get/parameters/2/schema",
	}

	valuesMap := make(map[string]struct{}, len(values))
	for _, v := range values {
		valuesMap[v] = struct{}{}
	}

	expected := []string{
		// Added shared parameters and responses
		"#/parameters/someParam/schema/properties/createdAt",
		"#/parameters/someParam/schema",
		"#/responses/someResponse/schema/properties/createdAt",
		"#/responses/someResponse/schema",
		"#/paths/~1some~1where~1{id}/parameters/1/schema/properties/createdAt",
		"#/paths/~1some~1where~1{id}/parameters/1/schema",
		"#/paths/~1some~1where~1{id}/get/parameters/2/schema/properties/createdAt",
		"#/paths/~1some~1where~1{id}/get/parameters/2/schema",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/id",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema/properties/value",
		"#/paths/~1some~1where~1{id}/get/responses/200/schema",
		"#/paths/~1some~1where~1{id}/get/responses/404/schema",
		"#/paths/~1some~1where~1{id}/get/responses/default/schema/properties/createdAt",
		"#/paths/~1some~1where~1{id}/get/responses/default/schema",
		"#/definitions/datedRecords/items/1/properties/createdAt",
		"#/definitions/datedTaggedRecords/items/1/properties/createdAt",
		"#/definitions/namedThing/properties/name/properties/id",
		"#/definitions/records/items/0/properties/createdAt",
		"#/definitions/datedTaggedRecords/additionalItems/properties/id",
		"#/definitions/datedTaggedRecords/additionalItems/properties/value",
		"#/definitions/otherRecords/items/properties/createdAt",
		"#/definitions/tags/additionalProperties/properties/id",
		"#/definitions/tags/additionalProperties/properties/value",
		"#/definitions/datedRecords/items/0",
		"#/definitions/datedRecords/items/1",
		"#/definitions/datedTag/allOf/0",
		"#/definitions/datedTag/allOf/1",
		"#/definitions/datedTag/properties/id",
		"#/definitions/datedTag/properties/value",
		"#/definitions/datedTaggedRecords/items/0",
		"#/definitions/datedTaggedRecords/items/1",
		"#/definitions/namedAgain/properties/id",
		"#/definitions/namedThing/properties/name",
		"#/definitions/pneumonoultramicroscopicsilicovolcanoconiosisAntidisestablishmentarianism/properties/" +
			"floccinaucinihilipilificationCreatedAt",
		"#/definitions/records/items/0",
		"#/definitions/datedTaggedRecords/additionalItems",
		"#/definitions/otherRecords/items",
		"#/definitions/tags/additionalProperties",
		"#/definitions/datedRecords",
		"#/definitions/datedTag",
		"#/definitions/datedTaggedRecords",
		"#/definitions/namedAgain",
		"#/definitions/namedThing",
		"#/definitions/otherRecords",
		"#/definitions/pneumonoultramicroscopicsilicovolcanoconiosisAntidisestablishmentarianism",
		"#/definitions/records",
		"#/definitions/tags",
	}

	assert.Equal(t, expected, DepthFirst(valuesMap))
}

func TestSortRef_TopmostFirst(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]string{"/a/b", "/a/b/c"},
		TopmostFirst([]string{"/a/b/c", "/a/b"}),
	)

	assert.Equal(t,
		[]string{"/a/b", "/a/c"},
		TopmostFirst([]string{"/a/c", "/a/b"}),
	)

	assert.Equal(t,
		[]string{"/a/b", "/a/c", "/a/b/c", "/a/b/d", "/a/a/b/d"},
		TopmostFirst([]string{"/a/a/b/d", "/a/b", "/a/b/c", "/a/b/d", "/a/c"}),
	)
}
