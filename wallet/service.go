package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.vegaprotocol.io/vega/logging"

	"github.com/rs/cors"
)

type Service struct {
	*http.ServeMux

	cfg         *Config
	log         *logging.Logger
	s           *http.Server
	handler     WalletHandler
	nodeForward NodeForward
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_handler_mock.go -package mocks code.vegaprotocol.io/vega/wallet WalletHandler
type WalletHandler interface {
	CreateWallet(wallet, passphrase string) (string, error)
	LoginWallet(wallet, passphrase string) (string, error)
	RevokeToken(token string) error
	GenerateKeypair(token, passphrase string) (string, error)
	ListPublicKeys(token string) ([]Keypair, error)
	SignTx(token, tx, pubkey string) (SignedBundle, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_forward_mock.go -package mocks code.vegaprotocol.io/vega/wallet NodeForward
type NodeForward interface {
	Send(context.Context, *SignedBundle) error
}

func NewServiceWith(log *logging.Logger, cfg *Config, rootPath string, h WalletHandler, n NodeForward) (*Service, error) {
	s := &Service{
		ServeMux:    http.NewServeMux(),
		log:         log,
		cfg:         cfg,
		handler:     h,
		nodeForward: n,
	}

	// all the endpoints are public for testing purpose
	s.HandleFunc("/api/v1/status", s.health)
	s.HandleFunc("/api/v1/create", s.CreateWallet)
	s.HandleFunc("/api/v1/login", s.Login)
	s.HandleFunc("/api/v1/revoke", ExtractToken(s.Revoke))
	s.HandleFunc("/api/v1/gen-keys", ExtractToken(s.GenerateKeypair))
	s.HandleFunc("/api/v1/list-keys", ExtractToken(s.ListPublicKeys))
	s.HandleFunc("/api/v1/sign", ExtractToken(s.SignTx))

	return s, nil

}

func NewService(log *logging.Logger, cfg *Config, rootPath string) (*Service, error) {
	log = log.Named(namedLogger)

	// ensure the folder exist
	if err := EnsureBaseFolder(rootPath); err != nil {
		return nil, err
	}
	auth, err := NewAuth(log, rootPath, cfg.TokenExpiry.Get())
	if err != nil {
		return nil, err
	}
	nodeForward, err := NewNodeForward(log, cfg.Node)
	if err != nil {
		return nil, err
	}
	handler := NewHandler(log, auth, rootPath)
	return NewServiceWith(log, cfg, rootPath, handler, nodeForward)
}

func (s *Service) Start() error {
	s.s = &http.Server{
		Addr:    fmt.Sprintf("%s:%v", s.cfg.IP, s.cfg.Port),
		Handler: cors.AllowAll().Handler(s), // middlewar with cors
	}

	s.log.Info("starting wallet http server", logging.String("address", s.s.Addr))
	return s.s.ListenAndServe()
}

func (s *Service) Stop() error {
	return s.s.Shutdown(context.Background())
}

func (s *Service) CreateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}
	// unmarshal request
	req := struct {
		Wallet     string `json:"wallet"`
		Passphrase string `json:"passphrase"`
	}{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	// validation
	if len(req.Wallet) <= 0 {
		writeError(w, newError("missing wallet field"), http.StatusBadRequest)
		return
	}
	if len(req.Passphrase) <= 0 {
		writeError(w, newError("missing passphrase field"), http.StatusBadRequest)
		return
	}

	token, err := s.handler.CreateWallet(req.Wallet, req.Passphrase)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}
	writeSuccess(w, token, http.StatusOK)
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}
	req := struct {
		Wallet     string `json:"wallet"`
		Passphrase string `json:"passphrase"`
	}{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	// validation
	if len(req.Wallet) <= 0 {
		writeError(w, newError("missing wallet field"), http.StatusBadRequest)
		return
	}
	if len(req.Passphrase) <= 0 {
		writeError(w, newError("missing passphrase field"), http.StatusBadRequest)
		return
	}

	token, err := s.handler.LoginWallet(req.Wallet, req.Passphrase)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}
	writeSuccess(w, token, http.StatusOK)
}

func (s *Service) Revoke(t string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}

	err := s.handler.RevokeToken(t)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	writeSuccess(w, true, http.StatusOK)
}

func (s *Service) GenerateKeypair(t string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}

	req := struct {
		Passphrase string `json:"passphrase"`
	}{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if len(req.Passphrase) <= 0 {
		writeError(w, newError("missing passphrase field"), http.StatusBadRequest)
		return
	}

	pubKey, err := s.handler.GenerateKeypair(t, req.Passphrase)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	writeSuccess(w, pubKey, http.StatusOK)
}

func (s *Service) ListPublicKeys(t string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}

	keys, err := s.handler.ListPublicKeys(t)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	writeSuccess(w, keys, http.StatusOK)
}

func (s *Service) SignTx(t string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}

	req := struct {
		Tx        string `json:"tx"`
		PubKey    string `json:"pubKey"`
		Propagate bool   `json:"propagate"`
	}{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if len(req.Tx) <= 0 {
		writeError(w, newError("missing tx field"), http.StatusBadRequest)
		return
	}
	if len(req.PubKey) <= 0 {
		writeError(w, newError("missing pubKey field"), http.StatusBadRequest)
		return
	}

	sb, err := s.handler.SignTx(t, req.Tx, req.PubKey)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	if req.Propagate {
		err := s.nodeForward.Send(r.Context(), &sb)
		if err != nil {
			writeError(w, newError(err.Error()), http.StatusInternalServerError)
			return
		}
	}

	writeSuccess(w, sb, http.StatusOK)
}

func (h *Service) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, ErrInvalidMethod, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func unmarshalBody(r *http.Request, into interface{}) error {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ErrInvalidRequest
	}
	return json.Unmarshal(body, into)
}

func writeError(w http.ResponseWriter, e error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(e)
	w.Write(buf)
}

type successResponse struct {
	Data interface{}
}

func writeSuccess(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(successResponse{data})
	w.Write(buf)
}

var (
	ErrInvalidRequest        = newError("invalid request")
	ErrInvalidMethod         = newError("invalid method")
	ErrInvalidOrMissingToken = newError("invalid or missing token")
)

type HttpError struct {
	ErrorStr string `json:"error"`
}

func (e HttpError) Error() string {
	return e.ErrorStr
}

func newError(e string) HttpError {
	return HttpError{
		ErrorStr: e,
	}
}
