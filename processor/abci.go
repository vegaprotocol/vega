package processor

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/protos/commands"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/genesis"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgtm "code.vegaprotocol.io/vega/libs/tm"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor/ratelimit"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/vegatime"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrPublicKeyCannotSubmitTransactionWithNoBalance  = errors.New("public key cannot submit transaction without balance")
	ErrTradingDisabled                                = errors.New("trading disabled")
	ErrNoTransactionAllowedDuringBootstrap            = errors.New("no transaction allowed during the bootstraping period")
	ErrMarketProposalDisabled                         = errors.New("market proposal disabled")
	ErrAssetProposalDisabled                          = errors.New("asset proposal disabled")
	ErrNonValidatorTransactionDisabledDuringBootstrap = errors.New("non validator transaction disabled during bootstrap")
	ErrCheckpointRestoreDisabledDuringBootstrap       = errors.New("checkpoint restore disaled during bootstrap")
	ErrAwaitingCheckpointRestore                      = errors.New("transactions not allowed while waiting for checkpoint restore")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/checkpoint_mock.go -package mocks code.vegaprotocol.io/vega/processor Checkpoint
type Checkpoint interface {
	BalanceCheckpoint(ctx context.Context) (*types.Snapshot, error)
	Checkpoint(ctx context.Context, now time.Time) (*types.Snapshot, error)
	Load(ctx context.Context, snap *types.Snapshot) error
	AwaitingRestore() bool
}

type SpamEngine interface {
	EndOfBlock(blockHeight uint64)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
}

type App struct {
	abci              *abci.App
	currentTimestamp  time.Time
	previousTimestamp time.Time
	txTotals          []uint64
	txSizes           []int
	cBlock            string
	blockCtx          context.Context // use this to have access to block hash + height in commit call
	reloadCP          bool

	cfg      Config
	log      *logging.Logger
	cancelFn func()
	rates    *ratelimit.Rates

	// service injection
	assets          Assets
	banking         Banking
	broker          Broker
	cmd             Commander
	witness         Witness
	evtfwd          EvtForwarder
	exec            ExecutionEngine
	ghandler        *genesis.Handler
	gov             GovernanceEngine
	notary          Notary
	stats           Stats
	time            TimeService
	top             ValidatorTopology
	netp            NetworkParameters
	oracles         *Oracle
	delegation      DelegationEngine
	limits          Limits
	stake           StakeVerifier
	stakingAccounts StakingAccounts
	checkpoint      Checkpoint
	spam            SpamEngine
}

