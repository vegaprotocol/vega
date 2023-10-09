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

package paths_test

import (
	"path/filepath"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/paths"

	"github.com/stretchr/testify/assert"
)

func TestCachePaths(t *testing.T) {
	t.Run("Joining cache path as CachePath succeeds", testCachePathsJoiningCachePathAsCachePathSucceeds)
	t.Run("Joining cache path as string succeeds", testCachePathsJoiningCachePathAsStringSucceeds)
}

func testCachePathsJoiningCachePathAsCachePathSucceeds(t *testing.T) {
	// given
	rootCachePath := paths.DataNodeCacheHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinCachePath(rootCachePath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.CachePath(filepath.Join("data-node", pathElem1, pathElem2)), builtPath)
}

func testCachePathsJoiningCachePathAsStringSucceeds(t *testing.T) {
	// given
	rootCachePath := paths.DataNodeCacheHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinCachePath(rootCachePath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.CachePath(filepath.Join("data-node", pathElem1, pathElem2)), builtPath)
}

func TestConfigPaths(t *testing.T) {
	t.Run("Joining config path as ConfigPath succeeds", testConfigPathsJoiningConfigPathAsConfigPathSucceeds)
	t.Run("Joining config path as string succeeds", testConfigPathsJoiningConfigPathAsStringSucceeds)
}

func testConfigPathsJoiningConfigPathAsConfigPathSucceeds(t *testing.T) {
	// given
	rootConfigPath := paths.NodeConfigHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinConfigPath(rootConfigPath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.ConfigPath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}

func testConfigPathsJoiningConfigPathAsStringSucceeds(t *testing.T) {
	// given
	rootConfigPath := paths.NodeConfigHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinConfigPath(rootConfigPath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.ConfigPath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}

func TestDataPaths(t *testing.T) {
	t.Run("Joining data path as DataPath succeeds", testDataPathsJoiningDataPathAsDataPathSucceeds)
	t.Run("Joining data path as string succeeds", testDataPathsJoiningDataPathAsStringSucceeds)
}

func testDataPathsJoiningDataPathAsDataPathSucceeds(t *testing.T) {
	// given
	rootDataPath := paths.NodeDataHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinDataPath(rootDataPath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.DataPath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}

func testDataPathsJoiningDataPathAsStringSucceeds(t *testing.T) {
	// given
	rootDataPath := paths.NodeDataHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinDataPath(rootDataPath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.DataPath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}

func TestStatePaths(t *testing.T) {
	t.Run("Joining state path as StatePath succeeds", testStatePathsJoiningStatePathAsStatePathSucceeds)
	t.Run("Joining state path as string succeeds", testStatePathsJoiningStatePathAsStringSucceeds)
}

func testStatePathsJoiningStatePathAsStatePathSucceeds(t *testing.T) {
	// given
	rootStatePath := paths.NodeStateHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinStatePath(rootStatePath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.StatePath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}

func testStatePathsJoiningStatePathAsStringSucceeds(t *testing.T) {
	// given
	rootStatePath := paths.NodeStateHome
	pathElem1 := vgrand.RandomStr(5)
	pathElem2 := vgrand.RandomStr(5)

	// when
	builtPath := paths.JoinStatePath(rootStatePath, pathElem1, pathElem2)

	// then
	assert.Equal(t, paths.StatePath(filepath.Join("node", pathElem1, pathElem2)), builtPath)
}
