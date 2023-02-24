package v1_test

import (
	"os"
	"path/filepath"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/network"
	v1 "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStoreV1(t *testing.T) {
	t.Run("New store succeeds", testNewStoreSucceeds)
	t.Run("Saving already existing network succeeds", testFileStoreV1SaveAlreadyExistingNetworkSucceeds)
	t.Run("Saving network succeeds", testFileStoreV1SaveNetworkSucceeds)
	t.Run("Saving network with bad name fails", testFileStoreV1SaveNetworkWithBadNameFails)
	t.Run("Verifying non-existing network fails", testFileStoreV1VerifyingNonExistingNetworkFails)
	t.Run("Verifying existing network succeeds", testFileStoreV1VerifyingExistingNetworkSucceeds)
	t.Run("Getting non-existing network fails", testFileStoreV1GetNonExistingNetworkFails)
	t.Run("Getting existing network succeeds", testFileStoreV1GetExistingNetworkSucceeds)
	t.Run("Getting network path succeeds", testFileStoreV1GetNetworkPathSucceeds)
	t.Run("Getting networks path succeeds", testFileStoreV1GetNetworksPathSucceeds)
	t.Run("Listing networks succeeds", testFileStoreV1ListingNetworksSucceeds)
	t.Run("Deleting network succeeds", testFileStoreV1DeleteNetworkSucceeds)
	t.Run("Renaming network succeeds", testFileStoreV1RenamingNetworkSucceeds)
	t.Run("Renaming non-existing network fails", testFileStoreV1RenamingNonExistingNetworkFails)
	t.Run("Renaming network with invalid name fails", testFileStoreV1RenamingNetworkWithInvalidNameFails)
}

func testNewStoreSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	s, err := v1.InitialiseStore(vegaHome)

	require.NoError(t, err)
	assert.NotNil(t, s)
	vgtest.AssertDirAccess(t, networksHome(t, vegaHome))
}

func testFileStoreV1SaveAlreadyExistingNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)
	net := &network.Network{
		Name: "test",
	}

	// when
	err := s.SaveNetwork(net)

	// then
	require.NoError(t, err)

	// when
	err = s.SaveNetwork(net)

	// then
	require.NoError(t, err)
}

func testFileStoreV1SaveNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)
	net := &network.Network{
		Name: vgrand.RandomStr(10),
	}

	// when
	err := s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, networkPath(t, vegaHome, net.Name))

	// confirm that the network name is not saved in the file
	netpath := networkPath(t, vegaHome, net.Name)
	b, err := os.ReadFile(netpath)
	assert.NoError(t, err)
	assert.NotContains(t, string(b), net.Name)

	// when
	returnedNet, err := s.GetNetwork(net.Name)

	// then
	require.NoError(t, err)
	assert.Equal(t, net, returnedNet)
	assert.Equal(t, net.Name, returnedNet.Name)
}

func testFileStoreV1SaveNetworkWithBadNameFails(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)

	tcs := []struct {
		name        string
		network     string
		expectedErr error
	}{
		{
			name:        "when empty",
			network:     "",
			expectedErr: v1.ErrNetworkNameCannotBeEmpty,
		}, {
			name:        "starting with `.`",
			network:     "." + vgrand.RandomStr(3),
			expectedErr: v1.ErrNetworkNameCannotStartWithDot,
		}, {
			name:        "with `/`",
			network:     "\\" + vgrand.RandomStr(3) + "/" + vgrand.RandomStr(3),
			expectedErr: v1.ErrNetworkNameCannotContainSlash,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			net := &network.Network{
				Name: tc.network,
			}

			// when
			err := s.SaveNetwork(net)

			// then
			require.ErrorIs(tt, err, tc.expectedErr)
		})
	}
}

func testFileStoreV1VerifyingNonExistingNetworkFails(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)

	// when
	exists, err := s.NetworkExists("test")

	// then
	assert.NoError(t, err)
	assert.False(t, exists)
}

func testFileStoreV1VerifyingExistingNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)
	net := &network.Network{
		Name: "test",
	}

	// when
	err := s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, networkPath(t, vegaHome, net.Name))

	// when
	exists, err := s.NetworkExists("test")

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func testFileStoreV1GetNonExistingNetworkFails(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)

	// when
	keys, err := s.GetNetwork("test")

	// then
	assert.Error(t, err)
	assert.Nil(t, keys)
}

func testFileStoreV1GetExistingNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)
	net := &network.Network{
		Name: "test",
	}

	// when
	err := s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, networkPath(t, vegaHome, net.Name))

	// when
	returnedNet, err := s.GetNetwork("test")

	// then
	require.NoError(t, err)
	assert.Equal(t, net, returnedNet)
}

func testFileStoreV1GetNetworkPathSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)

	// when
	returnedPath := s.GetNetworkPath("test")

	// then
	assert.Equal(t, networkPath(t, vegaHome, "test"), returnedPath)
}

