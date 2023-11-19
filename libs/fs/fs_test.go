// Copyright (C) 2023 Gobalsky Labs Limited
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

package fs_test

import (
	path2 "path"
	"testing"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSystemHelpers(t *testing.T) {
	t.Run("Ensuring presence of non-existing directories succeeds", testEnsuringPresenceOfNonExistingDirectoriesSucceeds)
	t.Run("Ensuring presence of existing directories succeeds", testEnsuringPresenceOfExistingDirectoriesSucceeds)
	t.Run("Verify path existence of non-existing one fails", testVerifyingPathExistenceOfNonExistingOneFails)
	t.Run("Verify path existence of existing one succeeds", testVerifyingPathExistenceOfExistingOneSucceeds)
	t.Run("Verify file existence of non-existing one fails", testVerifyingFileExistenceOfNonExistingOneFails)
	t.Run("Verify file existence of existing one succeeds", testVerifyingFileExistenceOfExistingOneSucceeds)
	t.Run("Verify file existence on a directory fails", testVerifyingExistenceOnDirectoryFails)
	t.Run("Writing file succeeds", testWritingFileSucceeds)
	t.Run("Rewriting file succeeds", testRewritingFileSucceeds)
	t.Run("Reading existing file succeeds", testReadingExistingFileSucceeds)
	t.Run("Reading non-existing file fails", testReadingNonExistingFileFails)
}

func testEnsuringPresenceOfNonExistingDirectoriesSucceeds(t *testing.T) {
	path := t.TempDir()
	err := vgfs.EnsureDir(path)
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, path)
}

func testEnsuringPresenceOfExistingDirectoriesSucceeds(t *testing.T) {
	path := t.TempDir()

	err := vgfs.EnsureDir(path)
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, path)

	err = vgfs.EnsureDir(path)
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, path)
}

func testVerifyingPathExistenceOfNonExistingOneFails(t *testing.T) {
	exists, err := vgfs.PathExists("/" + vgrand.RandomStr(10))
	require.NoError(t, err)
	assert.False(t, exists)
}

func testVerifyingPathExistenceOfExistingOneSucceeds(t *testing.T) {
	path := t.TempDir()

	err := vgfs.EnsureDir(path)
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, path)

	exists, err := vgfs.PathExists(path)
	require.NoError(t, err)
	assert.True(t, exists)
}

func testVerifyingFileExistenceOfNonExistingOneFails(t *testing.T) {
	exists, err := vgfs.FileExists("/" + vgrand.RandomStr(10))
	require.NoError(t, err)
	assert.False(t, exists)
}

func testVerifyingFileExistenceOfExistingOneSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")

	err := vgfs.WriteFile(path, []byte("Hello, World!"))
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	exists, err := vgfs.FileExists(path)
	require.NoError(t, err)
	assert.True(t, exists)
}

func testVerifyingExistenceOnDirectoryFails(t *testing.T) {
	path := t.TempDir()

	err := vgfs.EnsureDir(path)
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, path)

	exists, err := vgfs.FileExists(path)
	require.Error(t, err)
	assert.False(t, exists)
}

func testWritingFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := []byte("Hello, World!")

	err := vgfs.WriteFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData, err := vgfs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testRewritingFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := []byte("Hello, World!")

	err := vgfs.WriteFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData, err := vgfs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, readData)

	frenchData := []byte("Bonjour, le Monde!")

	err = vgfs.WriteFile(path, frenchData)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readFrenchData, err := vgfs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, frenchData, readFrenchData)
}

func testReadingExistingFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := []byte("Hello, World!")

	err := vgfs.WriteFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData, err := vgfs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testReadingNonExistingFileFails(t *testing.T) {
	path := t.TempDir()

	readData, err := vgfs.ReadFile(path)
	require.Error(t, err)
	assert.Empty(t, readData)
}
