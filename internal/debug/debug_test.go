// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package debug

import (
	"os"
	"runtime"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
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
