package analysis_test

import (
	"testing"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/assert"
)

func Test_FlattenAzure(t *testing.T) {
	t.SkipNow()

	// Local copy of https://raw.githubusercontent.com/Azure/azure-rest-api-specs/master/specification/network/resource-manager/Microsoft.Network/stable/2020-04-01/publicIpAddress.json
	url := "fixtures/azure/publicIpAddress.json"
	byts, err := swag.LoadFromFileOrHTTP(url)
	assert.NoError(t, err)
	swagger := &spec.Swagger{}
	err = swagger.UnmarshalJSON(byts)
	assert.NoError(t, err)
	spec := analysis.New(swagger)
	err = analysis.Flatten(analysis.FlattenOpts{Spec: spec, Expand: true, BasePath: url})
	assert.NoError(t, err)
}
