// Package schutils provides tools to save or clone a schema
// when flattening a spec.
package schutils

import (
	swspec "github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

// Save registers a schema as an entry in spec #/definitions
func Save(spec *swspec.Swagger, name string, schema *swspec.Schema) {
	if schema == nil {
		return
	}

	if spec.Definitions == nil {
		spec.Definitions = make(map[string]swspec.Schema, 150)
	}

	spec.Definitions[name] = *schema
}

// Clone deep-clones a schema
func Clone(schema *swspec.Schema) *swspec.Schema {
	var sch swspec.Schema
	_ = swag.FromDynamicJSON(schema, &sch)

	return &sch
}
