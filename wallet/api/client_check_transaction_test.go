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
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientCheckTransaction(t *testing.T) {
	t.Run("Checking a transaction with invalid params fails", testCheckingTransactionWithInvalidParamsFails)
	t.Run("Checking a transaction with valid params succeeds", testCheckingTransactionWithValidParamsSucceeds)
	t.Run("Checking a transaction without the needed permissions check the transaction", testCheckingTransactionWithoutNeededPermissionsDoesNotCheckTransaction)
	t.Run("Refusing the checking of a transaction does not check the transaction", testRefusingCheckingOfTransactionDoesNotCheckTransaction)
	t.Run("Cancelling the review does not check the transaction", testCancellingTheReviewDoesNotCheckTransaction)
	t.Run("Interrupting the request does not check the transaction", testInterruptingTheRequestDoesNotCheckTransaction)
	t.Run("Getting internal error during the review does not check the transaction", testGettingInternalErrorDuringReviewDoesNotCheckTransaction)
	t.Run("No healthy node available does not check the transaction", testNoHealthyNodeAvailableDoesNotCheckTransaction)
	t.Run("Failing to get the spam statistics does not check the transaction", testFailingToGetSpamStatsDoesNotCheckTransaction)
	t.Run("Failure when checking transaction returns an error", testFailureWhenCheckingTransactionReturnsAnError)
	t.Run("Failing spam checks aborts the transaction", testFailingSpamChecksAbortsCheckingTheTransaction)
	t.Run("Override TTL if the specified value is too high", testOverrideHighTTL)
}

func testCheckingTransactionWithInvalidParamsFails(t *testing.T) {
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
			params: api.ClientCheckTransactionParams{
				PublicKey:   "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with no transaction",
			params: api.ClientCheckTransactionParams{
				PublicKey:   vgrand.RandomStr(10),
				Transaction: nil,
			},
			expectedError: api.ErrTransactionIsRequired,
		},
		{
			name: "with transaction as invalid Vega command",
			params: api.ClientCheckTransactionParams{
				PublicKey: vgrand.RandomStr(10),
				Transaction: map[string]interface{}{
					"type": "not vega command",
				},
			},
			expectedError: errors.New("the transaction does not use a valid Vega command: unknown field \"type\" in vega.wallet.v1.SubmitTransactionRequest"),
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
			handler := newCheckTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, connectedWallet)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testCheckingTransactionWithValidParamsSucceeds(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)

	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), api.TransactionSuccessfullyChecked).Times(1)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), gomock.Any()).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}, nil)
	handler.node.EXPECT().CheckTransaction(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testCheckingTransactionWithoutNeededPermissionsDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingCheckingOfTransactionDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertUserRejectionError(t, errorDetails, api.ErrUserRejectedCheckingOfTransaction)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, api.ErrConnectionClosed)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotCheckTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
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

func testFailingToGetSpamStatsDoesNotCheckTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(types.SpamStatistics{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get the latest block information from the node: %w", assert.AnError)).Times(1)

	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
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

func testFailureWhenCheckingTransactionReturnsAnError(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(stats, nil)
	handler.node.EXPECT().Host().Times(1).Return(nodeHost)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &stats).Times(1)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), &stats).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}, nil)
	handler.node.EXPECT().CheckTransaction(ctx, gomock.Any()).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyFailedTransaction(ctx, traceID, uint8(2), gomock.Any(), gomock.Any(), assert.AnError, gomock.Any(), nodeHost).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, "the transaction failed: assert.AnError general error for testing", errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingSpamChecksAbortsCheckingTheTransaction(t *testing.T) {
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
	handler := newCheckTransactionHandler(t)

	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationError, gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
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

func testOverrideHighTTL(t *testing.T) {
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
		MaxTTL: 10,
	}

	// setup
	handler := newCheckTransactionHandler(t)
	pow := commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}

	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), api.TransactionSuccessfullyChecked).Times(1)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), gomock.Any()).Times(1).Return(&pow, nil)
	handler.node.EXPECT().CheckTransaction(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)

	// now do the exact same thing, but this time override the TTL to twice the max
	handler = newCheckTransactionHandler(t)
	// attempt twice the TTL for this Tx
	handler.overrideTTL = 2 * spamStats.MaxTTL
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	handler.walletStore.EXPECT().GetWallet(ctx, wallet1.Name()).Times(1).Return(wallet1, nil)
	handler.interactor.EXPECT().RequestTransactionReviewForChecking(ctx, traceID, uint8(1), hostname, wallet1.Name(), kp.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().SpamStatistics(ctx, kp.PublicKey()).Times(1).Return(spamStats, nil)
	handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), api.TransactionSuccessfullyChecked).Times(1)
	handler.spam.EXPECT().GenerateProofOfWork(kp.PublicKey(), gomock.Any()).Times(1).Return(&pow, nil)
	handler.node.EXPECT().CheckTransaction(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()
	// when
	result2, errorDetails2 := handler.handle(t, ctx, api.ClientCheckTransactionParams{
		PublicKey:   kp.PublicKey(),
		Transaction: testTransaction(t),
	}, connectedWallet)

	// then
	assert.Nil(t, errorDetails2)
	require.NotEmpty(t, result2)
	assert.NotEmpty(t, result2.Tx)
	// despite having TTL set to 20, because spamstats set the max at 10, this tx will have the max TTL of 10 set
	// we need to compare these two, but because the requests are sent a few nanoseconds apart, the signatures are different
	// assert.EqualValues(t, result.Tx, result2.Tx)
}

type checkTransactionHandler struct {
	*api.ClientCheckTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
	walletStore  *mocks.MockWalletStore
	spam         *mocks.MockSpamHandler
	overrideTTL  uint64
}

func (h *checkTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params, connectedWallet api.ConnectedWallet) (api.ClientCheckTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, connectedWallet, h.overrideTTL)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientCheckTransactionResult)
		if !ok {
			t.Fatal("ClientSendTransaction handler result is not a ClientCheckTransactionResult")
		}
		return result, err
	}
	return api.ClientCheckTransactionResult{}, err
}

func newCheckTransactionHandler(t *testing.T) *checkTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := nodemocks.NewMockSelector(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)
	proofOfWork := mocks.NewMockSpamHandler(ctrl)
	walletStore := mocks.NewMockWalletStore(ctrl)
	node := nodemocks.NewMockNode(ctrl)

	return &checkTransactionHandler{
		ClientCheckTransaction: api.NewClientCheckTransaction(walletStore, interactor, nodeSelector, proofOfWork),
		ctrl:                   ctrl,
		nodeSelector:           nodeSelector,
		interactor:             interactor,
		node:                   node,
		walletStore:            walletStore,
		spam:                   proofOfWork,
		overrideTTL:            0,
	}
}
