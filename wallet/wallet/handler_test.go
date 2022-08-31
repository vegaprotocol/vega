package wallet_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallet/mocks"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotateKey(t *testing.T) {
	t.Run("Annotating an existing key succeeds", testAnnotatingKeySucceeds)
}

func testAnnotatingKeySucceeds(t *testing.T) {
	tcs := []struct {
		name     string
		metadata []wallet.Meta
	}{
		{
			name: "with metadata",
			metadata: []wallet.Meta{
				{Key: "name", Value: "my-wallet"},
				{Key: "role", Value: "validation"},
			},
		}, {
			name:     "without metadata",
			metadata: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			w := newWalletWithKey(t)
			kp := w.ListKeyPairs()[0]

			req := &wallet.AnnotateKeyRequest{
				Wallet:     w.Name(),
				PubKey:     kp.PublicKey(),
				Metadata:   tc.metadata,
				Passphrase: "passphrase",
			}

			// setup
			store := handlerMocks(tt)
			store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
			store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
			store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

			// when
			err := wallet.AnnotateKey(store, req)

			// then
			require.NoError(tt, err)
			assert.Equal(tt, req.Metadata, w.ListKeyPairs()[0].Meta())
		})
	}
}

func TestGenerateKey(t *testing.T) {
	t.Run("Generating keys in non-existing wallet fails", testGenerateKeyInNonExistingWalletFails)
	t.Run("Generating keys in existing wallet succeeds", testGenerateKeyInExistingWalletSucceeds)
}

func testGenerateKeyInNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.GenerateKeyRequest{
		Wallet: vgrand.RandomStr(5),
		Metadata: []wallet.Meta{
			{Key: "name", Value: "my-wallet"},
			{Key: "role", Value: "validation"},
		},
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWalletPath(req.Wallet).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	resp, err := wallet.GenerateKey(store, req)

	// then
	require.ErrorIs(t, err, wallet.ErrWalletDoesNotExists)
	require.Nil(t, resp)
}

