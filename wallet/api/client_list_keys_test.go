package api_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKeys(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Run("Listing keys with valid params succeeds", testListingKeysWithValidParamsSucceeds)
	}
	t.Run("Listing keys with invalid params fails", testListingKeysWithInvalidParamsFails)
	t.Run("Listing keys excludes tainted keys", testListingKeysExcludesTaintedKeys)
	t.Run("Listing keys with invalid token fails", testListingKeysWithInvalidTokenFails)
	t.Run("Listing keys with long-living token succeeds", testListingKeysWithLongLivingTokenSucceeds)
	t.Run("Listing keys with long-living expiring token succeeds", testListingKeysWithLongLivingExpiringTokenSucceeds)
	t.Run("List keys with long expired token succeed", testListingKeysWithLongExpiredTokenSucceeds)
	t.Run("Listing keys with not enough permissions fails", testListingKeysWithNotEnoughPermissionsFails)
	t.Run("Cancelling the review does not update the permissions", testListingKeysCancellingTheReviewDoesNotUpdatePermissions)
	t.Run("Interrupting the request does not update the permissions", testListingKeysInterruptingTheRequestDoesNotUpdatePermissions)
	t.Run("Interrupting the request does not update the permissions", testListingKeysInterruptingTheRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the review does not update the permissions", testListingKeysGettingInternalErrorDuringReviewDoesNotUpdatePermissions)
	t.Run("Cancelling the passphrase request does not update the permissions", testListingKeysCancellingThePassphraseRequestDoesNotUpdatePermissions)
	t.Run("Interrupting the request during the passphrase request does not update the permissions", testListingKeysInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the passphrase request does not update the permissions", testListingKeysGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Using wrong passphrase does not update the permissions", testListingKeysUsingWrongPassphraseDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet retrieval does not update the permissions", testListingKeysGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet saving does not update the permissions", testListingKeysGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions)
	t.Run("Updating the permissions does not overwrite untracked changes", testListingKeysUpdatingPermissionsDoesNotOverwriteUntrackedChanges)
}

func testListingKeysWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty connection token",
			params: api.ClientListKeysParams{
				Token: "",
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()
			metadata := requestMetadataForTest()

			// setup
			handler := newListKeysHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, metadata)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testListingKeysWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	w, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	})
	_, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate key for tests: %v", err)
	}
	expectedPubKeys := make([]api.ClientNamedPublicKey, 0, len(w.ListPublicKeys()))
	for _, key := range w.ListPublicKeys() {
		expectedPubKeys = append(expectedPubKeys, api.ClientNamedPublicKey{
			Name:      key.Name(),
			PublicKey: key.Key(),
		})
	}

	sort.Slice(expectedPubKeys, func(i, j int) bool { return expectedPubKeys[i].PublicKey < expectedPubKeys[j].PublicKey })

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, expectedPubKeys, result.Keys)
}

func testListingKeysExcludesTaintedKeys(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	w, kp1 := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	})
	kp2, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate key for tests: %v", err)
	}
	if err = w.TaintKey(kp2.PublicKey()); err != nil {
		t.Fatalf("could not taint key for tests: %v", err)
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, []api.ClientNamedPublicKey{
		{
			Name:      kp1.Name(),
			PublicKey: kp1.PublicKey(),
		},
	}, result.Keys)
}

func testListingKeysWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()

	// setup
	handler := newListKeysHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: vgrand.RandomStr(5),
	}, metadata)

	// then
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, session.ErrNoWalletConnected)
}

func testListingKeysWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	w, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	// setup
	handler := newListKeysHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, w, time.Now(), nil); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions %v", err)
	}

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	assert.Equal(t, []api.ClientNamedPublicKey{
		{
			Name:      "Key 1",
			PublicKey: kp.PublicKey(),
		},
	}, result.Keys)
}

func testListingKeysWithLongLivingExpiringTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	w, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	now := time.Now()

	expiry := now.Add(24 * time.Hour)

	// setup
	handler := newListKeysHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, w, now, &expiry); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions %v", err)
	}

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	assert.Equal(t, []api.ClientNamedPublicKey{
		{
			Name:      "Key 1",
			PublicKey: kp.PublicKey(),
		},
	}, result.Keys)
}

