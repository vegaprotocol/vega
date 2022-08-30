package api_test

import (
	"context"
	"encoding/base64"
	"errors"
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

func TestSendTransaction(t *testing.T) {
	t.Run("Sending a transaction with invalid params fails", testSendingTransactionWithInvalidParamsFails)
	t.Run("Sending a transaction with with valid params succeeds", testSendingTransactionWithValidParamsSucceeds)
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
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty token",
			params: api.SendTransactionParams{
				Token:              "",
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		}, {
			name: "with empty public key permissions",
			params: api.SendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          "",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		}, {
			name: "with empty sending mode",
			params: api.SendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrSendingModeIsRequired,
		}, {
			name: "with unsupported sending mode",
			params: api.SendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_UNSPECIFIED",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: api.ErrSendingModeCannotBeTypeUnspecified,
		}, {
			name: "with unsupported sending mode",
			params: api.SendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_MANY_FAST",
				EncodedTransaction: "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K",
			},
			expectedError: errors.New(`sending mode "TYPE_MANY_FAST" is not a valid one`),
		}, {
			name: "with empty encoded transaction",
			params: api.SendTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "",
			},
			expectedError: api.ErrEncodedTransactionIsRequired,
		}, {
			name: "with invalid encoded transaction",
			params: api.SendTransactionParams{
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
			// -- unexpected calls
			handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
			handler.nodeSelector.EXPECT().Stop().Times(0)
			handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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

func testSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
		Height:            100,
		Hash:              vgrand.RandomStr(64),
		SpamPowDifficulty: 1,
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.pipeline.EXPECT().NotifyTransactionStatus(ctx, traceID, txHash, gomock.Any(), nil, gomock.Any()).Times(1)
	// -- unexpected calls
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
}

func testSendingTransactionWithInvalidTokenFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	// -- unexpected calls
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              vgrand.RandomStr(5),
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	token := connectWallet(t, handler.sessions, hostname, wallet1)
	// -- unexpected calls
	handler.pipeline.EXPECT().RequestTransactionReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSendingOfTransactionDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, nil)
	// -- unexpected calls
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertClientRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, api.ErrConnectionClosed)
	// -- unexpected calls
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.ServerError, api.ErrRequestInterrupted).Times(1)
	// -- unexpected calls
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.nodeSelector.EXPECT().Node(gomock.Any()).Times(0)
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSendTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not find an healthy node: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotSendTransaction(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(nil, assert.AnError)
	handler.pipeline.EXPECT().NotifyError(ctx, traceID, api.NetworkError, fmt.Errorf("could not get last block from node: %w", assert.AnError)).Times(1)
	// -- unexpected calls
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyTransactionStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailureWhenSendingTransactionReturnsAnError(t *testing.T) {
	// given
	ctx, traceID := contextWithTraceID()
	hostname := "vega.xyz"
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	decodedTransaction, _ := base64.StdEncoding.DecodeString(encodedTransaction)
	wallet1 := walletWithPerms(t, hostname, wallet.Permissions{
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
	handler.pipeline.EXPECT().RequestTransactionReview(ctx, traceID, hostname, wallet1.Name(), pubKey, string(decodedTransaction), gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
		Height:            100,
		Hash:              vgrand.RandomStr(64),
		SpamPowDifficulty: 1,
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
	handler.pipeline.EXPECT().NotifyTransactionStatus(ctx, traceID, "", gomock.Any(), assert.AnError, gomock.Any()).Times(1)
	// -- unexpected calls
	handler.node.EXPECT().LastBlock(gomock.Any()).Times(0)
	handler.node.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.nodeSelector.EXPECT().Stop().Times(0)
	handler.pipeline.EXPECT().RequestPassphrase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestPermissionsReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletConnectionReview(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().RequestWalletSelection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifyError(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.pipeline.EXPECT().NotifySuccessfulRequest(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.SendTransactionParams{
		Token:              token,
		PublicKey:          pubKey,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrTransactionFailed.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type sendTransactionHandler struct {
	*api.SendTransaction
	ctrl         *gomock.Controller
	pipeline     *mocks.MockPipeline
	sessions     *api.Sessions
	nodeSelector *mocks.MockNodeSelector
	node         *mocks.MockNode
}

func (h *sendTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.SendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.SendTransactionResult)
		if !ok {
			t.Fatal("SendTransaction handler result is not a SendTransactionResult")
		}
		return result, err
	}
	return api.SendTransactionResult{}, err
}

func newSendTransactionHandler(t *testing.T) *sendTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := mocks.NewMockNodeSelector(ctrl)
	pipeline := mocks.NewMockPipeline(ctrl)

	sessions := api.NewSessions()
	node := mocks.NewMockNode(ctrl)

	return &sendTransactionHandler{
		SendTransaction: api.NewSendTransaction(pipeline, nodeSelector, sessions),
		ctrl:            ctrl,
		nodeSelector:    nodeSelector,
		pipeline:        pipeline,
		sessions:        sessions,
		node:            node,
	}
}
