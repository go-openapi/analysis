// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package debug

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebug(t *testing.T) {
	tmpFile, err := os.CreateTemp(workaroundTempDir(t)(), "debug-test")
	require.NoError(t, err)

	output = tmpFile
	tmpName := tmpFile.Name()
	testLogger := GetLogger("test", true)

	testLogger("A debug: %s", "a string")
	tmpFile.Close()

	flushed, err := os.ReadFile(tmpName)
	require.NoError(t, err)

	assert.Contains(t, string(flushed), "A debug: a string")

	tmpEmptyFile, err := os.CreateTemp(workaroundTempDir(t)(), "debug-test")
	require.NoError(t, err)
	tmpEmpty := tmpEmptyFile.Name()
	testLogger = GetLogger("test", false)

	testLogger("A debug: %s", "a string")
	tmpFile.Close()

	flushed, err = os.ReadFile(tmpEmpty)
	require.NoError(t, err)

	assert.Empty(t, flushed)
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
