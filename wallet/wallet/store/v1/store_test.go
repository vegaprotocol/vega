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
	"code.vegaprotocol.io/vega/wallet/api"
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
	t.Run("Getting unlocked wallet succeeds", testFileStoreV1GetUnlockedWalletSucceeds)
	t.Run("Getting locked wallet succeeds", testFileStoreV1GetLockedWalletFails)
	t.Run("Getting non-existing wallet fails", testFileStoreV1GetNonExistingWalletFails)
	t.Run("Unlocking wallet succeeds", testFileStoreV1UnlockingWalletSucceeds)
	t.Run("Unlocking wallet without wrong passphrase fails", testFileStoreV1UnlockingWalletWithWrongPassphraseFails)
	t.Run("Locked wallet is not accessible anymore", testFileStoreV1LockedWalletIsNotAccessibleAnymore)
	t.Run("Getting wallet path succeeds", testFileStoreV1GetWalletPathSucceeds)
	t.Run("Verifying non-existing wallet fails", testFileStoreV1NonExistingWalletFails)
	t.Run("Verifying existing wallet succeeds", testFileStoreV1ExistingWalletSucceeds)
	t.Run("Updating wallet succeeds", testFileStoreV1UpdatingWalletSucceeds)
	t.Run("Updating locked wallet fails", testFileStoreV1UpdatingLockedWalletFails)
	t.Run("Renaming wallet succeeds", testFileStoreV1RenamingWalletSucceeds)
	t.Run("Renaming non-existing wallet fails", testFileStoreV1RenamingNonExistingWalletFails)
	t.Run("Renaming wallet with invalid name fails", testFileStoreV1RenamingWalletWithInvalidNameFails)
	t.Run("Deleting wallet succeeds", testFileStoreV1DeletingWalletSucceeds)
	t.Run("Updating passphrase succeeds", testFileStoreV1UpdatingPassphraseSucceeds)
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
		err := s.CreateWallet(context.Background(), w, passphrase)

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
		err := s.CreateWallet(context.Background(), w, passphrase)

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
		err := s.CreateWallet(context.Background(), w, passphrase)

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

func testFileStoreV1UnlockingWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	err = s.UnlockWallet(context.Background(), w.Name(), passphrase)

	// then
	require.NoError(t, err)
}

func testFileStoreV1UnlockingWalletWithWrongPassphraseFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)
	othPassphrase := "not-original-passphrase"

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	err = s.UnlockWallet(context.Background(), w.Name(), othPassphrase)

	// then
	assert.ErrorIs(t, err, wallet.ErrWrongPassphrase)
}

func testFileStoreV1LockedWalletIsNotAccessibleAnymore(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	w2, err := s.GetWallet(context.Background(), w.Name())

	// then
	assert.ErrorIs(t, err, api.ErrWalletIsLocked)
	assert.Nil(t, w2)
}

func testFileStoreV1GetUnlockedWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)
	assert.Equal(t, w, returnedWallet)
}

func testFileStoreV1GetLockedWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), w.Name())

	// then
	assert.ErrorIs(t, err, api.ErrWalletIsLocked)
	assert.Nil(t, returnedWallet)
}

func testFileStoreV1GetNonExistingWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	name := vgrand.RandomStr(5)

	// when
	returnedWallet, err := s.GetWallet(context.Background(), name)

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
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	exists, err := s.WalletExists(context.Background(), w.Name())

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func testFileStoreV1UpdatingWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	passphrase := vgrand.RandomStr(5)
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.UpdateWallet(context.Background(), w)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	buf, err := os.ReadFile(filepath.Join(walletsDir, w.Name()))
	if err != nil {
		t.Fatalf("couldn't read wallet file: %v", w.Name())
	}
	assert.NotEmpty(t, buf)
}

func testFileStoreV1UpdatingLockedWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	passphrase := vgrand.RandomStr(5)
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	err = s.UpdateWallet(context.Background(), w)

	// then
	assert.Error(t, err, api.ErrWalletIsLocked)
}

func testFileStoreV1RenamingWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	previousName := w.Name()
	newName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	// when
	err = s.RenameWallet(context.Background(), w.Name(), newName)

	// then
	assert.NoError(t, err)
	vgtest.AssertNoFile(t, filepath.Join(walletsDir, previousName))
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, newName))

	// when
	w1, err := s.GetWallet(context.Background(), previousName)

	// then
	assert.Error(t, err, api.ErrWalletDoesNotExist)
	assert.Nil(t, w1)

	// when
	w2, err := s.GetWallet(context.Background(), newName)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, w2)
}

func testFileStoreV1RenamingNonExistingWalletFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	unknownName := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// when
	err := s.RenameWallet(context.Background(), unknownName, newName)

	// then
	assert.ErrorIs(t, err, api.ErrWalletDoesNotExist)
	vgtest.AssertNoFile(t, filepath.Join(walletsDir, unknownName))
	vgtest.AssertNoFile(t, filepath.Join(walletsDir, newName))
}

func testFileStoreV1RenamingWalletWithInvalidNameFails(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	tcs := []struct {
		name    string
		newName string
		err     error
	}{
		{
			name:    "starting with a dot",
			newName: ".start-with-dot",
			err:     storev1.ErrWalletNameCannotStartWithDot,
		}, {
			name:    "containing slashes",
			newName: "contains/multiple/slashes/",
			err:     storev1.ErrWalletNameCannotContainSlashCharacters,
		}, {
			name:    "containing back-slashes",
			newName: "contains\\multiple\\slashes\\",
			err:     storev1.ErrWalletNameCannotContainSlashCharacters,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			err := s.RenameWallet(context.Background(), w.Name(), tc.newName)

			// then
			require.ErrorIs(tt, err, tc.err)
			vgtest.AssertNoFile(tt, filepath.Join(walletsDir, tc.newName))
		})
	}
}

func testFileStoreV1DeletingWalletSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	// when
	err = s.DeleteWallet(context.Background(), w.Name())

	// then
	assert.NoError(t, err)
	vgtest.AssertNoFile(t, filepath.Join(walletsDir, w.Name()))
}

func testFileStoreV1UpdatingPassphraseSucceeds(t *testing.T) {
	walletsDir := newWalletsDir(t)

	// given
	s := initialiseStore(t, walletsDir)
	w := newHDWalletWithKeys(t)
	passphrase := vgrand.RandomStr(5)
	newPassphrase := vgrand.RandomStr(5)

	// when
	err := s.CreateWallet(context.Background(), w, passphrase)

	// then
	require.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	// when
	err = s.UpdatePassphrase(context.Background(), w.Name(), newPassphrase)

	// then
	assert.NoError(t, err)
	vgtest.AssertFileAccess(t, filepath.Join(walletsDir, w.Name()))

	// when
	err = s.LockWallet(context.Background(), w.Name())

	// then
	require.NoError(t, err)

	// when
	err = s.UnlockWallet(context.Background(), w.Name(), newPassphrase)

	// then
	require.NoError(t, err)
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
