package normalize

import (
	"net/url"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

const (
	definitionA    = "#/definitions/A"
	definitionABC  = "#/definitions/abc"
	definitionBase = "#/definitions/base"
	exampleBase    = "https://example.com/base"
)

func TestNormalize_Path(t *testing.T) {
	t.Parallel()

	values := []struct{ Source, Expected string }{
		{definitionA, definitionA},
		{"http://somewhere.com/definitions/A", "http://somewhere.com/definitions/A"},
		{wrapWindowsPath("/definitions/A"), wrapWindowsPath("/definitions/A")}, // considered absolute on unix but not on windows
		{wrapWindowsPath("/definitions/errorModel.json") + definitionA, wrapWindowsPath("/definitions/errorModel.json") + definitionA},
		{"http://somewhere.com", "http://somewhere.com"},
		{wrapWindowsPath("./definitions/definitions.yaml") + definitionA, wrapWindowsPath("/abs/to/spec/definitions/definitions.yaml") + definitionA},
		{"#", wrapWindowsPath("/abs/to/spec")},
	}

	for _, v := range values {
		assert.Equal(t, v.Expected, Path(spec.MustCreateRef(v.Source), wrapWindowsPath("/abs/to/spec/spec.json")))
	}
}

func TestNormalize_RebaseRef(t *testing.T) {
	t.Parallel()

	assert.Equal(t, definitionABC, RebaseRef(definitionBase, definitionABC))
	assert.Equal(t, definitionABC, RebaseRef("", definitionABC))
	assert.Equal(t, definitionABC, RebaseRef(".", definitionABC))
	assert.Equal(t, "otherfile"+definitionABC, RebaseRef("file"+definitionBase, "otherfile"+definitionABC))
	assert.Equal(t, wrapWindowsPath("../otherfile")+definitionABC, RebaseRef(wrapWindowsPath("../file")+definitionBase, wrapWindowsPath("./otherfile")+definitionABC))
	assert.Equal(t, wrapWindowsPath("../otherfile")+definitionABC, RebaseRef(wrapWindowsPath("../file")+definitionBase, wrapWindowsPath("otherfile")+definitionABC))
	assert.Equal(t, wrapWindowsPath("local/remote/otherfile")+definitionABC, RebaseRef(wrapWindowsPath("local/file")+definitionBase, wrapWindowsPath("remote/otherfile")+definitionABC))
	assert.Equal(t, wrapWindowsPath("local/remote/otherfile.yaml"), RebaseRef(wrapWindowsPath("local/file.yaml"), wrapWindowsPath("remote/otherfile.yaml")))

	assert.Equal(t, "file#/definitions/abc", RebaseRef("file#/definitions/base", definitionABC))

	// with remote
	assert.Equal(t, exampleBase+definitionABC, RebaseRef(exampleBase, exampleBase+definitionABC))
	assert.Equal(t, exampleBase+definitionABC, RebaseRef(exampleBase, definitionABC))
	assert.Equal(t, exampleBase+"#/dir/definitions/abc", RebaseRef(exampleBase, "#/dir/definitions/abc"))
	assert.Equal(t, exampleBase+"/dir/definitions/abc", RebaseRef(exampleBase+"/spec.yaml", "dir/definitions/abc"))
	assert.Equal(t, exampleBase+"/dir/definitions/abc", RebaseRef(exampleBase+"/", "dir/definitions/abc"))
	assert.Equal(t, "https://example.com/dir/definitions/abc", RebaseRef(exampleBase, "dir/definitions/abc"))
}

// wrapWindowsPath adapts path expectations for tests running on windows
func wrapWindowsPath(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}

	pp := filepath.FromSlash(p)
	if !filepath.IsAbs(p) && []rune(pp)[0] == '\\' {
		pp, _ = filepath.Abs(p)
		u, _ := url.Parse(pp)

		return u.String()
	}

	return pp
}
