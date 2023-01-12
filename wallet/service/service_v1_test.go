package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"code.vegaprotocol.io/vega/wallet/service/v1"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testRecoveryPhrase = "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render"

	testRequestTimeout = 10 * time.Second
)

func TestServiceV1(t *testing.T) {
	t.Run("create wallet ok", testServiceCreateWalletOK)
	t.Run("create wallet fail invalid request", testServiceCreateWalletFailInvalidRequest)
	t.Run("Importing a wallet succeeds", testServiceImportWalletOK)
	t.Run("Importing a wallet with and invalid request fails", testServiceImportWalletFailInvalidRequest)
	t.Run("login wallet ok", testServiceLoginWalletOK)
	t.Run("login wallet fail invalid request", testServiceLoginWalletFailInvalidRequest)
	t.Run("revoke token ok", testServiceRevokeTokenOK)
	t.Run("revoke token fail invalid request", testServiceRevokeTokenFailInvalidRequest)
	t.Run("gen keypair ok", testServiceGenKeypairOK)
	t.Run("gen keypair fail invalid request", testServiceGenKeypairFailInvalidRequest)
	t.Run("list keypair ok", testServiceListPublicKeysOK)
	t.Run("list keypair fail invalid request", testServiceListPublicKeysFailInvalidRequest)
	t.Run("get keypair ok", testServiceGetPublicKeyOK)
	t.Run("get keypair fail invalid request", testServiceGetPublicKeyFailInvalidRequest)
	t.Run("get keypair fail key not found", testServiceGetPublicKeyFailKeyNotFound)
	t.Run("get keypair fail misc error", testServiceGetPublicKeyFailMiscError)
	t.Run("taint ok", testServiceTaintOK)
	t.Run("taint fail invalid request", testServiceTaintFailInvalidRequest)
	t.Run("update metadata", testServiceUpdateMetaOK)
	t.Run("update metadata invalid request", testServiceUpdateMetaFailInvalidRequest)
	t.Run("Signing transaction succeeds", testAcceptSigningTransactionSucceeds)
	t.Run("Checking transaction succeeds", testCheckTransactionSucceeds)
	t.Run("Checking transaction with rejected transaction succeeds", testCheckTransactionWithRejectedTransactionSucceeds)
	t.Run("Checking transaction with failed transaction fails", testCheckTransactionWithFailedTransactionFails)
	t.Run("Decline signing transaction manually succeeds", testDeclineSigningTransactionManuallySucceeds)
	t.Run("Signing transaction fails spam", testAcceptSigningTransactionFailsSpam)
	t.Run("Failed signing of transaction fails", testFailedTransactionSigningFails)
	t.Run("Signing transaction with invalid request fails", testSigningTransactionWithInvalidRequestFails)
	t.Run("Signing anything succeeds", testSigningAnythingSucceeds)
	t.Run("Signing anything with invalid request fails", testSigningAnyDataWithInvalidRequestFails)
	t.Run("Verifying anything succeeds", testVerifyingAnythingSucceeds)
	t.Run("Failed verification fails", testVerifyingAnythingFails)
	t.Run("Verifying anything with invalid request fails", testVerifyingAnyDataWithInvalidRequestFails)
	t.Run("Requesting the chain id is successful", testGetNetworkChainIDSuccess)
	t.Run("Requesting the chain id fails when node in available", testGetNetworkChainIDFailure)
	t.Run("Empty chain id from network fails", TestEmptyChainIDFromNetworkFails)
}

func testServiceCreateWalletOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"wallet": "%s", "passphrase": "%s"}`, walletName, passphrase)

	// setup
	s.handler.EXPECT().CreateWallet(walletName, passphrase).Times(1).Return(testRecoveryPhrase, nil)
	s.auth.EXPECT().NewSession(walletName).Times(1).Return("this is a token", nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, createWalletRequest(t, payload))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceCreateWalletFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		payload string
	}{
		{
			name:    "misspelled wallet property",
			payload: `{"wall": "jeremy", "passphrase": "oh yea?"}`,
		}, {
			name:    "misspelled passphrase property",
			payload: `{"wallet": "jeremy", "passrase": "oh yea?"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, createWalletRequest(tt, tc.payload))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceImportWalletOK(t *testing.T) {
	tcs := []struct {
		name    string
		version uint32
	}{
		{
			name:    "version 1",
			version: 1,
		}, {
			name:    "version 2",
			version: 2,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// given
			walletName := vgrand.RandomStr(5)
			passphrase := vgrand.RandomStr(5)
			payload := fmt.Sprintf(`{"wallet": "%s", "passphrase": "%s", "recoveryPhrase": "%s", "version": %d}`, walletName, passphrase, testRecoveryPhrase, tc.version)

			// setup
			s.handler.EXPECT().ImportWallet(walletName, passphrase, testRecoveryPhrase, tc.version).Times(1).Return(nil)
			s.auth.EXPECT().NewSession(walletName).Times(1).Return("this is a token", nil)

			// when
			statusCode, _, _ := s.serveHTTP(tt, importWalletRequest(tt, payload))

			// then
			assert.Equal(tt, http.StatusOK, statusCode)
		})
	}
}

func testServiceImportWalletFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		payload string
	}{
		{
			name:    "misspelled wallet property",
			payload: fmt.Sprintf(`{"wall": "jeremy", "passphrase": "oh yea?", "recoveryPhrase": %q}`, testRecoveryPhrase),
		}, {
			name:    "misspelled passphrase property",
			payload: fmt.Sprintf(`{"wallet": "jeremy", "password": "oh yea?", "recoveryPhrase": %q}`, testRecoveryPhrase),
		}, {
			name:    "misspelled recovery phrase property",
			payload: fmt.Sprintf(`{"wallet": "jeremy", "passphrase": "oh yea?", "little_words": %q}`, testRecoveryPhrase),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, importWalletRequest(tt, tc.payload))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceLoginWalletOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"wallet": "%s", "passphrase": "%s"}`, walletName, passphrase)

	// setup
	s.handler.EXPECT().LoginWallet(walletName, passphrase).Times(1).Return(nil)
	s.auth.EXPECT().NewSession(walletName).Times(1).Return("this is a token", nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, loginRequest(t, payload))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceLoginWalletFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		payload string
	}{
		{
			name:    "misspelled wallet property",
			payload: `{"wall": "jeremy", "passphrase": "oh yea?"}`,
		}, {
			name:    "misspelled passphrase property",
			payload: `{"wallet": "jeremy", "passrase": "oh yea?"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")
			t.Cleanup(func() {
				s.ctrl.Finish()
			})

			// when
			statusCode, _, _ := s.serveHTTP(tt, loginRequest(tt, tc.payload))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceRevokeTokenOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)

	// setup
	s.auth.EXPECT().Revoke(token).Times(1).Return(walletName, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, logoutRequest(t, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceRevokeTokenFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(t, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, logoutRequest(t, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceGenKeypairOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	ed25519 := crypto.NewEd25519()
	key := &wallet.HDPublicKey{
		PublicKey: vgrand.RandomStr(5),
		Algorithm: wallet.Algorithm{
			Name:    ed25519.Name(),
			Version: ed25519.Version(),
		},
		Tainted:      false,
		MetadataList: nil,
	}
	walletName := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	payload := fmt.Sprintf(`{"passphrase": "%s"}`, passphrase)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SecureGenerateKeyPair(walletName, passphrase, gomock.Len(0)).Times(1).Return(key.PublicKey, nil)
	s.handler.EXPECT().GetPublicKey(walletName, key.PublicKey).Times(1).Return(key, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, generateKeyRequest(t, payload, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceGenKeypairFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
		payload string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
			payload: `{"passphrase": "oh yea?"}`,
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
			payload: `{"passphrase": "oh yea?"}`,
		}, {
			name:    "invalid request",
			headers: authHeadersV1(t, vgrand.RandomStr(5)),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, generateKeyRequest(t, tc.payload, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceListPublicKeysOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().ListPublicKeys(walletName).Times(1).Return([]wallet.PublicKey{}, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, listKeysRequest(t, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceListPublicKeysFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, listKeysRequest(t, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceGetPublicKeyOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	hdPubKey := &wallet.HDPublicKey{
		Idx:       1,
		PublicKey: vgrand.RandomStr(5),
		Algorithm: wallet.Algorithm{
			Name:    "some/algo",
			Version: 1,
		},
		Tainted:      false,
		MetadataList: []wallet.Metadata{{Key: "a", Value: "b"}},
	}
	headers := authHeadersV1(t, token)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().GetPublicKey(walletName, hdPubKey.PublicKey).Times(1).Return(hdPubKey, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, getKeyRequest(t, hdPubKey.PublicKey, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceGetPublicKeyFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(t, getKeyRequest(t, vgrand.RandomStr(5), tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceGetPublicKeyFailKeyNotFound(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	pubKey := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().GetPublicKey(walletName, pubKey).Times(1).Return(nil, wallet.ErrPubKeyDoesNotExist)

	// when
	statusCode, _, _ := s.serveHTTP(t, getKeyRequest(t, pubKey, headers))

	// then
	assert.Equal(t, http.StatusNotFound, statusCode)
}

func testServiceGetPublicKeyFailMiscError(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	pubKey := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().GetPublicKey(walletName, pubKey).Times(1).Return(nil, assert.AnError)

	// when
	statusCode, _, _ := s.serveHTTP(t, getKeyRequest(t, pubKey, headers))

	// then
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

func testServiceTaintOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	pubKey := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	payload := fmt.Sprintf(`{"passphrase": "%s"}`, passphrase)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().TaintKey(walletName, pubKey, passphrase).Times(1).Return(nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, taintKeyRequest(t, pubKey, payload, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceTaintFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
		payload string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
			payload: `{"passphrase": "some data"}`,
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
			payload: `{"passphrase": "some data"}`,
		}, {
			name:    "misspelled passphrase property",
			headers: authHeadersV1(t, vgrand.RandomStr(5)),
			payload: `{"passhp": "some data"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, taintKeyRequest(tt, vgrand.RandomStr(5), tc.payload, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testServiceUpdateMetaOK(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// when
	walletName := vgrand.RandomStr(5)
	pubKey := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)
	metaRole := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	payload := fmt.Sprintf(`{"passphrase": "%s", "meta": [{"key":"role", "value":"%s"}]}`, passphrase, metaRole)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().UpdateMeta(walletName, pubKey, passphrase, []wallet.Metadata{{
		Key:   "role",
		Value: metaRole,
	}}).Times(1).Return(nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, annotateKeyRequest(t, pubKey, payload, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testServiceUpdateMetaFailInvalidRequest(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
		payload string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
			payload: `{"passphrase": "some data", "meta": [{"key": "role", "value": "signing"}]}`,
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
			payload: `{"passphrase": "some data", "meta": [{"key": "role", "value": "signing"}]}`,
		}, {
			name:    "misspelled passphrase property",
			headers: authHeadersV1(t, vgrand.RandomStr(5)),
			payload: `{"pssphrse": "some data", "meta": [{"key": "role", "value": "signing"}]}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, annotateKeyRequest(tt, vgrand.RandomStr(5), tc.payload, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testCheckTransactionSucceeds(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), chainID).Times(1).Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().CheckTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&api.CheckTransactionResponse{
		Success:   true,
		Code:      0,
		GasWanted: 300,
		GasUsed:   200,
	}, nil)
	s.pow.EXPECT().Generate(pubKey, &nodetypes.LastBlock{
		ChainID:                         blockHeightResponse.ChainId,
		BlockHeight:                     blockHeightResponse.Height,
		BlockHash:                       blockHeightResponse.Hash,
		ProofOfWorkHashFunction:         blockHeightResponse.SpamPowHashFunction,
		ProofOfWorkDifficulty:           blockHeightResponse.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           blockHeightResponse.SpamPowNumberOfPastBlocks,
		ProofOfWorkTxPerBlock:           blockHeightResponse.SpamPowNumberOfTxPerBlock,
		ProofOfWorkIncreasingDifficulty: blockHeightResponse.SpamPowIncreasingDifficulty,
	}).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   "",
		Nonce: 10,
	}, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)
	// when

	statusCode, _, body := s.serveHTTP(t, checkTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusOK, statusCode)

	resp := &api.CheckTransactionResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		t.Fatalf("couldn't unmarshal responde: %v", err)
	}
	assert.True(t, resp.Success)
	assert.Equal(t, uint32(0), resp.Code)
	assert.Equal(t, int64(300), resp.GasWanted)
}

func testCheckTransactionWithRejectedTransactionSucceeds(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// given
	walletName := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), chainID).Times(1).Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().CheckTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&api.CheckTransactionResponse{
		Success: false,
		Code:    4,
	}, nil)
	s.pow.EXPECT().Generate(pubKey, &nodetypes.LastBlock{
		ChainID:                         blockHeightResponse.ChainId,
		BlockHeight:                     blockHeightResponse.Height,
		BlockHash:                       blockHeightResponse.Hash,
		ProofOfWorkHashFunction:         blockHeightResponse.SpamPowHashFunction,
		ProofOfWorkDifficulty:           blockHeightResponse.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           blockHeightResponse.SpamPowNumberOfPastBlocks,
		ProofOfWorkTxPerBlock:           blockHeightResponse.SpamPowNumberOfTxPerBlock,
		ProofOfWorkIncreasingDifficulty: blockHeightResponse.SpamPowIncreasingDifficulty,
	}).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   "",
		Nonce: 10,
	}, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)
	// when

	statusCode, _, body := s.serveHTTP(t, checkTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusOK, statusCode)

	resp := &api.CheckTransactionResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		t.Fatalf("couldn't unmarshal responde: %v", err)
	}
	assert.False(t, resp.Success)
	assert.Equal(t, uint32(4), resp.Code)
	assert.Equal(t, int64(0), resp.GasWanted)
}

func testCheckTransactionWithFailedTransactionFails(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), chainID).Times(1).Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().CheckTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, assert.AnError)
	s.pow.EXPECT().Generate(pubKey, &nodetypes.LastBlock{
		ChainID:                         blockHeightResponse.ChainId,
		BlockHeight:                     blockHeightResponse.Height,
		BlockHash:                       blockHeightResponse.Hash,
		ProofOfWorkHashFunction:         blockHeightResponse.SpamPowHashFunction,
		ProofOfWorkDifficulty:           blockHeightResponse.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           blockHeightResponse.SpamPowNumberOfPastBlocks,
		ProofOfWorkTxPerBlock:           blockHeightResponse.SpamPowNumberOfTxPerBlock,
		ProofOfWorkIncreasingDifficulty: blockHeightResponse.SpamPowIncreasingDifficulty,
	}).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   "",
		Nonce: 10,
	}, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, checkTxRequest(t, payload, headers))

	// then
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

func testAcceptSigningTransactionSucceeds(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), chainID).Times(1).Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().SendTx(gomock.Any(), gomock.Any(), api.SubmitTransactionRequest_TYPE_ASYNC, gomock.Any()).Times(1).
		Return(&api.SubmitTransactionResponse{Success: true}, nil)
	s.pow.EXPECT().Generate(pubKey, &nodetypes.LastBlock{
		ChainID:                         blockHeightResponse.ChainId,
		BlockHeight:                     blockHeightResponse.Height,
		BlockHash:                       blockHeightResponse.Hash,
		ProofOfWorkHashFunction:         blockHeightResponse.SpamPowHashFunction,
		ProofOfWorkDifficulty:           blockHeightResponse.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           blockHeightResponse.SpamPowNumberOfPastBlocks,
		ProofOfWorkTxPerBlock:           blockHeightResponse.SpamPowNumberOfTxPerBlock,
		ProofOfWorkIncreasingDifficulty: blockHeightResponse.SpamPowIncreasingDifficulty,
	}).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   "",
		Nonce: 10,
	}, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)
	// when

	statusCode, _, _ := s.serveHTTP(t, signTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusOK, statusCode)
}

