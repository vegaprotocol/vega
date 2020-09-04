package processor

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type codec struct {
}

func (c *codec) Decode(payload []byte) (abci.Tx, error) {
	bundle := &types.SignedBundle{}
	if err := proto.Unmarshal(payload, bundle); err != nil {
		return nil, fmt.Errorf("unable to unmarshal signed bundle: %w", err)
	}

	tx := &types.Transaction{}
	if err := proto.Unmarshal(bundle.Tx, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction from signed bundle: %w", err)
	}

	return NewTx(tx)
}

type App struct {
	abci              *abci.App
	currentTimestamp  time.Time
	previousTimestamp time.Time
	hasRegistered     bool
	size              uint64
	seenPayloads      map[string]struct{}

	Config
	log      *logging.Logger
	cancelFn func()

	// service injection
	assets     Assets
	banking    Banking
	cmd        Commander
	exec       ExecutionEngine
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
	exec ExecutionEngine,
	stats Stats,
	time TimeService,
	top ValidatorTopology,
) *App {
	app := &App{
		abci:         abci.New(&codec{}),
		seenPayloads: map[string]struct{}{},

		log:      log,
		Config:   config,
		cancelFn: cancelFn,

		assets:  assets,
		banking: banking,
		exec:    exec,
		stats:   stats,
		time:    time,
		top:     top,
	}

	// setup handlers
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnDeliverTx = app.OnDeliverTx

	app.abci.
		HandleCheckTx(blockchain.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(blockchain.ChainEventCommand, app.RequireValidatorPubKey)

	app.abci.
		HandleDeliverTx(blockchain.SubmitOrderCommand, app.DeliverSubmitOrder).
		HandleDeliverTx(blockchain.CancelOrderCommand, app.DeliverCancelOrder).
		HandleDeliverTx(blockchain.WithdrawCommand, app.DeliverWithdraw)

	return app
}

func (app *App) cancel() {
	if fn := app.cancelFn; fn != nil {
		fn()
	}
}

// OnBeginBlock updates the internal lastBlockTime value with each new block
func (app *App) OnBeginBlock(tmtypes.RequestBeginBlock) (resp tmtypes.ResponseBeginBlock) {
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

func (app *App) OnCommit(req tmtypes.RequestCommit) (resp tmtypes.ResponseCommit) {
	// Compute the AppHash
	resp.Data = make([]byte, 8)
	binary.BigEndian.PutUint64(resp.Data, uint64(app.size))

	app.log.Debug("Processor COMMIT starting")
	app.updateStats()

	if err := app.exec.Generate(); err != nil {
		app.log.Error("failure generating data in execution engine (commit)")
		return
	}
	app.log.Debug("Processor COMMIT completed")

	return
}

// OnDeliverTx increments the internal tx counter and decorates the context with tracing information.
func (app *App) OnDeliverTx(ctx context.Context, req tmtypes.RequestDeliverTx) (context.Context, tmtypes.ResponseDeliverTx) {
	app.size++

	tx := abci.TxFromContext(ctx)

	return contextutil.WithTraceID(
		ctx,
		hex.EncodeToString([]byte(tx.Hash())),
	), tmtypes.ResponseDeliverTx{}
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

	// Submit the create order request to the execution engine
	conf, err := app.exec.SubmitOrder(ctx, order)
	if conf != nil {
		app.log.Debug("Order confirmed",
			logging.Order(*order),
			logging.OrderWithTag(*conf.Order, "aggressive-order"),
			logging.String("passive-trades", fmt.Sprintf("%+v", conf.Trades)),
			logging.String("passive-orders", fmt.Sprintf("%+v", conf.PassiveOrdersAffected)))

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
	order, err := tx.(*Tx).asOrderCancellation()
	if err != nil {
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
	w, err := tx.(*Tx).asWithdraw()
	if err != nil {
		return err
	}

	asset, err := app.assets.Get(w.Asset)
	if err != nil {
		app.log.Error("invalid vega asset ID for withdrawal",
			logging.Error(err),
			logging.String("party-id", w.PartyID),
			logging.Uint64("amount", w.Amount),
			logging.String("asset-id", w.Asset))
		return err
	}

	switch {
	case asset.IsBuiltinAsset():
		return app.banking.WithdrawalBuiltinAsset(ctx, w.PartyID, w.Asset, w.Amount)
	case asset.IsERC20():
		return errors.New("unimplemented withdrawal for ERC20")
	}

	return errors.New("unimplemented withdrawal")
}
