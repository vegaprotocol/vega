package test

import (
	"io/fs"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertDirAccess(t *testing.T, dirPath string) {
	t.Helper()
	stats, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, stats.IsDir())
	if runtime.GOOS == "windows" {
		assert.Equal(t, fs.FileMode(0o777), stats.Mode().Perm())
	} else {
		assert.Equal(t, fs.FileMode(0o700), stats.Mode().Perm())
	}
}

func AssertFileAccess(t *testing.T, filePath string) {
	t.Helper()
	stats, err := os.Stat(filePath)
	assert.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, fs.FileMode(0o666), stats.Mode().Perm())
	} else {
		assert.Equal(t, fs.FileMode(0o600), stats.Mode().Perm())
	}
}

func AssertNoFile(t *testing.T, filePath string) {
	t.Helper()
	_, err := os.Stat(filePath)
	require.Error(t, err)
}
