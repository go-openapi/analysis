package schutils

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

func TestFlattenSchema_SaveDefinition(t *testing.T) {
	t.Parallel()

	sp := &spec.Swagger{}
	Save(sp, "theName", spec.StringProperty())
	assert.Contains(t, sp.Definitions, "theName")

	saveNilSchema := func() {
		Save(sp, "ThisNilSchema", nil)
	}
	assert.NotPanics(t, saveNilSchema)
}

func TestFlattenSchema_Clone(t *testing.T) {
	sch := spec.RefSchema("#/definitions/x")
	assert.EqualValues(t, sch, Clone(sch))
}
