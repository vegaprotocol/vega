package wallet_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallet/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
