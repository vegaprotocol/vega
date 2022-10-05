package v1_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/wallet/wallet"
	storev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStoreV1(t *testing.T) {
	t.Run("Initialising store succeeds", testInitialisingStoreSucceeds)
	t.Run("Listing wallets succeeds", testFileStoreV1ListWalletsSucceeds)
	t.Run("Listing wallets does not show hidden files", testFileStoreV1ListWalletsDoesNotShowHiddenFiles)
	t.Run("Listing wallets does not show directories", testFileStoreV1ListWalletsDoesNotShowDirectories)
	t.Run("Getting wallet succeeds", testFileStoreV1GetWalletSucceeds)
	t.Run("Getting wallet without wrong passphrase fails", testFileStoreV1GetWalletWithWrongPassphraseFails)
	t.Run("Getting non-existing wallet fails", testFileStoreV1GetNonExistingWalletFails)
	t.Run("Getting wallet path succeeds", testFileStoreV1GetWalletPathSucceeds)
	t.Run("Verifying non-existing wallet fails", testFileStoreV1NonExistingWalletFails)
	t.Run("Verifying existing wallet succeeds", testFileStoreV1ExistingWalletSucceeds)
	t.Run("Saving wallet succeeds", testFileStoreV1SaveWalletSucceeds)
	t.Run("Saving wallet with invalid name fails ", testFileStoreV1SaveWalletWithInvalidNameFails)
}

func testInitialisingStoreSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	s, err := storev1.InitialiseStore(walletsDir)

	require.NoError(t, err)
	assert.NotNil(t, s)
	vgtest.AssertDirAccess(t, walletsDir)
}

func testFileStoreV1ListWalletsSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	passphrase := vgrand.RandomStr(5)

	var expectedWallets []string
	for i := 0; i < 3; i++ {
		w := newHDWalletWithKeys(t)

		// when
		err := s.SaveWallet(context.Background(), w, passphrase)

		// then
		require.NoError(t, err)

		expectedWallets = append(expectedWallets, w.Name())
	}
	sort.Strings(expectedWallets)

	// when
	returnedWallets, err := s.ListWallets(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedWallets, returnedWallets)
}

func testFileStoreV1ListWalletsDoesNotShowHiddenFiles(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	passphrase := vgrand.RandomStr(5)

	var expectedWallets []string
	for i := 0; i < 3; i++ {
		w := newHDWalletWithKeys(t)

		// when
		err := s.SaveWallet(context.Background(), w, passphrase)

		// then
		require.NoError(t, err)

		expectedWallets = append(expectedWallets, w.Name())
	}
	sort.Strings(expectedWallets)

	for i := 0; i < 3; i++ {
		hiddenFileName := "." + vgrand.RandomStr(4)
		hiddenFilePath := filepath.Join(walletsDir, hiddenFileName)
		if err := os.WriteFile(hiddenFilePath, []byte(""), 0o0600); err != nil {
			t.Fatalf("could not write hidden file: %v", err)
		}
	}

	// when
	returnedWallets, err := s.ListWallets(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedWallets, returnedWallets)
}

func testFileStoreV1ListWalletsDoesNotShowDirectories(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	passphrase := vgrand.RandomStr(5)

	var expectedWallets []string
	for i := 0; i < 3; i++ {
		w := newHDWalletWithKeys(t)

		// when
		err := s.SaveWallet(context.Background(), w, passphrase)

		// then
		require.NoError(t, err)

		expectedWallets = append(expectedWallets, w.Name())
	}
	sort.Strings(expectedWallets)

	for i := 0; i < 3; i++ {
		dirName := "." + vgrand.RandomStr(4)
		dirPath := filepath.Join(walletsDir, dirName)
		if err := os.Mkdir(dirPath, 0o0600); err != nil {
			t.Fatalf("could not create directory: %v", err)
		}
	}

	// when
	returnedWallets, err := s.ListWallets(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedWallets, returnedWallets)
}

func testFileStoreV1GetWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.SaveWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), w.Name(), passphrase)

	// then
	require.NoError(t, err)
	assert.Equal(t, w, returnedWallet)
}

func testFileStoreV1GetWalletWithWrongPassphraseFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)
	othPassphrase := "not-original-passphrase"

	// when
	err := s.SaveWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), w.Name(), othPassphrase)

	// then
	assert.ErrorIs(t, err, wallet.ErrWrongPassphrase)
	assert.Nil(t, returnedWallet)
}

func testFileStoreV1GetNonExistingWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	name := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), name, passphrase)

	// then
	assert.Error(t, err)
	assert.Nil(t, returnedWallet)
}

func testFileStoreV1GetWalletPathSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	name := vgrand.RandomStr(5)

	// when
	path := s.GetWalletPath(name)

	// then
	assert.Equal(t, filepath.Join(walletsDir, name), path)
}

func testFileStoreV1NonExistingWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	name := vgrand.RandomStr(5)

	// when
	exists, err := s.WalletExists(context.Background(), name)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func testFileStoreV1ExistingWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.SaveWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	exists, err := s.WalletExists(context.Background(), w.Name())

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func testFileStoreV1SaveWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	passphrase := vgrand.RandomStr(5)
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)

	// when
	err := s.SaveWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	buf, err := os.ReadFile(filepath.Join(walletsDir, w.Name()))
	if err != nil {
		t.Fatalf("couldn't read wallet file: %v", w.Name())
	}
	assert.NotEmpty(t, buf)
}

func testFileStoreV1SaveWalletWithInvalidNameFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	passphrase := vgrand.RandomStr(5)
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)

	tcs := []struct {
		name   string
		wallet string
		err    error
	}{
		{
			name:   "starting with a dot",
			wallet: ".start-with-dot",
			err:    storev1.ErrWalletNameCannotStartWithDot,
		}, {
			name:   "containing slashes",
			wallet: "contains/multiple/slashes/",
			err:    storev1.ErrWalletNameCannotContainSlashCharacters,
		}, {
			name:   "containing back-slashes",
			wallet: "contains\\multiple\\slashes\\",
			err:    storev1.ErrWalletNameCannotContainSlashCharacters,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			w.SetName(tc.wallet)

			// when
			err := s.SaveWallet(context.Background(), w, passphrase)

			// then
			require.ErrorIs(tt, err, tc.err)
			vgtest.AssertNoFile(tt, filepath.Join(walletsDir, w.Name()))
		})
	}
}

func initialiseStore(t *testing.T, walletsDir string) *storev1.Store {
	t.Helper()
	s, err := storev1.InitialiseStore(walletsDir)
	if err != nil {
		t.Fatalf("couldn't initialise store: %v", err)
	}

	return s
}

func newHDWalletWithKeys(t *testing.T) *wallet.HDWallet {
	t.Helper()
	w, _, err := wallet.NewHDWallet(fmt.Sprintf("my-wallet-%v", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("couldn't create wallet: %v", err)
	}

	_, err = w.GenerateKeyPair([]wallet.Metadata{})
	if err != nil {
		t.Fatalf("couldn't generate key: %v", err)
	}

	return w
}

func newWalletsDir(t *testing.T) string {
	t.Helper()
	rootPath := filepath.Join("/tmp", "vegawallet", vgrand.RandomStr(10))
	t.Cleanup(func() {
		if err := os.RemoveAll(rootPath); err != nil {
			t.Fatalf("couldn't remove vega home: %v", err)
		}
	})

	return rootPath
}