func testListingKeysWithLongExpiredTokenSucceeds(t *testing.T) {
	// given
	w, _ := walletWithKey(t)
	token := vgrand.RandomStr(10)

	// long expired token
	now := time.Now()
	expiry := now.Add(-24 * time.Hour)

	// setup
	handler := newListKeysHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, w, now, &expiry); !errors.Is(err, session.ErrAPITokenExpired) {
		t.Fatalf("expected %v got %v", session.ErrAPITokenExpired, err)
	}
}

func testListingKeysWithNotEnoughPermissionsFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.NoAccess,
			RestrictedKeys: []string{},
		},
	}
	originalPermissions := wallet.Permissions{}
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}
	w, _ := walletWithPerms(t, metadata.Hostname, expectedPermissions)
	_, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate key for tests: %v", err)
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, w)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, w.Name(), requestedPermissions).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)

	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysCancellingTheReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysInterruptingTheRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysGettingInternalErrorDuringReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the permissions review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysCancellingThePassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the passphrase failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysUsingWrongPassphraseDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(cancelCtx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(cancelCtx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(cancelCtx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(cancelCtx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(cancelCtx, wallet1.Name(), passphrase).Times(1).Return(nil, wallet.ErrWrongPassphrase)
	handler.interactor.EXPECT().NotifyError(cancelCtx, metadata.TraceID, api.UserError, wallet.ErrWrongPassphrase).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})

	// when
	result, errorDetails := handler.handle(t, cancelCtx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	passphrase := vgrand.RandomStr(5)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	walletName := vgrand.RandomStr(5)
	originalPermissions := wallet.Permissions{}
	wallet1, recoveryPhrase, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}
	if _, err := wallet1.GenerateKeyPair(nil); err != nil {
		t.Fatalf("could not generate key for test: %v", err)
	}

	// Clone the wallet1, so we can emulate a different instance returned by
	// the wallet store.
	loadedWallet, err := wallet.ImportHDWallet(walletName, recoveryPhrase, 2)
	if err != nil {
		t.Fatalf("could not import wallet for test: %v", err)
	}
	if _, err := loadedWallet.GenerateKeyPair(nil); err != nil {
		t.Fatalf("could not generate key for test: %v", err)
	}
	passphrase := vgrand.RandomStr(5)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(loadedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, loadedWallet, passphrase).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("could not save the wallet: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is not updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testListingKeysUpdatingPermissionsDoesNotOverwriteUntrackedChanges(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	walletName := vgrand.RandomStr(5)
	wallet1, recoveryPhrase, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	// Clone the wallet1, so we can modify the clone without tempering with wallet1.
	modifiedWallet, err := wallet.ImportHDWallet(walletName, recoveryPhrase, 2)
	if err != nil {
		t.Fatalf("could not import wallet for test: %v", err)
	}
	kp, _ := modifiedWallet.GenerateKeyPair([]wallet.Metadata{{Key: "name", Value: "hello"}})

	passphrase := vgrand.RandomStr(5)
	askedPermissions := wallet.PermissionsSummary{
		"public_keys": "read",
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), askedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(modifiedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).DoAndReturn(func(_ context.Context, w wallet.Wallet, _ string) error {
		// We verify that the saved wallet contains the modification from the modified wallet and the permissions update from wallet1.
		assert.Equal(t, []wallet.KeyPair{kp}, w.ListKeyPairs())
		assert.Equal(t, wallet.Permissions{
			PublicKeys: wallet.PublicKeysPermission{
				Access: wallet.ReadAccess,
			},
		}, w.Permissions(metadata.Hostname))
		return nil
	})
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, metadata.TraceID, api.PermissionsSuccessfullyUpdated).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientListKeysParams{
		Token: token,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, askedPermissions, connectedWallet.Permissions().Summary())
}

type listKeysHandler struct {
	*api.ClientListKeys
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	interactor  *mocks.MockInteractor
	sessions    *session.Sessions
}

func (h *listKeysHandler) handle(t *testing.T, ctx context.Context, params interface{}, metadata jsonrpc.RequestMetadata) (api.ClientListKeysResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, metadata)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientListKeysResult)
		if !ok {
			t.Fatal("ClientListKeys handler result is not a ClientListKeysResult")
		}
		return result, err
	}
	return api.ClientListKeysResult{}, err
}

func newListKeysHandler(t *testing.T) *listKeysHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)

	sessions := session.NewSessions()

	return &listKeysHandler{
		ClientListKeys: api.NewListKeys(walletStore, interactor, sessions),
		ctrl:           ctrl,
		walletStore:    walletStore,
		interactor:     interactor,
		sessions:       sessions,
	}
}
