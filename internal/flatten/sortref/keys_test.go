// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package sortref

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestName_UnitGuards(t *testing.T) {
	t.Parallel()

	parts := KeyParts("#/nowhere/arbitrary/pointer")

	res := parts.DefinitionName()
	assert.Empty(t, res)

	res = parts.ResponseName()
	assert.Empty(t, res)

	b := parts.isKeyName(-1)
	assert.False(t, b)
}
