package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPermissions(t *testing.T) {
	t.Run("Requesting permissions with invalid params fails", testRequestingPermissionsWithInvalidParamsFails)
	t.Run("Requesting permissions with with valid params succeeds", testRequestingPermissionsWithValidParamsSucceeds)
	t.Run("Requesting permissions with invalid token fails", testRequestingPermissionsWithInvalidTokenFails)
	t.Run("Refusing permissions update doesn't update the permissions", testRefusingPermissionsUpdateDoesNotUpdatePermissions)
	t.Run("Cancelling the review doesn't update the permissions", testCancellingTheReviewDoesNotUpdatePermissions)
	t.Run("Interrupting the request doesn't update the permissions", testInterruptingTheRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the review doesn't update the permissions", testGettingInternalErrorDuringReviewDoesNotUpdatePermissions)
	t.Run("Cancelling the passphrase request doesn't update the permissions", testCancellingThePassphraseRequestDoesNotUpdatePermissions)
	t.Run("Interrupting the request during the passphrase request doesn't update the permissions", testInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the passphrase request doesn't update the permissions", testGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions)
	t.Run("Using wrong passphrase doesn't update the permissions", testUsingWrongPassphraseDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet retrieval doesn't update the permissions", testGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet saving doesn't update the permissions", testGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions)
	t.Run("Updating the permissions doesn't overwrite untracked changes", testUpdatingPermissionsDoesNotOverwriteUntrackedChanges)
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
			params: api.RequestPermissionsParams{
				Token: "",
				RequestedPermissions: map[string]string{
					"public_keys": "read",
				},
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		}, {
			name: "without requested permissions",
			params: api.RequestPermissionsParams{
				Token:                vgrand.RandomStr(10),
				RequestedPermissions: map[string]string{},
			},
			expectedError: api.ErrRequestedPermissionsAreRequired,
		}, {
			name: "with unsupported access mode",
			params: api.RequestPermissionsParams{
				Token: vgrand.RandomStr(10),
				RequestedPermissions: map[string]string{
					"public_keys": "read",
					"everything":  "read",
				},
			},
			expectedError: errors.New("permission \"everything\" is not supported"),
		}, {
			name: "with unsupported access mode",
			params: api.RequestPermissionsParams{
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
			ctx, _ := contextWithTraceID()

			// setup
			handler := newRequestPermissionsHandler(tt)
			// -- unexpected calls
			handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
			handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

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
			name: "With write access on public keys",
			askedPermissions: wallet.PermissionsSummary{
				"public_keys": "write",
			},
			expectedPermissions: wallet.Permissions{
				PublicKeys: wallet.PublicKeysPermission{
					Access:         wallet.WriteAccess,
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
			ctx, traceID := contextWithTraceID()
			hostname := "vega.xyz"
			wallet1 := walletWithPerms(tt, hostname, wallet.Permissions{})
			passphrase := vgrand.RandomStr(5)

			// setup
			handler := newRequestPermissionsHandler(tt)
			token := connectWallet(tt, handler.sessions, hostname, wallet1)
			// -- expected calls
			handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), tc.askedPermissions).Times(1).Return(true, nil)
			handler.pipeline.EXPECT().RequestPassphrase(ctx, traceID, wallet1.Name()).Times(1).Return(passphrase, nil)
			handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(wallet1, nil)
			handler.walletStore.EXPECT().SaveWallet(ctx, wallet1, passphrase).Times(1).Return(nil)
			handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)
			// -- unexpected calls
			handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
			handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// when
			result, errorDetails := handler.handle(tt, ctx, api.RequestPermissionsParams{
				Token:                token,
				RequestedPermissions: tc.askedPermissions,
			})

			// then
			assert.Nil(tt, errorDetails)
			require.NotEmpty(tt, result)
			assert.Equal(tt, tc.askedPermissions, result.Permissions)
			// Verifying the connected wallet is updated.
			connectedWallet, err := handler.sessions.GetConnectedWallet(token)
			require.NoError(tt, err)
			assert.Equal(tt, tc.askedPermissions, connectedWallet.Permissions().Summary())
		})
	}
}

func testRequestingPermissionsWithInvalidTokenFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()

	// setup
	handler := newRequestPermissionsHandler(t)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token: vgrand.RandomStr(5),
		RequestedPermissions: map[string]string{
			"public_keys": "read",
		},
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testRefusingPermissionsUpdateDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertClientRejectionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testCancellingTheReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrConnectionClosed)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testInterruptingTheRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(false, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the permissions review failed: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testCancellingThePassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrConnectionClosed)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testInterruptingTheRequestDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringPassphraseRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(ctx, traceID, wallet1.Name()).Times(1).Return("", assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the passphrase failed: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testUsingWrongPassphraseDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(cancelCtx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(cancelCtx, traceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(cancelCtx, wallet1.Name(), passphrase).Times(1).Return(nil, wallet.ErrWrongPassphrase)
	handler.pipeline.EXPECT().NotifyError(cancelCtx, traceID, api.ClientError, wallet.ErrWrongPassphrase).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, cancelCtx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	originalPermissions := wallet.Permissions{}
	wallet1 := walletWithPerms(t, hostname, originalPermissions)
	passphrase := vgrand.RandomStr(5)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(ctx, traceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("couldn't retrieve the wallet: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testGettingInternalErrorDuringWalletSavingDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	walletName := vgrand.RandomStr(5)
	originalPermissions := wallet.Permissions{}
	wallet1, recoveryPhrase, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatal("couldn't create wallet for test: %w", err)
	}

	// Clone the wallet1, so we can emulate a different instance returned by
	// the wallet store.
	loadedWallet, err := wallet.ImportHDWallet(walletName, recoveryPhrase, 2)
	if err != nil {
		t.Fatal("couldn't import wallet for test: %w", err)
	}
	passphrase := vgrand.RandomStr(5)
	requestedPermissions := map[string]string{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), requestedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(ctx, traceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(loadedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, loadedWallet, passphrase).Times(1).Return(assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("couldn't save wallet: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: requestedPermissions,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotRequestPermissions)
	assert.Empty(t, result)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, originalPermissions.Summary(), connectedWallet.Permissions().Summary())
}

func testUpdatingPermissionsDoesNotOverwriteUntrackedChanges(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	walletName := vgrand.RandomStr(5)
	wallet1, recoveryPhrase, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatal("couldn't create wallet for test: %w", err)
	}

	// Clone the wallet1, so we can modify the clone without tempering with wallet1.
	modifiedWallet, err := wallet.ImportHDWallet(walletName, recoveryPhrase, 2)
	if err != nil {
		t.Fatal("couldn't import wallet for test: %w", err)
	}
	kp, _ := modifiedWallet.GenerateKeyPair([]wallet.Meta{{Key: "name", Value: "hello"}})

	passphrase := vgrand.RandomStr(5)
	askedPermissions := wallet.PermissionsSummary{
		"public_keys": "read",
	}

	// setup
	handler := newRequestPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestPermissionsReview(ctx, traceID, hostname, wallet1.Name(), askedPermissions).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestPassphrase(ctx, traceID, wallet1.Name()).Times(1).Return(passphrase, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(modifiedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).DoAndReturn(func(_ context.Context, w wallet.Wallet, _ string) error {
		// We verify that the saved wallet contains the modification from the modified wallet and the permissions update from wallet1.
		assert.Equal(t, []wallet.KeyPair{kp}, w.ListKeyPairs())
		assert.Equal(t, wallet.Permissions{
			PublicKeys: wallet.PublicKeysPermission{
				Access: wallet.ReadAccess,
			},
		}, w.Permissions(hostname))
		return nil
	})
	handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.RequestPermissionsParams{
		Token:                token,
		RequestedPermissions: askedPermissions,
	})

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, askedPermissions, result.Permissions)
	// Verifying the connected wallet is updated.
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	require.NoError(t, err)
	assert.Equal(t, askedPermissions, connectedWallet.Permissions().Summary())
}

type requestPermissionsHandler struct {
	*api.RequestPermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	pipeline    *mocks.MockPipeline
	sessions    *api.Sessions
}

func (h *requestPermissionsHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.RequestPermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.RequestPermissionsResult)
		if !ok {
			t.Fatal("RequestPermissions handler result is not a RequestPermissionsResult")
		}
		return result, err
	}
	return api.RequestPermissionsResult{}, err
}

func newRequestPermissionsHandler(t *testing.T) *requestPermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	pipeline := mocks.NewMockPipeline(ctrl)

	sessions := api.NewSessions()

	return &requestPermissionsHandler{
		RequestPermissions: api.NewRequestPermissions(walletStore, pipeline, sessions),
		ctrl:               ctrl,
		walletStore:        walletStore,
		pipeline:           pipeline,
		sessions:           sessions,
	}
}