func testGenerateKeyInExistingWalletSucceeds(t *testing.T) {
	// given
	w := newWallet(t)
	req := &wallet.GenerateKeyRequest{
		Wallet: w.Name(),
		Metadata: []wallet.Meta{
			{Key: "name", Value: "my-wallet"},
			{Key: "role", Value: "validation"},
		},
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(1).Return(nil)

	// when
	resp, err := wallet.GenerateKey(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	// verify updated wallet
	assert.Equal(t, req.Wallet, w.Name())
	require.Len(t, w.ListKeyPairs(), 1)
	keyPair := w.ListKeyPairs()[0]
	assert.Equal(t, req.Metadata, keyPair.Meta())
	// verify response
	assert.Equal(t, keyPair.PublicKey(), resp.PublicKey)
	assert.Equal(t, keyPair.AlgorithmName(), resp.Algorithm.Name)
	assert.Equal(t, keyPair.AlgorithmVersion(), resp.Algorithm.Version)
	assert.Equal(t, keyPair.Meta(), resp.Meta)
}

func TestTaintKey(t *testing.T) {
	t.Run("Tainting key succeeds", testTaintingKeySucceeds)
	t.Run("Tainting key of non-existing wallet fails", testTaintingKeyOfNonExistingWalletFails)
}

func testTaintingKeySucceeds(t *testing.T) {
	// given
	w := newWalletWithKey(t)
	kp := w.ListKeyPairs()[0]

	req := &wallet.TaintKeyRequest{
		Wallet:     w.Name(),
		PubKey:     kp.PublicKey(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err := wallet.TaintKey(store, req)

	// then
	require.NoError(t, err)
	assert.True(t, w.ListKeyPairs()[0].IsTainted())
}

func testTaintingKeyOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.TaintKeyRequest{
		Wallet:     vgrand.RandomStr(5),
		PubKey:     vgrand.RandomStr(25),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	err := wallet.TaintKey(store, req)

	// then
	require.Error(t, err)
}

func TestUntaintKey(t *testing.T) {
	t.Run("Untainting key succeeds", testUntaintingKeySucceeds)
	t.Run("Untainting key of non-existing wallet fails", testUntaintingKeyOfNonExistingWalletFails)
}

func testUntaintingKeySucceeds(t *testing.T) {
	// given
	w := newWalletWithKey(t)
	kp := w.ListKeyPairs()[0]
	err := w.TaintKey(kp.PublicKey())
	if err != nil {
		t.Fatalf("couldn't taint key: %v", err)
	}

	req := &wallet.UntaintKeyRequest{
		Wallet:     w.Name(),
		PubKey:     kp.PublicKey(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err = wallet.UntaintKey(store, req)

	// then
	require.NoError(t, err)
	assert.False(t, w.ListKeyPairs()[0].IsTainted())
}

func testUntaintingKeyOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.UntaintKeyRequest{
		Wallet:     vgrand.RandomStr(5),
		PubKey:     vgrand.RandomStr(25),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	err := wallet.UntaintKey(store, req)

	// then
	require.Error(t, err)
}

func TestIsolateKey(t *testing.T) {
	t.Run("Isolating key succeeds", testIsolatingKeySucceeds)
	t.Run("Isolating key of non-existing wallet fails", testIsolatingKeyOfNonExistingWalletFails)
}

func testIsolatingKeySucceeds(t *testing.T) {
	// given
	w := newWalletWithKey(t)
	kp := w.ListKeyPairs()[0]
	expectedResp := &wallet.IsolateKeyResponse{
		Wallet:   fmt.Sprintf("%s.%s.isolated", w.Name(), kp.PublicKey()[0:8]),
		FilePath: vgrand.RandomStr(10),
	}
	req := &wallet.IsolateKeyRequest{
		Wallet:     w.Name(),
		PubKey:     kp.PublicKey(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(1).Return(nil)
	store.EXPECT().GetWalletPath(gomock.Any()).Times(1).Return(expectedResp.FilePath)

	// when
	resp, err := wallet.IsolateKey(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedResp, resp)
}

func testIsolatingKeyOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.IsolateKeyRequest{
		Wallet:     vgrand.RandomStr(5),
		PubKey:     vgrand.RandomStr(25),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	resp, err := wallet.IsolateKey(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestListKeys(t *testing.T) {
	t.Run("List keys succeeds", testListKeysSucceeds)
	t.Run("List keys of non-existing wallet fails", testListKeysOfNonExistingWalletFails)
}

func testListKeysSucceeds(t *testing.T) {
	// given
	w := newWallet(t)
	keyCount := 3
	expectedKeys := &wallet.ListKeysResponse{
		Keys: make([]wallet.NamedPubKey, 0, keyCount),
	}
	for i := 0; i < keyCount; i++ {
		keyName := vgrand.RandomStr(5)
		kp, err := w.GenerateKeyPair([]wallet.Meta{{Key: "name", Value: keyName}})
		if err != nil {
			t.Fatalf("couldn't generate key: %v", err)
		}
		expectedKeys.Keys = append(expectedKeys.Keys, wallet.NamedPubKey{
			Name:      keyName,
			PublicKey: kp.PublicKey(),
		})
	}

	req := &wallet.ListKeysRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.ListKeys(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedKeys, resp)
}

func testListKeysOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.ListKeysRequest{
		Wallet:     vgrand.RandomStr(5),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.ListKeys(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestSignCommand(t *testing.T) {
	t.Run("Sign message succeeds", testSignCommandSucceeds)
	t.Run("Sign message of non-existing wallet fails", testSignCommandWithNonExistingWalletFails)
}

func testSignCommandSucceeds(t *testing.T) {
	// given
	w := importWalletWithKey(t)
	kp := w.ListKeyPairs()[0]

	req := &wallet.SignCommandRequest{
		Wallet: w.Name(),
		Request: &walletpb.SubmitTransactionRequest{
			PubKey:    kp.PublicKey(),
			Propagate: false,
			Command: &walletpb.SubmitTransactionRequest_VoteSubmission{
				VoteSubmission: &commandspb.VoteSubmission{
					ProposalId: vgrand.RandomStr(5),
					Value:      vega.Vote_VALUE_YES,
				},
			},
		},
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.SignCommand(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Base64Transaction)
}

func testSignCommandWithNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.SignCommandRequest{
		Wallet: vgrand.RandomStr(5),
		Request: &walletpb.SubmitTransactionRequest{
			PubKey:    vgrand.RandomStr(5),
			Propagate: false,
			Command: &walletpb.SubmitTransactionRequest_VoteSubmission{
				VoteSubmission: &commandspb.VoteSubmission{
					ProposalId: vgrand.RandomStr(5),
					Value:      vega.Vote_VALUE_YES,
				},
			},
		},
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.SignCommand(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestSignMessage(t *testing.T) {
	t.Run("Sign message succeeds", testSignMessageSucceeds)
	t.Run("Sign message of non-existing wallet fails", testSignMessageWithNonExistingWalletFails)
}

func testSignMessageSucceeds(t *testing.T) {
	// given
	w := importWalletWithKey(t)
	kp := w.ListKeyPairs()[0]

	expectedKeys := &wallet.SignMessageResponse{
		Base64: "StH82RHxjQ3yTeaSN25b6sJwAyLiq1CDvPWf0X4KIf/WTIjkunkWKn1Gq9ntCoGBfBZIyNfpPtGx0TSZsSrbCA==",
		Bytes:  []byte{0x4a, 0xd1, 0xfc, 0xd9, 0x11, 0xf1, 0x8d, 0xd, 0xf2, 0x4d, 0xe6, 0x92, 0x37, 0x6e, 0x5b, 0xea, 0xc2, 0x70, 0x3, 0x22, 0xe2, 0xab, 0x50, 0x83, 0xbc, 0xf5, 0x9f, 0xd1, 0x7e, 0xa, 0x21, 0xff, 0xd6, 0x4c, 0x88, 0xe4, 0xba, 0x79, 0x16, 0x2a, 0x7d, 0x46, 0xab, 0xd9, 0xed, 0xa, 0x81, 0x81, 0x7c, 0x16, 0x48, 0xc8, 0xd7, 0xe9, 0x3e, 0xd1, 0xb1, 0xd1, 0x34, 0x99, 0xb1, 0x2a, 0xdb, 0x8},
	}

	req := &wallet.SignMessageRequest{
		Wallet:     w.Name(),
		PubKey:     kp.PublicKey(),
		Message:    []byte("Je ne connaîtrai pas la peur car la peur tue l'esprit."),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.SignMessage(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedKeys, resp)
}

func testSignMessageWithNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.SignMessageRequest{
		Wallet:     vgrand.RandomStr(5),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.SignMessage(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestRotateKey(t *testing.T) {
	t.Run("Rotate key succeeds", testRotateKeySucceeds)
	t.Run("Rotate key with non existing wallet fails", testRotateWithNonExistingWalletFails)
	t.Run("Rotate key with non existing new public key fails", testRotateKeyWithNonExistingNewPublicKeyFails)
	t.Run("Rotate key with non existing current public key fails", testRotateKeyWithNonExistingCurrentPublicKeyFails)
	t.Run("Rotate key tainted public key fails", testRotateKeyWithTaintedPublicKeyFails)
}

func testRotateKeySucceeds(t *testing.T) {
	// given
	w := importWalletWithTwoKeys(t)
	chainID := vgrand.RandomStr(5)

	currentPubKey := w.ListPublicKeys()[0]
	newPubKey := w.ListPublicKeys()[1]

	masterKeyPair, err := w.GetMasterKeyPair()
	require.NoError(t, err)

	req := &wallet.RotateKeyRequest{
		Wallet:            w.Name(),
		Passphrase:        "passphrase",
		ChainID:           chainID,
		NewPublicKey:      newPubKey.Key(),
		CurrentPublicKey:  currentPubKey.Key(),
		TxBlockHeight:     20,
		TargetBlockHeight: 25,
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.RotateKey(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, masterKeyPair.PublicKey(), resp.MasterPublicKey)

	transactionRaw, err := base64.StdEncoding.DecodeString(resp.Base64Transaction)
	require.NoError(t, err)

	transaction := &commandspb.Transaction{}
	err = proto.Unmarshal(transactionRaw, transaction)
	require.NoError(t, err)

	inputData, err := commands.UnmarshalInputData(transaction.Version, transaction.InputData, chainID)
	require.NoError(t, err)

	keyRotate, ok := inputData.Command.(*commandspb.InputData_KeyRotateSubmission)
	require.True(t, ok)
	require.NotNil(t, keyRotate)

	require.Equal(t, req.TxBlockHeight, inputData.BlockHeight)
	require.Equal(t, newPubKey.Index(), keyRotate.KeyRotateSubmission.NewPubKeyIndex)
	require.Equal(t, req.TargetBlockHeight, keyRotate.KeyRotateSubmission.TargetBlock)
	require.Equal(t, req.NewPublicKey, keyRotate.KeyRotateSubmission.NewPubKey)
}

func testRotateWithNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.RotateKeyRequest{
		Wallet:            vgrand.RandomStr(5),
		Passphrase:        "passphrase",
		NewPublicKey:      "nonexisting",
		TxBlockHeight:     20,
		TargetBlockHeight: 25,
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.RotateKey(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func testRotateKeyWithNonExistingNewPublicKeyFails(t *testing.T) {
	// given
	w := importWalletWithKey(t)

	req := &wallet.RotateKeyRequest{
		Wallet:            w.Name(),
		Passphrase:        "passphrase",
		NewPublicKey:      "nonexisting",
		TxBlockHeight:     20,
		TargetBlockHeight: 25,
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.RotateKey(store, req)

	// then
	require.Nil(t, resp)
	require.Error(t, err)
}

func testRotateKeyWithNonExistingCurrentPublicKeyFails(t *testing.T) {
	// given
	w := importWalletWithKey(t)

	newPubKey := w.ListPublicKeys()[0]

	req := &wallet.RotateKeyRequest{
		Wallet:            w.Name(),
		Passphrase:        "passphrase",
		NewPublicKey:      newPubKey.Key(),
		CurrentPublicKey:  "non-existing",
		TxBlockHeight:     20,
		TargetBlockHeight: 25,
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.RotateKey(store, req)

	// then
	require.Nil(t, resp)
	require.Error(t, err)
}

func testRotateKeyWithTaintedPublicKeyFails(t *testing.T) {
	// given
	w := importWalletWithTwoKeys(t)

	currentPubKey := w.ListPublicKeys()[0]
	newPubKey := w.ListPublicKeys()[1]

	err := w.TaintKey(newPubKey.Key())
	require.NoError(t, err)

	req := &wallet.RotateKeyRequest{
		Wallet:            w.Name(),
		Passphrase:        "passphrase",
		NewPublicKey:      newPubKey.Key(),
		CurrentPublicKey:  currentPubKey.Key(),
		TxBlockHeight:     20,
		TargetBlockHeight: 25,
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.RotateKey(store, req)

	// then
	require.Nil(t, resp)
	require.ErrorIs(t, err, wallet.ErrPubKeyIsTainted)
}

func TestListPermissions(t *testing.T) {
	t.Run("List permissions succeeds", testListPermissionsSucceeds)
	t.Run("List permissions of non-existing wallet fails", testListPermissionsOfNonExistingWalletFails)
}

func testListPermissionsSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	// when
	_, err := w.GenerateKeyPair(nil)

	// then
	require.NoError(t, err)

	// when
	err = w.UpdatePermissions("vega.xyz", wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	})

	// then
	require.NoError(t, err)

	// when
	err = w.UpdatePermissions("token.vega.xyz", wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	})

	// then
	require.NoError(t, err)

	// given
	req := &wallet.ListPermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.ListPermissions(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &wallet.ListPermissionsResponse{
		Hostnames: []string{"token.vega.xyz", "vega.xyz"},
	}, resp)
}

func testListPermissionsOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.ListPermissionsRequest{
		Wallet:     vgrand.RandomStr(3),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.ListPermissions(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestDescribePermissions(t *testing.T) {
	t.Run("Describe permissions succeeds", testDescribePermissionsSucceeds)
	t.Run("Describe permissions of non-existing wallet fails", testDescribePermissionsOfNonExistingWalletFails)
	t.Run("Describe permissions for unknown hostname succeeds", testDescribePermissionsForUnknownHostnameSucceeds)
}

func testDescribePermissionsSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	// when
	_, err := w.GenerateKeyPair(nil)

	// then
	require.NoError(t, err)

	// when
	vegaPerms := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	}
	err = w.UpdatePermissions("vega.xyz", vegaPerms)

	// then
	require.NoError(t, err)

	// when
	err = w.UpdatePermissions("token.vega.xyz", wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	})

	// then
	require.NoError(t, err)

	// given
	req := &wallet.DescribePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
		Hostname:   "vega.xyz",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.DescribePermissions(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &wallet.DescribePermissionsResponse{
		Permissions: vegaPerms,
	}, resp)
}

func testDescribePermissionsOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.DescribePermissionsRequest{
		Wallet:     vgrand.RandomStr(3),
		Passphrase: "passphrase",
		Hostname:   vgrand.RandomStr(5),
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)

	// when
	resp, err := wallet.DescribePermissions(store, req)

	// then
	require.Error(t, err)
	assert.Nil(t, resp)
}

func testDescribePermissionsForUnknownHostnameSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	req := &wallet.DescribePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
		Hostname:   vgrand.RandomStr(5),
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)

	// when
	resp, err := wallet.DescribePermissions(store, req)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, wallet.DefaultPermissions(), resp.Permissions)
}

func TestRevokePermissions(t *testing.T) {
	t.Run("Revoke permissions succeeds", testRevokePermissionsSucceeds)
	t.Run("Revoke permissions of non-existing wallet fails", testRevokePermissionsOfNonExistingWalletFails)
	t.Run("Revoke permissions for unknown hostname succeeds", testRevokePermissionsForUnknownHostnameSucceeds)
}

func testRevokePermissionsSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	// when
	_, err := w.GenerateKeyPair(nil)

	// then
	require.NoError(t, err)

	// when
	vegaPerms := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	}
	err = w.UpdatePermissions("vega.xyz", vegaPerms)

	// then
	require.NoError(t, err)

	// given
	tokenPerms := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	}

	// when
	err = w.UpdatePermissions("token.vega.xyz", tokenPerms)

	// then
	require.NoError(t, err)

	// given
	req := &wallet.RevokePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
		Hostname:   "vega.xyz",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err = wallet.RevokePermissions(store, req)

	// then
	require.NoError(t, err)
	assert.Equal(t, wallet.DefaultPermissions(), w.Permissions("vega.xyz"))
	assert.Equal(t, tokenPerms, w.Permissions("token.vega.xyz"))
}

func testRevokePermissionsOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.RevokePermissionsRequest{
		Wallet:     vgrand.RandomStr(3),
		Passphrase: "passphrase",
		Hostname:   vgrand.RandomStr(5),
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	err := wallet.RevokePermissions(store, req)

	// then
	require.Error(t, err)
}

func testRevokePermissionsForUnknownHostnameSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	req := &wallet.RevokePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
		Hostname:   vgrand.RandomStr(5),
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err := wallet.RevokePermissions(store, req)

	// then
	require.NoError(t, err)
}

func TestPurgePermissions(t *testing.T) {
	t.Run("Purge permissions succeeds", testPurgePermissionsSucceeds)
	t.Run("Purge permissions of non-existing wallet fails", testPurgePermissionsOfNonExistingWalletFails)
	t.Run("Purge permissions without existing permissions succeeds", testPurgePermissionsWithExistingPermissionsSucceeds)
}

func testPurgePermissionsSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	// when
	_, err := w.GenerateKeyPair(nil)

	// then
	require.NoError(t, err)

	// when
	vegaPerms := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	}
	err = w.UpdatePermissions("vega.xyz", vegaPerms)

	// then
	require.NoError(t, err)

	// when
	err = w.UpdatePermissions("token.vega.xyz", wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: wallet.ReadAccess,
		},
	})

	// then
	require.NoError(t, err)

	// given
	req := &wallet.PurgePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err = wallet.PurgePermissions(store, req)

	// then
	require.NoError(t, err)
	assert.Equal(t, wallet.DefaultPermissions(), w.Permissions("vega.xyz"))
	assert.Equal(t, wallet.DefaultPermissions(), w.Permissions("token.vega.xyz"))
}

func testPurgePermissionsOfNonExistingWalletFails(t *testing.T) {
	// given
	req := &wallet.PurgePermissionsRequest{
		Wallet:     vgrand.RandomStr(3),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(false, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(0)
	store.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), req.Passphrase).Times(0)

	// when
	err := wallet.PurgePermissions(store, req)

	// then
	require.Error(t, err)
}

func testPurgePermissionsWithExistingPermissionsSucceeds(t *testing.T) {
	// given
	w := newWallet(t)

	req := &wallet.PurgePermissionsRequest{
		Wallet:     w.Name(),
		Passphrase: "passphrase",
	}

	// setup
	store := handlerMocks(t)
	store.EXPECT().WalletExists(gomock.Any(), req.Wallet).Times(1).Return(true, nil)
	store.EXPECT().GetWallet(gomock.Any(), req.Wallet, req.Passphrase).Times(1).Return(w, nil)
	store.EXPECT().SaveWallet(gomock.Any(), w, req.Passphrase).Times(1).Return(nil)

	// when
	err := wallet.PurgePermissions(store, req)

	// then
	require.NoError(t, err)
}

func newWalletWithKey(t *testing.T) *wallet.HDWallet {
	t.Helper()
	return newWalletWithKeys(t, 1)
}

func newWalletWithKeys(t *testing.T, n int) *wallet.HDWallet {
	t.Helper()
	w := newWallet(t)

	for i := 0; i < n; i++ {
		if _, err := w.GenerateKeyPair(nil); err != nil {
			t.Fatalf("couldn't generate key: %v", err)
		}
	}
	return w
}

func importWalletWithTwoKeys(t *testing.T) *wallet.HDWallet {
	t.Helper()
	w := importWalletWithKey(t)
	if _, err := w.GenerateKeyPair(nil); err != nil {
		t.Fatalf("couldn't generate second key: %v", err)
	}

	return w
}

func importWalletWithKey(t *testing.T) *wallet.HDWallet {
	t.Helper()
	w, err := wallet.ImportHDWallet(
		vgrand.RandomStr(5),
		"swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
		2,
	)
	if err != nil {
		t.Fatalf("couldn't import wallet: %v", err)
	}

	if _, err := w.GenerateKeyPair(nil); err != nil {
		t.Fatalf("couldn't generate key: %v", err)
	}

	return w
}

func newWallet(t *testing.T) *wallet.HDWallet {
	t.Helper()
	w, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatalf("couldn't create HD wallet: %v", err)
	}
	return w
}

func handlerMocks(t *testing.T) *mocks.MockStore {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStore(ctrl)
	return store
}