func NewApp(
	log *logging.Logger,
	config Config,
	cancelFn func(),
	assets Assets,
	banking Banking,
	broker Broker,
	witness Witness,
	evtfwd EvtForwarder,
	exec ExecutionEngine,
	cmd Commander,
	ghandler *genesis.Handler,
	gov GovernanceEngine,
	notary Notary,
	stats Stats,
	time TimeService,
	epoch EpochService,
	top ValidatorTopology,
	netp NetworkParameters,
	oracles *Oracle,
	delegation DelegationEngine,
	limits Limits,
	stake StakeVerifier,
	stakingAccounts StakingAccounts,
	checkpoint Checkpoint,
	spam SpamEngine,
) *App {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	if err := vgfs.EnsureDir(config.CheckpointsPath); err != nil {
		log.Panic("Could not create checkpoints directory",
			logging.String("checkpoint-dir", config.CheckpointsPath),
			logging.Error(err))
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
		reloadCP:        checkpoint.AwaitingRestore(),
		assets:          assets,
		banking:         banking,
		broker:          broker,
		cmd:             cmd,
		witness:         witness,
		evtfwd:          evtfwd,
		exec:            exec,
		ghandler:        ghandler,
		gov:             gov,
		notary:          notary,
		stats:           stats,
		time:            time,
		top:             top,
		netp:            netp,
		oracles:         oracles,
		delegation:      delegation,
		limits:          limits,
		stake:           stake,
		stakingAccounts: stakingAccounts,
		checkpoint:      checkpoint,
		spam:            spam,
	}

	// setup handlers
	app.abci.OnInitChain = app.OnInitChain
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnEndBlock = app.OnEndBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnCheckTx = app.OnCheckTx
	app.abci.OnDeliverTx = app.OnDeliverTx

	app.abci.
		HandleCheckTx(txn.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.NodeVoteCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.ChainEventCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.SubmitOracleDataCommand, app.CheckSubmitOracleData)

	app.abci.
		HandleDeliverTx(txn.SubmitOrderCommand,
			app.SendEventOnError(app.DeliverSubmitOrder)).
		HandleDeliverTx(txn.CancelOrderCommand,
			app.SendEventOnError(app.DeliverCancelOrder)).
		HandleDeliverTx(txn.AmendOrderCommand,
			app.SendEventOnError(app.DeliverAmendOrder)).
		HandleDeliverTx(txn.WithdrawCommand,
			app.SendEventOnError(addDeterministicID(app.DeliverWithdraw))).
		HandleDeliverTx(txn.ProposeCommand,
			app.SendEventOnError(addDeterministicID(app.DeliverPropose))).
		HandleDeliverTx(txn.VoteCommand,
			app.SendEventOnError(app.DeliverVote)).
		HandleDeliverTx(txn.NodeSignatureCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeSignature)).
		HandleDeliverTx(txn.LiquidityProvisionCommand,
			app.SendEventOnError(addDeterministicID(app.DeliverLiquidityProvision))).
		HandleDeliverTx(txn.NodeVoteCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeVote)).
		HandleDeliverTx(txn.ChainEventCommand,
			app.RequireValidatorPubKeyW(addDeterministicID(app.DeliverChainEvent))).
		HandleDeliverTx(txn.SubmitOracleDataCommand, app.DeliverSubmitOracleData).
		HandleDeliverTx(txn.DelegateCommand,
			app.SendEventOnError(app.DeliverDelegate)).
		HandleDeliverTx(txn.UndelegateCommand,
			app.SendEventOnError(app.DeliverUndelegate)).
		HandleDeliverTx(txn.CheckpointRestoreCommand,
			app.SendEventOnError(app.DeliverReloadSnapshot))

	app.time.NotifyOnTick(app.onTick)

	return app
}

// addDeterministicID will build the command id and .
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

func (app *App) SendEventOnError(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := f(ctx, tx); err != nil {
			app.broker.Send(events.NewTxErrEvent(ctx, err, tx.Party(), tx.GetCmd()))
			return err
		}
		return nil
	}
}

