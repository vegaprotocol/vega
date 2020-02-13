package wallet

import (
	"net/http"
	"sync"

	"code.vegaprotocol.io/vega/logging"
)

type Auth interface {
	NewSession(walletname string) (string, error)
	VerifyToken(token string) (string, error)
	Revoke(token string) error
}

type handler struct {
	log      *logging.Logger
	auth     Auth
	rootPath string

	// wallet name -> wallet
	store map[string]*Wallet

	// just to make sure we do not access same file conccurently or the map
	mu sync.RWMutex
}

func newHandler(log *logging.Logger, auth Auth, rootPath string) *handler {
	return &handler{
		log:      log,
		auth:     auth,
		rootPath: rootPath,
		store:    map[string]*Wallet{},
	}
}

// return the actual token
func (h *handler) CreateWallet(wallet, passphrase string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.store[wallet]; ok {
		return "", ErrWalletAlreadyExist
	}

	w, err := Create(h.rootPath, wallet, passphrase)
	if err != nil {
		return "", err
	}

	h.store[wallet] = w
	return h.auth.NewSession(wallet)
}

func (h *handler) LoginWallet(wallet, passphrase string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// first check if the user own the wallet
	w, err := Read(h.rootPath, wallet, passphrase)
	if err != nil {
		return "", ErrWalletDoesNotExist
	}

	// then store it in the memory store then
	if _, ok := h.store[wallet]; !ok {
		h.store[wallet] = w
	}

	return h.auth.NewSession(wallet)
}

func (h *handler) RevokeToken(token string) error {
	return h.auth.Revoke(token)
}

func (h *handler) GenerateKeypair(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

}

func (h *handler) ListPublicKeys(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
}

func (h *handler) SignAndSubmitTx(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
}

func (h *handler) SignTx(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
}
