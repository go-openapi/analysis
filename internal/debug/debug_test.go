// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package debug

import (
	"os"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestDebug(t *testing.T) {
	folder := t.TempDir()

	tmpFile, err := os.CreateTemp(folder, "debug-test")
	require.NoError(t, err)
	tmpFileClosed := false
	defer func() {
		if tmpFileClosed {
			return
		}
		_ = tmpFile.Close()
	}()

	output = tmpFile
	tmpName := tmpFile.Name()
	testLogger := GetLogger("test", true)

	testLogger("A debug: %s", "a string")
	tmpFile.Close()
	tmpFileClosed = true

	flushed, err := os.ReadFile(tmpName)
	require.NoError(t, err)

	assert.Contains(t, string(flushed), "A debug: a string")

	tmpEmptyFile, err := os.CreateTemp(folder, "debug-empty-test")
	require.NoError(t, err)
	tmpEmptyFileClosed := false
	defer func() {
		if tmpEmptyFileClosed {
			return
		}
		_ = tmpEmptyFile.Close()
	}()
	tmpEmpty := tmpEmptyFile.Name()
	testLogger = GetLogger("test", false)

	testLogger("A debug: %s", "a string")
	tmpEmptyFile.Close()
	tmpEmptyFileClosed = true

	flushed, err = os.ReadFile(tmpEmpty)
	require.NoError(t, err)

	assert.Empty(t, flushed)
}
