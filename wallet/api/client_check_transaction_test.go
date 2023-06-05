package api_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

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
	t.Run("Documentation matches the code", testClientCheckTransactionSchemaCorrect)
	t.Run("Checking a transaction with invalid params fails", testCheckingTransactionWithInvalidParamsFails)
	t.Run("Checking a transaction with valid params succeeds", testCheckingTransactionWithValidParamsSucceeds)
	t.Run("Checking a transaction in parallel blocks on same party but not on different parties", testCheckingTransactionInParallelBlocksOnSamePartyButNotOnDifferentParties)
	t.Run("Checking a transaction without the needed permissions check the transaction", testCheckingTransactionWithoutNeededPermissionsDoesNotCheckTransaction)
	t.Run("Refusing the checking of a transaction does not check the transaction", testRefusingCheckingOfTransactionDoesNotCheckTransaction)
	t.Run("Cancelling the review does not check the transaction", testCancellingTheReviewDoesNotCheckTransaction)
	t.Run("Interrupting the request does not check the transaction", testInterruptingTheRequestDoesNotCheckTransaction)
	t.Run("Getting internal error during the review does not check the transaction", testGettingInternalErrorDuringReviewDoesNotCheckTransaction)
	t.Run("No healthy node available does not check the transaction", testNoHealthyNodeAvailableDoesNotCheckTransaction)
	t.Run("Failing to get the spam statistics does not check the transaction", testFailingToGetSpamStatsDoesNotCheckTransaction)
	t.Run("Failure when checking transaction returns an error", testFailureWhenCheckingTransactionReturnsAnError)
	t.Run("Failing spam checks aborts the transaction", testFailingSpamChecksAbortsCheckingTheTransaction)
}

func testClientCheckTransactionSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "client.check_transaction", api.ClientCheckTransactionParams{}, api.ClientCheckTransactionResult{})
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
	assert.NotEmpty(t, result.Transaction)
}

