package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	nodemock "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignTransaction(t *testing.T) {
	t.Run("Signing a transaction with invalid params fails", testSigningTransactionWithInvalidParamsFails)
	t.Run("Signing a transaction with valid params succeeds", testSigningTransactionWithValidParamsSucceeds)
	t.Run("Signing a transaction without the needed permissions sign the transaction", testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction)
	t.Run("Refusing the signing of a transaction does not sign the transaction", testRefusingSigningOfTransactionDoesNotSignTransaction)
	t.Run("Cancelling the review does not sign the transaction", testCancellingTheReviewDoesNotSignTransaction)
	t.Run("Interrupting the request does not sign the transaction", testInterruptingTheRequestDoesNotSignTransaction)
	t.Run("Getting internal error during the review does not sign the transaction", testGettingInternalErrorDuringReviewDoesNotSignTransaction)
	t.Run("No healthy node available does not sign the transaction", testNoHealthyNodeAvailableDoesNotSignTransaction)
	t.Run("Failing to get spam statistics does not sign the transaction", testFailingToGetSpamStatsDoesNotSignTransaction)
	t.Run("Failing spam check aborts signing the transaction", testFailingSpamChecksAbortsSigningTheTransaction)
}

func testSigningTransactionWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty public key permissions",
			params: api.ClientSignTransactionParams{
				PublicKey:   "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		}, {
			name: "with no transaction",
			params: api.ClientSignTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				Transaction: nil,
			},
			expectedError: api.ErrTransactionIsRequired,
		}, {
			name: "with transaction as invalid Vega command",
			params: api.ClientSignTransactionParams{
				PublicKey: vgrand.RandomStr(10),
				Transaction: map[string]interface{}{
					"type": "not vega command",
				},
			},
			expectedError: errors.New("the transaction is not a valid Vega command: unknown field \"type\" in vega.wallet.v1.SubmitTransactionRequest"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := clientContextForTest()
			hostname := vgrand.RandomStr(5)
			w, _ := walletWithKeys(t, 2)
			connectedWallet, err := api.NewConnectedWallet(hostname, w)
			if err != nil {
				t.Fatalf(err.Error())
			}

			// setup
			handler := newSignTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, connectedWallet)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSigningTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	spamStats := types.SpamStatistics{
		ChainID:           vgrand.RandomStr(5),
		LastBlockHeight:   100,
		Proposals:         &types.SpamStatistic{MaxForEpoch: 1},
		NodeAnnouncements: &types.SpamStatistic{MaxForEpoch: 1},
		Delegations:       &types.SpamStatistic{MaxForEpoch: 1},
		Transfers:         &types.SpamStatistic{MaxForEpoch: 1},
		Votes:             &types.VoteSpamStatistics{MaxForEpoch: 1},
		PoW: &types.PoWStatistics{
			PowBlockStates: []types.PoWBlockState{{}},
		},
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), &spamStats).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)

	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), api.TransactionSuccessfullySigned).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, _ := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSigningOfTransactionDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertUserRejectionError(t, errorDetails, api.ErrUserRejectedSigningOfTransaction)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, api.ErrConnectionClosed)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSignTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetSpamStatsDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(types.SpamStatistics{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get the latest block from the node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingSpamChecksAbortsSigningTheTransaction(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp, err := wallet1.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	spamStats := types.SpamStatistics{
		ChainID:           vgrand.RandomStr(5),
		LastBlockHeight:   100,
		Proposals:         &types.SpamStatistic{MaxForEpoch: 1},
		NodeAnnouncements: &types.SpamStatistic{MaxForEpoch: 1},
		Delegations:       &types.SpamStatistic{MaxForEpoch: 1},
		Transfers:         &types.SpamStatistic{MaxForEpoch: 1},
		Votes:             &types.VoteSpamStatistics{MaxForEpoch: 1},
		PoW: &types.PoWStatistics{
			PowBlockStates: []types.PoWBlockState{{}},
		},
	}

	// setup
	handler := newSignTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestHasBeenCancelledByApplication, errorDetails.Code)
	assert.Equal(t, "Application error", errorDetails.Message)
	assert.Equal(t, assert.AnError.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type signTransactionHandler struct {
	*api.ClientSignTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	nodeSelector *nodemock.MockSelector
	node         *nodemock.MockNode
	walletStore  *mocks.MockWalletStore
	spam         *mocks.MockSpamHandler
}

func (h *signTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params, connectedWallet api.ConnectedWallet) (api.ClientSignTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, connectedWallet)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientSignTransactionResult)
		if !ok {
			t.Fatal("ClientSignTransaction handler result is not a ClientSignTransactionResult")
		}
		return result, err
	}
	return api.ClientSignTransactionResult{}, err
}

func newSignTransactionHandler(t *testing.T) *signTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)
	nodeSelector := nodemock.NewMockSelector(ctrl)
	node := nodemock.NewMockNode(ctrl)
	proofOfWork := mocks.NewMockSpamHandler(ctrl)

	return &signTransactionHandler{
		ClientSignTransaction: api.NewClientSignTransaction(walletStore, interactor, nodeSelector, proofOfWork),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interactor,
		node:                  node,
		walletStore:           walletStore,
		spam:                  proofOfWork,
	}
}
