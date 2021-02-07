package analysis_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	rex = regexp.MustCompile(`"\$ref":\s*"(.*?)"`)
)

func assertRefResolve(t *testing.T, jazon, exclude string, root interface{}, opts ...*spec.ExpandOptions) {
	assertRefWithFunc(t, "resolve", jazon, exclude, func(t *testing.T, match string) {
		ref := spec.MustCreateRef(match)
		var (
			sch *spec.Schema
			err error
		)
		if len(opts) > 0 {
			options := *opts[0]
			sch, err = spec.ResolveRefWithBase(root, &ref, &options)
		} else {
			sch, err = spec.ResolveRef(root, &ref)
		}

		require.NoErrorf(t, err, `%v: for "$ref": %q`, err, match)
		require.NotNil(t, sch)
	})
}

// assertNoRef ensures that no $ref is remaining in json doc
func assertNoRef(t testing.TB, jazon string) {
	m := rex.FindAllStringSubmatch(jazon, -1)
	require.Nil(t, m)
}

func assertRefInJSONRegexp(t testing.TB, jazon, match string) {
	// assert a match in a references
	m := rex.FindAllStringSubmatch(jazon, -1)
	require.NotNil(t, m)

	refMatch, err := regexp.Compile(match)
	require.NoError(t, err)

	for _, matched := range m {
		subMatch := matched[1]
		assert.True(t, refMatch.MatchString(subMatch),
			"expected $ref to match %q, got: %s", match, matched[0])
	}
}

// assertRefResolve ensures that all $ref in some json doc verify some asserting func.
//
// "exclude" is a regexp pattern to ignore certain $ref (e.g. some specs may embed $ref that are not processed, such as extensions).
func assertRefWithFunc(t *testing.T, name, jazon, exclude string, asserter func(*testing.T, string)) {
	filterRex := regexp.MustCompile(exclude)
	m := rex.FindAllStringSubmatch(jazon, -1)
	require.NotNil(t, m)

	allRefs := make(map[string]struct{}, len(m))
	for _, toPin := range m {
		matched := toPin
		subMatch := matched[1]
		if exclude != "" && filterRex.MatchString(subMatch) {
			continue
		}

		_, ok := allRefs[subMatch]
		if ok {
			continue
		}

		allRefs[subMatch] = struct{}{}

		t.Run(fmt.Sprintf("%s-%s-%s", t.Name(), name, subMatch), func(t *testing.T) {
			t.Parallel()
			asserter(t, subMatch)
		})
	}
}

func asJSON(t testing.TB, sp interface{}) string {
	bbb, err := json.MarshalIndent(sp, "", " ")
	require.NoError(t, err)

	return string(bbb)
}
