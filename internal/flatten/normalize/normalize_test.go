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

func TestNormalize_Path(t *testing.T) {
	t.Parallel()

	values := []struct{ Source, Expected string }{
		{"#/definitions/A", "#/definitions/A"},
		{"http://somewhere.com/definitions/A", "http://somewhere.com/definitions/A"},
		{wrapWindowsPath("/definitions/A"), wrapWindowsPath("/definitions/A")}, // considered absolute on unix but not on windows
		{wrapWindowsPath("/definitions/errorModel.json") + "#/definitions/A", wrapWindowsPath("/definitions/errorModel.json") + "#/definitions/A"},
		{"http://somewhere.com", "http://somewhere.com"},
		{wrapWindowsPath("./definitions/definitions.yaml") + "#/definitions/A", wrapWindowsPath("/abs/to/spec/definitions/definitions.yaml") + "#/definitions/A"},
		{"#", wrapWindowsPath("/abs/to/spec")},
	}

	for _, v := range values {
		assert.Equal(t, v.Expected, Path(spec.MustCreateRef(v.Source), wrapWindowsPath("/abs/to/spec/spec.json")))
	}
}

func TestNormalize_RebaseRef(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "#/definitions/abc", RebaseRef("#/definitions/base", "#/definitions/abc"))
	assert.Equal(t, "#/definitions/abc", RebaseRef("", "#/definitions/abc"))
	assert.Equal(t, "#/definitions/abc", RebaseRef(".", "#/definitions/abc"))
	assert.Equal(t, "otherfile#/definitions/abc", RebaseRef("file#/definitions/base", "otherfile#/definitions/abc"))
	assert.Equal(t, wrapWindowsPath("../otherfile")+"#/definitions/abc", RebaseRef(wrapWindowsPath("../file")+"#/definitions/base", wrapWindowsPath("./otherfile")+"#/definitions/abc"))
	assert.Equal(t, wrapWindowsPath("../otherfile")+"#/definitions/abc", RebaseRef(wrapWindowsPath("../file")+"#/definitions/base", wrapWindowsPath("otherfile")+"#/definitions/abc"))
	assert.Equal(t, wrapWindowsPath("local/remote/otherfile")+"#/definitions/abc", RebaseRef(wrapWindowsPath("local/file")+"#/definitions/base", wrapWindowsPath("remote/otherfile")+"#/definitions/abc"))
	assert.Equal(t, wrapWindowsPath("local/remote/otherfile.yaml"), RebaseRef(wrapWindowsPath("local/file.yaml"), wrapWindowsPath("remote/otherfile.yaml")))

	assert.Equal(t, "file#/definitions/abc", RebaseRef("file#/definitions/base", "#/definitions/abc"))

	// with remote
	assert.Equal(t, "https://example.com/base#/definitions/abc", RebaseRef("https://example.com/base", "https://example.com/base#/definitions/abc"))
	assert.Equal(t, "https://example.com/base#/definitions/abc", RebaseRef("https://example.com/base", "#/definitions/abc"))
	assert.Equal(t, "https://example.com/base#/dir/definitions/abc", RebaseRef("https://example.com/base", "#/dir/definitions/abc"))
	assert.Equal(t, "https://example.com/base/dir/definitions/abc", RebaseRef("https://example.com/base/spec.yaml", "dir/definitions/abc"))
	assert.Equal(t, "https://example.com/base/dir/definitions/abc", RebaseRef("https://example.com/base/", "dir/definitions/abc"))
	assert.Equal(t, "https://example.com/dir/definitions/abc", RebaseRef("https://example.com/base", "dir/definitions/abc"))
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
