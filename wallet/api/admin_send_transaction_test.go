package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAdminSendTransaction(t *testing.T) {
	t.Run("Sending transaction with invalid params fails", testAdminSendingTransactionWithInvalidParamsFails)
	t.Run("Sending transaction with valid params succeeds", testAdminSendingTransactionWithValidParamsSucceeds)
	t.Run("Getting internal error during wallet verification fails", testAdminSendTransactionGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Sending transaction with wallet that doesn't exist fails", testAdminSendingTransactionWithWalletThatDoesntExistFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminSendTransactionGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Sending transaction with malformed transaction fails", testAdminSendingTransactionWithMalformedTransactionFails)
	t.Run("Sending transaction which is invalid fails", testAdminSendingTransactionWithInvalidTransactionFails)
}

func testAdminSendingTransactionWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminSendTransactionParams{
				Wallet:      "",
				Passphrase:  vgrand.RandomStr(5),
				PublicKey:   vgrand.RandomStr(5),
				Transaction: testTransaction(t),
				Network:     vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		},
		{
			name: "with empty passphrase",
			params: api.AdminSendTransactionParams{
				Wallet:      vgrand.RandomStr(5),
				Passphrase:  "",
				PublicKey:   vgrand.RandomStr(5),
				Transaction: testTransaction(t),
				Network:     vgrand.RandomStr(5),
			},
			expectedError: api.ErrPassphraseIsRequired,
		},
		{
			name: "with empty public key",
			params: api.AdminSendTransactionParams{
				Wallet:      vgrand.RandomStr(5),
				Passphrase:  vgrand.RandomStr(5),
				PublicKey:   "",
				Transaction: testTransaction(t),
				Network:     vgrand.RandomStr(5),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty transaction",
			params: api.AdminSendTransactionParams{
				Wallet:      vgrand.RandomStr(5),
				Passphrase:  vgrand.RandomStr(5),
				PublicKey:   vgrand.RandomStr(5),
				Transaction: "",
				Network:     vgrand.RandomStr(5),
			},
			expectedError: api.ErrTransactionIsRequired,
		},
		{
			name: "with no network or node address",
			params: api.AdminSendTransactionParams{
				Wallet:      vgrand.RandomStr(5),
				Passphrase:  vgrand.RandomStr(5),
				PublicKey:   vgrand.RandomStr(5),
				Network:     "",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrNetworkOrNodeAddressIsRequired,
		},
		{
			name: "with no network and node address",
			params: api.AdminSendTransactionParams{
				Wallet:      vgrand.RandomStr(5),
				Passphrase:  vgrand.RandomStr(5),
				PublicKey:   vgrand.RandomStr(5),
				Network:     "some_network",
				NodeAddress: "some_node_address",
				Transaction: testTransaction(t),
			},
			expectedError: api.ErrSpecifyingNetworkAndNodeAddressIsNotSupported,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newAdminSendTransactionHandler(tt, unexpectedNodeSelectorCall(tt))

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testAdminSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)
	passphrase := vgrand.RandomStr(5)
	nodeHost := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)
	hash := "hashy mchashface"

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
			BlockHeight:             150,
			BlockHash:               vgrand.RandomStr(64),
			ProofOfWorkHashFunction: vgcrypto.Sha3,
			ProofOfWorkDifficulty:   1,
			ChainID:                 vgrand.RandomStr(5),
		}, nil)
		node.EXPECT().SendTransaction(ctx, gomock.Any(), gomock.Any()).Times(1).Return(hash, nil)
		node.EXPECT().Host().Times(1).Return(nodeHost)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, w.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Wallet:      w.Name(),
		Passphrase:  passphrase,
		PublicKey:   kp.PublicKey(),
		Network:     network.Name,
		Transaction: testTransaction(t),
	})

	// then
	assert.Nil(t, errorDetails)
	assert.Equal(t, hash, result.TxHash)
	assert.NotEmpty(t, result.Tx)
}

func testAdminSendTransactionGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
			BlockHeight:             150,
			BlockHash:               vgrand.RandomStr(64),
			ProofOfWorkHashFunction: vgcrypto.Sha3,
			ProofOfWorkDifficulty:   1,
			ChainID:                 vgrand.RandomStr(5),
		}, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, walletName).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Wallet:      walletName,
		Passphrase:  passphrase,
		PublicKey:   vgrand.RandomStr(5),
		Network:     network.Name,
		Transaction: testTransaction(t),
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithWalletThatDoesntExistFails(t *testing.T) {
	// given
	ctx := context.Background()

	params := api.AdminSendTransactionParams{
		Wallet:      vgrand.RandomStr(5),
		Passphrase:  vgrand.RandomStr(5),
		PublicKey:   vgrand.RandomStr(5),
		Network:     "fairground",
		Transaction: testTransaction(t),
	}

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, params.Wallet).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, params)

	// then
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
	assert.Empty(t, result)
}

func testAdminSendTransactionGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
			BlockHeight:             150,
			BlockHash:               vgrand.RandomStr(64),
			ProofOfWorkHashFunction: vgcrypto.Sha3,
			ProofOfWorkDifficulty:   1,
			ChainID:                 vgrand.RandomStr(5),
		}, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, walletName).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, walletName, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, walletName).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Wallet:      walletName,
		Passphrase:  passphrase,
		PublicKey:   vgrand.RandomStr(5),
		Network:     network.Name,
		Transaction: testTransaction(t),
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithMalformedTransactionFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, w.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Wallet:      w.Name(),
		Passphrase:  passphrase,
		PublicKey:   kp.PublicKey(),
		Network:     network,
		Transaction: map[string]int{"bob": 5},
	})

	// then
	assertInvalidParams(t, errorDetails, errors.New("the transaction does not use a valid Vega command: unknown field \"bob\" in vega.wallet.v1.SubmitTransactionRequest"))
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithInvalidTransactionFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)
	passphrase := vgrand.RandomStr(5)
	w, kp := walletWithKey(t)

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, w.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, w.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, w.Name()).Times(1).Return(w, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Wallet:      w.Name(),
		Passphrase:  passphrase,
		PublicKey:   kp.PublicKey(),
		Network:     network.Name,
		Transaction: testMalformedTransaction(t),
	})

	// then
	assertInvalidParams(t, errorDetails, fmt.Errorf("vote_submission.proposal_id (should be a valid vega ID)"))
	assert.Empty(t, result)
}

type AdminSendTransactionHandler struct {
	*api.AdminSendTransaction
	ctrl         *gomock.Controller
	walletStore  *mocks.MockWalletStore
	networkStore *mocks.MockNetworkStore
}

func (h *AdminSendTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminSendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminSendTransactionResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminSendTransactionResult")
		}
		return result, err
	}
	return api.AdminSendTransactionResult{}, err
}

func newAdminSendTransactionHandler(t *testing.T, builder api.NodeSelectorBuilder) *AdminSendTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &AdminSendTransactionHandler{
		AdminSendTransaction: api.NewAdminSendTransaction(walletStore, networkStore, builder),
		ctrl:                 ctrl,
		walletStore:          walletStore,
		networkStore:         networkStore,
	}
}
