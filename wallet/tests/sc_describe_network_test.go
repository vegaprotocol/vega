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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescribeNetwork(t *testing.T) {
	// given
	home := t.TempDir()
	networkFile := NewFile(t, home, "my-network-1.toml", FakeNetwork("my-network-1"))

	// when
	importNetworkResp, err := NetworkImport(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
	})

	// then
	require.NoError(t, err)
	AssertImportNetwork(t, importNetworkResp).
		WithName("my-network-1").
		LocatedUnder(home)

	// when
	describeResp, err := NetworkDescribe(t, []string{
		"--home", home,
		"--output", "json",
		"--network", "my-network-1",
	})

	// then
	require.NoError(t, err)
	AssertDescribeNetwork(t, describeResp).
		WithName("my-network-1").
		WithGRPCConfig([]string{"example.com:3007"}).
		WithRESTConfig([]string{"https://example.com/rest"}).
		WithGraphQLConfig([]string{"https://example.com/gql/query"})

	// when
	describeResp, err = NetworkDescribe(t, []string{
		"--home", home,
		"--output", "json",
		"--from-file", networkFile,
		"--network", "i-do-not-exist",
	})

	// then
	require.Error(t, err)
	require.Nil(t, describeResp)
}
