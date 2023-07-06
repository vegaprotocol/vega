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
	nodemock "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientSignTransaction(t *testing.T) {
	t.Run("Documentation matches the code", testClientSignTransactionSchemaCorrect)
	t.Run("Signing a transaction with invalid params fails", testSigningTransactionWithInvalidParamsFails)
	t.Run("Signing a transaction with valid params succeeds", testSigningTransactionWithValidParamsSucceeds)
	t.Run("Signing a transaction in parallel blocks on same party but not on different parties", testSigningTransactionInParallelBlocksOnSamePartyButNotOnDifferentParties)
	t.Run("Signing a transaction without the needed permissions does not sign the transaction", testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction)
	t.Run("Refusing the signing of a transaction does not sign the transaction", testRefusingSigningOfTransactionDoesNotSignTransaction)
	t.Run("Cancelling the review does not sign the transaction", testCancellingTheReviewDoesNotSignTransaction)
	t.Run("Interrupting the request does not sign the transaction", testInterruptingTheRequestDoesNotSignTransaction)
	t.Run("Getting internal error during the review does not sign the transaction", testGettingInternalErrorDuringReviewDoesNotSignTransaction)
	t.Run("No healthy node available does not sign the transaction", testNoHealthyNodeAvailableDoesNotSignTransaction)
	t.Run("Failing to get spam statistics does not sign the transaction", testFailingToGetSpamStatsDoesNotSignTransaction)
	t.Run("Failing spam check aborts signing the transaction", testFailingSpamChecksAbortsSigningTheTransaction)
}

func testClientSignTransactionSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "client.sign_transaction", api.ClientSignTransactionParams{}, api.ClientSignTransactionResult{})
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
			expectedError: errors.New("the transaction does not use a valid Vega command: unknown field \"type\" in vega.wallet.v1.SubmitTransactionRequest"),
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
	assert.NotEmpty(t, result.Transaction)
}

func testSigningTransactionInParallelBlocksOnSamePartyButNotOnDifferentParties(t *testing.T) {
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
	handler := newSignTransactionHandler(t)

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
		handler.interactor.EXPECT().RequestTransactionReviewForSigning(r1Ctx, r1TraceID, uint8(1), hostname, wallet1.Name(), kp1.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil),
		// Third request.
		handler.interactor.EXPECT().RequestTransactionReviewForSigning(r3Ctx, r3TraceID, uint8(1), hostname, wallet1.Name(), kp2.PublicKey(), fakeTransaction, gomock.Any()).Times(1).Return(true, nil),
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
		handler.interactor.EXPECT().NotifySuccessfulRequest(r1Ctx, r1TraceID, uint8(2), api.TransactionSuccessfullySigned).Times(1).Do(func(_ context.Context, _ string, _ uint8, _ string) {
			// Unblock the second and third requests, and trigger the signing.
			close(sendSecondRequests)
			<-waitForSecondRequestToExit
		}),
		// Third request.
		handler.interactor.EXPECT().NotifySuccessfulRequest(r3Ctx, r3TraceID, uint8(2), api.TransactionSuccessfullySigned).Times(1),
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
		result, errorDetails := handler.handle(t, r1Ctx, api.ClientSignTransactionParams{
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
		result, errorDetails := handler.handle(t, r2Ctx, api.ClientSignTransactionParams{
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
		result, errorDetails := handler.handle(t, r3Ctx, api.ClientSignTransactionParams{
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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, api.ErrConnectionClosed)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerErrorType, api.ErrRequestInterrupted).Times(1)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkErrorType, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkErrorType, fmt.Errorf("could not get the latest block information from the node: %w", assert.AnError)).Times(1)
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
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ApplicationErrorType, gomock.Any()).Times(1)
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
	interact := mocks.NewMockInteractor(ctrl)
	nodeSelector := nodemock.NewMockSelector(ctrl)
	node := nodemock.NewMockNode(ctrl)
	proofOfWork := mocks.NewMockSpamHandler(ctrl)

	requestController := api.NewRequestController(
		api.WithMaximumAttempt(1),
		api.WithIntervalDelayBetweenRetries(1*time.Second),
	)

	return &signTransactionHandler{
		ClientSignTransaction: api.NewClientSignTransaction(walletStore, interact, nodeSelector, proofOfWork, requestController),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interact,
		node:                  node,
		walletStore:           walletStore,
		spam:                  proofOfWork,
	}
}
