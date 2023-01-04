// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package faucet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	vghttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	types "code.vegaprotocol.io/vega/protos/vega"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cenkalti/backoff"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

var (
	// ErrNotABuiltinAsset is raised when a party try to top up for a non builtin asset.
	ErrNotABuiltinAsset = errors.New("asset is not a builtin asset")

	// ErrAssetNotFound is raised when an asset id is not found.
	ErrAssetNotFound = errors.New("asset was not found")

	// ErrInvalidMintAmount is raised when the mint amount is too high.
	ErrInvalidMintAmount = errors.New("mint amount is invalid")

	HealthCheckResponse = struct {
		Success bool `json:"success"`
	}{true}
)

type Faucet struct {
	*httprouter.Router

	log    *logging.Logger
	cfg    Config
	wallet *faucetWallet
	s      *http.Server
	rl     *vghttp.RateLimit
	cfunc  context.CancelFunc
	stopCh chan struct{}

	// node connections stuff
	clt     api.CoreServiceClient
	coreclt api.CoreStateServiceClient
	conn    *grpc.ClientConn
}

type MintRequest struct {
	Party  string `json:"party"`
	Amount string `json:"amount"`
	Asset  string `json:"asset"`
}

type MintResponse struct {
	Success bool `json:"success"`
}

func NewService(log *logging.Logger, vegaPaths paths.Paths, cfg Config, passphrase string) (*Faucet, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)

	walletLoader, err := InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise faucet wallet loader: %w", err)
	}

	wallet, err := walletLoader.load(cfg.WalletName, passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load faucet wallet %s: %w", cfg.WalletName, err)
	}

	nodeAddr := fmt.Sprintf("%v:%v", cfg.Node.IP, cfg.Node.Port)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := api.NewCoreServiceClient(conn)
	coreClient := api.NewCoreStateServiceClient(conn)
	ctx, cfunc := context.WithCancel(context.Background())

	rl, err := vghttp.NewRateLimit(ctx, cfg.RateLimit)
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
		coreclt: coreClient,
		conn:    conn,
		cfunc:   cfunc,
		rl:      rl,
		stopCh:  make(chan struct{}),
	}

	f.POST("/api/v1/mint", f.Mint)
	f.GET("/api/v1/health", f.Health)
	return f, nil
}

func (f *Faucet) Health(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	writeSuccess(w, HealthCheckResponse, http.StatusOK)
}

func (f *Faucet) Mint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req := MintRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		writeError(w, newError(err.Error()), http.StatusBadRequest)
		return
	}

	if len(req.Party) <= 0 {
		writeError(w, newError("missing party field"), http.StatusBadRequest)
		return
	}
	if len(req.Amount) <= 0 {
		writeError(w, newError("amount need to be a > 0 unsigned integer"), http.StatusBadRequest)
		return
	}
	amount, overflowed := num.UintFromString(req.Amount, 10)
	if overflowed {
		writeError(w, newError("amount overflowed or was not base 10"), http.StatusBadRequest)
	}

	if len(req.Asset) <= 0 {
		writeError(w, newError("missing asset field"), http.StatusBadRequest)
		return
	}

	if err := f.getAllowedAmount(r.Context(), amount, req.Asset); err != nil {
		if errors.Is(err, ErrAssetNotFound) || errors.Is(err, ErrInvalidMintAmount) {
			writeError(w, newError(err.Error()), http.StatusBadRequest)
			return
		}
		writeError(w, newError(err.Error()), http.StatusInternalServerError)
		return
	}

	// rate limit minting by source IP address, party, asset
	ip, err := vghttp.RemoteAddr(r)
	if err != nil {
		writeError(w, newError(fmt.Sprintf("failed to get request remote address: %v", err)), http.StatusBadRequest)
		return
	}
	rlkey := fmt.Sprintf("minting for party %s and asset %s", req.Party, req.Asset)
	if err := f.rl.NewRequest(rlkey, ip); err != nil {
		f.log.Debug("Mint denied - rate limit",
			logging.String("ip", vfmt.Escape(ip)),
			logging.String("rlkey", vfmt.Escape(rlkey)),
		)
		writeError(w, newError(err.Error()), http.StatusForbidden)
		return
	}

	ce := &commandspb.ChainEvent{
		Nonce: vgrand.NewNonce(),
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
		return
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

func (f *Faucet) getAllowedAmount(ctx context.Context, amount *num.Uint, asset string) error {
	req := &api.ListAssetsRequest{
		Asset: asset,
	}
	resp, err := f.coreclt.ListAssets(ctx, req)
	if err != nil {
		return err
	}
	if len(resp.Assets) <= 0 {
		return ErrAssetNotFound
	}
	source := resp.Assets[0].Details.GetBuiltinAsset()
	if source == nil {
		return ErrNotABuiltinAsset
	}
	maxAmount, overflowed := num.UintFromString(source.MaxFaucetAmountMint, 10)
	if overflowed {
		return ErrInvalidMintAmount
	}
	if maxAmount.LT(amount) {
		return fmt.Errorf("amount request exceed maximal amount of %v: %w", maxAmount, ErrInvalidMintAmount)
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

var ErrInvalidRequest = newError("invalid request")

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
