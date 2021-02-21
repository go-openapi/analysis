package analysis

import (
	"net/url"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

// wrapWindowsPath adapts path expectations for tests running on windows
func wrapWindowsPath(p string) string {
	if goruntime.GOOS != "windows" {
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

func definitionPtr(key string) string {
	if !strings.HasPrefix(key, "#/definitions") {
		return key
	}
	return strings.Join(strings.Split(key, "/")[:3], "/")
}
