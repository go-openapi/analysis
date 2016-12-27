package analysis

import "github.com/go-openapi/spec"

// Schema analysis, will classify the schema according to known
// patterns.
func Schema(schema *spec.Schema) *AnalyzedSchema {
	a := &AnalyzedSchema{}
	a.initializeFlags(schema)
	a.inferKnownType(schema.Type, schema.Format)
	a.inferMap(schema)
	a.inferArray(schema)
	a.inferTuple(schema)
	a.inferEnum(schema)
	return a
}

// AnalyzedSchema indicates what the schema represents
type AnalyzedSchema struct {
	hasProps           bool
	hasAllOf           bool
	hasItems           bool
	hasAdditionalProps bool
	hasAdditionalItems bool

	IsKnownType      bool
	IsArray          bool
	IsMap            bool
	IsExtendedObject bool
	IsTuple          bool
	IsTupleWithExtra bool
	IsBaseType       bool
	IsSubType        bool
	IsEnum           bool
}

func (a *AnalyzedSchema) inferKnownType(tpe spec.StringOrArray, format string) {
	a.IsKnownType = tpe.Contains("boolean") ||
		tpe.Contains("integer") ||
		tpe.Contains("number") ||
		tpe.Contains("string") ||
		format != ""
}

func (a *AnalyzedSchema) inferMap(sch *spec.Schema) {
	if a.isObjectType(sch) {
		hasExtra := a.hasProps || a.hasAllOf
		a.IsMap = a.hasAdditionalProps && !hasExtra
		a.IsExtendedObject = a.hasAdditionalProps && hasExtra
	}
}

func (a *AnalyzedSchema) inferArray(sch *spec.Schema) {
	fromValid := a.isArrayType(sch) && (sch.Items == nil || sch.Items.Len() < 2)
	a.IsArray = fromValid || (a.hasItems && sch.Items.Len() < 2)
}

func (a *AnalyzedSchema) inferTuple(sch *spec.Schema) {
	tuple := a.hasItems && sch.Items.Len() > 1
	a.IsTuple = tuple && !a.hasAdditionalItems
	a.IsTupleWithExtra = tuple && a.hasAdditionalItems
}

func (a *AnalyzedSchema) inferBaseType(sch *spec.Schema) {
	if a.isObjectType(sch) {
		a.IsBaseType = sch.Discriminator != ""
	}
}

func (a *AnalyzedSchema) inferEnum(sch *spec.Schema) {
	a.IsEnum = len(sch.Enum) > 0
}

func (a *AnalyzedSchema) initializeFlags(sch *spec.Schema) {
	a.hasProps = len(sch.Properties) > 0
	a.hasAllOf = len(sch.AllOf) > 0

	a.hasItems = sch.Items != nil &&
		(sch.Items.Schema != nil || len(sch.Items.Schemas) > 0)

	a.hasAdditionalProps = sch.AdditionalProperties != nil &&
		(sch.AdditionalProperties.Allows || sch.AdditionalProperties.Schema != nil)

	a.hasAdditionalItems = sch.AdditionalItems != nil &&
		(sch.AdditionalItems.Allows || sch.AdditionalItems.Schema != nil)

}

func (a *AnalyzedSchema) isObjectType(sch *spec.Schema) bool {
	return sch.Type == nil || sch.Type.Contains("") || sch.Type.Contains("object")
}

func (a *AnalyzedSchema) isArrayType(sch *spec.Schema) bool {
	return sch.Type != nil && sch.Type.Contains("array")
}
