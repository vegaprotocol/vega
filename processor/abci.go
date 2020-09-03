package processor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

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
	abci          *abci.App
	lastBlockTime time.Time

	Config
	log *logging.Logger

	// service injection
	assets  Assets
	banking Banking
	exec    ExecutionEngine
	stats   Stats
	top     ValidatorTopology
}

func NewApp(
	log *logging.Logger,
	config Config,
	assets Assets,
	banking Banking,
	exec ExecutionEngine,
	stats Stats,
	top ValidatorTopology,
) *App {
	app := &App{
		abci:          abci.New(&codec{}),
		lastBlockTime: time.Now(),

		log:    log,
		Config: config,

		assets:  assets,
		banking: banking,
		exec:    exec,
		stats:   stats,
		top:     top,
	}

	// setup handlers
	app.abci.OnBeginBlock = app.OnBeginBlock

	app.abci.
		HandleCheckTx(blockchain.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(blockchain.ChainEventCommand, app.RequireValidatorPubKey)

	app.abci.
		HandleDeliverTx(blockchain.SubmitOrderCommand, app.DeliverSubmitOrder).
		HandleDeliverTx(blockchain.CancelOrderCommand, app.DeliverCancelOrder).
		HandleDeliverTx(blockchain.WithdrawCommand, app.DeliverWithdraw)

	return app
}

// OnBeginBlock updates the internal lastBlockTime value with each new block
func (app *App) OnBeginBlock(tmtypes.RequestBeginBlock) (resp tmtypes.ResponseBeginBlock) {
	app.lastBlockTime = time.Now()
	return
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
	order.CreatedAt = app.lastBlockTime.UnixNano()

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
