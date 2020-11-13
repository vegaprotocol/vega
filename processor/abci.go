package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/processor/ratelimit"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/vegatime"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type App struct {
	abci              *abci.App
	currentTimestamp  time.Time
	previousTimestamp time.Time
	size              uint64
	txTotals          []uint64
	txSizes           []int

	cfg      Config
	log      *logging.Logger
	cancelFn func()
	rates    *ratelimit.Rates

	// service injection
	assets     Assets
	banking    Banking
	broker     Broker
	cmd        Commander
	erc        ExtResChecker
	evtfwd     EvtForwarder
	exec       ExecutionEngine
	ghandler   *genesis.Handler
	gov        GovernanceEngine
	notary     Notary
	stats      Stats
	time       TimeService
	top        ValidatorTopology
	vegaWallet nodewallet.Wallet
}

func NewApp(
	log *logging.Logger,
	config Config,
	cancelFn func(),
	assets Assets,
	banking Banking,
	broker Broker,
	erc ExtResChecker,
	evtfwd EvtForwarder,
	exec ExecutionEngine,
	cmd Commander,
	ghandler *genesis.Handler,
	gov GovernanceEngine,
	notary Notary,
	stats Stats,
	time TimeService,
	top ValidatorTopology,
	wallet Wallet,
) (*App, error) {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	vegaWallet, ok := wallet.Get(nodewallet.Vega)
	if !ok {
		return nil, ErrVegaWalletRequired
	}

	app := &App{
		abci: abci.New(&codec{}),

		log:      log,
		cfg:      config,
		cancelFn: cancelFn,
		rates: ratelimit.New(
			config.Ratelimit.Requests,
			config.Ratelimit.PerNBlocks,
		),
		assets:     assets,
		banking:    banking,
		broker:     broker,
		cmd:        cmd,
		erc:        erc,
		evtfwd:     evtfwd,
		exec:       exec,
		ghandler:   ghandler,
		gov:        gov,
		notary:     notary,
		stats:      stats,
		time:       time,
		top:        top,
		vegaWallet: vegaWallet,
	}

	// setup handlers
	app.abci.OnInitChain = app.OnInitChain
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnCheckTx = app.OnCheckTx
	app.abci.OnDeliverTx = app.OnDeliverTx

	app.abci.
		HandleCheckTx(txn.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.NodeVoteCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.ChainEventCommand, app.RequireValidatorPubKey)

	app.abci.
		HandleDeliverTx(txn.SubmitOrderCommand, app.DeliverSubmitOrder).
		HandleDeliverTx(txn.CancelOrderCommand, app.DeliverCancelOrder).
		HandleDeliverTx(txn.AmendOrderCommand, app.DeliverAmendOrder).
		HandleDeliverTx(txn.WithdrawCommand, addDeterministicID(app.DeliverWithdraw)).
		HandleDeliverTx(txn.ProposeCommand, addDeterministicID(app.DeliverPropose)).
		HandleDeliverTx(txn.VoteCommand, app.DeliverVote).
		HandleDeliverTx(txn.NodeSignatureCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeSignature)).
		HandleDeliverTx(txn.LiquidityProvisionCommand, addDeterministicID(app.DeliverLiquidityProvision)).
		HandleDeliverTx(txn.NodeVoteCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeVote)).
		HandleDeliverTx(txn.ChainEventCommand,
			app.RequireValidatorPubKeyW(addDeterministicID(app.DeliverChainEvent)))

	app.time.NotifyOnTick(app.onTick)

	return app, nil
}

// addDeteremisticID will build the command id and .
// the command id is built using the signature of the proposer of the command
// the signature is then hashed with sha3_256
// the hash is the hex string encoded
func addDeterministicID(
	f func(context.Context, abci.Tx, string) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		return f(ctx, tx, hex.EncodeToString(crypto.Hash(tx.Signature())))
	}
}

func (app *App) RequireValidatorPubKeyW(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := app.RequireValidatorPubKey(ctx, tx); err != nil {
			return err
		}
		return f(ctx, tx)
	}
}

// ReloadConf updates the internal configuration
func (a *App) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	a.cfg = cfg
}

func (app *App) Abci() *abci.App {
	return app.abci
}

