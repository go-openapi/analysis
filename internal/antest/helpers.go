package antest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/require"
)

func init() {
	spec.PathLoader = func(path string) (json.RawMessage, error) {
		ext := filepath.Ext(path)
		if ext == ".yml" || ext == ".yaml" {
			return swag.YAMLDoc(path)
		}

		data, err := swag.LoadFromFileOrHTTP(path)
		if err != nil {
			return nil, err
		}

		return json.RawMessage(data), nil
	}
}

// LoadSpec loads a json a yaml spec
func LoadSpec(path string) (*spec.Swagger, error) {
	data, err := swag.YAMLDoc(path)
	if err != nil {
		return nil, err
	}

	var sw spec.Swagger
	if err := json.Unmarshal(data, &sw); err != nil {
		return nil, err
	}

	return &sw, nil
}

// LoadOrFail fetches a spec from a relative path or dies if the spec cannot be loaded properly
func LoadOrFail(t testing.TB, relative string) *spec.Swagger {
	cwd, _ := os.Getwd()
	sp, err := LoadSpec(filepath.Join(cwd, relative))
	require.NoError(t, err)

	return sp
}

// AsJSON unmarshals anything as JSON or dies
func AsJSON(t testing.TB, in interface{}) string {
	bbb, err := json.MarshalIndent(in, "", " ")
	require.NoError(t, err)

	return string(bbb)
}
