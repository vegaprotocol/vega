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

func TestClientListKeys(t *testing.T) {
	t.Run("Documentation matches the code", testClientListKeysSchemaCorrect)
	t.Run("Listing keys with enough permissions succeeds", testListingKeysWithEnoughPermissionsSucceeds)
	t.Run("Listing keys without enough permissions succeeds", testListingKeysWithoutEnoughPermissionsSucceeds)
	t.Run("Getting internal error during wallet retrieval does not update the permissions", testListingKeysGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions)
	t.Run("Retrieving a locked wallet does not update the permissions", testListingKeysRetrievingLockedWalletDoesNotUpdatePermissions)
	t.Run("Refusing permissions update does not update the permissions", testListingKeysRefusingPermissionsUpdateDoesNotUpdatePermissions)
	t.Run("Cancelling the permissions review does not update the permissions", testListingKeysCancellingTheReviewDoesNotUpdatePermissions)
	t.Run("Interrupting the request does not update the permissions", testListingKeysInterruptingTheRequestDoesNotUpdatePermissions)
	t.Run("Getting internal error during the review does not update the permissions", testListingKeysGettingInternalErrorDuringReviewDoesNotUpdatePermissions)
	t.Run("Getting internal error during the wallet update does not update the permissions", testListingKeysGettingInternalErrorDuringWalletUpdateDoesNotUpdatePermissions)
}

func testClientListKeysSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "client.list_keys", nil, api.ClientListKeysResult{})
}

func testListingKeysWithEnoughPermissionsSucceeds(t *testing.T) {
	// given
	ctx, _ := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, kps := walletWithKeys(t, 2)
	if err := w.UpdatePermissions(hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	}); err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, []api.ClientNamedPublicKey{
		{
			Name:      kps[0].Name(),
			PublicKey: kps[0].PublicKey(),
		}, {
			Name:      kps[1].Name(),
			PublicKey: kps[1].PublicKey(),
		},
	}, result.Keys)
}

func testListingKeysWithoutEnoughPermissionsSucceeds(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, kps := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, connectedWallet.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, w).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), api.PermissionsSuccessfullyUpdated)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, []api.ClientNamedPublicKey{
		{
			Name:      kps[0].Name(),
			PublicKey: kps[0].PublicKey(),
		}, {
			Name:      kps[1].Name(),
			PublicKey: kps[1].PublicKey(),
		},
	}, result.Keys)
}

func testListingKeysGettingInternalErrorDuringWalletRetrievalDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, connectedWallet.Name()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("could not retrieve the wallet for the permissions update: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotListKeys)
	assert.Empty(t, result)
}

func testListingKeysRetrievingLockedWalletDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, connectedWallet.Name()).Times(1).Return(nil, api.ErrWalletIsLocked)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, api.ErrWalletIsLocked).Times(1)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotListKeys)
	assert.Empty(t, result)
}

func testListingKeysRefusingPermissionsUpdateDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertUserRejectionError(t, errorDetails, api.ErrUserRejectedAccessToKeys)
	assert.Empty(t, result)
}

func testListingKeysCancellingTheReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(false, api.ErrUserCloseTheConnection)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, api.ErrConnectionClosed)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testListingKeysInterruptingTheRequestDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerErrorType, api.ErrRequestInterrupted).Times(1)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testListingKeysGettingInternalErrorDuringReviewDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("requesting the permissions review failed: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotListKeys)
	assert.Empty(t, result)
}

func testListingKeysGettingInternalErrorDuringWalletUpdateDoesNotUpdatePermissions(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	w, _ := walletWithKeys(t, 2)
	connectedWallet, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newListKeysHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.PermissionRequestWorkflow, uint8(2)).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.interactor.EXPECT().RequestPermissionsReview(ctx, traceID, uint8(1), hostname, w.Name(), map[string]string{
		"public_keys": "read",
	}).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, w).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("could not save the permissions update on the wallet: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotListKeys)
	assert.Empty(t, result)
}

type listKeysHandler struct {
	*api.ClientListKeys
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	interactor  *mocks.MockInteractor
}

func (h *listKeysHandler) handle(t *testing.T, ctx context.Context, connectedWallet api.ConnectedWallet) (api.ClientListKeysResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, connectedWallet)
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

	return &listKeysHandler{
		ClientListKeys: api.NewListKeys(walletStore, interactor),
		ctrl:           ctrl,
		walletStore:    walletStore,
		interactor:     interactor,
	}
}
