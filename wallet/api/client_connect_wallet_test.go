package api_test

import (
	"context"
	"fmt"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/preferences"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectWallet(t *testing.T) {
	t.Run("Connecting to a wallet with valid params succeeds", testConnectingToWalletWithValidParamsSucceeds)
	t.Run("Connecting to a connected wallet disconnects the previous one and generates a new token", testConnectingToConnectedWalletDisconnectsPreviousOneAndGeneratesNewToken)
	t.Run("Connecting to a wallet without wallets fails", testConnectingWalletWithoutWalletsFails)
	t.Run("Getting internal error during the wallet listing does not connect to a wallet", testGettingInternalErrorDuringWalletListingDoesNotConnectToWallet)
	t.Run("Refusing a wallet connection does not connect to a wallet", testRefusingWalletConnectionDoesNotConnectToWallet)
	t.Run("Canceling the review does not connect to a wallet", testCancelingTheReviewDoesNotConnectToWallet)
	t.Run("Interrupting the request during the review does not connect to a wallet", testInterruptingTheRequestDuringReviewDoesNotConnectToWallet)
	t.Run("Getting internal error during the review does not connect to a wallet", testGettingInternalErrorDuringReviewDoesNotConnectToWallet)
	t.Run("Cancelling the wallet selection does not connect to a wallet", testCancellingTheWalletSelectionDoesNotConnectToWallet)
	t.Run("Interrupting the request during the wallet selection does not connect to a wallet", testInterruptingTheRequestDuringWalletSelectionDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet selection does not connect to a wallet", testGettingInternalErrorDuringWalletSelectionDoesNotConnectToWallet)
	t.Run("Selecting a non-existing wallet does not connect to a wallet", testSelectingNonExistingWalletDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet verification does not connect to a wallet", testGettingInternalErrorDuringWalletVerificationDoesNotConnectToWallet)
	t.Run("Using the wrong passphrase does not connect to a wallet", testUsingWrongPassphraseDoesNotConnectToWallet)
	t.Run("Getting internal error during the wallet retrieval does not connect to a wallet", testGettingInternalErrorDuringWalletRetrievalDoesNotConnectToWallet)
}

func testConnectingToWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: []string{},
		},
	}
	expectedSelectedWallet := walletWithPerms(t, hostname, expectedPermissions)
	nonSelectedWallet := walletWithPerms(t, hostname, wallet.Permissions{})

	passphrase := vgrand.RandomStr(5)
	availableWallets := []string{
		expectedSelectedWallet.Name(),
		nonSelectedWallet.Name(),
	}

	// setup
	// -- expected calls
	handler := newConnectWalletHandler(t)
	handler.walletStore.EXPECT().WalletExists(ctx, expectedSelectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(availableWallets, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name()).Times(1).Return(expectedSelectedWallet, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, api.WalletConnectionSuccessfullyEstablished).Times(1)

	// when
	token, errorDetails := handler.Handle(ctx, hostname)

	// then
	require.Nil(t, errorDetails)
	assert.NotEmpty(t, token)
}

func testConnectingToConnectedWalletDisconnectsPreviousOneAndGeneratesNewToken(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: []string{},
		},
	}
	expectedSelectedWallet := walletWithPerms(t, hostname, expectedPermissions)
	nonSelectedWallet := walletWithPerms(t, hostname, wallet.Permissions{})

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
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name()).Times(1).Return(expectedSelectedWallet, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, api.WalletConnectionSuccessfullyEstablished).Times(1)

	// when
	wallet1, errorDetails := handler.Handle(ctx, hostname)

	// then
	assert.Nil(t, errorDetails)
	assert.NotEmpty(t, wallet1)

	// setup
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedSelectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(availableWallets, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedSelectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedSelectedWallet.Name()).Times(1).Return(expectedSelectedWallet, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, availableWallets).Times(1).Return(api.SelectedWallet{
		Wallet:     expectedSelectedWallet.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, api.WalletConnectionSuccessfullyEstablished).Times(1)

	// when
	wallet2, errorDetails := handler.Handle(ctx, hostname)

	// then
	assert.Nil(t, errorDetails)
	assert.Equal(t, wallet2, wallet1)
}

func testConnectingWalletWithoutWalletsFails(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, api.ErrNoWalletToConnectTo).Times(1)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{}, nil)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertApplicationCancellationError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletListingDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not list the available wallets: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testRefusingWalletConnectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.RejectedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name()}, nil)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertUserRejectionError(t, errorDetails, api.ErrUserRejectedWalletConnection)
	assert.Empty(t, result)
}

func testCancelingTheReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return("", api.ErrUserCloseTheConnection)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name()}, nil)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDuringReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return("", api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name()}, nil)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("reviewing the wallet connection failed: %w", assert.AnError)).Times(1)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name()}, nil)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testCancellingTheWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDuringWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletSelectionDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the wallet selection failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testSelectingNonExistingWalletDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})
	nonExistingWallet := vgrand.RandomStr(5)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(cancelCtx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(cancelCtx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(cancelCtx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(cancelCtx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(cancelCtx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     nonExistingWallet,
		Passphrase: vgrand.RandomStr(4),
	}, nil)
	handler.walletStore.EXPECT().WalletExists(cancelCtx, nonExistingWallet).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyError(cancelCtx, traceID, api.UserError, api.ErrWalletDoesNotExist).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})

	// when
	result, errorDetails := handler.Handle(cancelCtx, hostname)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: vgrand.RandomStr(5),
	}, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, wallet1.Name()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

func testUsingWrongPassphraseDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	cancelCtx, cancelFn := context.WithCancel(ctx)
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})
	passphrase := vgrand.RandomStr(4)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(cancelCtx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(cancelCtx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(cancelCtx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(cancelCtx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(cancelCtx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.walletStore.EXPECT().WalletExists(cancelCtx, wallet1.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(cancelCtx, wallet1.Name(), passphrase).Times(1).Return(wallet.ErrWrongPassphrase)
	handler.interactor.EXPECT().NotifyError(cancelCtx, traceID, api.UserError, wallet.ErrWrongPassphrase).Times(1).Do(func(_ context.Context, _ string, _ api.ErrorType, _ error) {
		// Once everything has been called once, we cancel the handler to break the loop.
		cancelFn()
	})

	// when
	result, errorDetails := handler.Handle(cancelCtx, hostname)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotConnectToWallet(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5) + ".xyz"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	wallet2 := walletWithPerms(t, hostname, wallet.Permissions{})
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newConnectWalletHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestWalletConnectionReview(ctx, traceID, hostname).Times(1).Return(string(preferences.ApprovedOnlyThisTime), nil)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return([]string{wallet1.Name(), wallet2.Name()}, nil)
	handler.interactor.EXPECT().RequestWalletSelection(ctx, traceID, hostname, []string{wallet1.Name(), wallet2.Name()}).Times(1).Return(api.SelectedWallet{
		Wallet:     wallet1.Name(),
		Passphrase: passphrase,
	}, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, wallet1.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, wallet1.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.Handle(ctx, hostname)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotConnectToWallet)
	assert.Empty(t, result)
}

type connectWalletHandler struct {
	*api.ClientConnectWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	interactor  *mocks.MockInteractor
}

func newConnectWalletHandler(t *testing.T) *connectWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)

	return &connectWalletHandler{
		ClientConnectWallet: api.NewConnectWallet(walletStore, interactor),
		ctrl:                ctrl,
		walletStore:         walletStore,
		interactor:          interactor,
	}
}
