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
	"path/filepath"
	"testing"

	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomPaths(t *testing.T) {
	t.Run("Getting custom path for cache file succeeds", testGettingCustomPathForCacheFileSucceeds)
	t.Run("Getting custom path for config file succeeds", testGettingCustomPathForConfigFileSucceeds)
	t.Run("Getting custom path for data file succeeds", testGettingCustomPathForDataFileSucceeds)
	t.Run("Getting custom path for state file succeeds", testGettingCustomPathForStateFileSucceeds)
	t.Run("Getting custom path from struct for cache file succeeds", testGettingCustomPathFromStructForCacheFileSucceeds)
	t.Run("Getting custom path from struct for config file succeeds", testGettingCustomPathFromStructForConfigFileSucceeds)
	t.Run("Getting custom path from struct for data file succeeds", testGettingCustomPathFromStructForDataFileSucceeds)
	t.Run("Getting custom path from struct for state file succeeds", testGettingCustomPathFromStructForStateFileSucceeds)
}

func testGettingCustomPathForCacheFileSucceeds(t *testing.T) {
	home := t.TempDir()
	path, err := paths.CreateCustomCachePathFor(home, "fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "cache", "fake-file.empty"), path)
}

func testGettingCustomPathForConfigFileSucceeds(t *testing.T) {
	home := t.TempDir()
	path, err := paths.CreateCustomConfigPathFor(home, "fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "config", "fake-file.empty"), path)
}

func testGettingCustomPathForDataFileSucceeds(t *testing.T) {
	home := t.TempDir()
	path, err := paths.CreateCustomDataPathFor(home, "fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "data", "fake-file.empty"), path)
}

func testGettingCustomPathForStateFileSucceeds(t *testing.T) {
	home := t.TempDir()
	path, err := paths.CreateCustomStatePathFor(home, "fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "state", "fake-file.empty"), path)
}

func testGettingCustomPathFromStructForCacheFileSucceeds(t *testing.T) {
	home := t.TempDir()
	customPaths := &paths.CustomPaths{CustomHome: home}
	path, err := customPaths.CreateCachePathFor("fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "cache", "fake-file.empty"), path)
}

func testGettingCustomPathFromStructForConfigFileSucceeds(t *testing.T) {
	home := t.TempDir()
	customPaths := &paths.CustomPaths{CustomHome: home}
	path, err := customPaths.CreateConfigPathFor("fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "config", "fake-file.empty"), path)
}

func testGettingCustomPathFromStructForDataFileSucceeds(t *testing.T) {
	home := t.TempDir()
	customPaths := &paths.CustomPaths{CustomHome: home}
	path, err := customPaths.CreateDataPathFor("fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "data", "fake-file.empty"), path)
}

func testGettingCustomPathFromStructForStateFileSucceeds(t *testing.T) {
	home := t.TempDir()
	customPaths := &paths.CustomPaths{CustomHome: home}
	path, err := customPaths.CreateStatePathFor("fake-file.empty")
	require.NoError(t, err)
	vgtest.AssertDirAccess(t, filepath.Dir(home))
	assert.Equal(t, filepath.Join(home, "state", "fake-file.empty"), path)
}
