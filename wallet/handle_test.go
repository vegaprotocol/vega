package wallet_test

import (
	"errors"
	"os"
	"testing"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testHandler struct {
	*wallet.Handler
	ctrl    *gomock.Controller
	auth    *mocks.MockAuth
	rootDir string
}

func getTestHandler(t *testing.T) *testHandler {
	ctrl := gomock.NewController(t)
	auth := mocks.NewMockAuth(ctrl)
	rootPath := rootDir()
	fsutil.EnsureDir(rootPath)
	wallet.EnsureBaseFolder(rootPath)

	h := wallet.NewHandler(logging.NewTestLogger(), auth, rootPath)
	return &testHandler{
		Handler: h,
		ctrl:    ctrl,
		auth:    auth,
		rootDir: rootPath,
	}
}

func TestHandler(t *testing.T) {
	t.Run("create a wallet success then login", testHandlerCreateWalletThenLogin)
	t.Run("create a wallet failure - already exists", testHandlerCreateWalletFailureAlreadyExists)
	t.Run("login failure on non wallet", testHandlerLoginFailureOnNonCreatedWallet)
	t.Run("revoke token success", testHandlerRevokeTokenSuccess)
	t.Run("revoke token failure", testHandlerRevokeTokenFailure)
	t.Run("generate keypair success and list public keys", testVerifyTokenSuccess)
	t.Run("generate keypair failure - invalid token", testVerifyTokenInvalidToken)
	t.Run("generate keypair failure - wallet not found", testVerifyTokenWalletNotFound)
	t.Run("list public key failure - invalid token", testListPubInvalidToken)
	t.Run("list public key failure - wallet not found", testListPubWalletNotFound)
}

func testHandlerCreateWalletThenLogin(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	h.auth.EXPECT().NewSession(gomock.Any()).Times(2).
		Return("some fake token", nil)

	tok, err := h.CreateWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	tok, err = h.LoginWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testHandlerCreateWalletFailureAlreadyExists(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	h.auth.EXPECT().NewSession(gomock.Any()).Times(1).
		Return("some fake token", nil)

	// create the wallet once.
	tok, err := h.CreateWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	// try to create it again
	tok, err = h.CreateWallet("jeremy", "we can use a different passphrase yo!")
	assert.EqualError(t, err, wallet.ErrWalletAlreadyExist.Error())
	assert.Empty(t, tok)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testHandlerLoginFailureOnNonCreatedWallet(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	tok, err := h.LoginWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.EqualError(t, err, wallet.ErrWalletDoesNotExist.Error())
	assert.Empty(t, tok)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testHandlerRevokeTokenSuccess(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	h.auth.EXPECT().NewSession(gomock.Any()).Times(1).
		Return("some fake token", nil)

	tok, err := h.CreateWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	h.auth.EXPECT().Revoke(gomock.Any()).Times(1).
		Return(nil)
	err = h.RevokeToken(tok)
	assert.NoError(t, err)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testHandlerRevokeTokenFailure(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	h.auth.EXPECT().NewSession(gomock.Any()).Times(1).
		Return("some fake token", nil)

	tok, err := h.CreateWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	h.auth.EXPECT().Revoke(gomock.Any()).Times(1).
		Return(errors.New("bad token"))
	err = h.RevokeToken(tok)
	assert.EqualError(t, err, "bad token")

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testVerifyTokenSuccess(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	// first create the wallet
	h.auth.EXPECT().NewSession(gomock.Any()).Times(1).
		Return("some fake token", nil)

	tok, err := h.CreateWallet("jeremy", "thisisasecurepassphraseinnit")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	// then start the test
	h.auth.EXPECT().VerifyToken(gomock.Any()).Times(2).
		Return("jeremy", nil)

	key, err := h.GenerateKeypair(tok)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	// now make sure we have the new key saved
	keys, err := h.ListPublicKeys(tok)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, key, keys[0].Pub)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testVerifyTokenInvalidToken(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	// then start the test
	h.auth.EXPECT().VerifyToken(gomock.Any()).Times(1).
		Return("", errors.New("bad token"))

	key, err := h.GenerateKeypair("yolo token")
	assert.EqualError(t, err, "bad token")
	assert.Empty(t, key)

	assert.NoError(t, os.RemoveAll(h.rootDir))

}

// this should never happend but beeeh....
func testVerifyTokenWalletNotFound(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	// then start the test
	h.auth.EXPECT().VerifyToken(gomock.Any()).Times(1).
		Return("jeremy", nil)

	key, err := h.GenerateKeypair("yolo token")
	assert.EqualError(t, err, "could not found wallet")
	assert.Empty(t, key)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}

func testListPubInvalidToken(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	// then start the test
	h.auth.EXPECT().VerifyToken(gomock.Any()).Times(1).
		Return("", errors.New("bad token"))

	key, err := h.ListPublicKeys("yolo token")
	assert.EqualError(t, err, "bad token")
	assert.Empty(t, key)

	assert.NoError(t, os.RemoveAll(h.rootDir))

}

// this should never happend but beeeh....
func testListPubWalletNotFound(t *testing.T) {
	h := getTestHandler(t)
	defer h.ctrl.Finish()

	// then start the test
	h.auth.EXPECT().VerifyToken(gomock.Any()).Times(1).
		Return("jeremy", nil)

	key, err := h.ListPublicKeys("yolo token")
	assert.EqualError(t, err, "could not found wallet")
	assert.Empty(t, key)

	assert.NoError(t, os.RemoveAll(h.rootDir))
}
