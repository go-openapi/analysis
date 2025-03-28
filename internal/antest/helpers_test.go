package antest

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLongTestEnabled(t *testing.T) {
	t.Run("should be false by default", func(t *testing.T) {
		require.False(t, LongTestsEnabled())
	})
}

func TestLoadSpecErrorCases(t *testing.T) {
	t.Run("should not load invalid path", func(t *testing.T) {
		_, err := LoadSpec("nowhere.json")
		require.Error(t, err)
	})

	t.Run("should not load invalid YAML", func(t *testing.T) {
		invalidYAMLFile, clean := prepareBadDoc(t, "yaml", true)
		t.Cleanup(clean)

		_, err := LoadSpec(invalidYAMLFile)
		require.Error(t, err)
	})

	t.Run("should not load invalid JSON", func(t *testing.T) {
		invalidJSONFile, clean := prepareBadDoc(t, "json", true)
		t.Cleanup(clean)

		_, err := LoadSpec(invalidJSONFile)
		require.Error(t, err)
	})

	t.Run("should not load invalid spec", func(t *testing.T) {
		invalidJSONFile, clean := prepareBadDoc(t, "json", false)
		t.Cleanup(clean)

		_, err := LoadSpec(invalidJSONFile)
		require.Error(t, err)
	})
}

func prepareBadDoc(t testing.TB, kind string, invalidFormat bool) (string, func()) {
	t.Helper()

	var (
		file string
		data []byte
	)

	switch kind {
	case "yaml", "yml":
		f, err := os.CreateTemp(workaroundTempDir(t)(), "*.yaml")
		require.NoError(t, err)
		file = f.Name()

		if invalidFormat {
			data = []byte(`--
zig:
  zag 3, 4
`)
		} else {
			data = []byte(`--
swagger: 2
info:
  title: true
`)
		}

	case "json":
		f, err := os.CreateTemp(workaroundTempDir(t)(), "*.json")
		require.NoError(t, err)
		file = f.Name()

		if invalidFormat {
			data = []byte(`{
"zig": {
  "zag"
}`)
		} else {
			data = []byte(`{
"swagger": 2
"info": {
  "title": true
}
}`)
		}

	default:
		panic("supports only yaml or json")
	}

	require.NoError(t,
		os.WriteFile(file, data, 0600),
	)

	return file, func() {}
}

func workaroundTempDir(t testing.TB) func() string {
	// Workaround for go testing bug on Windows: https://github.com/golang/go/issues/71544
	// On windows, t.TempDir() doesn't properly release file handles yet,
	// se we just leave it unchecked (no cleanup would take place).
	if runtime.GOOS == "windows" {
		return func() string {
			return ""
		}
	}

	return t.TempDir
}
