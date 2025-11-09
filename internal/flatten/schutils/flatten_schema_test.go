// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schutils

import (
	"testing"

	_ "github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
)

func TestFlattenSchema_Save(t *testing.T) {
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
	assert.Equal(t, sch, Clone(sch))
}