// ReloadConf updates the internal configuration
func (app *App) ReloadConf(cfg Config) {
	app.log.Info("reloading configuration")
	if app.log.GetLevel() != cfg.Level.Get() {
		app.log.Info("updating log level",
			logging.String("old", app.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		app.log.SetLevel(cfg.Level.Get())
	}

	app.cfg = cfg
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
	ctx := vgcontext.WithBlockHeight(context.Background(), 0)
	ctx = vgcontext.WithTraceID(ctx, hash)
	app.blockCtx = ctx

	vators := make([]string, 0, len(req.Validators))
	// get just the pubkeys out of the validator list
	for _, v := range req.Validators {
		if len(v.PubKey.GetEd25519()) > 0 {
			vators = append(vators, vgtm.PubKeyToString(v.PubKey))
		}
	}

	app.top.UpdateValidatorSet(vators)
	if err := app.ghandler.OnGenesis(ctx, req.Time, req.AppStateBytes); err != nil {
		app.cancel()
		app.log.Panic("something happened when initializing vega with the genesis block", logging.Error(err))
	}

	return tmtypes.ResponseInitChain{}
}

func (app *App) OnEndBlock(req tmtypes.RequestEndBlock) (ctx context.Context, resp tmtypes.ResponseEndBlock) {
	app.log.Debug("ABCI service END block completed",
		logging.Int64("current-timestamp", app.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", app.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(app.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(app.previousTimestamp)),
	)

	if app.spam != nil {
		app.spam.EndOfBlock(uint64(req.Height))
	}
	return
}

// OnBeginBlock updates the internal lastBlockTime value with each new block
func (app *App) OnBeginBlock(req tmtypes.RequestBeginBlock) (ctx context.Context, resp tmtypes.ResponseBeginBlock) {
	hash := hex.EncodeToString(req.Hash)
	app.cBlock = hash
	ctx = vgcontext.WithBlockHeight(vgcontext.WithTraceID(context.Background(), hash), req.Header.Height)
	app.blockCtx = ctx

	now := req.Header.Time

	app.time.SetTimeNow(ctx, now)
	app.rates.NextBlock()
	app.currentTimestamp = app.time.GetTimeNow()
	app.previousTimestamp = app.time.GetTimeLastBatch()

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

	resp.Data = app.exec.Hash()
	// Snapshot can be nil if it wasn't time to create a snapshot
	if snap, _ := app.checkpoint.Checkpoint(app.blockCtx, app.currentTimestamp); snap != nil {
		resp.Data = append(resp.Data, snap.Hash...)
		_ = app.handleCheckpoint(snap)
	}
	// Compute the AppHash and update the response

	app.updateStats()
	app.setBatchStats()

	return resp
}

func (app *App) handleCheckpoint(snap *types.Snapshot) error {
	f, err := os.Create(
		filepath.Join(
			app.cfg.CheckpointsPath,
			fmt.Sprintf(
				"%s-%s.cp", app.cBlock, hex.EncodeToString(snap.Hash),
			),
		),
	)
	if err != nil {
		return err
	}
	defer f.Close()
	// write data
	if _, err = f.Write(snap.State); err != nil {
		return err
	}
	// emit the event indicating a new checkpoint was created
	// this function is called both for interval checkpoints and withdrawal checkpoints
	event := events.NewCheckpointEvent(app.blockCtx, snap)
	app.broker.Send(event)
	return nil
}

// OnCheckTx performs soft validations.
func (app *App) OnCheckTx(ctx context.Context, _ tmtypes.RequestCheckTx, tx abci.Tx) (context.Context, tmtypes.ResponseCheckTx) {
	resp := tmtypes.ResponseCheckTx{}

	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = abci.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
		return ctx, resp
	}

	// Check ratelimits
	// FIXME(): temporary disable all rate limiting
	_, isval := app.limitPubkey(tx.PubKeyHex())
	if isval {
		return ctx, resp
	}

	if app.spam != nil {
		if _, err := app.spam.PreBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = abci.AbciSpamError
			resp.Data = []byte(err.Error())
			return ctx, resp
		}
	}

	return ctx, resp
}

// limitPubkey returns whether a request should be rate limited or not
func (app *App) limitPubkey(pk string) (limit bool, isValidator bool) {
	// Do not rate limit validators nodes.
	if app.top.Exists(pk) {
		return false, true
	}

	key := ratelimit.Key(pk).String()
	if !app.rates.Allow(key) {
		app.log.Debug("Rate limit exceeded", logging.String("key", key))
		return true, false
	}

	app.log.Debug("RateLimit allowance", logging.String("key", key), logging.Int("count", app.rates.Count(key)))
	return false, false
}

func (app *App) canSubmitTx(tx abci.Tx) (err error) {
	defer func() {
		if err != nil {
			app.log.Error("cannot submit transaction", logging.Error(err))
		}
	}()

	// are we in a bootstrapping period?
	if !app.limits.BootstrapFinished() {
		// only validators can send transaction at this point.
		party := tx.Party()
		if !app.top.Exists(party) {
			return ErrNoTransactionAllowedDuringBootstrap
		}
		cmd := tx.Command()
		// make sure this is a validator command and not a checkpoint.
		// checkpoints are only allow when the bootstrap period is done.
		if !cmd.IsValidatorCommand() {
			return ErrNonValidatorTransactionDisabledDuringBootstrap
		}
		if cmd == txn.CheckpointRestoreCommand {
			return ErrCheckpointRestoreDisabledDuringBootstrap
		}
	}

	switch tx.Command() {
	case txn.WithdrawCommand:
		if app.reloadCP {
			// we haven't reloaded the collateral data, withdrawals are going to fail
			return ErrAwaitingCheckpointRestore
		}
	case txn.SubmitOrderCommand, txn.AmendOrderCommand, txn.CancelOrderCommand, txn.LiquidityProvisionCommand:
		if !app.limits.CanTrade() {
			return ErrTradingDisabled
		}
		if app.reloadCP {
			return ErrAwaitingCheckpointRestore
		}
	case txn.ProposeCommand:
		if app.reloadCP {
			return ErrAwaitingCheckpointRestore
		}
		praw := &commandspb.ProposalSubmission{}
		if err := tx.Unmarshal(praw); err != nil {
			return fmt.Errorf("could not unmarshal proposal submission: %w", err)
		}
		p := types.NewProposalSubmissionFromProto(praw)
		if p.Terms == nil {
			return errors.New("invalid proposal submission")
		}
		switch p.Terms.Change.GetTermType() {
		case types.ProposalTerms_NEW_MARKET:
			if !app.limits.CanProposeMarket() {
				return ErrMarketProposalDisabled
			}
		case types.ProposalTerms_NEW_ASSET:
			if !app.limits.CanProposeAsset() {
				return ErrAssetProposalDisabled
			}
		}
	}
	return nil
}

// OnDeliverTx increments the internal tx counter and decorates the context with tracing information.
func (app *App) OnDeliverTx(ctx context.Context, req tmtypes.RequestDeliverTx, tx abci.Tx) (context.Context, tmtypes.ResponseDeliverTx) {
	app.setTxStats(len(req.Tx))

	var resp tmtypes.ResponseDeliverTx
	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = abci.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
	}

	if app.spam != nil {
		if _, err := app.spam.PostBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = abci.AbciSpamError
			resp.Data = []byte(err.Error())
		}
	}

	// we don't need to set trace ID on context, it's been handled with OnBeginBlock
	return ctx, resp
}

