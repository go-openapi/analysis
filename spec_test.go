package analysis_test

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enableLongTests bool

func init() {
	flag.BoolVar(&enableLongTests, "enable-long", false, "enable long runnning tests")
}

func skipNotify(t *testing.T) {
	t.Log("To enable this long running test, use -args -enable-long in your go test command line")
}

func Test_FlattenAzure(t *testing.T) {
	if !enableLongTests {
		skipNotify(t)
		t.SkipNow()
	}
	t.Parallel()

	// Local copy of https://raw.githubusercontent.com/Azure/azure-rest-api-specs/master/specification/network/resource-manager/Microsoft.Network/stable/2020-04-01/publicIpAddress.json
	url := "fixtures/azure/publicIpAddress.json"
	byts, err := swag.LoadFromFileOrHTTP(url)
	assert.NoError(t, err)
	swagger := &spec.Swagger{}
	require.NoError(t, swagger.UnmarshalJSON(byts))

	analyzed := analysis.New(swagger)
	require.NoError(t, analysis.Flatten(analysis.FlattenOpts{Spec: analyzed, Expand: true, BasePath: url}))

	jazon := asJSON(t, swagger)

	assertRefInJSONRegexp(t, jazon, `^(#/definitions/)|(\./example)`)

	t.Run("resolve local $ref azure", func(t *testing.T) {
		assertRefResolve(t, jazon, `\./example`, swagger, &spec.ExpandOptions{RelativeBase: url})
	})
}

func TestRemoteFlattenAzure_Expand(t *testing.T) {
	if !enableLongTests {
		skipNotify(t)
		t.SkipNow()
	}
	t.Parallel()

	server := httptest.NewServer(http.FileServer(http.Dir("fixtures/azure")))
	defer server.Close()

	basePath := server.URL + "/publicIpAddress.json"

	swagger, err := loads.Spec(basePath)
	require.NoError(t, err)

	require.NoError(t, analysis.Flatten(analysis.FlattenOpts{Spec: swagger.Analyzer, Expand: true, BasePath: basePath}))

	jazon := asJSON(t, swagger.Spec())

	assertRefInJSONRegexp(t, jazon, `^(#/definitions/)|(\./example)`)

	t.Run("resolve remote $ref azure [after expansion]", func(t *testing.T) {
		assertRefResolve(t, jazon, `\./example`, swagger.Spec(), &spec.ExpandOptions{RelativeBase: basePath})
	})
}

func TestRemoteFlattenAzure_Flatten(t *testing.T) {
	if !enableLongTests {
		skipNotify(t)
		t.SkipNow()
	}
	t.Parallel()

	server := httptest.NewServer(http.FileServer(http.Dir("fixtures/azure")))
	defer server.Close()

	basePath := server.URL + "/publicIpAddress.json"

	swagger, err := loads.Spec(basePath)
	require.NoError(t, err)

	require.NoError(t, analysis.Flatten(analysis.FlattenOpts{Spec: swagger.Analyzer, Expand: false, BasePath: basePath}))

	jazon := asJSON(t, swagger.Spec())

	assertRefInJSONRegexp(t, jazon, `^(#/definitions/)|(\./example)`)

	t.Run("resolve remote $ref azure [minimal flatten]", func(t *testing.T) {
		assertRefResolve(t, jazon, `\./example`, swagger.Spec(), &spec.ExpandOptions{RelativeBase: basePath})
	})
}

func TestIssue66(t *testing.T) {
	// no BasePath provided: assume current working directory
	file, clean := makeFileSpec(t)
	defer clean()

	// analyze and expand
	doc, err := loads.Spec(file)
	require.NoError(t, err)
	an := analysis.New(doc.Spec()) // Analyze spec
	require.NoError(t, analysis.Flatten(analysis.FlattenOpts{
		Spec:   an,
		Expand: true,
	}))
	jazon := asJSON(t, doc.Spec())
	assertNoRef(t, jazon)

	// reload and flatten
	doc, err = loads.Spec(file)
	require.NoError(t, err)
	require.NoError(t, analysis.Flatten(analysis.FlattenOpts{
		Spec:   an,
		Expand: false,
	}))
	jazon = asJSON(t, doc.Spec())
	t.Run("resolve $ref issue66", func(t *testing.T) {
		assertRefResolve(t, jazon, "", doc.Spec(), &spec.ExpandOptions{})
	})
}

func makeFileSpec(t testing.TB) (string, func()) {
	file := "./openapi.yaml"
	require.NoError(t, ioutil.WriteFile(file, fixtureIssue66(), 0600))
	return file, func() {
		_ = os.Remove(file)
	}
}

func fixtureIssue66() []byte {
	return []byte(`
	x-google-endpoints:
  - name: bravo-api.endpoints.dev-srplatform.cloud.goog
    allowCors: true
host: bravo-api.endpoints.dev-srplatform.cloud.goog
swagger: '2.0'
info:
  description: Demo API for Bravo team testing
  title: BRAVO Team API
  version: 0.0.0
basePath: /bravo
x-google-allow: all
consumes:
  - application/json
produces:
  - application/json
schemes:
  - http
  - https
paths:
  /bravo-api:
    get:
      description: List expansions
      operationId: default
      responses:
        200:
          description: Default Path
          schema:
            $ref: '#/definitions/heartbeatResponse'
        403:
          description: Forbidden
        500:
          description: Internal Server Error
  /bravo-api/healthN:
    get:
      description: N Health
      operationId: healthN
      responses:
        200:
          description: Default Path
          schema:
            $ref: '#/definitions/heartbeatResponse'
        403:
          description: Forbidden
        500:
          description: Internal Server Error
  /bravo-api/internal/heartbeat:
    get:
      description: Heartbeat endpoint
      operationId: heartbeat
      produces:
        - application/json
      responses:
        200:
          description: Health Status
          schema:
            $ref: '#/definitions/heartbeatResponse'
  /bravo-api/internal/version:
    get:
      description: Version endpoint
      operationId: version
      produces:
        - application/json
      responses:
        200:
          description: Version Information
  /bravo-api/internal/cpuload:
    get:
      description: CPU Load endpoint
      operationId: cpuload
      produces:
        - application/json
      responses:
        200:
          description: Run a CPU load

definitions:
  heartbeatResponse:
    properties:
      Status:
        type: string
      ProjectID:
        type: string
      Version:
        type: string
securityDefinitions:
  okta_jwt:
    authorizationUrl: "http://okta.example.com"
    flow: "implicit"
    type: "oauth2"
    scopes:
      com.sr.messaging: 'View and manage messaging content, criteria and definitions.'
    x-google-issuer: "http://okta.example.com"
    x-google-jwks_uri: "http://okta.example.com/v1/keys"
    x-google-audiences: "http://api.example.com"
	`)
}
