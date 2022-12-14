package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendTransaction(t *testing.T) {
	t.Run("Sending a transaction with invalid params fails", testSendingTransactionWithInvalidParamsFails)
	t.Run("Sending a transaction with valid params succeeds", testSendingTransactionWithValidParamsSucceeds)
	t.Run("Sending a transaction with invalid token fails", testSendingTransactionWithInvalidTokenFails)
	t.Run("Sending a transaction with a long-living token succeeds", testSendingTransactionWithLongLivingTokenSucceeds)
	t.Run("Sending a transaction with a long-living expired token succeeds", testSendingTransactionWithLongLivingExpiredTokenFails)
	t.Run("Sending a transaction with a long-living  token succeeds", testSendingTransactionWithLongLivingExpiringTokenSucceed)
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
			ctx := context.Background()
			metadata := requestMetadataForTest()

			// setup
			handler := newSendTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, metadata)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
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
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.interactor.EXPECT().NotifySuccessfulTransaction(ctx, metadata.TraceID, txHash, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testSendingTransactionWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       vgrand.RandomStr(5),
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertInvalidParams(t, errorDetails, session.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testSendingTransactionWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)
	txHash := vgrand.RandomStr(64)

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, time.Now(), nil); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.interactor.EXPECT().NotifySuccessfulTransaction(ctx, metadata.TraceID, txHash, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testSendingTransactionWithLongLivingExpiredTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	now := time.Now()
	expiring := now.Add(1 * time.Hour)
	afterExpiry := now.Add(2 * time.Hour)

	// setup
	handler := newSendTransactionHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, now, &expiring); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}
	// -- expected calls
	handler.time.EXPECT().Now().Times(1).Return(afterExpiry)

	// when
	_, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	}, metadata)

	assert.EqualError(t, errorDetails, "the token has expired (Invalid params -32602)")
}

func testSendingTransactionWithLongLivingExpiringTokenSucceed(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)
	txHash := vgrand.RandomStr(64)

	now := time.Now()
	expiry := now.Add(1 * time.Hour)
	nextT := now.Add(30 * time.Minute)

	// setup
	handler := newSendTransactionHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, now, &expiry); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
	handler.interactor.EXPECT().NotifySuccessfulTransaction(ctx, metadata.TraceID, txHash, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()
	handler.time.EXPECT().Now().Times(1).Return(nextT)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: encodedTransaction,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testSendingTransactionWithoutNeededPermissionsDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())

	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSendingOfTransactionDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, nil)
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSendTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotSendTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.NetworkError, fmt.Errorf("could not get the latest block from node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailureWhenSendingTransactionReturnsAnError(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: nil,
		},
	})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSendTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(time.Now())
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSending(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:           100,
		BlockHash:             vgrand.RandomStr(64),
		ProofOfWorkDifficulty: 1,
		ChainID:               "chain-id",
	}, nil)
	handler.node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
	handler.interactor.EXPECT().NotifyFailedTransaction(ctx, metadata.TraceID, gomock.Any(), gomock.Any(), assert.AnError, gomock.Any()).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSendTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		SendingMode: "TYPE_SYNC",
		Transaction: testTransaction(t),
	}, metadata)

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
	sessions     *session.Sessions
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
	time         *mocks.MockTimeProvider
}

func (h *sendTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}, metadata jsonrpc.RequestMetadata) (api.ClientSendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, metadata)
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

	sessions := session.NewSessions()
	node := nodemocks.NewMockNode(ctrl)
	tp := mocks.NewMockTimeProvider(ctrl)

	return &sendTransactionHandler{
		ClientSendTransaction: api.NewSendTransaction(interactor, nodeSelector, sessions, tp),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interactor,
		sessions:              sessions,
		node:                  node,
		time:                  tp,
	}
}
