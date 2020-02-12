package wallet

import (
	"net/http"

	"code.vegaprotocol.io/vega/logging"
)

type handler struct {
	*http.ServeMux
	log *logging.Logger
}

func newHandler(log *logging.Logger) *handler {
	h := &handler{log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	h.ServeMux = mux
	return h
}

func (h *handler) CreateWallet(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) LoginWallet(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) LogoutWallet(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) GenerateKeypair(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) ListPublicKeys(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) SignAndSubmitTx(w http.ResponseWriter, r *http.Request) {
}

func (h *handler) SignTx(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}
