package faucet

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

const (
	defaultVegaFaucetOwner = "vega-faucet"
)

var (
	// ErrNotABuiltinAsset is raised when a party try to top up for a non builtin asset
	ErrNotABuiltinAsset = errors.New("asset is not a builtin asset")

	// ErrAssetNotFound is raised when an asset id is not found
	ErrAssetNotFound = errors.New("asset was not found")
)

type Faucet struct {
	*httprouter.Router

	log    *logging.Logger
	cfg    Config
	wal    *wallet.Wallet
	s      *http.Server
	rl     *gateway.RateLimit
	cfunc  context.CancelFunc
	stopCh chan struct{}

	// node connections stuff
	clt     api.TradingClient
	cltdata api.TradingDataClient
	conn    *grpc.ClientConn
}

type MintRequest struct {
	Party  string `json:"party"`
	Amount uint64 `json:"amount"`
	Asset  string `json:"asset"`
}

type MintResponse struct {
	Success bool `json:"success"`
}

func New(log *logging.Logger, cfg Config, passphrase string) (*Faucet, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)
	wal, err := wallet.ReadWalletFile(cfg.WalletPath, passphrase)
	if err != nil {
		return nil, err
	}
	nodeAddr := fmt.Sprintf("%v:%v", cfg.Node.IP, cfg.Node.Port)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewTradingClient(conn)
	clientData := api.NewTradingDataClient(conn)

	ctx, cfunc := context.WithCancel(context.Background())

	f := &Faucet{
		Router:  httprouter.New(),
		log:     log,
		cfg:     cfg,
		wal:     wal,
		clt:     client,
		cltdata: clientData,
		conn:    conn,
		cfunc:   cfunc,
		rl:      gateway.NewRateLimit(ctx, cfg.RateLimit),
		stopCh:  make(chan struct{}),
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
	if req.Amount == 0 {
		writeError(w, newError("amount need to be a > 0 unsigned integer"), http.StatusBadRequest)
		return
	}
	if len(req.Asset) <= 0 {
		writeError(w, newError("missing asset field"), http.StatusBadRequest)
		return
	}

	if err := f.getAllowedAmount(r.Context(), req.Amount, req.Asset); err != nil {
		if errors.Is(err, ErrAssetNotFound) {
			writeError(w, newError(err.Error()), http.StatusBadRequest)
			return
		}
		writeError(w, newError(err.Error()), http.StatusInternalServerError)
		return
	}
	rlkey := fmt.Sprintf("party-%s-asset-%s", req.Party, req.Asset)
	if err := f.rl.NewRequest(rlkey); err != nil {
		f.log.Debug("Mint denied - rate limit", logging.String("rlkey", rlkey))
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	ce := &types.ChainEvent{
		Nonce: makeNonce(),
		Event: &types.ChainEvent_Builtin{
			Builtin: &types.BuiltinAssetEvent{
				Action: &types.BuiltinAssetEvent_Deposit{
					Deposit: &types.BuiltinAssetDeposit{
						VegaAssetID: req.Asset,
						PartyID:     req.Party,
						Amount:      req.Amount,
					},
				},
			},
		},
	}

	msg, err := proto.Marshal(ce)
	if err != nil {
		writeError(w, newError("unable to marshal"), http.StatusInternalServerError)
		return
	}

	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		f.log.Error("unable to instanciate new algorithm", logging.Error(err))
		writeError(w, newError("unable to instanciate crypto"), http.StatusInternalServerError)
		return
	}

	sig, err := wallet.Sign(alg, &f.wal.Keypairs[0], msg)
	if err != nil {
		f.log.Error("unable to sign", logging.Error(err))
		writeError(w, newError("unable to sign crypto"), http.StatusInternalServerError)
	}

	preq := &api.PropagateChainEventRequest{
		Evt:       ce,
		PubKey:    f.wal.Keypairs[0].Pub,
		Signature: sig,
	}

	var ok bool
	err = backoff.Retry(
		func() error {
			resp, err := f.clt.PropagateChainEvent(context.Background(), preq)
			if err != nil {
				return err
			}
			ok = resp.Success
			return nil
		},
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), f.cfg.Node.Retries),
	)
	if err != nil {
		writeError(w, newError(err.Error()), http.StatusInternalServerError)
		return
	}

	resp := MintResponse{ok}
	writeSuccess(w, resp, http.StatusOK)
}

func (f *Faucet) getAllowedAmount(ctx context.Context, amount uint64, asset string) error {
	req := &api.AssetByIDRequest{
		ID: asset,
	}
	resp, err := f.cltdata.AssetByID(ctx, req)
	if err != nil {
		if resp == nil {
			return ErrAssetNotFound
		}
		return err
	}
	source := resp.Asset.Source.GetBuiltinAsset()
	if source == nil {
		return ErrNotABuiltinAsset
	}
	maxAmount, err := strconv.ParseUint(source.MaxFaucetAmountMint, 10, 64)
	if err != nil {
		return err
	}
	if maxAmount < amount {
		return fmt.Errorf("amount request exceed maximal amount of %v", maxAmount)
	}

	return nil
}

func (f *Faucet) Start() error {
	f.s = &http.Server{
		Addr:    fmt.Sprintf("%s:%v", f.cfg.IP, f.cfg.Port),
		Handler: cors.AllowAll().Handler(f), // middleware with cors
	}

	f.log.Info("starting faucet server", logging.String("address", f.s.Addr))

	errCh := make(chan error)
	go func() {
		errCh <- f.s.ListenAndServe()
	}()

	defer func() {
		f.cfunc()
		f.conn.Close()
	}()

	// close the rate limit
	select {
	case err := <-errCh:
		return err
	case <-f.stopCh:
		f.s.Shutdown(context.Background())
		return nil
	}
}

func (f *Faucet) Stop() error {
	f.stopCh <- struct{}{}
	return nil
}

func Init(path, passphrase string) (string, error) {
	if ok, _ := fsutil.PathExists(path); ok {
		return "", fmt.Errorf("faucet file already exists %v", path)
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
