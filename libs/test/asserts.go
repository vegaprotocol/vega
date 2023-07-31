package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertDirAccess(t *testing.T, dirPath string) {
	t.Helper()
	stats, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, stats.IsDir())
}

func AssertFileAccess(t *testing.T, filePath string) {
	t.Helper()
	stats, err := os.Stat(filePath)
	assert.NoError(t, err)
	assert.True(t, !stats.IsDir())
}

func AssertNoFile(t *testing.T, filePath string) {
	t.Helper()
	_, err := os.Stat(filePath)
	require.Error(t, err)
}
