package processor

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	tmtypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proto/crypto/keys"
)

type App struct {
	abci              *abci.App
	currentTimestamp  time.Time
	previousTimestamp time.Time
	hasRegistered     bool
	size              uint64

	Config
	log      *logging.Logger
	cancelFn func()
	idGen    *IDgenerator

	// service injection
	assets     Assets
	banking    Banking
	broker     Broker
	cmd        Commander
	col        Collateral
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
	col Collateral,
	ghandler *genesis.Handler,
	gov GovernanceEngine,
	notary Notary,
	stats Stats,
	time TimeService,
	top ValidatorTopology,
	wallet Wallet,
) (*App, error) {
	vegaWallet, ok := wallet.Get(nodewallet.Vega)
	if !ok {
		return nil, ErrVegaWalletRequired
	}

	app := &App{
		abci: abci.New(&codec{}),

		log:      log,
		Config:   config,
		cancelFn: cancelFn,
		idGen:    NewIDGen(),

		assets:     assets,
		banking:    banking,
		broker:     broker,
		cmd:        cmd,
		col:        col,
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
	app.abci.OnDeliverTx = app.OnDeliverTx

	app.abci.
		HandleCheckTx(blockchain.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(blockchain.ChainEventCommand, app.RequireValidatorPubKey)

	app.abci.
		HandleDeliverTx(blockchain.SubmitOrderCommand, app.DeliverSubmitOrder).
		HandleDeliverTx(blockchain.CancelOrderCommand, app.DeliverCancelOrder).
		HandleDeliverTx(blockchain.WithdrawCommand, app.DeliverWithdraw).
		HandleDeliverTx(blockchain.ProposeCommand, app.DeliverPropose).
		HandleDeliverTx(blockchain.VoteCommand, app.DeliverVote).
		HandleDeliverTx(blockchain.RegisterNodeCommand, app.DeliverRegisterNode).
		HandleDeliverTx(blockchain.NodeVoteCommand, app.DeliverNodeVote).
		HandleDeliverTx(blockchain.ChainEventCommand, app.DeliverChainEvent)

	app.time.NotifyOnTick(app.onTick)

	return app, nil
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
	vators := make([][]byte, 0, len(req.Validators))
	// get just the pubkeys out of the validator list
	for _, v := range req.Validators {
		var data []byte
		switch t := v.PubKey.Sum.(type) {
		case *keys.PublicKey_Ed25519:
			data = t.Ed25519
		}

		if len(data) > 0 {
			vators = append(vators, data)
		}
	}

	if err := app.ghandler.OnGenesis(req.Time, req.AppStateBytes, vators); err != nil {
		app.log.Error("something happened when initializing vega with the genesis block", logging.Error(err))
		panic(err)
	}

	return tmtypes.ResponseInitChain{}

}

// OnBeginBlock updates the internal lastBlockTime value with each new block
func (app *App) OnBeginBlock(req tmtypes.RequestBeginBlock) (resp tmtypes.ResponseBeginBlock) {
	hash := hex.EncodeToString(req.Hash)
	ctx := contextutil.WithTraceID(context.Background(), hash)

	now := req.Header.Time
	app.time.SetTimeNow(ctx, now)

	var err error
	if app.currentTimestamp, err = app.time.GetTimeNow(); err != nil {
		app.cancel()
		return
	}

	if app.previousTimestamp, err = app.time.GetTimeLastBatch(); err != nil {
		app.cancel()
		return
	}

	if !app.hasRegistered && app.top.IsValidator() && !app.top.Ready() {
		if pk := app.top.SelfChainPubKey(); pk != nil {
			payload := &types.NodeRegistration{
				ChainPubKey: pk,
				PubKey:      app.vegaWallet.PubKeyOrAddress(),
			}
			if err := app.cmd.Command(blockchain.RegisterNodeCommand, payload); err != nil {
				app.cancel()
				return
			}
			app.hasRegistered = true
		}
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
	resp.Data = make([]byte, 8)
	binary.BigEndian.PutUint64(resp.Data, uint64(app.size))

	app.updateStats()

	return resp
}

func (app *App) updateStats() {
	app.stats.IncTotalBatches()
	avg := app.stats.TotalOrders() / app.stats.TotalBatches()
	app.stats.SetAverageOrdersPerBatch(avg)
	duration := time.Duration(app.currentTimestamp.UnixNano() - app.previousTimestamp.UnixNano()).Seconds()
	var (
		currentOrders, currentTrades uint64
	)
	app.stats.SetBlockDuration(uint64(duration * float64(time.Second.Nanoseconds())))
	if duration > 0 {
		currentOrders, currentTrades = uint64(float64(app.stats.CurrentOrdersInBatch())/duration),
			uint64(float64(app.stats.CurrentTradesInBatch())/duration)
	}
	app.stats.SetOrdersPerSecond(currentOrders)
	app.stats.SetTradesPerSecond(currentTrades)
	// log stats
	app.log.Debug("Processor batch stats",
		logging.Int64("previousTimestamp", app.previousTimestamp.UnixNano()),
		logging.Int64("currentTimestamp", app.currentTimestamp.UnixNano()),
		logging.Float64("duration", duration),
		logging.Uint64("currentOrdersInBatch", app.stats.CurrentOrdersInBatch()),
		logging.Uint64("currentTradesInBatch", app.stats.CurrentTradesInBatch()),
		logging.Uint64("total-batches", app.stats.TotalBatches()),
		logging.Uint64("avg-orders-batch", avg),
		logging.Uint64("orders-per-sec", currentOrders),
		logging.Uint64("trades-per-sec", currentTrades),
	)
	app.stats.NewBatch() // sets previous batch orders/trades to current, zeroes current tally
}

// OnDeliverTx increments the internal tx counter and decorates the context with tracing information.
func (app *App) OnDeliverTx(ctx context.Context, req tmtypes.RequestDeliverTx, tx abci.Tx) (context.Context, tmtypes.ResponseDeliverTx) {
	app.size++

	// update the context with Tracing Info.
	hash := hex.EncodeToString(tx.Hash())
	ctx = contextutil.WithTraceID(ctx, hash)

	return ctx, tmtypes.ResponseDeliverTx{}
}

func (app *App) RequireValidatorPubKey(ctx context.Context, tx abci.Tx) error {
	if !app.top.Exists(tx.PubKey()) {
		return ErrNodeSignatureFromNonValidator
	}
	return nil
}

func (app *App) DeliverSubmitOrder(ctx context.Context, tx abci.Tx) error {
	order, err := tx.(*Tx).asOrderSubmission()
	if err != nil {
		return err
	}
	order.CreatedAt = app.currentTimestamp.UnixNano()
	app.stats.IncTotalCreateOrder()

	// Submit the create order request to the execution engine
	conf, err := app.exec.SubmitOrder(ctx, order)
	if conf != nil {
		if app.log.GetLevel() == logging.DebugLevel {
			app.log.Debug("Order confirmed",
				logging.Order(*order),
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

	if err != nil {
		app.log.Error("error message on creating order",
			logging.Order(*order),
			logging.Error(err))
	}

	return nil
}

func (app *App) DeliverCancelOrder(ctx context.Context, tx abci.Tx) error {
	order := &types.OrderCancellation{}
	if err := tx.(*Tx).Unmarshal(order); err != nil {
		return err
	}

	app.stats.IncTotalCancelOrder()
	app.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", order.OrderID))

	// Submit the cancel new order request to the Vega trading core
	msg, err := app.exec.CancelOrder(ctx, order)
	if err != nil {
		app.log.Error("error on cancelling order", logging.String("order-id", order.OrderID), logging.Error(err))
		return err
	}
	if app.LogOrderCancelDebug {
		for _, v := range msg {
			app.log.Debug("Order cancelled", logging.Order(*v.Order))
		}
	}

	return nil
}

func (app *App) DeliverWithdraw(ctx context.Context, tx abci.Tx) error {
	w := &types.WithdrawSubmission{}
	if err := tx.(*Tx).Unmarshal(w); err != nil {
		return err
	}

	return app.processWithdraw(ctx, w)
}

func (app *App) DeliverPropose(ctx context.Context, tx abci.Tx) error {
	prop := &types.Proposal{}
	if err := tx.(*Tx).Unmarshal(prop); err != nil {
		return err
	}

	app.log.Debug("Submitting proposal",
		logging.String("proposal-id", prop.ID),
		logging.String("proposal-reference", prop.Reference),
		logging.String("proposal-party", prop.PartyID),
		logging.String("proposal-terms", prop.Terms.String()))

	// TODO(JEREMY): use hash of the signature here.
	app.idGen.SetProposalID(prop)
	prop.Timestamp = app.currentTimestamp.UnixNano()

	return app.gov.SubmitProposal(ctx, *prop)
}

func (app *App) DeliverVote(ctx context.Context, tx abci.Tx) error {
	vote := &types.Vote{}
	if err := tx.(*Tx).Unmarshal(vote); err != nil {
		return err
	}

	app.log.Debug("Voting on proposal",
		logging.String("proposal-id", vote.ProposalID),
		logging.String("vote-party", vote.PartyID),
		logging.String("vote-value", vote.Value.String()))

	vote.Timestamp = app.currentTimestamp.UnixNano()
	return app.gov.AddVote(ctx, *vote)
}

func (app *App) DeliverRegisterNode(ctx context.Context, tx abci.Tx) error {
	node := &types.NodeRegistration{}
	if err := tx.(*Tx).Unmarshal(node); err != nil {
		return err
	}

	return app.top.AddNodeRegistration(node)
}

func (app *App) DeliverNodeVote(ctx context.Context, tx abci.Tx) error {
	vote := &types.NodeVote{}
	if err := tx.(*Tx).Unmarshal(vote); err != nil {
		return err
	}

	return app.erc.AddNodeCheck(ctx, vote)
}

func (app *App) DeliverChainEvent(ctx context.Context, tx abci.Tx) error {
	ce := &types.ChainEvent{}
	if err := tx.(*Tx).Unmarshal(ce); err != nil {
		return err
	}

	return app.processChainEvent(ctx, ce, tx.PubKey())
}

func (app *App) onTick(ctx context.Context, t time.Time) {
	app.idGen.NewBatch()
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
	if err := app.cmd.Command(blockchain.NodeSignatureCommand, payload); err != nil {
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
