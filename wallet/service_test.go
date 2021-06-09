package wallet_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	vhttp "code.vegaprotocol.io/vega/http"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto/api"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"code.vegaprotocol.io/vega/wallet/mocks"
	"github.com/stretchr/testify/require"

	"github.com/golang/mock/gomock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*wallet.Service

	ctrl        *gomock.Controller
	handler     *mocks.MockWalletHandler
	nodeForward *mocks.MockNodeForward
	nodeClient  *mocks.MockNodeClient
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	handler := mocks.NewMockWalletHandler(ctrl)
	nodeForward := mocks.NewMockNodeForward(ctrl)
	nodeClient := mocks.NewMockNodeClient(ctrl)
	cfg := &wallet.Config{
		RateLimit: vhttp.RateLimitConfig{
			CoolDown: encoding.Duration{Duration: 1 * time.Minute},
		},
	}
	s, _ := wallet.NewServiceWith(logging.NewTestLogger(), cfg, handler, nodeForward, nodeClient)
	return &testService{
		Service:     s,
		ctrl:        ctrl,
		handler:     handler,
		nodeForward: nodeForward,
		nodeClient:  nodeClient,
	}
}

func TestService(t *testing.T) {
	t.Run("create wallet ok", testServiceCreateWalletOK)
	t.Run("create wallet fail invalid request", testServiceCreateWalletFailInvalidRequest)
	t.Run("create wallet fail rate limit", testServiceCreateWalletFailRateLimit)
	t.Run("login wallet ok", testServiceLoginWalletOK)
	t.Run("Downloading the wallet succeeds", testServiceDownloadingWalletSucceeds)
	t.Run("login wallet fail invalid request", testServiceLoginWalletFailInvalidRequest)
	t.Run("revoke token ok", testServiceRevokeTokenOK)
	t.Run("revoke token fail invalid request", testServiceRevokeTokenFailInvalidRequest)
	t.Run("gen keypair ok", testServiceGenKeypairOK)
	t.Run("gen keypair fail invalid request", testServiceGenKeypairFailInvalidRequest)
	t.Run("gen keypair fail rate limit", testServiceGenKeypairFailRateLimit)
	t.Run("list keypair ok", testServiceListPublicKeysOK)
	t.Run("list keypair fail invalid request", testServiceListPublicKeysFailInvalidRequest)
	t.Run("get keypair ok", testServiceGetPublicKeyOK)
	t.Run("get keypair fail invalid request", testServiceGetPublicKeyFailInvalidRequest)
	t.Run("get keypair fail key not found", testServiceGetPublicKeyFailKeyNotFound)
	t.Run("get keypair fail misc error", testServiceGetPublicKeyFailMiscError)
	t.Run("taint ok", testServiceTaintOK)
	t.Run("taint fail invalid request", testServiceTaintFailInvalidRequest)
	t.Run("update meta", testServiceUpdateMetaOK)
	t.Run("update meta invalid request", testServiceUpdateMetaFailInvalidRequest)
	t.Run("Signing transaction succeeds", testSigningTransactionSucceeds)
	t.Run("Signing transaction with propagation succeeds", testSigningTransactionWithPropagationSucceeds)
	t.Run("Signing transaction with failed propagation fails", testSigningTransactionWithFailedPropagationFails)
	t.Run("Failed signing of transaction fails", testFailedSigningTransactionFails)
	t.Run("Signing transaction with invalid payload fails", testSigningTransactionWithInvalidPayloadFails)
	t.Run("Signing transaction without pub-key fails", testSigningTransactionWithoutPubKeyFails)
	t.Run("Signing transaction without command fails", testSigningTransactionWithoutCommandFails)
}

func testServiceCreateWalletOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().CreateWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)

	payload := `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.CreateWallet(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceCreateWalletFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	payload := `{"wall": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.CreateWallet(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.CreateWallet(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func testServiceCreateWalletFailRateLimit(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// create wallet - OK
	s.handler.EXPECT().CreateWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)
	payload := `{"wallet": "someone", "passphrase": "123"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	s.CreateWallet(w, r, nil)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// create wallet - rate limit
	payload = `{"wallet": "someoneelse", "passphrase": "pass"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	s.CreateWallet(w, r, nil)
	resp = w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func testServiceLoginWalletOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().LoginWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)

	payload := `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.Login(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceDownloadingWalletSucceeds(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().LoginWallet(gomock.Any(), gomock.Any()).
		Times(1).
		Return("this is a token", nil)

	payload := `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r := newAuthenticatedRequest(payload)

	w := httptest.NewRecorder()

	s.Login(w, r, nil)
	resp := w.Result()
	var token struct {
		Data string
	}
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	raw, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	_ = json.Unmarshal(raw, &token)

	tmpFile, _ := ioutil.TempFile(".", "test-wallet")
	defer func() {
		name := tmpFile.Name()
		tmpFile.Close()
		os.Remove(name)
	}()
	s.handler.EXPECT().GetWalletPath(token.Data).Times(1).Return(tmpFile.Name(), nil)

	// now get the file:
	r = httptest.NewRequest(http.MethodGet, "scheme://host/path", bytes.NewBufferString(""))
	w = httptest.NewRecorder()

	s.DownloadWallet(token.Data, w, r, nil)
	resp = w.Result()

	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceLoginWalletFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	payload := `{"wall": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.Login(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.Login(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func testServiceRevokeTokenOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().RevokeToken(gomock.Any()).Times(1).Return(nil)

	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceRevokeTokenFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func testServiceGenKeypairOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().GenerateKeypair(gomock.Any(), gomock.Any()).Times(1).Return("", nil)
	s.handler.EXPECT().GetWalletName(gomock.Any()).Times(1).Return("walletname", nil)
	s.handler.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Times(1).Return(&wallet.Keypair{}, nil)

	payload := `{"passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceGenKeypairFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// token but no payload
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

}

func testServiceGenKeypairFailRateLimit(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()
	const passphrase = "p4ssphr4s3"

	// log in
	s.handler.EXPECT().LoginWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)
	payload := fmt.Sprintf(`{"wallet": "walletname", "passphrase": "%s"}`, passphrase)
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	s.Login(w, r, nil)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// generate keypair - OK
	s.handler.EXPECT().GetWalletName(gomock.Any()).Times(1).Return("walletname", nil)
	s.handler.EXPECT().GenerateKeypair(gomock.Any(), gomock.Any()).Times(1).Return("", nil)
	s.handler.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Times(1).Return(&wallet.Keypair{}, nil)
	payload = fmt.Sprintf(`{"passphrase": "%s"}`, passphrase)
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")
	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// generate keypair - ratelimit
	s.handler.EXPECT().GetWalletName(gomock.Any()).Times(1).Return("walletname", nil)
	payload = fmt.Sprintf(`{"passphrase": "%s"}`, passphrase)
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")
	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)
	resp = w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func testServiceListPublicKeysOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().ListPublicKeys(gomock.Any()).Times(1).
		Return([]wallet.Keypair{}, nil)

	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceListPublicKeysFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func testServiceGetPublicKeyOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	kp := wallet.Keypair{
		Pub:       "pub",
		Priv:      "",
		Algorithm: crypto.NewEd25519(),
		Tainted:   false,
		Meta:      []wallet.Meta{{Key: "a", Value: "b"}},
	}
	s.handler.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Times(1).
		Return(&kp, nil)

	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GetPublicKey)(w, r, httprouter.Params{{Key: "keyid", Value: "apubkey"}})

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceGetPublicKeyFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GetPublicKey)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.GetPublicKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func testServiceGetPublicKeyFailKeyNotFound(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Times(1).
		Return(nil, wallet.ErrPubKeyDoesNotExist)

	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GetPublicKey)(w, r, httprouter.Params{{Key: "keyid", Value: "apubkey"}})

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func testServiceGetPublicKeyFailMiscError(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Times(1).
		Return(nil, errors.New("an error"))

	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GetPublicKey)(w, r, httprouter.Params{{Key: "keyid", Value: "apubkey"}})

	resp := w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func testServiceTaintOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().TaintKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(nil)
	payload := `{"passphrase": "some data"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, httprouter.Params{{Key: "keyid", Value: "asdasasdasd"}})

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceTaintFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// token but invalid payload
	payload := `{"passhp": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	payload = `{"passphrase": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

}

func testServiceUpdateMetaOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().UpdateMeta(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(nil)
	payload := `{"passphrase": "some data", "meta": [{"key":"ok", "value":"primary"}]}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Set("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, httprouter.Params{{Key: "keyid", Value: "asdasasdasd"}})

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testServiceUpdateMetaFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Set("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// token but invalid payload
	payload := `{"passhp": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	payload = `{"passphrase": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

}

func testSigningTransactionSucceeds(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "eyXXzA"
	payload := `{"pubKey": "0xCAFEDUDE", "orderCancellation": {}}`
	request := newAuthenticatedRequest(payload)
	response := httptest.NewRecorder()

	// setup
	s.handler.EXPECT().
		SignTxV2(token, gomock.Any()).
		Times(1).
		Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().
		SendTxV2(gomock.Any(), &commandspb.Transaction{}, api.SubmitTransactionV2Request_TYPE_ASYNC).
		Times(0)

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func testSigningTransactionWithPropagationSucceeds(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "eyXXzA"
	payload := `{"propagate": true, "pubKey": "0xCAFEDUDE", "orderCancellation": {}}`
	request := newAuthenticatedRequest(payload)
	response := httptest.NewRecorder()

	// setup
	s.handler.EXPECT().
		SignTxV2(token, gomock.Any()).
		Times(1).
		Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().
		SendTxV2(gomock.Any(), &commandspb.Transaction{}, api.SubmitTransactionV2Request_TYPE_SYNC).
		Times(1).
		Return(nil)

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func testSigningTransactionWithFailedPropagationFails(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "eyXXzA"
	payload := `{"propagate": true, "pubKey": "0xCAFEDUDE", "orderCancellation": {}}`
	request := newAuthenticatedRequest(payload)
	response := httptest.NewRecorder()

	// setup
	s.handler.EXPECT().
		SignTxV2(token, gomock.Any()).
		Times(1).
		Return(&commandspb.Transaction{}, nil)
	s.nodeForward.EXPECT().
		SendTxV2(gomock.Any(), &commandspb.Transaction{}, api.SubmitTransactionV2Request_TYPE_SYNC).
		Times(1).
		Return(errors.New("failure"))

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
}

func testFailedSigningTransactionFails(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "eyXXzA"
	payload := `{"pubKey": "0xCAFEDUDE", "orderCancellation": {}}`
	request := newAuthenticatedRequest(payload)
	response := httptest.NewRecorder()

	// setup
	s.handler.EXPECT().
		SignTxV2(token, gomock.Any()).
		Times(1).
		Return(nil, errors.New("failure"))

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
}

func testSigningTransactionWithInvalidPayloadFails(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "eyXXzA"
	payload := `{"badKey": "0xCAFEDUDE"}`
	request := newAuthenticatedRequest(payload)
	response := httptest.NewRecorder()

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func testSigningTransactionWithoutPubKeyFails(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "0xDEADBEEF"
	payload := `{"orderSubmission": {}}`
	response := httptest.NewRecorder()
	request := newAuthenticatedRequest(payload)

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	require.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func testSigningTransactionWithoutCommandFails(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// given
	token := "0xDEADBEEF"
	payload := `{"pubKey": "0xCAFEDUDE"}`
	response := httptest.NewRecorder()
	request := newAuthenticatedRequest(payload)

	// when
	s.SignTxSyncV2(token, response, request, nil)

	// then
	result := response.Result()
	require.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func newAuthenticatedRequest(payload string) *http.Request {
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Set("Authorization", "Bearer eyXXzA")
	return r
}
