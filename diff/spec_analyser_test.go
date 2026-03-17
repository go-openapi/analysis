// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/analysis/internal/antest"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestDiffForVariousCombinations - computes the diffs for a number
// of scenarios and compares the computed diff with expected diffs.
func TestDiffForVariousCombinations(t *testing.T) {
	pattern := fixturePath("*.diff.txt")
	allTests, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.NotEmpty(t, allTests)

	// To filter cases for debugging poke an individual case here eg "path", "enum" etc
	// see the test cases in fixtures/diff
	// Don't forget to remove it once you're done.
	// (There's a test at the end to check all cases were run)
	matches := allTests
	// matches := []string{"enum"}

	testCases := makeTestCases(t, matches)

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("should diff", func(t *testing.T) {
				diffs, err := getDiffs(tc.oldSpec, tc.newSpec)
				require.NoError(t, err)

				t.Run("should report all diff", func(t *testing.T) {
					out, err, warn := diffs.ReportAllDiffs(false)
					require.NoError(t, err)

					if !assertReadersContent(t, true, tc.expectedLines, out) {
						t.Logf("unexpected content for fixture %q[%d] (file: %s)", tc.name, i, tc.expectedFile)
					}

					if diffs.BreakingChangeCount() > 0 {
						t.Run("should error when breaking changes are detected", func(t *testing.T) {
							require.Error(t, warn)
						})
					}
				})
			})
		})
	}

	require.Lenf(t, allTests, len(matches), "All test cases were not run. Remove filter")
}

func getDiffs(oldSpecPath, newSpecPath string) (SpecDifferences, error) {
	spec1, err := antest.LoadSpec(oldSpecPath)
	if err != nil {
		return nil, err
	}

	spec2, err := antest.LoadSpec(newSpecPath)
	if err != nil {
		return nil, err
	}

	return Compare(spec1, spec2)
}

func makeTestCases(t testing.TB, matches []string) []testCaseData {
	t.Helper()

	testCases := make([]testCaseData, 0, len(matches))
	for _, eachFile := range matches {
		namePart := fixturePart(eachFile)

		if _, err := os.Stat(fixturePath(namePart, ".v1.json")); err == nil {
			testCases = append(
				testCases, testCaseData{
					name:          namePart,
					oldSpec:       fixturePath(namePart, ".v1.json"),
					newSpec:       fixturePath(namePart, ".v2.json"),
					expectedLines: linesInFile(t, fixturePath(namePart, ".diff.txt")),
				})
		}

		if _, err := os.Stat(fixturePath(namePart, ".v1.yml")); err == nil {
			testCases = append(
				testCases, testCaseData{
					name:          namePart,
					oldSpec:       fixturePath(namePart, ".v1.yml"),
					newSpec:       fixturePath(namePart, ".v2.yml"),
					expectedLines: linesInFile(t, fixturePath(namePart, ".diff.txt")),
				})
		}
	}

	return testCases
}

func TestIssue2962(t *testing.T) {
	oldSpec := filepath.Join("fixtures", "bugs", "2962", "old.json")
	newSpec := filepath.Join("fixtures", "bugs", "2962", "new.json")

	t.Run("should diff", func(t *testing.T) {
		diffs, err := getDiffs(oldSpec, newSpec)
		require.NoError(t, err)

		const (
			expectedChanges  = 3
			expectedBreaking = 1
		)

		t.Run(fmt.Sprintf("should find %d breaking changes", expectedBreaking), func(t *testing.T) {
			require.Len(t, diffs, expectedChanges)
			require.EqualT(t, expectedBreaking, diffs.BreakingChangeCount())
		})
	})
}

func fixturePath(file string, parts ...string) string {
	return filepath.Join("fixtures", strings.Join(append([]string{file}, parts...), ""))
}

type testCaseData struct {
	name          string
	oldSpec       string
	newSpec       string
	expectedLines io.Reader
	expectedFile  string
}

func fixturePart(file string) string {
	base := filepath.Base(file)
	parts := strings.Split(base, ".diff.txt")
	return parts[0]
}

func linesInFile(t testing.TB, fileName string) io.Reader {
	t.Helper()

	file, err := os.ReadFile(fileName)
	require.NoError(t, err)

	// consumes a bit of extra memory, but no longer leaks open files
	return bytes.NewBuffer(file)
}

func assertReadersContent(t testing.TB, noBlanks bool, expected, actual io.Reader) bool {
	t.Helper()

	e, err := io.ReadAll(expected)
	require.NoError(t, err)

	a, err := io.ReadAll(actual)
	require.NoError(t, err)

	var wants, got strings.Builder
	_, _ = wants.Write(e)
	_, _ = got.Write(a)

	if noBlanks {
		r := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "")
		return assert.EqualTf(t, r.Replace(wants.String()), r.Replace(got.String()), "expected:\n%s\ngot:\n%s", wants.String(), got.String())
	}

	return assert.EqualT(t, wants.String(), got.String())
}
