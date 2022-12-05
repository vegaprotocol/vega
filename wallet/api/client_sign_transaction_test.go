package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	nodemock "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignTransaction(t *testing.T) {
	t.Run("Signing a transaction with invalid params fails", testSigningTransactionWithInvalidParamsFails)
	t.Run("Signing a transaction with valid params succeeds", testSigningTransactionWithValidParamsSucceeds)
	t.Run("Signing a transaction with invalid token fails", testSigningTransactionWithInvalidTokenFails)
	t.Run("Signing a transaction with a long-living token succeeds", testSigningTransactionWithLongLivingTokenSucceeds)
	t.Run("Signing a transaction with a long-living token expired fails", testSigningTransactionWithLongLivingExpiredTokenFails)
	t.Run("Signing a transaction with a long-living valid token succeeds", testSigningTransactionWithLongLivingValidTokenSucceeds)
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
			params: api.ClientSignTransactionParams{
				Token:       "",
				PublicKey:   vgrand.RandomStr(10),
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		}, {
			name: "with empty public key permissions",
			params: api.ClientSignTransactionParams{
				Token:       vgrand.RandomStr(10),
				PublicKey:   "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		}, {
			name: "with empty encoded transaction",
			params: api.ClientSignTransactionParams{
				Token:              vgrand.RandomStr(10),
				PublicKey:          vgrand.RandomStr(10),
				EncodedTransaction: "",
			},
			expectedError: api.ErrTransactionIsRequired,
		}, {
			name: "with invalid encoded transaction",
			params: api.ClientSignTransactionParams{
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
			ctx := context.Background()
			metadata := requestMetadataForTest()

			// setup
			handler := newSignTransactionHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params, metadata)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testSigningTransactionWithValidParamsSucceeds(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:             100,
		BlockHash:               vgrand.RandomStr(64),
		ProofOfWorkHashFunction: "sha3_24_rounds",
		ProofOfWorkDifficulty:   1,
		ChainID:                 vgrand.RandomStr(5),
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, metadata.TraceID, api.TransactionSuccessfullySigned).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testSigningTransactionWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       vgrand.RandomStr(5),
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertInvalidParams(t, errorDetails, session.ErrNoWalletConnected)
	assert.Empty(t, result)
}

func testSigningTransactionWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	// setup
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, time.Now(), nil); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:             100,
		BlockHash:               vgrand.RandomStr(64),
		ProofOfWorkHashFunction: "sha3_24_rounds",
		ProofOfWorkDifficulty:   1,
		ChainID:                 vgrand.RandomStr(5),
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, metadata.TraceID, api.TransactionSuccessfullySigned).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		EncodedTransaction: encodedTransaction,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testSigningTransactionWithLongLivingExpiredTokenFails(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)
	nextNow := now.Add(2 * time.Hour)

	// setup
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(nextNow)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, now, &expiresAt); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}

	// when
	_, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		EncodedTransaction: encodedTransaction,
	}, metadata)

	// then
	assert.EqualError(t, errorDetails, "the token has expired (Invalid params -32602)")
}

func testSigningTransactionWithLongLivingValidTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	encodedTransaction := "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
	wallet1, kp := walletWithKey(t)
	token := vgrand.RandomStr(10)

	now := time.Now()
	expiresAt := now.Add(2 * time.Hour)
	nextNow := now.Add(1 * time.Hour)

	// setup
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1).Return(nextNow)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, wallet1, now, &expiresAt); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		BlockHeight:             100,
		BlockHash:               vgrand.RandomStr(64),
		ProofOfWorkHashFunction: "sha3_24_rounds",
		ProofOfWorkDifficulty:   1,
		ChainID:                 vgrand.RandomStr(5),
	}, nil)
	handler.interactor.EXPECT().NotifySuccessfulRequest(ctx, metadata.TraceID, api.TransactionSuccessfullySigned).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:              token,
		PublicKey:          kp.PublicKey(),
		EncodedTransaction: encodedTransaction,
	}, metadata)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.NotEmpty(t, result.Tx)
}

func testSigningTransactionWithoutNeededPermissionsDoesNotSignTransaction(t *testing.T) {
	// given
	ctx := context.Background()
	metadata := requestMetadataForTest()
	wallet1, _ := walletWithPerms(t, metadata.Hostname, wallet.Permissions{})
	_, _ = wallet1.GenerateKeyPair(nil)
	pubKey := wallet1.ListPublicKeys()[0].Key()

	// setup
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertRequestNotPermittedError(t, errorDetails, api.ErrPublicKeyIsNotAllowedToBeUsed)
	assert.Empty(t, result)
}

func testRefusingSigningOfTransactionDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertUserRejectionError(t, errorDetails)
	assert.Empty(t, result)
}

func testCancellingTheReviewDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrUserCloseTheConnection)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertConnectionClosedError(t, errorDetails)
	assert.Empty(t, result)
}

func testInterruptingTheRequestDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, api.ErrRequestInterrupted)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.ServerError, api.ErrRequestInterrupted).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertRequestInterruptionError(t, errorDetails)
	assert.Empty(t, result)
}

func testGettingInternalErrorDuringReviewDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(false, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.InternalError, fmt.Errorf("requesting the transaction review failed: %w", assert.AnError)).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	assertInternalError(t, errorDetails, api.ErrCouldNotSignTransaction)
	assert.Empty(t, result)
}

func testNoHealthyNodeAvailableDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.NetworkError, fmt.Errorf("could not find a healthy node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotSignTransaction(t *testing.T) {
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
	handler := newSignTransactionHandler(t)
	handler.time.EXPECT().Now().Times(1)
	token := connectWallet(t, handler.sessions, metadata.Hostname, wallet1)
	// -- expected calls
	handler.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, gomock.Any()).Times(1).Return(nil)
	handler.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, gomock.Any()).Times(1)
	handler.interactor.EXPECT().RequestTransactionReviewForSigning(ctx, metadata.TraceID, metadata.Hostname, wallet1.Name(), pubKey, testTransactionJSON, gomock.Any()).Times(1).Return(true, nil)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{}, assert.AnError)
	handler.interactor.EXPECT().NotifyError(ctx, metadata.TraceID, api.NetworkError, fmt.Errorf("could not get the latest block from the node: %w", assert.AnError)).Times(1)
	handler.interactor.EXPECT().Log(ctx, metadata.TraceID, gomock.Any(), gomock.Any()).AnyTimes()

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientSignTransactionParams{
		Token:       token,
		PublicKey:   pubKey,
		Transaction: testTransaction(t),
	}, metadata)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type signTransactionHandler struct {
	*api.ClientSignTransaction
	ctrl         *gomock.Controller
	interactor   *mocks.MockInteractor
	sessions     *session.Sessions
	nodeSelector *nodemock.MockSelector
	node         *nodemock.MockNode
	time         *mocks.MockTimeProvider
}

func (h *signTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}, metadata jsonrpc.RequestMetadata) (api.ClientSignTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, metadata)
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
	nodeSelector := nodemock.NewMockSelector(ctrl)
	interactor := mocks.NewMockInteractor(ctrl)

	sessions := session.NewSessions()
	node := nodemock.NewMockNode(ctrl)
	tp := mocks.NewMockTimeProvider(ctrl)

	return &signTransactionHandler{
		ClientSignTransaction: api.NewSignTransaction(interactor, nodeSelector, sessions, tp),
		ctrl:                  ctrl,
		nodeSelector:          nodeSelector,
		interactor:            interactor,
		sessions:              sessions,
		node:                  node,
		time:                  tp,
	}
}
