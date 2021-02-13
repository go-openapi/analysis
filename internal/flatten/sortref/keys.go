package sortref

import (
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
)

const (
	paths       = "paths"
	responses   = "responses"
	parameters  = "parameters"
	definitions = "definitions"
)

var (
	ignoredKeys  map[string]struct{}
	validMethods map[string]struct{}
)

func init() {
	ignoredKeys = map[string]struct{}{
		"schema":     {},
		"properties": {},
		"not":        {},
		"anyOf":      {},
		"oneOf":      {},
	}

	validMethods = map[string]struct{}{
		"GET":     {},
		"HEAD":    {},
		"OPTIONS": {},
		"PATCH":   {},
		"POST":    {},
		"PUT":     {},
		"DELETE":  {},
	}
}

// Key ...
type Key struct {
	Segments int
	Key      string
}

// Keys is a sortable collable collection of Keys
type Keys []Key

func (k Keys) Len() int      { return len(k) }
func (k Keys) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k Keys) Less(i, j int) bool {
	return k[i].Segments > k[j].Segments || (k[i].Segments == k[j].Segments && k[i].Key < k[j].Key)
}

// KeyParts ...
func KeyParts(key string) SplitKey {
	var res []string
	for _, part := range strings.Split(key[1:], "/") {
		if part != "" {
			res = append(res, jsonpointer.Unescape(part))
		}
	}

	return res
}

type SplitKey []string

func (s SplitKey) IsDefinition() bool {
	return len(s) > 1 && s[0] == definitions
}

func (s SplitKey) DefinitionName() string {
	if !s.IsDefinition() {
		return ""
	}

	return s[1]
}

func (s SplitKey) isKeyName(i int) bool {
	if i <= 0 {
		return false
	}

	count := 0
	for idx := i - 1; idx > 0; idx-- {
		if s[idx] != "properties" {
			break
		}
		count++
	}

	return count%2 != 0
}

// PartAdder know how to construct the components of a new name
type PartAdder func(string) []string

func (s SplitKey) BuildName(segments []string, startIndex int, adder PartAdder) string {
	for i, part := range s[startIndex:] {
		if _, ignored := ignoredKeys[part]; !ignored || s.isKeyName(startIndex+i) {
			segments = append(segments, adder(part)...)
		}
	}

	return strings.Join(segments, " ")
}

func (s SplitKey) IsOperation() bool {
	return len(s) > 1 && s[0] == paths
}

func (s SplitKey) IsSharedOperationParam() bool {
	return len(s) > 2 && s[0] == paths && s[2] == parameters
}

func (s SplitKey) IsSharedParam() bool {
	return len(s) > 1 && s[0] == parameters
}

func (s SplitKey) IsOperationParam() bool {
	return len(s) > 3 && s[0] == paths && s[3] == parameters
}

func (s SplitKey) IsOperationResponse() bool {
	return len(s) > 3 && s[0] == paths && s[3] == responses
}

func (s SplitKey) IsSharedResponse() bool {
	return len(s) > 1 && s[0] == responses
}

func (s SplitKey) IsDefaultResponse() bool {
	return len(s) > 4 && s[0] == paths && s[3] == responses && s[4] == "default"
}

func (s SplitKey) IsStatusCodeResponse() bool {
	isInt := func() bool {
		_, err := strconv.Atoi(s[4])

		return err == nil
	}

	return len(s) > 4 && s[0] == paths && s[3] == responses && isInt()
}

func (s SplitKey) ResponseName() string {
	if s.IsStatusCodeResponse() {
		code, _ := strconv.Atoi(s[4])

		return http.StatusText(code)
	}

	if s.IsDefaultResponse() {
		return "Default"
	}

	return ""
}

func (s SplitKey) PathItemRef() spec.Ref {
	if len(s) < 3 {
		return spec.Ref{}
	}

	pth, method := s[1], s[2]
	if _, isValidMethod := validMethods[strings.ToUpper(method)]; !isValidMethod && !strings.HasPrefix(method, "x-") {
		return spec.Ref{}
	}

	return spec.MustCreateRef("#" + path.Join("/", paths, jsonpointer.Escape(pth), strings.ToUpper(method)))
}

func (s SplitKey) PathRef() spec.Ref {
	if !s.IsOperation() {
		return spec.Ref{}
	}

	return spec.MustCreateRef("#" + path.Join("/", paths, jsonpointer.Escape(s[1])))
}
