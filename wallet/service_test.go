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
}

func testServiceCreateWalletOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().CreateWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)

	payload := `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.CreateWallet(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceCreateWalletFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	payload := `{"wall": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.CreateWallet(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.CreateWallet(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r = httptest.NewRequest("GET", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.CreateWallet(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusMethodNotAllowed)
}

func testServiceLoginWalletOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().LoginWallet(gomock.Any(), gomock.Any()).Times(1).Return("this is a token", nil)

	payload := `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.Login(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceLoginWalletFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	payload := `{"wall": "jeremy", "passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	s.Login(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passrase": "oh yea?"}`
	r = httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.Login(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"wallet": "jeremy", "passphrase": "oh yea?"}`
	r = httptest.NewRequest("GET", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()

	s.Login(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusMethodNotAllowed)
}

func testServiceRevokeTokenOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().RevokeToken(gomock.Any()).Times(1).Return(nil)

	r := httptest.NewRequest("POST", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceRevokeTokenFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.Revoke)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func testServiceGenKeypairOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().GenerateKeypair(gomock.Any(), gomock.Any()).Times(1).Return("", nil)

	payload := `{"passphrase": "oh yea?"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceGenKeypairFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.GenerateKeypair)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but no payload
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	wallet.ExtractToken(s.GenerateKeypair)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}

func testServiceListPublicKeysOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().ListPublicKeys(gomock.Any()).Times(1).
		Return([]wallet.Keypair{}, nil)

	r := httptest.NewRequest("GET", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceListPublicKeysFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// invalid token
	r := httptest.NewRequest("POST", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.ListPublicKeys)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func testServiceSignOK(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	s.handler.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).Return(wallet.SignedBundle{}, nil)
	payload := `{"tx": "some data", "pubKey": "asdasasdasd"}`
	r := httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	w := httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func testServiceSignFailInvalidRequest(t *testing.T) {
	s := getTestService(t)
	defer s.ctrl.Finish()

	// InvalidMethod
	r := httptest.NewRequest("GET", "http://example.com/create", nil)
	w := httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// invalid token
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	r.Header.Add("Authorization", "Bearer")

	w = httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// no token
	r = httptest.NewRequest("POST", "http://example.com/create", nil)
	w = httptest.NewRecorder()

	wallet.ExtractToken(s.SignTx)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// token but invalid payload
	payload := `{"t": "some data", "pubKey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	wallet.ExtractToken(s.SignTx)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	payload = `{"tx": "some data", "puey": "asdasasdasd"}`
	r = httptest.NewRequest("POST", "http://example.com/create", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODIyMDYwMDMsImlzcyI6InZlZ2Egd2FsbGV0IiwiU2Vzc2lvbiI6ImI1NjFkMDMxMGFhNjA5YWQxZDhkZGJjMTJiZmU5OWI2ZGNhZGNkM2E4NDMzNjRkM2I0N2YzNmQ2MmQ2ZDkyYWYiLCJXYWxsZXQiOiJlZHdhcmQifQ.C5m4_-CEhjUxouruvW_S2rr4rbOKFxvyz1uYf4Aa-1pK3yG0e97a3_fG1MXXH5-9uxdbvc0khsrxaSbGKQTQH1ySSuAGgmJ3-1_Uvj64dbc0bOteeOd1b65jJcRm7chrWmw_cb0uPp6T75_W3nKRVpJ8jmElcXOf9yKfRIojVgy8belY01V5yQQAdWSBRMG9uC-KjQOkVfjagvVSL3uWNbgApNR-RnORp8JMYs5ETXztan5KXjkh6ncaA9dC1Gc4u2X4FAMciWl5ddBjnEy9CSxnzoJkHSWeq23Kb0LRglb35Tikrq1QXohy3PDtsRl3NNDTLq95tMwzpzW_uvq8zA")

	wallet.ExtractToken(s.SignTx)(w, r)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

}