func (app *App) cancel() {
	if fn := app.cancelFn; fn != nil {
		fn()
	}
}

func (app *App) OnInitChain(req tmtypes.RequestInitChain) tmtypes.ResponseInitChain {
	hash := hex.EncodeToString(crypto.Hash(req.AppStateBytes))
	// let's assume genesis block is block 0
	ctx := contextutil.WithBlockHeight(context.Background(), 0)
	ctx = contextutil.WithTraceID(ctx, hash)

	vators := make([][]byte, 0, len(req.Validators))
	// get just the pubkeys out of the validator list
	for _, v := range req.Validators {
		if len(v.PubKey.Data) > 0 {
			vators = append(vators, v.PubKey.Data)
		}
	}

	app.top.UpdateValidatorSet(vators)
	if err := app.ghandler.OnGenesis(ctx, req.Time, req.AppStateBytes, vators); err != nil {
		app.log.Fatal("something happened when initializing vega with the genesis block", logging.Error(err))
	}

	return tmtypes.ResponseInitChain{}

}

// OnBeginBlock updates the internal lastBlockTime value with each new block
func (app *App) OnBeginBlock(req tmtypes.RequestBeginBlock) (ctx context.Context, resp tmtypes.ResponseBeginBlock) {
	hash := hex.EncodeToString(req.Hash)
	ctx = contextutil.WithBlockHeight(contextutil.WithTraceID(context.Background(), hash), req.Header.Height)

	now := req.Header.Time
	app.time.SetTimeNow(ctx, now)

	app.rates.NextBlock()

	var err error
	if app.currentTimestamp, err = app.time.GetTimeNow(); err != nil {
		app.cancel()
		return
	}

	if app.previousTimestamp, err = app.time.GetTimeLastBatch(); err != nil {
		app.cancel()
		return
	}

	app.log.Debug("ABCI service BEGIN completed",
		logging.Int64("current-timestamp", app.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", app.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(app.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(app.previousTimestamp)),
	)
	return
}

func (app *App) OnCommit() (resp tmtypes.ResponseCommit) {
	app.log.Debug("Processor COMMIT starting")
	defer app.log.Debug("Processor COMMIT completed")

	// Compute the AppHash and update the response
	resp.Data = app.exec.Hash()

	app.updateStats()
	app.setBatchStats()

	return resp
}

// OnCheckTxHandler performs validations like ratelimiting
func (app *App) OnCheckTx(ctx context.Context, _ tmtypes.RequestCheckTx, tx abci.Tx) (context.Context, tmtypes.ResponseCheckTx) {
	resp := tmtypes.ResponseCheckTx{}

	// Check ratelimits
	if app.limitPubkey(tx.PubKey()) {
		resp.Code = abci.AbciTxnValidationFailure
	}

	return ctx, resp
}

// limitPubkey returns whether a request should be rate limited or not
func (app *App) limitPubkey(pk []byte) bool {
	// Do not rate limit validators nodes.
	if app.top.Exists(pk) {
		return false
	}

	key := ratelimit.Key(pk).String()
	if !app.rates.Allow(key) {
		app.log.Error("Rate limit exceeded", logging.String("key", key))
		return true
	}

	app.log.Debug("RateLimit allowance", logging.String("key", key), logging.Int("count", app.rates.Count(key)))
	return false
}

// OnDeliverTx increments the internal tx counter and decorates the context with tracing information.
func (app *App) OnDeliverTx(ctx context.Context, req tmtypes.RequestDeliverTx, tx abci.Tx) (context.Context, tmtypes.ResponseDeliverTx) {
	app.size++
	app.setTxStats(len(req.Tx))

	// we don't need to set trace ID on context, it's been handled with OnBeginBlock
	return ctx, tmtypes.ResponseDeliverTx{}
}

func (app *App) RequireValidatorPubKey(ctx context.Context, tx abci.Tx) error {
	if !app.top.Exists(tx.PubKey()) {
		return ErrNodeSignatureFromNonValidator
	}
	return nil
}

func (app *App) DeliverSubmitOrder(ctx context.Context, tx abci.Tx) error {
	order := &types.OrderSubmission{}
	err := tx.(*Tx).Unmarshal(order)
	if err != nil {
		return err
	}
	party := hex.EncodeToString(tx.PubKey())

	app.stats.IncTotalCreateOrder()

	// Submit the create order request to the execution engine
	conf, err := app.exec.SubmitOrder(ctx, party, order)
	if conf != nil {
		if app.log.GetLevel() <= logging.DebugLevel {
			app.log.Debug("Order confirmed",
				logging.OrderSubmission(order),
				logging.OrderWithTag(*conf.Order, "aggressive-order"),
				logging.String("passive-trades", fmt.Sprintf("%+v", conf.Trades)),
				logging.String("passive-orders", fmt.Sprintf("%+v", conf.PassiveOrdersAffected)))
		}

		app.stats.AddCurrentTradesInBatch(uint64(len(conf.Trades)))
		app.stats.AddTotalTrades(uint64(len(conf.Trades)))
		app.stats.IncCurrentOrdersInBatch()
	}

	// increment total orders, even for failures so current ID strategy is valid.
	app.stats.IncTotalOrders()

	if err != nil && app.log.GetLevel() <= logging.DebugLevel {
		app.log.Debug("error message on creating order",
			logging.OrderSubmission(order),
			logging.Error(err))
	}

	return nil
}

func (app *App) DeliverCancelOrder(ctx context.Context, tx abci.Tx) error {
	order := &types.OrderCancellation{}
	if err := tx.Unmarshal(order); err != nil {
		return err
	}
	party := hex.EncodeToString(tx.PubKey())

	app.stats.IncTotalCancelOrder()
	app.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", order.OrderID))

	// Submit the cancel new order request to the Vega trading core
	msg, err := app.exec.CancelOrder(ctx, party, order)
	if err != nil {
		app.log.Error("error on cancelling order", logging.String("order-id", order.OrderID), logging.Error(err))
		return err
	}
	if app.cfg.LogOrderCancelDebug {
		for _, v := range msg {
			app.log.Debug("Order cancelled", logging.Order(*v.Order))
		}
	}

	return nil
}

func (app *App) DeliverAmendOrder(ctx context.Context, tx abci.Tx) error {
	order := &types.OrderAmendment{}
	if err := tx.Unmarshal(order); err != nil {
		return err
	}
	party := hex.EncodeToString(tx.PubKey())

	app.stats.IncTotalAmendOrder()
	app.log.Debug("Blockchain service received a AMEND ORDER request", logging.String("order-id", order.OrderID))

	// Submit the cancel new order request to the Vega trading core
	msg, err := app.exec.AmendOrder(ctx, party, order)
	if err != nil {
		app.log.Error("error on amending order", logging.String("order-id", order.OrderID), logging.Error(err))
		return err
	}
	if app.cfg.LogOrderAmendDebug {
		app.log.Debug("Order amended", logging.Order(*msg.Order))
	}

	return nil
}

func (app *App) DeliverWithdraw(
	ctx context.Context, tx abci.Tx, id string) error {
	w := &types.WithdrawSubmission{}
	if err := tx.Unmarshal(w); err != nil {
		return err
	}

	party := hex.EncodeToString(tx.PubKey())
	return app.processWithdraw(ctx, party, w, id)
}

func (app *App) DeliverPropose(ctx context.Context, tx abci.Tx, id string) error {
	prop := &types.Proposal{}
	if err := tx.Unmarshal(prop); err != nil {
		return err
	}

	app.log.Debug("Submitting proposal",
		logging.String("proposal-id", prop.ID),
		logging.String("proposal-reference", prop.Reference),
		logging.String("proposal-party", prop.PartyID),
		logging.String("proposal-terms", prop.Terms.String()))

	return app.gov.SubmitProposal(ctx, *prop, id)
}

func (app *App) DeliverVote(ctx context.Context, tx abci.Tx) error {
	vote := &types.Vote{}
	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	app.log.Debug("Voting on proposal",
		logging.String("proposal-id", vote.ProposalID),
		logging.String("vote-party", vote.PartyID),
		logging.String("vote-value", vote.Value.String()))

	vote.Timestamp = app.currentTimestamp.UnixNano()
	return app.gov.AddVote(ctx, *vote)
}

func (app *App) DeliverNodeSignature(ctx context.Context, tx abci.Tx) error {
	ns := &types.NodeSignature{}
	if err := tx.Unmarshal(ns); err != nil {
		return err
	}
	_, _, err := app.notary.AddSig(ctx, tx.PubKey(), *ns)
	return err
}

func (app *App) DeliverLiquidityProvision(ctx context.Context, tx abci.Tx, id string) error {
	sub := &types.LiquidityProvisionSubmission{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}

	partyID := hex.EncodeToString(tx.PubKey())
	return app.exec.SubmitLiquidityProvision(ctx, sub, partyID, id)
}

func (app *App) DeliverNodeVote(ctx context.Context, tx abci.Tx) error {
	vote := &types.NodeVote{}
	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	return app.erc.AddNodeCheck(ctx, vote)
}

func (app *App) DeliverChainEvent(ctx context.Context, tx abci.Tx, id string) error {
	ce := &types.ChainEvent{}
	if err := tx.Unmarshal(ce); err != nil {
		return err
	}

	return app.processChainEvent(ctx, ce, tx.PubKey(), id)
}

func (app *App) onTick(ctx context.Context, t time.Time) {
	acceptedProposals := app.gov.OnChainTimeUpdate(ctx, t)
	for _, toEnact := range acceptedProposals {
		prop := toEnact.Proposal()
		switch {
		case toEnact.IsNewMarket():
			app.enactMarket(ctx, prop, toEnact.NewMarket())
		case toEnact.IsNewAsset():
			app.enactAsset(ctx, prop, toEnact.NewAsset())
		case toEnact.IsUpdateMarket():
			app.log.Error("update market enactment is not implemented")
		case toEnact.IsUpdateNetwork():
			app.log.Error("update network enactment is not implemented")
		default:
			prop.State = types.Proposal_STATE_FAILED
			app.log.Error("unknown proposal cannot be enacted", logging.String("proposal-id", prop.ID))
		}
		app.broker.Send(events.NewProposalEvent(ctx, *prop))
	}
}

func (app *App) enactAsset(ctx context.Context, prop *types.Proposal, _ *types.Asset) {
	prop.State = types.Proposal_STATE_ENACTED
	// first check if this asset is real
	asset, err := app.assets.Get(prop.ID)
	if err != nil {
		// this should not happen
		app.log.Error("invalid asset is getting enacted",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.Proposal_STATE_FAILED
		return
	}

	// if this is a builtin asset nothing needs to be done, just start the asset
	// straigh away
	if asset.IsBuiltinAsset() {
		err = app.banking.EnableBuiltinAsset(ctx, asset.ProtoAsset().ID)
		if err != nil {
			// this should not happen
			app.log.Error("unable to get builtin asset enabled",
				logging.String("asset-id", prop.ID),
				logging.Error(err))
			prop.State = types.Proposal_STATE_FAILED
		}
		return
	}

	// then instruct the notary to start getting signature from validators
	if err := app.notary.StartAggregate(prop.ID, types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW); err != nil {
		prop.State = types.Proposal_STATE_FAILED
		app.log.Error("unable to enact proposal",
			logging.String("proposal-id", prop.ID),
			logging.Error(err))
		return
	}

	// if we are not a validator the job is done here
	if !app.top.IsValidator() {
		// nothing to do
		return
	}

	var sig []byte
	switch {
	case asset.IsERC20():
		asset, _ := asset.ERC20()
		_, sig, err = asset.SignBridgeWhitelisting()
	}
	if err != nil {
		app.log.Error("unable to sign whitelisting transaction",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.Proposal_STATE_FAILED
		return
	}
	payload := &types.NodeSignature{
		ID:   prop.ID,
		Sig:  sig,
		Kind: types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW,
	}
	if err := app.cmd.Command(ctx, txn.NodeSignatureCommand, payload); err != nil {
		// do nothing for now, we'll need a retry mechanism for this and all command soon
		app.log.Error("unable to send command for notary",
			logging.Error(err))
	}
}

func (app *App) enactMarket(ctx context.Context, prop *types.Proposal, mkt *types.Market) {
	prop.State = types.Proposal_STATE_ENACTED
	if err := app.exec.SubmitMarket(ctx, mkt); err != nil {
		prop.State = types.Proposal_STATE_FAILED
		app.log.Error("failed to submit new market",
			logging.String("market-id", mkt.Id),
			logging.Error(err))
	}
}