func testCheckingTransactionInParallelBlocksOnSamePartyButNotOnDifferentParties(t *testing.T) {
	// setup

	// Use channels to orchestrate requests.
	sendSecondRequests := make(chan interface{})
	sendThirdRequests := make(chan interface{})
	waitForSecondRequestToExit := make(chan interface{})
	waitForThirdRequestToExit := make(chan interface{})

	hostname := vgrand.RandomStr(5)

	// One context for each request.
	r1Ctx, r1TraceID := clientContextForTest()
	r2Ctx, _ := clientContextForTest()
	r3Ctx, r3TraceID := clientContextForTest()

	// A wallet with 2 keys to have 2 different parties.
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	})
	kp1, err := wallet1.GenerateKeyPair(nil)
	require.NoError(t, err)
	kp2, err := wallet1.GenerateKeyPair(nil)
	require.NoError(t, err)

	// We can have a single connection as the implementation only cares about the
	// party.
	connectedWallet, err := api.NewConnectedWallet(hostname, wallet1)
	require.NoError(t, err)

	// Some mock data. Their value is irrelevant to test parallelism, so we recycle
	// them.
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
	pow := &commandspb.ProofOfWork{
		Tid:   vgrand.RandomStr(5),
		Nonce: 12345678,
	}

	// Setting up the mocked calls. The second request shouldn't trigger any of
	// them, since it should be rejected because it uses the same party as the
	// first request, which only unblock at the end.
	handler := newCheckTransactionHandler(t)

	gomock.InOrder(
		// First request.
		handler.spam.EXPECT().GenerateProofOfWork(kp1.PublicKey(), &spamStats).Times(1).Return(pow, nil),
		// Third request.
		handler.spam.EXPECT().GenerateProofOfWork(kp2.PublicKey(), &spamStats).Times(1).Return(pow, nil),
	)
	gomock.InOrder(
		// First request.
		handler.interactor.EXPECT().NotifyInteractionSessionBegan(r1Ctx, r1TraceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil),
		// Third request.
		handler.interactor.EXPECT().NotifyInteractionSessionBegan(r3Ctx, r3TraceID, api.TransactionReviewWorkflow, uint8(2)).Times(1).Return(nil),
	)
	gomock.InOrder(
		// Third request is expected before because the first request get unblocked
		// when the third request finishes.
		handler.interactor.EXPECT().NotifyInteractionSessionEnded(r3Ctx, r3TraceID).Times(1),
		// First request.
		handler.interactor.EXPECT().NotifyInteractionSessionEnded(r1Ctx, r1TraceID).Times(1),
	)
	gomock.InOrder(
		// First request.
		handler.interactor.EXPECT().RequestTransactionReviewForChecking(r1Ctx, r1TraceID, uint8(1), hostname, wallet1.Name(), kp1.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil),
		// Third request.
		handler.interactor.EXPECT().RequestTransactionReviewForChecking(r3Ctx, r3TraceID, uint8(1), hostname, wallet1.Name(), kp2.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil),
	)
	gomock.InOrder(
		// First request.
		handler.nodeSelector.EXPECT().Node(r1Ctx, gomock.Any()).Times(1).Return(handler.node, nil),
		// Third request.
		handler.nodeSelector.EXPECT().Node(r3Ctx, gomock.Any()).Times(1).Return(handler.node, nil),
	)
	gomock.InOrder(
		// First request.
		handler.walletStore.EXPECT().GetWallet(r1Ctx, wallet1.Name()).Times(1).Return(wallet1, nil),
		// Second request.
		handler.walletStore.EXPECT().GetWallet(r2Ctx, wallet1.Name()).Times(1).DoAndReturn(func(_ context.Context, _ string) (wallet.Wallet, error) {
			close(sendThirdRequests)
			return wallet1, nil
		}),
		// Third request.
		handler.walletStore.EXPECT().GetWallet(r3Ctx, wallet1.Name()).Times(1).Return(wallet1, nil),
	)
	gomock.InOrder(
		// First request.
		handler.node.EXPECT().SpamStatistics(r1Ctx, kp1.PublicKey()).Times(1).Return(spamStats, nil),
		// Third request.
		handler.node.EXPECT().SpamStatistics(r3Ctx, kp2.PublicKey()).Times(1).Return(spamStats, nil),
	)
	gomock.InOrder(
		// First request.
		handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil),
		// Third request.
		handler.spam.EXPECT().CheckSubmission(gomock.Any(), &spamStats).Times(1).Return(nil),
	)
	gomock.InOrder(
		// First request.
		handler.interactor.EXPECT().NotifySuccessfulRequest(r1Ctx, r1TraceID, uint8(2), api.TransactionSuccessfullyChecked).Times(1).Do(func(_ context.Context, _ string, _ uint8, _ string) {
			// Unblock the second and third requests, and trigger the signing.
			close(sendSecondRequests)
			<-waitForSecondRequestToExit
		}),
		// Third request.
		handler.interactor.EXPECT().NotifySuccessfulRequest(r3Ctx, r3TraceID, uint8(2), api.TransactionSuccessfullyChecked).Times(1),
	)
	gomock.InOrder(
		// First request.
		handler.node.EXPECT().CheckTransaction(r1Ctx, gomock.Any()).AnyTimes().Return(nil),
		// Third request.
		handler.node.EXPECT().CheckTransaction(r3Ctx, gomock.Any()).AnyTimes().Return(nil),
	)
	gomock.InOrder(
		// First request.
		handler.interactor.EXPECT().Log(r1Ctx, r1TraceID, gomock.Any(), gomock.Any()).AnyTimes(),
		// Third request.
		handler.interactor.EXPECT().Log(r3Ctx, r3TraceID, gomock.Any(), gomock.Any()).AnyTimes(),
	)

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		// when
		result, errorDetails := handler.handle(t, r1Ctx, api.ClientCheckTransactionParams{
			PublicKey:   kp1.PublicKey(),
			Transaction: testTransaction(t),
		}, connectedWallet)

		<-waitForSecondRequestToExit
		<-waitForThirdRequestToExit

		// then
		assert.Nil(t, errorDetails)
		require.NotEmpty(t, result)
		assert.NotEmpty(t, result.Transaction)
	}()

	go func() {
		defer wg.Done()

		// Closing this resume, unblock the first request.
		defer close(waitForSecondRequestToExit)

		// Ensure the first request acquire the "lock" on the public key.
		<-sendSecondRequests

		// when
		result, errorDetails := handler.handle(t, r2Ctx, api.ClientCheckTransactionParams{
			PublicKey:   kp1.PublicKey(),
			Transaction: testTransaction(t),
		}, connectedWallet)

		// then
		assert.NotNil(t, errorDetails)
		assertRequestNotPermittedError(t, errorDetails, fmt.Errorf("this public key %q is already in use, retry later", kp1.PublicKey()))
		require.Empty(t, result)
	}()

	go func() {
		defer wg.Done()
		defer close(waitForThirdRequestToExit)

		// Ensure the first request acquire the "lock" on the public key, and
		// we second request calls `GetWallet()` before the third request.
		<-sendThirdRequests

		// then
		result, errorDetails := handler.handle(t, r3Ctx, api.ClientCheckTransactionParams{
			PublicKey:   kp2.PublicKey(),
			Transaction: testTransaction(t),
		}, connectedWallet)

		// then
		assert.Nil(t, errorDetails)
		require.NotEmpty(t, result)
		assert.NotEmpty(t, result.Transaction)
	}()

	wg.Wait()
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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, api.ErrConnectionClosed)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerErrorType, api.ErrRequestInterrupted).Times(1)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkErrorType, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkErrorType, fmt.Errorf("could not get the latest block information from the node: %w", assert.AnError)).Times(1)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, gomock.Any()).Times(1)
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

type checkTransactionHandler struct {
	*api.ClientCheckTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
	walletStore  *mocks.MockWalletStore
	spam         *mocks.MockSpamHandler
}

func (h *checkTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params, connectedWallet api.ConnectedWallet) (api.ClientCheckTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, connectedWallet)
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

	requestController := api.NewRequestController(
		api.WithMaximumAttempt(1),
		api.WithIntervalDelayBetweenRetries(1*time.Second),
	)

	return &checkTransactionHandler{
		ClientCheckTransaction: api.NewClientCheckTransaction(walletStore, interactor, nodeSelector, proofOfWork, requestController),
		ctrl:                   ctrl,
		nodeSelector:           nodeSelector,
		interactor:             interactor,
		node:                   node,
		walletStore:            walletStore,
		spam:                   proofOfWork,
	}
}
