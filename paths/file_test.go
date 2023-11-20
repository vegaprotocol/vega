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

package paths_test

import (
	path2 "path"
	"testing"

	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileHelpers(t *testing.T) {
	t.Run("Writing structured file succeeds", testWritingStructuredFileSucceeds)
	t.Run("Rewriting structured file succeeds", testRewritingStructuredFileSucceeds)
	t.Run("Reading structured file succeeds", testReadingStructuredFileSucceeds)
	t.Run("Reading non-existing structured file fails", testReadingNonExistingStructuredFileFails)
	t.Run("Writing encrypted file succeeds", testWritingEncryptedFileSucceeds)
	t.Run("Rewriting encrypted file succeeds", testRewritingEncryptedFileSucceeds)
	t.Run("Reading encrypted file succeeds", testReadingEncryptedFileSucceeds)
	t.Run("Reading non-existing encrypted file fails", testReadingNonExistingEncryptedFileFails)
	t.Run("Reading encrypted file with wrong passphrase fails", testReadingEncryptedFileWithWrongPassphraseFails)
}

func testWritingStructuredFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteStructuredFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadStructuredFile(path, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testRewritingStructuredFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteStructuredFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadStructuredFile(path, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)

	newData := &DummyData{
		Name: "John",
		Age:  30,
	}

	err = paths.WriteStructuredFile(path, newData)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readNewData := &DummyData{}
	err = paths.ReadStructuredFile(path, readNewData)
	require.NoError(t, err)
	assert.Equal(t, newData, readNewData)
}

func testReadingStructuredFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteStructuredFile(path, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadStructuredFile(path, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testReadingNonExistingStructuredFileFails(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")

	readData := &DummyData{}
	err := paths.ReadStructuredFile(path, readData)
	require.Error(t, err)
	assert.Empty(t, readData)
}

func testWritingEncryptedFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	passphrase := "pa$$w0rd"
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteEncryptedFile(path, passphrase, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadEncryptedFile(path, passphrase, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testRewritingEncryptedFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	passphrase := "pa$$w0rd"
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteEncryptedFile(path, passphrase, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadEncryptedFile(path, passphrase, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)

	newData := &DummyData{
		Name: "John",
		Age:  30,
	}

	err = paths.WriteEncryptedFile(path, passphrase, newData)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readNewData := &DummyData{}
	err = paths.ReadEncryptedFile(path, passphrase, readNewData)
	require.NoError(t, err)
	assert.Equal(t, newData, readNewData)
}

func testReadingEncryptedFileSucceeds(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	passphrase := "pa$$w0rd"
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteEncryptedFile(path, passphrase, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadEncryptedFile(path, passphrase, readData)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
}

func testReadingNonExistingEncryptedFileFails(t *testing.T) {
	path := t.TempDir()
	passphrase := "pa$$w0rd"

	readData := &DummyData{}
	err := paths.ReadEncryptedFile(path, passphrase, readData)
	require.Error(t, err)
	assert.Empty(t, readData)
}

func testReadingEncryptedFileWithWrongPassphraseFails(t *testing.T) {
	path := path2.Join(t.TempDir(), "file.txt")
	passphrase := "pa$$w0rd"
	wrongPassphrase := "HaXx0r"
	data := &DummyData{
		Name: "Jane",
		Age:  40,
	}

	err := paths.WriteEncryptedFile(path, passphrase, data)
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, path)

	readData := &DummyData{}
	err = paths.ReadEncryptedFile(path, wrongPassphrase, readData)
	require.Error(t, err)
	assert.Empty(t, readData)
}

type DummyData struct {
	Name string
	Age  uint8
}