func testDeclineSigningTransactionManuallySucceeds(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// given
	token := vgrand.RandomStr(5)
	walletName := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := "toBeDeclined"
	payload := fmt.Sprintf(`{"propagate": true, "pubKey": "%s", "orderCancellation": {}}`, pubKey)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-s.consentRequestsCh:
				req.Confirmation <- v1.ConsentConfirmation{
					TxID:     req.TxID,
					Decision: false,
				}
				return
			}
		}
	}()

	// when
	statusCode, _, _ := s.serveHTTP(t, signTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusUnauthorized, statusCode)
}

func testFailedTransactionSigningFails(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"propagate": true, "pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignTx(walletName, gomock.Any(), gomock.Any(), chainID).Times(1).Return(nil, assert.AnError)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, signTxRequest(t, payload, headers))

	// then
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

func testSigningTransactionWithInvalidRequestFails(t *testing.T) {
	token := vgrand.RandomStr(5)

	tcs := []struct {
		name    string
		headers map[string]string
		payload string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
			payload: `{"propagate": true, "pubKey": "0xCAFEDUDE", "orderCancellation": {}}`,
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
			payload: `{"propagate": true, "pubKey": "0xCAFEDUDE", "orderCancellation": {}}`,
		}, {
			name:    "misspelled pubKey property",
			headers: authHeadersV1(t, token),
			payload: `{"propagate": true, "puey": "0xCAFEDUDE", "orderCancellation": {}}`,
		}, {
			name:    "without command",
			headers: authHeadersV1(t, token),
			payload: `{"propagate": true, "pubKey": "0xCAFEDUDE", "robMoney": {}}`,
		}, {
			name:    "with unknown command",
			headers: authHeadersV1(t, token),
			payload: `{"propagate": true, "pubKey": "0xCAFEDUDE"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")
			if tc.name != "no header" && tc.name != "no token" {
				s.auth.EXPECT().VerifyToken(token).Times(1)
			}
			// when
			statusCode, _, _ := s.serveHTTP(tt, signTxRequest(tt, tc.payload, tc.headers))
			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testAcceptSigningTransactionFailsSpam(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	chainID := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)
	blockHeightResponse := &api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     chainID,
	}

	// setup
	s.auth.EXPECT().VerifyToken(token).AnyTimes().Return(walletName, nil)
	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), chainID).AnyTimes().Return(&commandspb.Transaction{}, nil)
	s.pow.EXPECT().Generate(pubKey, &nodetypes.LastBlock{
		ChainID:                         blockHeightResponse.ChainId,
		BlockHeight:                     blockHeightResponse.Height,
		BlockHash:                       blockHeightResponse.Hash,
		ProofOfWorkHashFunction:         blockHeightResponse.SpamPowHashFunction,
		ProofOfWorkDifficulty:           blockHeightResponse.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           blockHeightResponse.SpamPowNumberOfPastBlocks,
		ProofOfWorkTxPerBlock:           blockHeightResponse.SpamPowNumberOfTxPerBlock,
		ProofOfWorkIncreasingDifficulty: blockHeightResponse.SpamPowIncreasingDifficulty,
	}).Times(1).Return(&commandspb.ProofOfWork{
		Tid:   "",
		Nonce: 10,
	}, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(blockHeightResponse, 0, nil)
	// when

	s.nodeForward.EXPECT().SendTx(gomock.Any(), gomock.Any(), api.SubmitTransactionRequest_TYPE_ASYNC, gomock.Any()).Times(1).
		Return(&api.SubmitTransactionResponse{Success: false, Code: 89}, nil)

	statusCode, _, _ := s.serveHTTP(t, signTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusTooManyRequests, statusCode)
}

func testSigningAnythingSucceeds(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	walletName := vgrand.RandomStr(5)
	pubKey := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	payload := fmt.Sprintf(`{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "%s"}`, pubKey)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.handler.EXPECT().SignAny(walletName, []byte("spice of dune"), pubKey).Times(1).Return([]byte("some sig"), nil)

	// when
	statusCode, _, _ := s.serveHTTP(t, signAnyRequest(t, payload, headers))

	// then
	assert.Equal(t, http.StatusOK, statusCode)
}

func testSigningAnyDataWithInvalidRequestFails(t *testing.T) {
	tcs := []struct {
		name    string
		headers map[string]string
		payload string
	}{
		{
			name:    "no header",
			headers: map[string]string{},
			payload: `{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "asdasasdasd"}`,
		}, {
			name:    "no token",
			headers: authHeadersV1(t, ""),
			payload: `{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "asdasasdasd"}`,
		}, {
			name:    "misspelled pubKey property",
			headers: authHeadersV1(t, vgrand.RandomStr(5)),
			payload: `{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "puey": "asdasasdasd"}`,
		}, {
			name:    "misspelled inputData property",
			headers: authHeadersV1(t, vgrand.RandomStr(5)),
			payload: `{"data": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "asdasasdasd"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, signAnyRequest(tt, tc.payload, tc.headers))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testVerifyingAnythingSucceeds(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "%s", "signature": "U2lldGNoIFRhYnI="}`, pubKey)

	// setup
	s.handler.EXPECT().VerifyAny([]byte("spice of dune"), []byte("Sietch Tabr"), pubKey).Times(1).Return(true, nil)

	// when
	statusCode, _, body := s.serveHTTP(t, verifyAnyRequest(t, payload))

	// then
	assert.Equal(t, http.StatusOK, statusCode)

	resp := &v1.VerifyAnyResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		t.Fatalf("couldn't unmarshal responde: %v", err)
	}
	assert.True(t, resp.Valid)
}

func testVerifyingAnythingFails(t *testing.T) {
	s := getTestServiceV1(t, "automatic")

	// given
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "%s", "signature": "U2lldGNoIFRhYnI="}`, pubKey)

	// setup
	s.handler.EXPECT().VerifyAny([]byte("spice of dune"), []byte("Sietch Tabr"), pubKey).Times(1).Return(false, nil)

	// when
	statusCode, _, body := s.serveHTTP(t, verifyAnyRequest(t, payload))

	// then
	assert.Equal(t, http.StatusOK, statusCode)

	resp := &v1.VerifyAnyResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		t.Fatalf("couldn't unmarshal responde: %v", err)
	}
	assert.False(t, resp.Valid)
}

