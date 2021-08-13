package faucet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	vhttp "code.vegaprotocol.io/vega/http"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"google.golang.org/grpc"
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
	wallet *faucetWallet
	s      *http.Server
	rl     *vhttp.RateLimit
	cfunc  context.CancelFunc
	stopCh chan struct{}

	// node connections stuff
	clt     api.TradingServiceClient
	cltdata api.TradingDataServiceClient
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

	wallet, err := loadWallet(cfg.WalletPath, passphrase)
	if err != nil {
		return nil, err
	}

	nodeAddr := fmt.Sprintf("%v:%v", cfg.Node.IP, cfg.Node.Port)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewTradingServiceClient(conn)
	clientData := api.NewTradingDataServiceClient(conn)

	ctx, cfunc := context.WithCancel(context.Background())

	rl, err := vhttp.NewRateLimit(ctx, cfg.RateLimit)
	if err != nil {
		cfunc()
		return nil, fmt.Errorf("failed to create RateLimit: %v", err)
	}

	f := &Faucet{
		Router:  httprouter.New(),
		log:     log,
		cfg:     cfg,
		wallet:  wallet,
		clt:     client,
		cltdata: clientData,
		conn:    conn,
		cfunc:   cfunc,
		rl:      rl,
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

	// rate limit minting by source IP address, party, asset
	ip, err := vhttp.RemoteAddr(r)
	if err != nil {
		writeError(w, newError(fmt.Sprintf("failed to get request remote address: %v", err)), http.StatusBadRequest)
		return
	}
	rlkey := fmt.Sprintf("minting for party %s and asset %s", req.Party, req.Asset)
	if err := f.rl.NewRequest(rlkey, ip); err != nil {
		f.log.Debug("Mint denied - rate limit",
			logging.String("ip", ip),
			logging.String("rlkey", rlkey),
		)
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	ce := &commandspb.ChainEvent{
		Nonce: crypto.NewNonce(),
		Event: &commandspb.ChainEvent_Builtin{
			Builtin: &types.BuiltinAssetEvent{
				Action: &types.BuiltinAssetEvent_Deposit{
					Deposit: &types.BuiltinAssetDeposit{
						VegaAssetId: req.Asset,
						PartyId:     req.Party,
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

	sig, pubKey, err := f.wallet.Sign(msg)
	if err != nil {
		f.log.Error("unable to sign", logging.Error(err))
		writeError(w, newError("unable to sign crypto"), http.StatusInternalServerError)
	}

	preq := &api.PropagateChainEventRequest{
		Event:     msg,
		PubKey:    pubKey,
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
		Id: asset,
	}
	resp, err := f.cltdata.AssetByID(ctx, req)
	if err != nil {
		if resp == nil {
			return ErrAssetNotFound
		}
		return err
	}
	source := resp.Asset.Details.GetBuiltinAsset()
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