func testFileStoreV1GetNetworksPathSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)

	// when
	returnedPath := s.GetNetworksPath()

	// then
	assert.Equal(t, networksHome(t, vegaHome), returnedPath)
}

func testFileStoreV1ListingNetworksSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s := initialiseFromPath(t, vegaHome)
	net := &network.Network{
		// we use "toml" as name on purpose since we want to verify it's not
		// stripped by the ListNetwork() function.
		Name: "toml",
	}

	// when
	err := s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, networkPath(t, vegaHome, net.Name))

	// when
	nets, err := s.ListNetworks()

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"toml"}, nets)
}

func testFileStoreV1DeleteNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// Create a network for us to delete
	s, err := v1.InitialiseStore(vegaHome)
	require.NoError(t, err)
	assert.NotNil(t, s)

	net := &network.Network{
		Name: "test",
	}

	err = s.SaveNetwork(net)
	require.NoError(t, err)

	// Check it's really there
	returnedNet, err := s.GetNetwork("test")
	require.NoError(t, err)
	assert.Equal(t, net, returnedNet)

	// Now delete it
	err = s.DeleteNetwork("test")
	require.NoError(t, err)

	// Check it's no longer there
	returnedNet, err = s.GetNetwork("test")
	require.Error(t, err)
	assert.Nil(t, returnedNet)
}

func testFileStoreV1RenamingNetworkSucceeds(t *testing.T) {
	vegaHome := newVegaHome(t)

	// Create a network for us to rename
	s, err := v1.InitialiseStore(vegaHome)
	require.NoError(t, err)
	assert.NotNil(t, s)

	// given
	net := &network.Network{
		Name: "test",
	}

	// when
	err = s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, s.GetNetworkPath(net.Name))

	// given
	newName := vgrand.RandomStr(5)

	// when
	err = s.RenameNetwork(net.Name, newName)

	// then
	assert.NoError(t, err)
	vgtest.AssertNoFile(t, s.GetNetworkPath(net.Name))
	vgtest.AssertFileAccess(t, s.GetNetworkPath(newName))

	// when
	w1, err := s.GetNetwork(net.Name)

	// then
	assert.Error(t, err, api.ErrNetworkDoesNotExist)
	assert.Nil(t, w1)

	// when
	w2, err := s.GetNetwork(newName)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, w2)
}

func testFileStoreV1RenamingNonExistingNetworkFails(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s, err := v1.InitialiseStore(vegaHome)
	require.NoError(t, err)
	assert.NotNil(t, s)

	// given
	unknownName := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// when
	err = s.RenameNetwork(unknownName, newName)

	// then
	assert.ErrorIs(t, err, api.ErrNetworkDoesNotExist)
	vgtest.AssertNoFile(t, s.GetNetworkPath(unknownName))
	vgtest.AssertNoFile(t, s.GetNetworkPath(newName))
}

func testFileStoreV1RenamingNetworkWithInvalidNameFails(t *testing.T) {
	vegaHome := newVegaHome(t)

	// given
	s, err := v1.InitialiseStore(vegaHome)
	require.NoError(t, err)
	assert.NotNil(t, s)

	// given
	net := &network.Network{
		Name: "test",
	}

	// when
	err = s.SaveNetwork(net)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, s.GetNetworkPath(net.Name))

	tcs := []struct {
		name    string
		newName string
		err     error
	}{
		{
			name:    "empty",
			newName: "",
			err:     v1.ErrNetworkNameCannotBeEmpty,
		}, {
			name:    "starting with a dot",
			newName: ".start-with-dot",
			err:     v1.ErrNetworkNameCannotStartWithDot,
		}, {
			name:    "containing slashes",
			newName: "contains/multiple/slashes/",
			err:     v1.ErrNetworkNameCannotContainSlash,
		}, {
			name:    "containing back-slashes",
			newName: "contains\\multiple\\slashes\\",
			err:     v1.ErrNetworkNameCannotContainSlash,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			err := s.RenameNetwork(net.Name, tc.newName)

			// then
			require.ErrorIs(tt, err, tc.err)
			vgtest.AssertNoFile(tt, s.GetNetworkPath(tc.newName))
		})
	}
}

func initialiseFromPath(t *testing.T, vegaHome paths.Paths) *v1.FileStore {
	t.Helper()
	s, err := v1.InitialiseStore(vegaHome)
	if err != nil {
		t.Fatalf("couldn't initialise store: %v", err)
	}
	return s
}

func newVegaHome(t *testing.T) *paths.CustomPaths {
	t.Helper()
	return &paths.CustomPaths{CustomHome: t.TempDir()}
}

func networksHome(t *testing.T, vegaHome *paths.CustomPaths) string {
	t.Helper()
	return vegaHome.ConfigPathFor(paths.WalletServiceNetworksConfigHome)
}

func networkPath(t *testing.T, vegaHome *paths.CustomPaths, name string) string {
	t.Helper()
	return filepath.Join(networksHome(t, vegaHome), name+".toml")
}
