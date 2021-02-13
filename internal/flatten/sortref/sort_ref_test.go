package sortref

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/* TOO(fred)
func TestSortRef_DepthFirstSort(t *testing.T) {
	bp := filepath.Join("fixtures", "inline_schemas.yml")
	sp := antest.LoadOrFail(t, bp)

	values := []string{
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

	a := New(sp)
	result := DepthFirst(a.allSchemas)
	assert.Equal(t, values, result)
}
*/

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
