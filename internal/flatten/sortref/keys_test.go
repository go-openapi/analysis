package sortref

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName_UnitGuards(t *testing.T) {
	t.Parallel()

	parts := KeyParts("#/nowhere/arbitrary/pointer")

	res := parts.DefinitionName()
	assert.Equal(t, "", res)

	res = parts.ResponseName()
	assert.Equal(t, "", res)

	b := parts.isKeyName(-1)
	assert.False(t, b)
}
