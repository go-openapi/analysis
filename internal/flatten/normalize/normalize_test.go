// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package normalize

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	definitionA    = "#/definitions/A"
	definitionABC  = "#/definitions/abc"
	definitionBase = "#/definitions/base"
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
		assert.EqualT(t, v.Expected, Path(spec.MustCreateRef(v.Source), wrapWindowsPath("/abs/to/spec/spec.json")))
	}
}

func TestNormalize_RebaseRef(t *testing.T) {
	t.Parallel()

	t.Run("with local refs", func(t *testing.T) {
		t.Parallel()

		assert.EqualT(t, definitionABC, RebaseRef(definitionBase, definitionABC))
		assert.EqualT(t, definitionABC, RebaseRef("", definitionABC))
		assert.EqualT(t, definitionABC, RebaseRef(".", definitionABC))
		assert.EqualT(t, "otherfile"+definitionABC, RebaseRef("file"+definitionBase, "otherfile"+definitionABC))
		assert.EqualT(t,
			wrapWindowsPath("../otherfile")+definitionABC,
			RebaseRef(wrapWindowsPath("../file")+definitionBase, wrapWindowsPath("./otherfile")+definitionABC),
		)
		assert.EqualT(t,
			wrapWindowsPath("../otherfile")+definitionABC,
			RebaseRef(wrapWindowsPath("../file")+definitionBase, wrapWindowsPath("otherfile")+definitionABC),
		)
		assert.EqualT(t,
			wrapWindowsPath("local/remote/otherfile")+definitionABC,
			RebaseRef(wrapWindowsPath("local/file")+definitionBase, wrapWindowsPath("remote/otherfile")+definitionABC),
		)
		assert.EqualT(t,
			wrapWindowsPath("local/remote/otherfile.yaml"),
			RebaseRef(wrapWindowsPath("local/file.yaml"), wrapWindowsPath("remote/otherfile.yaml")),
		)

		assert.EqualT(t, "file#/definitions/abc", RebaseRef("file#/definitions/base", definitionABC))
	})

	t.Run("with remote refs", func(t *testing.T) {
		t.Parallel()

		server := remoteSpecServer(t)
		remoteBase := server.URL + "/base"

		assert.EqualT(t, remoteBase+definitionABC, RebaseRef(remoteBase, remoteBase+definitionABC))
		assert.EqualT(t, remoteBase+definitionABC, RebaseRef(remoteBase, definitionABC))
		assert.EqualT(t, remoteBase+"#/dir/definitions/abc", RebaseRef(remoteBase, "#/dir/definitions/abc"))
		assert.EqualT(t, remoteBase+"/dir/definitions/abc", RebaseRef(remoteBase+"/spec.yaml", "dir/definitions/abc"))
		assert.EqualT(t, remoteBase+"/dir/definitions/abc", RebaseRef(remoteBase+"/", "dir/definitions/abc"))
		assert.EqualT(t, server.URL+"/dir/definitions/abc", RebaseRef(remoteBase, "dir/definitions/abc"))

		t.Run("rebased ref should point to a reachable document", func(t *testing.T) {
			rebased := RebaseRef(remoteBase, "dir/definitions/abc")

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rebased, nil)
			require.NoError(t, err)

			resp, err := server.Client().Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			assert.EqualT(t, http.StatusOK, resp.StatusCode)
		})
	})
}

// remoteSpecServer starts a local TLS server standing in for a remote spec host.
//
// It serves a stub document at any path, so that refs rebased against it
// resolve to a real, locally owned origin instead of a domain we don't control.
func remoteSpecServer(t testing.TB) *httptest.Server {
	t.Helper()

	const stub = `{"definitions":{"abc":{"type":"string"}}}`

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, stub)
	}))
	t.Cleanup(server.Close)

	return server
}

// wrapWindowsPath adapts path expectations for tests running on windows.
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
