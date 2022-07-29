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
	stats, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, stats.IsDir())
	if runtime.GOOS == "windows" {
		assert.Equal(t, fs.FileMode(0777), stats.Mode().Perm())
	} else {
		assert.Equal(t, fs.FileMode(0700), stats.Mode().Perm())
	}
}

func AssertFileAccess(t *testing.T, filePath string) {
	stats, err := os.Stat(filePath)
	assert.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, fs.FileMode(0666), stats.Mode().Perm())
	} else {
		assert.Equal(t, fs.FileMode(0600), stats.Mode().Perm())
	}
}
