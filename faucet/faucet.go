package faucet

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

const (
	defaultVegaFaucetOwner = "vega-faucet"
)

type Faucet struct {
	*httprouter.Router

	log *logging.Logger
	cfg Config
	wal *wallet.Wallet
	s   *http.Server
}

type MintRequest struct {
	Party  string `json:"party"`
	Amount uint64 `json:"amount"`
	Asset  string `json:"asset"`
}

func New(log *logging.Logger, cfg Config, passphrase string) (*Faucet, error) {
	wal, err := wallet.ReadWalletFile(cfg.WalletPath, passphrase)
	if err != nil {
		return nil, err
	}
	f := &Faucet{
		Router: httprouter.New(),
		log:    log,
		cfg:    cfg,
		wal:    wal,
	}

	f.POST("/api/v1/mint", f.Mint)
	return f, nil
}

func (f *Faucet) Mint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// unmarshal request
	req := MintRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, newError(err.Error()), http.StatusBadRequest)
		return
	}

	// validation
	if len(req.Party) <= 0 {
		writeError(w, newError("missing party field"), http.StatusBadRequest)
		return
	}
	if len(req.Amount) <= 0 {
		writeError(w, newError("amount need to be a > 0 integer"), http.StatusBadRequest)
		return
	}
	if len(req.Asset) <= 0 {
		writeError(w, newError("missing asset field"), http.StatusBadRequest)
		return
	}

}

func (f *Faucet) Start() error {
	f.s = &http.Server{
		Addr:    fmt.Sprintf("%s:%v", f.cfg.IP, f.cfg.Port),
		Handler: cors.AllowAll().Handler(f), // middleware with cors
	}

	f.log.Info("starting wallet http server", logging.String("address", f.s.Addr))
	return f.s.ListenAndServe()

}

func (f *Faucet) Stop() error { return f.s.Shutdown(context.Background()) }

func Init(path, passphrase string) (string, error) {
	if ok, _ := fsutil.PathExists(path); ok {
		return "", fmt.Errorf("faucet folder already exists %v", path)
	}

	w, err := wallet.CreateWalletFile(path, defaultVegaFaucetOwner, passphrase)
	if err != nil {
		return "", err
	}

	// gen the keypair
	algo := crypto.NewEd25519()
	kp, err := wallet.GenKeypair(algo.Name())
	if err != nil {
		return "", fmt.Errorf("unable to generate new key pair: %v", err)
	}

	w.Keypairs = append(w.Keypairs, *kp)
	_, err = wallet.WriteWalletFile(w, path, passphrase)
	if err != nil {
		return "", err
	}

	return kp.Pub, nil
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

func writeSuccess(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(data)
	w.Write(buf)
}

var (
	ErrInvalidRequest = newError("invalid request")
)

type HTTPError struct {
	ErrorStr string `json:"error"`
}

func (e HTTPError) Error() string {
	return e.ErrorStr
}

func newError(e string) HTTPError {
	return HTTPError{
		ErrorStr: e,
	}
}

func makeNonce() uint64 {
	max := &big.Int{}
	// set it to the max value of the uint64
	max.SetUint64(^uint64(0))
	nonce, _ := rand.Int(rand.Reader, max)
	return nonce.Uint64()
}