func testVerifyingAnyDataWithInvalidRequestFails(t *testing.T) {
	tcs := []struct {
		name    string
		payload string
	}{
		{
			name:    "misspelled pubKey property",
			payload: `{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "puey": "asdasasdasd", "signature": "U2lldGNoIFRhYnI="}`,
		}, {
			name:    "misspelled inputData property",
			payload: `{"data": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "asdasasdasd", "signature": "U2lldGNoIFRhYnI="}`,
		}, {
			name:    "misspelled signature property",
			payload: `{"inputData": "c3BpY2Ugb2YgZHVuZQ==", "pubKey": "asdasasdasd", "sign": "U2lldGNoIFRhYnI="}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			s := getTestServiceV1(tt, "automatic")

			// when
			statusCode, _, _ := s.serveHTTP(tt, verifyAnyRequest(tt, tc.payload))

			// then
			assert.Equal(tt, http.StatusBadRequest, statusCode)
		})
	}
}

func testGetNetworkChainIDSuccess(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// setup
	expectedChainID := "some-chain-id"
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).AnyTimes().Return(&api.LastBlockHeightResponse{
		ChainId: expectedChainID,
	}, 0, nil)

	// when
	statusCode, _, body := s.serveHTTP(t, chainIDRequest(t))
	assert.Equal(t, http.StatusOK, statusCode)

	resp := struct {
		ChainID string `json:"chainID"`
	}{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("couldn't unmarshal responde: %v", err)
	}
	assert.Equal(t, resp.ChainID, expectedChainID)
}

func testGetNetworkChainIDFailure(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// setup
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).AnyTimes().Return(nil, 0, assert.AnError)

	// when
	statusCode, _, _ := s.serveHTTP(t, chainIDRequest(t))
	assert.Equal(t, http.StatusFailedDependency, statusCode)
}

func TestEmptyChainIDFromNetworkFails(t *testing.T) {
	s := getTestServiceV1(t, "manual")

	// given
	walletName := vgrand.RandomStr(5)
	token := vgrand.RandomStr(5)
	headers := authHeadersV1(t, token)
	pubKey := vgrand.RandomStr(5)
	payload := fmt.Sprintf(`{"pubKey": "%s", "orderCancellation": {}}`, pubKey)

	// setup
	s.auth.EXPECT().VerifyToken(token).Times(1).Return(walletName, nil)
	s.nodeForward.EXPECT().LastBlockHeightAndHash(gomock.Any()).Times(1).Return(&api.LastBlockHeightResponse{
		Height:                      42,
		Hash:                        "0292041e2f0cf741894503fb3ead4cb817bca2375e543aa70f7c4d938157b5a6",
		SpamPowHashFunction:         "sha3_24_rounds",
		SpamPowDifficulty:           2,
		SpamPowNumberOfPastBlocks:   2,
		SpamPowNumberOfTxPerBlock:   2,
		SpamPowIncreasingDifficulty: false,
		ChainId:                     "",
	}, 0, nil)
	// when

	statusCode, _, _ := s.serveHTTP(t, checkTxRequest(t, payload, headers))
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

func loginRequest(t *testing.T, payload string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/auth/token", payload, nil)
}

func logoutRequest(t *testing.T, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodDelete, "/api/v1/auth/token", "", headers)
}

func createWalletRequest(t *testing.T, payload string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/wallets", payload, nil)
}

func importWalletRequest(t *testing.T, payload string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/wallets/import", payload, nil)
}

func generateKeyRequest(t *testing.T, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/keys", payload, headers)
}

func listKeysRequest(t *testing.T, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodGet, "/api/v1/keys", "", headers)
}

func getKeyRequest(t *testing.T, keyID string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/keys/%s", keyID), "", headers)
}

func taintKeyRequest(t *testing.T, id, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPut, fmt.Sprintf("/api/v1/keys/%s/taint", id), payload, headers)
}

func annotateKeyRequest(t *testing.T, id, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPut, fmt.Sprintf("/api/v1/keys/%s/metadata", id), payload, headers)
}

func checkTxRequest(t *testing.T, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/command/check", payload, headers)
}

func signTxRequest(t *testing.T, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/command", payload, headers)
}

func signAnyRequest(t *testing.T, payload string, headers map[string]string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/sign", payload, headers)
}

func verifyAnyRequest(t *testing.T, payload string) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodPost, "/api/v1/verify", payload, nil)
}

func chainIDRequest(t *testing.T) *http.Request {
	t.Helper()
	return buildRequest(t, http.MethodGet, "/api/v1/network/chainid", "", map[string]string{})
}

func authHeadersV1(t *testing.T, token string) map[string]string {
	t.Helper()
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}
