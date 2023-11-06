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

package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/stretchr/testify/require"
)

func TestImportNetwork(t *testing.T) {
	// given
	home := t.TempDir()
	networkFile1 := NewFile(t, home, "my-network-1.toml", FakeNetwork("my-network-1"))

	// when
	importNetworkResp1, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile1,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp1).
		WithName("my-network-1").
		LocatedUnder(home)

	// when
	listNetsResp1, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp1).
		WithNetworks("my-network-1")

	// given
	networkFile2 := NewFile(t, home, "my-network-2.toml", FakeNetwork("my-network-2"))

	// when
	importNetworkResp2, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile2,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp2).
		WithName("my-network-2").
		LocatedUnder(home)

	// when
	listNetsResp2, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp2).
		WithNetworks("my-network-1", "my-network-2")
}

func TestForceImportNetwork(t *testing.T) {
	// given
	home := t.TempDir()
	networkFile := NewFile(t, home, "my-network.toml", FakeNetwork("my-network"))

	// when
	importNetworkResp1, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp1).
		WithName("my-network").
		LocatedUnder(home)

	// when
	importNetworkResp2, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
	})

	// then
	require.Error(t, err)
	require.Nil(t, importNetworkResp2)

	// when
	importNetworkResp3, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
		"--force",
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp3).
		WithName("my-network").
		LocatedUnder(home)

	// when
	listNetsResp, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp).
		WithNetworks("my-network")
}

func TestImportNetworkWithNewName(t *testing.T) {
	// given
	home := t.TempDir()
	networkFile := NewFile(t, home, "my-network.toml", FakeNetwork("my-network"))

	// when
	importNetworkResp1, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp1).
		WithName("my-network").
		LocatedUnder(home)

	// when
	listNetsResp1, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp1).
		WithNetworks("my-network")

	// given
	networkName := vgrand.RandomStr(5)

	// when
	importNetworkResp2, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
		"--with-name", networkName,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp2).
		WithName(networkName).
		LocatedUnder(home)

	// when
	listNetsResp2, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp2).
		WithNetworks("my-network", networkName)
}

func TestRenameANetworkFileRenamesNetwork(t *testing.T) {
	// given
	home := t.TempDir()
	networkFile := NewFile(t, home, "my-network.toml", FakeNetwork("my-network"))

	// when
	importNetworkResp, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp).
		WithName("my-network").
		LocatedUnder(home)

	// when
	newNetworkName := "renamed-network"
	require.NoError(t,
		os.Rename(
			importNetworkResp.FilePath,
			filepath.Join(filepath.Dir(importNetworkResp.FilePath), newNetworkName+".toml"),
		),
	)

	// when
	listNetsResp, err := NetworkList(t, []string{
		"--home", home,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertListNetwork(t, listNetsResp).
		WithNetworks(newNetworkName)
}
