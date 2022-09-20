package api_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAdminSignTransaction(t *testing.T) {
	t.Run("Signing transaction with invalid params fails", testAdminSigningTransactionWithInvalidParamsFails)
	t.Run("Signing transaction with valid params succeeds", testAdminSigningTransactionWithValidParamsSucceeds)
	t.Run("Getting internal error during wallet verification fails", testAdminSignTransactionGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Signing transaction with wallet that doesn't exist fails", testAdminSigningTransactionWithWalletThatDoesntExistFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminSignTransactionGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Signing transaction with malformed transaction fails", testAdminSigningTransactionWithMalformedTransactionFails)
	t.Run("Signing transaction which is invalid fails", testAdminSigningTransactionWithInvalidTransactionFails)
}

func testAdminSigningTransactionWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty wallet",
			params: api.AdminSignTransactionParams{
				Wallet:         "",
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      vgrand.RandomStr(5),
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
				Network:        vgrand.RandomStr(5),
				LastBlockData:  nil,
			},
			expectedError: api.ErrWalletIsRequired,
		},
		{
			name: "with empty passphrase",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     "",
				PublicKey:      vgrand.RandomStr(5),
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
				Network:        vgrand.RandomStr(5),
				LastBlockData:  nil,
			},
			expectedError: api.ErrPassphraseIsRequired,
		},
		{
			name: "with empty public key",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      "",
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
				Network:        vgrand.RandomStr(5),
				LastBlockData:  nil,
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty encoded transaction",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      vgrand.RandomStr(5),
				EncodedCommand: "",
				Network:        vgrand.RandomStr(5),
				LastBlockData:  nil,
			},
			expectedError: api.ErrEncodedTransactionIsRequired,
		},
		{
			name: "with badly encoded transaction",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      vgrand.RandomStr(5),
				EncodedCommand: "{}",
				Network:        vgrand.RandomStr(5),
				LastBlockData:  nil,
			},
			expectedError: api.ErrEncodedTransactionIsNotValidBase64String,
		},
		{
			name: "with no network of block data",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      vgrand.RandomStr(5),
				Network:        "",
				LastBlockData:  nil,
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrLastBlockDataOrNetworkIsRequired,
		},
		{
			name: "with both network and block data",
			params: api.AdminSignTransactionParams{
				Wallet:         vgrand.RandomStr(5),
				Passphrase:     vgrand.RandomStr(5),
				PublicKey:      vgrand.RandomStr(5),
				Network:        "fairground",
				LastBlockData:  &api.AdminLastBlockData{},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrSpecifyingNetworkAndLastBlockDataIsNotSupported,
		},
		{
			name: "with block data without chain ID",
			params: api.AdminSignTransactionParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				PublicKey:  vgrand.RandomStr(5),
				LastBlockData: &api.AdminLastBlockData{
					ChainID:                 "",
					BlockHeight:             12,
					BlockHash:               vgrand.RandomStr(64),
					ProofOfWorkHashFunction: "sha3_24_rounds",
					ProofOfWorkDifficulty:   12,
				},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrChainIDIsRequired,
		},
		{
			name: "with block data without block hash",
			params: api.AdminSignTransactionParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				PublicKey:  vgrand.RandomStr(5),
				LastBlockData: &api.AdminLastBlockData{
					ChainID:                 "chain-id",
					BlockHeight:             12,
					BlockHash:               "",
					ProofOfWorkHashFunction: "sha3_24_rounds",
					ProofOfWorkDifficulty:   12,
				},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrBlockHashIsRequired,
		},
		{
			name: "with block data without pow difficulty",
			params: api.AdminSignTransactionParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				PublicKey:  vgrand.RandomStr(5),
				LastBlockData: &api.AdminLastBlockData{
					ChainID:                 "chain-id",
					BlockHeight:             12,
					BlockHash:               vgrand.RandomStr(64),
					ProofOfWorkHashFunction: "sha3_24_rounds",
					ProofOfWorkDifficulty:   0,
				},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrProofOfWorkDifficultyRequired,
		},
		{
			name: "with block data without block height",
			params: api.AdminSignTransactionParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				PublicKey:  vgrand.RandomStr(5),
				LastBlockData: &api.AdminLastBlockData{
					BlockHeight:             0,
					ChainID:                 "chain-id",
					BlockHash:               vgrand.RandomStr(64),
					ProofOfWorkDifficulty:   12,
					ProofOfWorkHashFunction: "sha3_24_rounds",
				},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrBlockHeightIsRequired,
		},
		{
			name: "with block data without hash function",
			params: api.AdminSignTransactionParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				PublicKey:  vgrand.RandomStr(5),
				LastBlockData: &api.AdminLastBlockData{
					BlockHeight:             150,
					ChainID:                 "chain-id",
					BlockHash:               vgrand.RandomStr(64),
					ProofOfWorkDifficulty:   12,
					ProofOfWorkHashFunction: "",
				},
				EncodedCommand: base64.StdEncoding.EncodeToString([]byte("{}")),
			},
			expectedError: api.ErrProofOfWorkHashFunctionRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newAdminSignTransactionHandler(tt, unexpectedNodeSelectorCall(tt))

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testAdminSigningTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)
	passphrase := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)
	encodedCommand := base64.StdEncoding.EncodeToString([]byte("{\"voteSubmission\": {\"proposalId\": \"bbf5079b800a93c7e977547f45a7d0353aa2e52b0c745ff090fc795d9012a604\", \"value\": \"VALUE_YES\"}}"))

	// setup
	handler := newAdminSignTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
			Height:              150,
			Hash:                vgrand.RandomStr(64),
			SpamPowHashFunction: vgcrypto.Sha3,
			SpamPowDifficulty:   1,
			ChainId:             vgrand.RandomStr(5),
		}, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name(), passphrase).Times(1).Return(w, nil)
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSignTransactionParams{
		Wallet:         w.Name(),
		Passphrase:     passphrase,
		PublicKey:      kp.PublicKey(),
		Network:        network.Name,
		EncodedCommand: encodedCommand,
	})

	// then
	assert.Nil(t, errorDetails)
	assert.NotEmpty(t, result.EncodedTransaction)
	assert.NotEmpty(t, result.Tx)
}

func testAdminSignTransactionGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	encodedCommand := base64.StdEncoding.EncodeToString([]byte("{\"voteSubmission\": {\"proposalId\": \"bbf5079b800a93c7e977547f45a7d0353aa2e52b0c745ff090fc795d9012a604\", \"value\": \"VALUE_YES\"}}"))

	// setup
	handler := newAdminSignTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
			Height:              150,
			Hash:                vgrand.RandomStr(64),
			SpamPowHashFunction: vgcrypto.Sha3,
			SpamPowDifficulty:   1,
			ChainId:             vgrand.RandomStr(5),
		}, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, walletName).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSignTransactionParams{
		Wallet:         walletName,
		Passphrase:     passphrase,
		PublicKey:      vgrand.RandomStr(5),
		Network:        network.Name,
		EncodedCommand: encodedCommand,
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSigningTransactionWithWalletThatDoesntExistFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	params := paramsWithTransaction(t, "{\"voteSubmission\": {\"proposalId\": \"bbf5079b800a93c7e977547f45a7d0353aa2e52b0c745ff090fc795d9012a604\", \"value\": \"VALUE_YES\"}}")

	// setup
	handler := newAdminSignTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, params.Wallet).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, params)

	// then
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
	assert.Empty(t, result)
}

func testAdminSignTransactionGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	encodedCommand := base64.StdEncoding.EncodeToString([]byte("{\"voteSubmission\": {\"proposalId\": \"bbf5079b800a93c7e977547f45a7d0353aa2e52b0c745ff090fc795d9012a604\", \"value\": \"VALUE_YES\"}}"))

	// setup
	handler := newAdminSignTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
			Height:              150,
			Hash:                vgrand.RandomStr(64),
			SpamPowHashFunction: vgcrypto.Sha3,
			SpamPowDifficulty:   1,
			ChainId:             vgrand.RandomStr(5),
		}, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, walletName).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, walletName, passphrase).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSignTransactionParams{
		Wallet:         walletName,
		Passphrase:     passphrase,
		PublicKey:      vgrand.RandomStr(5),
		Network:        network.Name,
		EncodedCommand: encodedCommand,
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSigningTransactionWithMalformedTransactionFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)
	encodedCommand := base64.StdEncoding.EncodeToString([]byte("I am not a transaction"))

	// setup
	handler := newAdminSignTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name(), passphrase).Times(1).Return(w, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSignTransactionParams{
		Wallet:         w.Name(),
		Passphrase:     passphrase,
		PublicKey:      kp.PublicKey(),
		Network:        network,
		EncodedCommand: encodedCommand,
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrTransactionIsMalformed)
	assert.Empty(t, result)
}

func testAdminSigningTransactionWithInvalidTransactionFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)
	passphrase := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)
	encodedCommand := base64.StdEncoding.EncodeToString([]byte("{\"voteSubmission\": {\"proposalId\": \"not real id\", \"value\": \"VALUE_YES\"}}"))

	// setup
	handler := newAdminSignTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name(), passphrase).Times(1).Return(w, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSignTransactionParams{
		Wallet:         w.Name(),
		Passphrase:     passphrase,
		PublicKey:      kp.PublicKey(),
		Network:        network.Name,
		EncodedCommand: encodedCommand,
	})

	// then
	assertInvalidParams(t, errorDetails, fmt.Errorf("vote_submission.proposal_id (should be a valid vega ID)"))
	assert.Empty(t, result)
}

type AdminSignTransactionHandler struct {
	*api.AdminSignTransaction
	ctrl         *gomock.Controller
	walletStore  *mocks.MockWalletStore
	networkStore *mocks.MockNetworkStore
}

func (h *AdminSignTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminSignTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminSignTransactionResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminSignTransactionResult")
		}
		return result, err
	}
	return api.AdminSignTransactionResult{}, err
}

func newAdminSignTransactionHandler(t *testing.T, builder api.NodeSelectorBuilder) *AdminSignTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &AdminSignTransactionHandler{
		AdminSignTransaction: api.NewAdminSignTransaction(walletStore, networkStore, builder),
		ctrl:                 ctrl,
		walletStore:          walletStore,
		networkStore:         networkStore,
	}
}

func paramsWithTransaction(t *testing.T, tx string) api.AdminSignTransactionParams {
	t.Helper()

	encoded := base64.StdEncoding.EncodeToString([]byte(tx))
	return api.AdminSignTransactionParams{
		Wallet:         vgrand.RandomStr(5),
		Passphrase:     vgrand.RandomStr(5),
		PublicKey:      vgrand.RandomStr(5),
		Network:        "fairground",
		EncodedCommand: encoded,
	}
}
