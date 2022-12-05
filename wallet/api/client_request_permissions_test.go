package api_test

import (
	"context"
	"errors"
	"fmt"
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

func TestRequestPermissions(t *testing.T) {
	t.Run("Requesting permissions with invalid params fails", testRequestingPermissionsWithInvalidParamsFails)
	t.Run("Requesting permissions with with valid params succeeds", testRequestingPermissionsWithValidParamsSucceeds)
	t.Run("Requesting permissions with invalid token fails", testRequestingPermissionsWithInvalidTokenFails)
	t.Run("Refusing permissions update does not update the permissions", testRefusingPermissionsUpdateDoesNotUpdatePermissions)
	t.Run("Cancelling the review does not update the permissions", testCancellingTheReviewDoesNotUpdatePermissions)
	t.Run("Interrupting the request does not update the permissions", testInterruptingTheRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the review does not update the permissions", testGettingInternalErrorDuringReviewDoesNotUpdatePermissions)
	t.Run("Cancelling the passphrase request does not update the permissions", testCancellingThePassphraseRequestDoesNotUpdatePermissions)
	t.Run("Interrupting the request during the passphrase request does not update the permissions", testInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the passphrase request does not update the permissions", testGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Using wrong passphrase does not update the permissions", testUsingWrongPassphraseDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet retrieval does not update the permissions", testGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet saving does not update the permissions", testGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions)
	t.Run("Updating the permissions does not overwrite untracked changes", testUpdatingPermissionsDoesNotOverwriteUntrackedChanges)
}

func testRequestingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty token",
			params: api.ClientRequestPermissionsParams{
				Token: "",
				RequestedPermissions: map[string]string{
					"public_keys": "read",
				},
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		}, {
			name: "without requested permissions",
			params: api.ClientRequestPermissionsParams{
				Token:                vgrand.RandomStr(10),
				RequestedPermissions: map[string]string{},
			},
			expectedError: api.ErrRequestedPermissionsAreRequired,
		}, {
			name: "with unsupported access mode",
			params: api.ClientRequestPermissionsParams{
				Token: vgrand.RandomStr(10),
				RequestedPermissions: map[string]string{
					"public_keys": "read",
					"everything":  "read",
				},
			},
			expectedError: errors.New("permission \"everything\" is not supported"),
		}, {
			name: "with unsupported access mode",
			params: api.ClientRequestPermissionsParams{
				Token: vgrand.RandomStr(10),
				RequestedPermissions: map[string]string{
					"public_keys": "full-access",
				},
			},
			expectedError: errors.New("access mode \"full-access\" is not supported"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()
			metadata := requestMetadataForTest()

			// setup
			handler := newRequestPermissionsHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, metadata)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRequestingPermissionsWithValidParamsSucceeds(t *testing.T) {
	tcs := []struct {
		name                string
		askedPermissions    wallet.PermissionsSummary
		expectedPermissions wallet.Permissions
	}{
		{
			name: "With read access on public keys",
			askedPermissions: wallet.PermissionsSummary{
				"public_keys": "read",
			},
			expectedPermissions: wallet.Permissions{
				PublicKeys: wallet.PublicKeysPermission{
					Access:         wallet.ReadAccess,
					RestrictedKeys: nil,
				},
			},
		}, {
			name: "With no access on public keys",
			askedPermissions: wallet.PermissionsSummary{
				"public_keys": "none",
			},
			expectedPermissions: wallet.Permissions{
				PublicKeys: wallet.PublicKeysPermission{
					Access:         wallet.NoAccess,
					RestrictedKeys: nil,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()
			metadata := requestMetadataForTest()
			wallet1, _ := walletWithPerms(tt, metadata.Hostname, wallet.Permissions{})
			passphrase := vgrand.RandomStr(5)

			// setup
			handler := newRequestPermissionsHandler(tt)
			token := connectWallet(tt, handler.sessions, metadata.Hostname, wallet1)
			// -- expected calls
			handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
			handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
			handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), tc.askedPermissions).Times(1).Return(true, nil)
			handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
			handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(wallet1, nil)
			handler.walletStore.EXPECT().SaveWallet(ctx, wallet1, passphrase).Times(1).Return(nil)
			handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, metadata.TraceID, api.PermissionsSuccessfullyUpdated).Times(1)

			// when
			result, errorDetails := handler.handle(tt, ctx, api.ClientRequestPermissionsParams{
				Token:                token,
				RequestedPermissions: tc.askedPermissions,
			}, metadata)

			// then
			assert.Nil(tt, errorDetails)
			require.NotEmpty(tt, result)
			assert.Equal(tt, tc.askedPermissions, result.Permissions)
			// Verifying the connected wallet is updated.
			connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
			require.NoError(tt, err)
			assert.Equal(tt, tc.askedPermissions, connectedWallet.Permissions().Summary())
		})
	}
}

func testRequestingPermissionsWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()

	// setup
	handler := newRequestPermissionsHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token: vgrand.RandomStr(5),
		RequestedPermissions: map[string]string{
			"public_keys": "read",
		},
	}, metadata)

	// then
	assertInvalidParams(t, errorDetails, session.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testRefusingPermissionsUpdateDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testCancellingTheReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testInterruptingTheRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the permissions review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testCancellingThePassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	originalPermissions := wallet.Permissions{}
	wallet1, _ := walletWithPerms(t, metadata.Hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the passphrase failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testUsingWrongPassphraseDoesNotUpdatePermissions(t *testing.T) {
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
	handler := newRequestPermissionsHandler(t)
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
	result, errorDetails := handler.handle(t, cancelCtx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions(t *testing.T) {
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
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.interactor.EXPECT().RequestPassphrase(ctx, metadata.TraceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions(t *testing.T) {
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
	handler := newRequestPermissionsHandler(t)
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
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is not updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testUpdatingPermissionsDoesNotOverwriteUntrackedChanges(t *testing.T) {
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
	handler := newRequestPermissionsHandler(t)
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
	result, errorDetails := handler.handle(t, ctx, api.ClientRequestPermissionsParams{
		Token:                token,
		RequestedPermissions: askedPermissions,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, askedPermissions, result.Permissions)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	require.NoError(t, err)
	assert.Equal(t, askedPermissions, connectedWallet.Permissions().Summary())
}

type requestPermissionsHandler struct {
	*api.ClientRequestPermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	interactor  *mocks.MockInteractor
	sessions    *session.Sessions
}

func (h *requestPermissionsHandler) handle(t *testing.T, ctx context.Context, params interface{}, metadata jsonrpc.RequestMetadata) (api.ClientRequestPermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, metadata)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientRequestPermissionsResult)
		if !ok {
			t.Fatal("ClientRequestPermissions handler result is not a ClientRequestPermissionsResult")
		}
		return result, err
	}
	return api.ClientRequestPermissionsResult{}, err
}

func newRequestPermissionsHandler(t *testing.T) *requestPermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)

	sessions := session.NewSessions()

	return &requestPermissionsHandler{
		ClientRequestPermissions: api.NewRequestPermissions(walletStore, interactor, sessions),
		ctrl:                     ctrl,
		walletStore:              walletStore,
		interactor:               interactor,
		sessions:                 sessions,
	}
}
