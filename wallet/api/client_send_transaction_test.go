package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendTransaction(t *testing.T) {
	t.Run("Sending a transaction with invalid params fails", testSendingTransactionWithInvalidParamsFails)
	t.Run("Sending a transaction with valid params succeeds", testSendingTransactionWithValidParamsSucceeds)
	t.Run("Sending a transaction without the needed permissions send the transaction", testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction)
	t.Run("Refusing the sending of a transaction does not send the transaction", testRefusingSendingOfTransactionDoesNotSendTransaction)
	t.Run("Cancelling the review does not send the transaction", testCancellingTheReviewDoesNotSendTransaction)
	t.Run("Interrupting the request does not send the transaction", testInterruptingTheRequestDoesNotSendTransaction)
	t.Run("Getting internal error during the review does not send the transaction", testGettingInternalErrorDuringReviewDoesNotSendTransaction)
	t.Run("No healthy node available does not send the transaction", testNoHealthyNodeAvailableDoesNotSendTransaction)
	t.Run("Failing to get the spam statistics does not send the transaction", testFailingToGetSpamStatsDoesNotSendTransaction)
	t.Run("Failure when sending transaction returns an error", testFailureWhenSendingTransactionReturnsAnError)
	t.Run("Failing spam checks aborts the transaction", testFailingSpamChecksAbortsTheTransaction)
}

func testSendingTransactionWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		},
		{
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		},
		{
			name: "with empty public key permissions",
			params: api.ClientSendTransactionParams{
				PublicKey:   "",
				SendingMode: "TYPE_SYNC",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty sending mode",
			params: api.ClientSendTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrSendingModeIsRequired,
		},
		{
			name: "with unsupported sending mode",
			params: api.ClientSendTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_UNSPECIFIED",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrSendingModeCannotBeTypeUnspecified,
		},
		{
			name: "with unsupported sending mode",
			params: api.ClientSendTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_MANY_FAST",
				Transaction: testTransaction(t),
			},
			expectedError: errors.New(`the sending mode "TYPE_MANY_FAST" is not a valid one`),
		},
		{
			name: "with no transaction",
			params: api.ClientSendTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_SYNC",
				Transaction: nil,
			},
			expectedError: api.ErrTransactionIsRequired,
		},
		{
			name: "with transaction as invalid Vega command",
			params: api.ClientSendTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_SYNC",
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
			wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
				PublicKeys: wallet.PublicKeysPermission{
					Access:      wallet.ReadAccess,
					AllowedKeys: nil,
				},
			})
			connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
			if err != nil {
				t.Fatalf(err.Error())
			}

			// setup
			handler := newSendTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, connectedWallet)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	nodeHost := vgrand.RandomStr(5)
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
	txHash := vgrand.RandomStr(64)
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
	handler := newSendTransactionHandler(t)

	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.node.EXPECT().Host().Times(1).Return(nodeHost)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), gomock.Any()).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.interactor.EXPECT().NotifySuccessfulTransaction(ctx, traceID, txHash, gomock.Any(), gomock.Any(), gomock.Any(), nodeHost).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSendingOfTransactionDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertUserRejectionError(t, errorDetails, api.ErrUserRejectedSendingOfTransaction)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSendTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetSpamStatsDoesNotSendTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(types.SpamStatistics{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get the latest block from node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailureWhenSendingTransactionReturnsAnError(t *testing.T) {
	// given
	ctx, traceID := clientContextForTest()
	hostname := vgrand.RandomStr(5)
	nodeHost := vgrand.RandomStr(5)
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
	stats := types.SpamStatistics{
		ChainID:         vgrand.RandomStr(5),
		LastBlockHeight: 100,
	}

	// setup
	handler := newSendTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(stats, nil)
	handler.node.EXPECT().Host().Times(1).Return(nodeHost)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &stats).Times(1)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), &stats).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyFailedTransaction(ctx, traceID, gomock.Any(), gomock.Any(), assert.AnError, gomock.Any(), nodeHost).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, "the transaction failed: assert.AnError general error for testing", errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingSpamChecksAbortsTheTransaction(t *testing.T) {
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
	handler := newSendTransactionHandler(t)

	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		PublicKey:   kp.PublicKey(),
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestHasBeenCanceledByApplication, errorDetails.Code)
	assert.Equal(t, "Application error", errorDetails.Message)
	assert.Equal(t, api.ErrTransactionBlockedBySpamRules.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type sendTransactionHandler struct {
	*api.ClientSendTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
	walletStore  *mocks.MockWalletStore
	spam         *mocks.MockSpamHandler
}

func (h *sendTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params, connectedWallet api.ConnectedWallet) (api.ClientSendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, connectedWallet)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientSendTransactionResult)
		if !ok {
			t.Fatal("ClientSendTransaction handler result is not a ClientSendTransactionResult")
		}
		return result, err
	}
	return api.ClientSendTransactionResult{}, err
}

func newSendTransactionHandler(t *testing.T) *sendTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := nodemocks.NewMockSelector(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)
	proofOfWork := mocks.NewMockSpamHandler(ctrl)
	walletStore := mocks.NewMockWalletStore(ctrl)
	node := nodemocks.NewMockNode(ctrl)

	return &sendTransactionHandler{
		ClientSendTransaction: api.NewClientSendTransaction(walletStore, interactor, nodeSelector, proofOfWork),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interactor,
		node:                  node,
		walletStore:           walletStore,
		spam:                  proofOfWork,
	}
}
