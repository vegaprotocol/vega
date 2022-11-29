package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
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
	t.Run("Sending a transaction with invalid token fails", testSendingTransactionWithInvalidTokenFails)
	t.Run("Sending a transaction without the needed permissions send the transaction", testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction)
	t.Run("Refusing the sending of a transaction does not send the transaction", testRefusingSendingOfTransactionDoesNotSendTransaction)
	t.Run("Cancelling the review does not send the transaction", testCancellingTheReviewDoesNotSendTransaction)
	t.Run("Interrupting the request does not send the transaction", testInterruptingTheRequestDoesNotSendTransaction)
	t.Run("Getting internal error during the review does not send the transaction", testGettingInternalErrorDuringReviewDoesNotSendTransaction)
	t.Run("No healthy node available does not send the transaction", testNoHealthyNodeAvailableDoesNotSendTransaction)
	t.Run("Failing to get the last block does not send the transaction", testFailingToGetLastBlockDoesNotSendTransaction)
	t.Run("Failure when sending transaction returns an error", testFailureWhenSendingTransactionReturnsAnError)
}

var testTransactionJSON = `{"voteSubmission":{"proposalId":"eb2d3902fdda9c3eb6e369f2235689b871c7322cf3ab284dde3e9dfc13863a17","value":"VALUE_YES"}}`

func testTransaction(t *testing.T) map[string]any {
	t.Helper()
	testTransaction := make(map[string]any)
	assert.NoError(t, json.Unmarshal([]byte(testTransactionJSON), &testTransaction))
	return testTransaction
}

var testEncodedTransaction = "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"

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
			name: "with empty token",
			params: api.ClientSendTransactionParams{
				Token:       "",
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_SYNC",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		},
		{
			name: "with empty public key permissions",
			params: api.ClientSendTransactionParams{
				Token:       vgrand.RandomStr(10),
				PublicKey:   "",
				SendingMode: "TYPE_SYNC",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty sending mode",
			params: api.ClientSendTransactionParams{
				Token:       vgrand.RandomStr(10),
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrSendingModeIsRequired,
		},
		{
			name: "with unsupported sending mode",
			params: api.ClientSendTransactionParams{
				Token:       vgrand.RandomStr(10),
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_UNSPECIFIED",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrSendingModeCannotBeTypeUnspecified,
		},
		{
			name: "with unsupported sending mode",
			params: api.ClientSendTransactionParams{
				Token:       vgrand.RandomStr(10),
				PublicKey:   vgrand.RandomStr(10),
				SendingMode: "TYPE_MANY_FAST",
				Transaction: testTransaction(t),
			},
			expectedError: errors.New(`the sending mode "TYPE_MANY_FAST" is not a valid one`),
		},
		{
			name: "with empty encoded transaction and no transaction",
			params: api.ClientSendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "",
			},
			expectedError: api.ErrTransactionIsRequired,
		},
		{
			name: "with both encoded transaction and transaction",
			params: api.ClientSendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: testEncodedTransaction,
				Transaction:        testTransaction(t),
			},
			expectedError: api.ErrEncodedTransactionAndTransactionSupplied,
		},
		{
			name: "with invalid encoded transaction",
			params: api.ClientSendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: `{ "voteSubmission": {} }`,
			},
			expectedError: api.ErrEncodedTransactionIsNotValidBase64String,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newSendTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()
	txHash := vgrand.RandomStr(64)

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.interactor.EXPECT().NotifySuccessfulTransaction(ctx, traceID, txHash, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testSendingTransactionWithInvalidTokenFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       vgrand.RandomStr(5),
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSendingOfTransactionDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSendTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get the latest block from node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailureWhenSendingTransactionReturnsAnError(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, traceID, hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyFailedTransaction(ctx, traceID, gomock.Any(), gomock.Any(), assert.AnError, gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, traceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrTransactionFailed.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type sendTransactionHandler struct {
	*api.ClientSendTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	sessions     *api.Sessions
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
}

func (h *sendTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.ClientSendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
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

	sessions := api.NewSessions()
	node := nodemocks.NewMockNode(ctrl)

	return &sendTransactionHandler{
		ClientSendTransaction: api.NewSendTransaction(interactor, nodeSelector, sessions),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interactor,
		sessions:              sessions,
		node:                  node,
	}
}
