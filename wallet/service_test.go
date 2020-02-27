package wallet_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/mocks"

	"github.com/golang/mock/gomock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

// this tests in general ensure request / response contracts are not broken for the service

type testService struct {
	*wallet.Service

	ctrl        *gomock.Controller
	handler     *mocks.MockWalletHandler
	nodeForward *mocks.MockNodeForward
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	handler := mocks.NewMockWalletHandler(ctrl)
	nodeForward := mocks.NewMockNodeForward(ctrl)
	// no needs of the conf or path as we do not run an actual service
	s, _ := wallet.NewServiceWith(logging.NewTestLogger(), nil, "", handler, nodeForward)
	return &testService{
		Service:     s,
		ctrl:        ctrl,
		handler:     handler,
		nodeForward: nodeForward,
	}
}

func TestService(t *testing.T) {
	t.Run("create wallet ok", testServiceCreateWalletOK)
	t.Run("create wallet fail invalid request", testServiceCreateWalletFailInvalidRequest)
	t.Run("login wallet ok", testServiceLoginWalletOK)
	t.Run("login wallet fail invalid request", testServiceLoginWalletFailInvalidRequest)
	t.Run("revoke token ok", testServiceRevokeTokenOK)
	t.Run("revoke token fail invalid request", testServiceRevokeTokenFailInvalidRequest)
	t.Run("gen keypair ok", testServiceGenKeypairOK)
	t.Run("gen keypair fail invalid request", testServiceGenKeypairFailInvalidRequest)
	t.Run("list keypair ok", testServiceListPublicKeysOK)
	t.Run("list keypair fail invalid request", testServiceListPublicKeysFailInvalidRequest)
	t.Run("sign ok", testServiceSignOK)
	t.Run("sign fail invalid request", testServiceSignFailInvalidRequest)
	t.Run("taint ok", testServiceTaintOK)
	t.Run("taint fail invalid request", testServiceTaintFailInvalidRequest)
	t.Run("update meta", testServiceUpdateMetaOK)
	t.Run("update meta invalid request", testServiceUpdateMetaFailInvalidRequest)
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
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceCreateWalletFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	payload := `{"wall": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.CreateWallet(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.CreateWallet(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
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
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.Login(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func testServiceRevokeTokenOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().RevokeToken(gomock.Any()).Times(1).Return(nil)

	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceRevokeTokenFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func testServiceGenKeypairOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().GenerateKeypair(gomock.Any(), gomock.Any()).Times(1).Return("", nil)

	payload := `{"passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceGenKeypairFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but no payload
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.GenerateKeypair)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}

func testServiceListPublicKeysOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().ListPublicKeys(gomock.Any()).Times(1).
		Return([]wallet.Keypair{}, nil)

	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceListPublicKeysFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func testServiceSignOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(wallet.SignedBundle{}, nil)
	payload := `{"tx": "some data", "pubKey": "asdasasdasd"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceSignFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// InvalidMethod
	r := httptest.NewRequest("GET", "scheme://host/path", nil)
	w := httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// invalid token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w = httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but invalid payload
	payload := `{"t": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"tx": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.SignTx)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}

func testServiceTaintOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().TaintKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(nil)
	payload := `{"passphrase": "some data"}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, httprouter.Params{{Key: "keyid", Value: "asdasasdasd"}})

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceTaintFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but invalid payload
	payload := `{"passhp": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"passphrase": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.TaintKey)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}

func testServiceUpdateMetaOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().UpdateMeta(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(nil)
	payload := `{"passphrase": "some data", "meta": [{"key":"ok", "value":"primary"}]}`
	r := httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyXXzA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, httprouter.Params{{Key: "keyid", Value: "asdasasdasd"}})

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceUpdateMetaFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "scheme://host/path", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "scheme://host/path", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but invalid payload
	payload := `{"passhp": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"passphrase": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "scheme://host/path", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyXXzA")

	wallet.ExtractToken(s.UpdateMeta)(w, r, nil)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}
