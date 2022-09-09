package api_test

import (
	"context"
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

func TestConnectWallet(t *testing.T) {
	t.Run("Connecting to a wallet with invalid params fails", testConnectingToWalletWithInvalidParamsFails)
	t.Run("Connecting to a wallet with valid params succeeds", testConnectingToWalletWithValidParamsSucceeds)
	t.Run("Connecting to a connected wallet disconnects the previous one and generates a new token", testConnectingToConnectedWalletDisconnectsPreviousOneAndGeneratesNewToken)
	t.Run("Refusing a wallet connection does not connect to a wallet", testRefusingWalletConnectionDoesNotConnectToWallet)
	t.Run("Canceling the review does not connect to a wallet", testCancelingTheReviewDoesNotConnectToWallet)
	t.Run("Interrupting the request during the review does not connect to a wallet", testInterruptingTheRequestDuringReviewDoesNotConnectToWallet)
	t.Run("Getting internal error during the review does not connect to a wallet", testGettingInternalErrorDuringReviewDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet listing does not connect to a wallet", testGettingInternalErrorDuringWalletListingDoesNotConnectToWallet)
	t.Run("Cancelling the wallet selection does not connect to a wallet", testCancellingTheWalletSelectionDoesNotConnectToWallet)
	t.Run("Interrupting the request during the wallet selection does not connect to a wallet", testInterruptingTheRequestDuringWalletSelectionDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet selection does not connect to a wallet", testGettingInternalErrorDuringWalletSelectionDoesNotConnectToWallet)
	t.Run("Selecting a non-existing wallet does not connect to a wallet", testSelectingNonExistingWalletDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet verification does not connect to a wallet", testGettingInternalErrorDuringWalletVerificationDoesNotConnectToWallet)
	t.Run("Using the wrong passphrase does not connect to a wallet", testUsingWrongPassphraseDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet retrieval does not connect to a wallet", testGettingInternalErrorDuringWalletRetrievalDoesNotConnectToWallet)
}

func testConnectingToWalletWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty hostname",
			params: api.ConnectWalletParams{
				Hostname: "",
			},
			expectedError: api.ErrHostnameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newConnectWalletHandler(tt)
			// -- unexpected calls
			handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
			handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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

func testConnectingToWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	}
	expectedHostname := "vega.xyz"
	expectedSelectedWallet := walletWithPerms(t, expectedHostname, expectedPermissions)
	nonSelectedWallet := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	passphrase := vgrand.RandomStr(5)
	availableWallets := []string{
		expectedSelectedWallet.Name(),
		nonSelectedWallet.Name(),
	}

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedSelectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(availableWallets, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(expectedSelectedWallet, nil)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotEmpty(t, result.Token)
}

func testConnectingToConnectedWalletDisconnectsPreviousOneAndGeneratesNewToken(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	}
	expectedHostname := "vega.xyz"
	expectedSelectedWallet := walletWithPerms(t, expectedHostname, expectedPermissions)
	nonSelectedWallet := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	passphrase := vgrand.RandomStr(5)
	availableWallets := []string{
		expectedSelectedWallet.Name(),
		nonSelectedWallet.Name(),
	}

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedSelectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(availableWallets, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(expectedSelectedWallet, nil)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result1, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assert.Nil(t, errorDetails)
	assert.NotEmpty(t, result1.Token)

	// setup
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedSelectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(availableWallets, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(expectedSelectedWallet, nil)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result2, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assert.Nil(t, errorDetails)
	assert.NotEqual(t, result2.Token, result1.Token)
}

func testRefusingWalletConnectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(false, nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertClientRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancelingTheReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(false, api.ErrConnectionClosed)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDuringReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(false, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("reviewing the wallet connection failed: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletListingDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not list available wallets: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testCancellingTheWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, api.ErrConnectionClosed)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDuringWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the wallet selection failed: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testSelectingNonExistingWalletDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	nonExistingWallet := vgrand.RandomStr(5)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(cancelCtx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(cancelCtx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(cancelCtx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     nonExistingWallet,
		Passphrase: vgrand.RandomStr(4),
	}, nil)
	handler.walletStore.EXPECT().WalletExists(cancelCtx, nonExistingWallet).Times(1).Return(false, nil)
	handler.pipeline.EXPECT().NotifyError(cancelCtx, traceID, api.ClientError, api.ErrWalletDoesNotExist).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})
	// -- unexpected calls
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, cancelCtx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: vgrand.RandomStr(5),
	}, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, wallet1.Name()).Times(1).Return(false, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testUsingWrongPassphraseDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	passphrase := vgrand.RandomStr(4)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(cancelCtx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(cancelCtx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(cancelCtx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.walletStore.EXPECT().WalletExists(cancelCtx, wallet1.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(cancelCtx, wallet1.Name(), passphrase).Times(1).Return(nil, wallet.ErrWrongPassphrase)
	handler.pipeline.EXPECT().NotifyError(cancelCtx, traceID, api.ClientError, wallet.ErrWrongPassphrase).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, cancelCtx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	expectedHostname := "vega.xyz"
	wallet1 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, expectedHostname, wallet.Permissions{})
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.pipeline.EXPECT().RequestWalletConnectionReview(ctx, traceID, expectedHostname).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.pipeline.EXPECT().RequestWalletSelection(ctx, traceID, expectedHostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, wallet1.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestTransactionSendingReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ConnectWalletParams{
		Hostname: expectedHostname,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

type connectWalletHandler struct {
	*api.ConnectWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	pipeline    *mocks.MockPipeline
}

func (h *connectWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.ConnectWalletResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.ConnectWalletResult)
		if !ok {
			t.Fatal("ConnectWallet handler result is not a ConnectWalletResult")
		}
		return result, err
	}
	return api.ConnectWalletResult{}, err
}

func newConnectWalletHandler(t *testing.T) *connectWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	pipeline := mocks.NewMockPipeline(ctrl)

	sessions := api.NewSessions()

	return &connectWalletHandler{
		ConnectWallet: api.NewConnectWallet(walletStore, pipeline, sessions),
		ctrl:          ctrl,
		walletStore:   walletStore,
		pipeline:      pipeline,
	}
}
