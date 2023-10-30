// Copyright (C) 2023  Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