func (app *App) RequireValidatorPubKey(ctx context.Context, tx abci.Tx) error {
	if !app.top.Exists(tx.PubKeyHex()) {
		return ErrNodeSignatureFromNonValidator
	}
	return nil
}

func (app *App) DeliverSubmitOrder(ctx context.Context, tx abci.Tx) error {
	s := &commandspb.OrderSubmission{}
	if err := tx.Unmarshal(s); err != nil {
		return err
	}

	app.stats.IncTotalCreateOrder()

	// Convert from proto to domain type
	os, err := types.NewOrderSubmissionFromProto(s)
	if err != nil {
		return err
	}
	// Submit the create order request to the execution engine
	conf, err := app.exec.SubmitOrder(ctx, os, tx.Party())
	if conf != nil {
		if app.log.GetLevel() <= logging.DebugLevel {
			app.log.Debug("Order confirmed",
				logging.OrderSubmission(os),
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
			logging.OrderSubmission(os),
			logging.Error(err))
	}

	return nil
}

func (app *App) DeliverCancelOrder(ctx context.Context, tx abci.Tx) error {
	porder := &commandspb.OrderCancellation{}
	if err := tx.Unmarshal(porder); err != nil {
		return err
	}

	app.stats.IncTotalCancelOrder()
	app.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", porder.OrderId))

	order := types.OrderCancellationFromProto(porder)
	// Submit the cancel new order request to the Vega trading core
	msg, err := app.exec.CancelOrder(ctx, order, tx.Party())
	if err != nil {
		app.log.Error("error on cancelling order", logging.String("order-id", order.OrderId), logging.Error(err))
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
	order := &commandspb.OrderAmendment{}
	if err := tx.Unmarshal(order); err != nil {
		return err
	}

	app.stats.IncTotalAmendOrder()
	app.log.Debug("Blockchain service received a AMEND ORDER request", logging.String("order-id", order.OrderId))

	// Convert protobuf into local domain type
	oa, err := types.NewOrderAmendmentFromProto(order)
	if err != nil {
		return err
	}

	// Submit the cancel new order request to the Vega trading core
	msg, err := app.exec.AmendOrder(ctx, oa, tx.Party())
	if err != nil {
		app.log.Error("error on amending order", logging.String("order-id", order.OrderId), logging.Error(err))
		return err
	}
	if app.cfg.LogOrderAmendDebug {
		app.log.Debug("Order amended", logging.Order(*msg.Order))
	}

	return nil
}

func (app *App) DeliverWithdraw(
	ctx context.Context, tx abci.Tx, id string) error {
	w := &commandspb.WithdrawSubmission{}
	if err := tx.Unmarshal(w); err != nil {
		return err
	}

	// Convert protobuf to local domain type
	ws, err := types.NewWithdrawSubmissionFromProto(w)
	if err != nil {
		return err
	}
	if err := app.processWithdraw(ctx, ws, id, tx.Party()); err != nil {
		return err
	}
	snap, err := app.checkpoint.BalanceCheckpoint(ctx)
	if err != nil {
		return err
	}
	return app.handleCheckpoint(snap)
}

func (app *App) DeliverPropose(ctx context.Context, tx abci.Tx, id string) error {
	prop := &commandspb.ProposalSubmission{}
	if err := tx.Unmarshal(prop); err != nil {
		return err
	}

	party := tx.Party()

	if app.log.GetLevel() <= logging.DebugLevel {
		app.log.Debug("submitting proposal",
			logging.ProposalID(id),
			logging.String("proposal-reference", prop.Reference),
			logging.String("proposal-party", party),
			logging.String("proposal-terms", prop.Terms.String()))
	}

	propSubmission := types.NewProposalSubmissionFromProto(prop)
	toSubmit, err := app.gov.SubmitProposal(ctx, *propSubmission, id, party)
	if err != nil {
		app.log.Debug("could not submit proposal",
			logging.ProposalID(id),
			logging.Error(err))
		return err
	}

	if toSubmit.IsNewMarket() {
		nm := toSubmit.NewMarket()

		// TODO(): for now we are using a hash of the market ID to create
		// the lp provision ID (well it's still deterministic...)
		lpid := hex.EncodeToString(crypto.Hash([]byte(nm.Market().ID)))
		err := app.exec.SubmitMarketWithLiquidityProvision(
			ctx, nm.Market(), nm.LiquidityProvisionSubmission(), party, lpid)
		if err != nil {
			app.log.Debug("unable to submit new market with liquidity submission",
				logging.ProposalID(nm.Market().ID),
				logging.Error(err))
			// an error happened when submitting the market + liquidity
			// we should cancel this proposal now
			if err := app.gov.RejectProposal(ctx, toSubmit.Proposal(), types.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET, err); err != nil {
				// this should never happen
				app.log.Panic("tried to reject an non-existing proposal",
					logging.String("proposal-id", toSubmit.Proposal().ID),
					logging.Error(err))
			}
			return err
		}
	}

	return nil
}

func (app *App) DeliverVote(ctx context.Context, tx abci.Tx) error {
	vote := &commandspb.VoteSubmission{}

	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	party := tx.Party()
	app.log.Debug("Voting on proposal",
		logging.String("proposal-id", vote.ProposalId),
		logging.String("vote-party", party),
		logging.String("vote-value", vote.Value.String()))

	if err := commands.CheckVoteSubmission(vote); err != nil {
		return err
	}

	v := types.NewVoteSubmissionFromProto(vote)

	return app.gov.AddVote(ctx, *v, party)
}

func (app *App) DeliverNodeSignature(ctx context.Context, tx abci.Tx) error {
	ns := &commandspb.NodeSignature{}
	if err := tx.Unmarshal(ns); err != nil {
		return err
	}
	_, _, err := app.notary.AddSig(ctx, tx.PubKeyHex(), *ns)
	return err
}

func (app *App) DeliverLiquidityProvision(ctx context.Context, tx abci.Tx, id string) error {
	sub := &commandspb.LiquidityProvisionSubmission{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}

	// Convert protobuf message to local domain type
	lps, err := types.LiquidityProvisionSubmissionFromProto(sub)
	if err != nil {
		if app.log.GetLevel() <= logging.DebugLevel {
			app.log.Debug("Unable to convert LiquidityProvisionSubmission protobuf message to domain type",
				logging.LiquidityProvisionSubmissionProto(sub), logging.Error(err))
		}
		return err
	}

	partyID := tx.Party()
	return app.exec.SubmitLiquidityProvision(ctx, lps, partyID, id)
}

func (app *App) DeliverNodeVote(ctx context.Context, tx abci.Tx) error {
	vote := &commandspb.NodeVote{}
	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	return app.witness.AddNodeCheck(ctx, vote)
}

func (app *App) DeliverChainEvent(ctx context.Context, tx abci.Tx, id string) error {
	ce := &commandspb.ChainEvent{}
	if err := tx.Unmarshal(ce); err != nil {
		return err
	}

	return app.processChainEvent(ctx, ce, tx.PubKeyHex(), id)
}

func (app *App) DeliverSubmitOracleData(ctx context.Context, tx abci.Tx) error {
	data := &commandspb.OracleDataSubmission{}
	if err := tx.Unmarshal(data); err != nil {
		return err
	}

	oracleData, err := app.oracles.Adaptors.Normalise(*data)
	if err != nil {
		return err
	}

	return app.oracles.Engine.BroadcastData(ctx, *oracleData)
}

func (app *App) CheckSubmitOracleData(_ context.Context, tx abci.Tx) error {
	data := &commandspb.OracleDataSubmission{}
	if err := tx.Unmarshal(data); err != nil {
		return err
	}

	_, err := app.oracles.Adaptors.Normalise(*data)
	return err
}

func (app *App) onTick(ctx context.Context, t time.Time) {
	if app.reloadCP {
		app.log.Debug("This would call on chain time update for governance. We've skipped all tx, so just ignore")
		return
	}
	toEnactProposals, voteClosedProposals := app.gov.OnChainTimeUpdate(ctx, t)
	for _, voteClosed := range voteClosedProposals {
		prop := voteClosed.Proposal()
		switch {
		case voteClosed.IsNewMarket():
			// Here we panic in both case as we should never reach a point
			// where we try to Reject or start the opening auction of a
			// non-existing market or any other error would be quite critical
			// anyway...
			nm := voteClosed.NewMarket()
			if nm.Rejected() {
				if err := app.exec.RejectMarket(ctx, prop.ID); err != nil {
					app.log.Panic("unable to reject market",
						logging.String("market-id", prop.ID),
						logging.Error(err))
				}
			} else if nm.StartAuction() {
				if err := app.exec.StartOpeningAuction(ctx, prop.ID); err != nil {
					app.log.Panic("unable to start market opening auction",
						logging.String("market-id", prop.ID),
						logging.Error(err))
				}
			}
		}
	}

	for _, toEnact := range toEnactProposals {
		prop := toEnact.Proposal()
		switch {
		case toEnact.IsNewMarket():
			app.enactMarket(ctx, prop)
		case toEnact.IsNewAsset():
			app.enactAsset(ctx, prop, toEnact.NewAsset())
		case toEnact.IsUpdateMarket():
			app.log.Error("update market enactment is not implemented")
		case toEnact.IsUpdateNetworkParameter():
			app.enactNetworkParameterUpdate(ctx, prop, toEnact.UpdateNetworkParameter())
		default:
			prop.State = types.ProposalStateFailed
			app.log.Error("unknown proposal cannot be enacted", logging.ProposalID(prop.ID))
		}
		app.broker.Send(events.NewProposalEvent(ctx, *prop))
	}

}

func (app *App) enactAsset(ctx context.Context, prop *types.Proposal, _ *types.Asset) {
	prop.State = types.ProposalStateEnacted
	// first check if this asset is real
	asset, err := app.assets.Get(prop.ID)
	if err != nil {
		// this should not happen
		app.log.Error("invalid asset is getting enacted",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.ProposalStateFailed
		return
	}

	// if this is a builtin asset nothing needs to be done, just start the asset
	// straight away
	if asset.IsBuiltinAsset() {
		err = app.banking.EnableBuiltinAsset(ctx, asset.Type().ID)
		if err != nil {
			// this should not happen
			app.log.Error("unable to get builtin asset enabled",
				logging.String("asset-id", prop.ID),
				logging.Error(err))
			prop.State = types.ProposalStateFailed
		}
		return
	}

	// then instruct the notary to start getting signature from validators
	if err := app.notary.StartAggregate(prop.ID, types.NodeSignatureKindAssetNew); err != nil {
		prop.State = types.ProposalStateFailed
		app.log.Error("unable to enact proposal",
			logging.ProposalID(prop.ID),
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
		_, sig, err = asset.SignBridgeListing()
	}
	if err != nil {
		app.log.Error("unable to sign allowlisting transaction",
			logging.String("asset-id", prop.ID),
			logging.Error(err))
		prop.State = types.ProposalStateFailed
		return
	}
	payload := &commandspb.NodeSignature{
		Id:   prop.ID,
		Sig:  sig,
		Kind: commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW,
	}

	// no callbacks needed there, core should retry if nothging
	// is received back at some point
	app.cmd.Command(ctx, txn.NodeSignatureCommand, payload, nil)
}

func (app *App) enactMarket(ctx context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted

	// TODO: add checks for end of auction in here
}

func (app *App) enactNetworkParameterUpdate(ctx context.Context, prop *types.Proposal, np *types.NetworkParameter) {
	prop.State = types.ProposalStateEnacted
	if err := app.netp.Update(ctx, np.Key, np.Value); err != nil {
		prop.State = types.ProposalStateFailed
		app.log.Error("failed to update network parameters",
			logging.ProposalID(prop.ID),
			logging.Error(err))
		return
	}

	// we call the dispatch updates here then
	// just so we are sure all netparams updates are dispatches one by one
	// in a deterministic order
	app.netp.DispatchChanges(ctx)
}

func (app *App) DeliverDelegate(ctx context.Context, tx abci.Tx) (err error) {
	if app.reloadCP {
		app.log.Debug("Skipping transaction while waiting for checkpoint restore")
		return nil
	}
	ce := &commandspb.DelegateSubmission{}
	if err := tx.Unmarshal(ce); err != nil {
		return err
	}

	amount, overflowed := num.UintFromString(ce.Amount, 10)
	if overflowed {
		return errors.New("amount is not a valid base 10 number")
	}

	return app.delegation.Delegate(ctx, tx.Party(), ce.NodeId, amount)
}

func (app *App) DeliverUndelegate(ctx context.Context, tx abci.Tx) (err error) {
	if app.reloadCP {
		app.log.Debug("Skipping transaction while waiting for checkpoint restore")
		return nil
	}
	ce := &commandspb.UndelegateSubmission{}
	if err := tx.Unmarshal(ce); err != nil {
		return err
	}

	switch ce.Method {
	case commandspb.UndelegateSubmission_METHOD_NOW:
		return app.delegation.UndelegateNow(ctx, tx.Party(), ce.NodeId, num.Zero())
	case commandspb.UndelegateSubmission_METHOD_AT_END_OF_EPOCH:
		amount, overflowed := num.UintFromString(ce.Amount, 10)
		if overflowed {
			return errors.New("amount is not a valid base 10 number")
		}
		return app.delegation.UndelegateAtEndOfEpoch(ctx, tx.Party(), ce.NodeId, amount)
	default:
		return errors.New("unimplemented")
	}
}

func (app *App) DeliverReloadSnapshot(ctx context.Context, tx abci.Tx) (rerr error) {
	cmd := &commandspb.RestoreSnapshot{}
	defer func() {
		if rerr != nil {
			app.log.Error("Restoring checkpoint failed",
				logging.Error(rerr),
			)
			return
		}
		app.log.Info("Checkpoint restored!")
	}()

	if err := tx.Unmarshal(cmd); err != nil {
		return err
	}

	// convert to snapshot type:
	snap := &types.Snapshot{}
	if err := snap.SetState(cmd.Data); err != nil {
		return err
	}
	bh, err := snap.GetBlockHeight()
	if err != nil {
		app.log.Panic("Failed to get blockheight from checkpoint", logging.Error(err))
	}
	// ensure block height is set
	ctx = vgcontext.WithBlockHeight(ctx, bh)
	app.blockCtx = ctx
	err = app.checkpoint.Load(ctx, snap)
	if err != nil && err != types.ErrSnapshotStateInvalid && err != types.ErrSnapshotHashIncorrect {
		app.log.Panic("Failed to restore checkpoint", logging.Error(err))
	}
	// set flag in case the CP has been reloaded
	app.reloadCP = app.checkpoint.AwaitingRestore()
	// now we can call onTick for the governance engine updates, and enable the markets
	app.onTick(ctx, app.time.GetTimeNow())
	// @TODO if the snapshot hash was invalid, or its payload incorrect, the data was potentially tampered with
	// emit an error event perhaps, log, etc...?
	return err
}
