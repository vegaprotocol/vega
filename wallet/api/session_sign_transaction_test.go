package api_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignTransaction(t *testing.T) {
	t.Run("Signing a transaction with invalid params fails", testSigningTransactionWithInvalidParamsFails)
	t.Run("Signing a transaction with with valid params succeeds", testSigningTransactionWithValidParamsSucceeds)
	t.Run("Signing a transaction with invalid token fails", testSigningTransactionWithInvalidTokenFails)
	t.Run("Signing a transaction without the needed permissions sign the transaction", testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction)
	t.Run("Refusing the signing of a transaction does not sign the transaction", testRefusingSigningOfTransactionDoesNotSignTransaction)
	t.Run("Cancelling the review does not sign the transaction", testCancellingTheReviewDoesNotSignTransaction)
	t.Run("Interrupting the request does not sign the transaction", testInterruptingTheRequestDoesNotSignTransaction)
	t.Run("Getting internal error during the review does not sign the transaction", testGettingInternalErrorDuringReviewDoesNotSignTransaction)
	t.Run("No healthy node available does not sign the transaction", testNoHealthyNodeAvailableDoesNotSignTransaction)
	t.Run("Failing to get the last block does not sign the transaction", testFailingToGetLastBlockDoesNotSignTransaction)
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
			name: "with empty token",
			params: api.SignTransactionParams{
				Token:              "",
				PublicKey:          vgrand.RandomStr(10),
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		}, {
			name: "with empty public key permissions",
			params: api.SignTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          "",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		}, {
			name: "with empty encoded transaction",
			params: api.SignTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				EncodedTransaction: "",
			},
			expectedError: api.ErrEncodedTransactionIsRequired,
		}, {
			name: "with invalid encoded transaction",
			params: api.SignTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
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
			handler := newSignTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSigningTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
		Height:            100,
		Hash:              vgrand.RandomStr(64),
		SpamPowDifficulty: 1,
	}, nil)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(ctx, traceID).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testSigningTransactionWithInvalidTokenFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              vgrand.RandomStr(5),
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSigningOfTransactionDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSignTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find an healthy node: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotSignTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- expected calls
	handler.pipeline.EXPECT().RequestTransactionSigningReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get last block from node: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SignTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		EncodedTransaction: encodedTransaction,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type signTransactionHandler struct {
	*api.SignTransaction
	ctrl         *gomock.Controller
	pipeline     *mocks.MockPipeline
	sessions     *api.Sessions
	nodeSelector *mocks.MockNodeSelector
	node         *mocks.MockNode
}

func (h *signTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.SignTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.SignTransactionResult)
		if !ok {
			t.Fatal("SignTransaction handler result is not a SignTransactionResult")
		}
		return result, err
	}
	return api.SignTransactionResult{}, err
}

func newSignTransactionHandler(t *testing.T) *signTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := mocks.NewMockNodeSelector(ctrl)
	pipeline := mocks.NewMockPipeline(ctrl)

	sessions := api.NewSessions()
	node := mocks.NewMockNode(ctrl)

	return &signTransactionHandler{
		SignTransaction: api.NewSignTransaction(pipeline, nodeSelector, sessions),
		ctrl:            ctrl,
		nodeSelector:    nodeSelector,
		pipeline:        pipeline,
		sessions:        sessions,
		node:            node,
	}
}
