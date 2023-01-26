package v1_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	v1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore(t *testing.T) {
	t.Run("List tokens succeeds", testFileStoreListTokensSucceeds)
	t.Run("Verifying an existing token succeeds", testFileStoreVerifyingExistingTokenSucceeds)
	t.Run("Verifying an unknown token fails", testFileStoreVerifyingUnknownTokenFails)
	t.Run("Deleting an existing token succeeds", testFileStoreDeletingExistingTokenSucceeds)
	t.Run("Changes to the token file are propagated to the listeners", testFileStoreChangesToTokenFileArePropagatedToListeners)
}

func testFileStoreListTokensSucceeds(t *testing.T) {
	vegaPaths := testHome(t)
	store := newTestFileStore(t, vegaPaths)

	// given
	description1 := connections.TokenDescription{
		Description:    vgrand.RandomStr(5),
		CreationDate:   time.Now().Add(-2 * time.Hour),
		ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
		Token:          connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// when
	err := store.SaveToken(description1)

	// then
	require.NoError(t, err)
	// given
	description2 := connections.TokenDescription{
		Description:  vgrand.RandomStr(5),
		CreationDate: time.Now().Add(-4 * time.Hour),
		Token:        connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// when
	err = store.SaveToken(description2)

	// then
	require.NoError(t, err)
	// when
	tokenSummaries, err := store.ListTokens()

	// then
	require.NoError(t, err)
	assert.Equal(t, description1.Description, tokenSummaries[0].Description)
	assert.Equal(t, description1.Token, tokenSummaries[0].Token)
	assert.WithinDuration(t, description1.CreationDate, tokenSummaries[0].CreationDate, 0)
	assert.WithinDuration(t, *description1.ExpirationDate, *tokenSummaries[0].ExpirationDate, 0)
	assert.Equal(t, description2.Description, tokenSummaries[1].Description)
	assert.Equal(t, description2.Token, tokenSummaries[1].Token)
	assert.WithinDuration(t, description2.CreationDate, tokenSummaries[1].CreationDate, 0)
	assert.Equal(t, description2.ExpirationDate, tokenSummaries[1].ExpirationDate, 0)
}

func testFileStoreVerifyingExistingTokenSucceeds(t *testing.T) {
	vegaPaths := testHome(t)
	store := newTestFileStore(t, vegaPaths)

	// given
	description1 := connections.TokenDescription{
		Description:    vgrand.RandomStr(5),
		CreationDate:   time.Now().Add(-2 * time.Hour),
		ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
		Token:          connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// when
	err := store.SaveToken(description1)

	// then
	require.NoError(t, err)

	// when
	exists, err := store.TokenExists(description1.Token)

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func testFileStoreVerifyingUnknownTokenFails(t *testing.T) {
	vegaPaths := testHome(t)
	store := newTestFileStore(t, vegaPaths)

	// given
	unknownToken := connections.GenerateToken()

	// when
	exists, err := store.TokenExists(unknownToken)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func testFileStoreDeletingExistingTokenSucceeds(t *testing.T) {
	vegaPaths := testHome(t)
	store := newTestFileStore(t, vegaPaths)

	// given
	description1 := connections.TokenDescription{
		Description:    vgrand.RandomStr(5),
		CreationDate:   time.Now().Add(-2 * time.Hour),
		ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
		Token:          connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// when
	err := store.SaveToken(description1)

	// then
	require.NoError(t, err)

	// given
	description2 := connections.TokenDescription{
		Description:  vgrand.RandomStr(5),
		CreationDate: time.Now().Add(-4 * time.Hour),
		Token:        connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// when
	err = store.SaveToken(description2)

	// then
	require.NoError(t, err)

	// when
	err = store.DeleteToken(description1.Token)

	// then
	require.NoError(t, err)

	// when
	exists, err := store.TokenExists(description1.Token)

	// then
	require.NoError(t, err)
	assert.False(t, exists)

	// when
	exists, err = store.TokenExists(description2.Token)

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func testFileStoreChangesToTokenFileArePropagatedToListeners(t *testing.T) {
	vegaPaths := testHome(t)
	store := newTestFileStore(t, vegaPaths)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()

	// given
	description1 := connections.TokenDescription{
		Description:    vgrand.RandomStr(5),
		CreationDate:   time.Now().Add(-2 * time.Hour),
		ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
		Token:          connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}
	description2 := connections.TokenDescription{
		Description:  vgrand.RandomStr(5),
		CreationDate: time.Now().Add(-4 * time.Hour),
		Token:        connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}
	description3 := connections.TokenDescription{
		Description:  vgrand.RandomStr(5),
		CreationDate: time.Now().Add(-2 * time.Hour).Truncate(time.Second),
		Token:        connections.GenerateToken(),
		Wallet: connections.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	wg := sync.WaitGroup{}
	checkStep := 1
	completed := false
	store.OnUpdate(func(_ context.Context, tokenDescriptions ...connections.TokenDescription) {
		if checkStep == 1 && len(tokenDescriptions) == 1 && tokenDescriptions[0].Token == description1.Token {
			checkStep = 2
			wg.Done()
		} else if checkStep == 2 && len(tokenDescriptions) == 2 && tokenDescriptions[0].Token == description1.Token && tokenDescriptions[1].Token == description2.Token {
			checkStep = 3
			wg.Done()
		} else if checkStep == 3 && len(tokenDescriptions) == 1 && tokenDescriptions[0].Token == description2.Token {
			checkStep = 4
			wg.Done()
		} else if checkStep == 4 && len(tokenDescriptions) == 0 {
			checkStep = 5
			wg.Done()
		} else if checkStep == 5 && len(tokenDescriptions) == 1 && tokenDescriptions[0].Token == description3.Token {
			completed = true
			wg.Done()
			cancelFunc()
		} else {
			t.Logf("A file system event has been received but didn't match any checks: %v", tokenDescriptions)
		}
	})

	// 1. Ensure the creation of a token is broadcast to listeners, a first time.
	wg.Add(1)

	// when
	err := store.SaveToken(description1)

	// then
	require.NoError(t, err)
	wg.Wait()

	// 2. Ensure the creation of a token is broadcast to listeners, a second time.
	wg.Add(1)

	// when
	err = store.SaveToken(description2)

	// then
	require.NoError(t, err)
	wg.Wait()

	// 3. Verifying the deleted token is not broadcast to listeners.
	wg.Add(1)

	// when
	err = store.DeleteToken(description1.Token)

	// then
	require.NoError(t, err)
	wg.Wait()

	// 4. Verifying the deletion of the file is interpreted as a removal of all
	//    the long-living tokens.
	wg.Add(1)

	// when
	err = os.RemoveAll(vegaPaths.DataPathFor(paths.WalletServiceTokensDataFile))

	// then
	require.NoError(t, err)
	wg.Wait()

	// 5. Verifying we can write after a deletion of the file.
	wg.Add(1)

	// when
	err = store.SaveToken(description3)

	// then
	require.NoError(t, err)
	wg.Wait()

	<-ctx.Done()

	if !completed {
		t.Errorf("The test couldn't be completed in due time. No FS event could pass the check %d.", checkStep)
	}
}

type testFileStore struct {
	*v1.FileStore
}

func testHome(t *testing.T) paths.Paths {
	t.Helper()
	return paths.New(t.TempDir())
}

func newTestFileStore(t *testing.T, vegaPaths paths.Paths) *testFileStore {
	t.Helper()

	tokenStore, err := v1.ReinitialiseStore(vegaPaths, vgrand.RandomStr(5))
	if err != nil {
		t.Fatalf("could not initialise the file store for tests: %v", err)
	}
	t.Cleanup(func() {
		tokenStore.Close()
	})

	return &testFileStore{
		FileStore: tokenStore,
	}
}
