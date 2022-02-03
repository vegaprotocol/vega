package processor

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/protos/commands"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/genesis"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgtm "code.vegaprotocol.io/vega/libs/tm"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor/ratelimit"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"code.vegaprotocol.io/vega/vegatime"

	tmtypes "github.com/tendermint/tendermint/abci/types"
	tmtypesint "github.com/tendermint/tendermint/types"
)

var (
	ErrPublicKeyCannotSubmitTransactionWithNoBalance  = errors.New("public key cannot submit transaction without balance")
	ErrTradingDisabled                                = errors.New("trading disabled")
	ErrNoTransactionAllowedDuringBootstrap            = errors.New("no transaction allowed during the bootstraping period")
	ErrMarketProposalDisabled                         = errors.New("market proposal disabled")
	ErrAssetProposalDisabled                          = errors.New("asset proposal disabled")
	ErrNonValidatorTransactionDisabledDuringBootstrap = errors.New("non validator transaction disabled during bootstrap")
	ErrCheckpointRestoreDisabledDuringBootstrap       = errors.New("checkpoint restore disabled during bootstrap")
	ErrAwaitingCheckpointRestore                      = errors.New("transactions not allowed while waiting for checkpoint restore")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/checkpoint_mock.go -package mocks code.vegaprotocol.io/vega/processor Checkpoint
type Checkpoint interface {
	BalanceCheckpoint(ctx context.Context) (*types.CheckpointState, error)
	Checkpoint(ctx context.Context, now time.Time) (*types.CheckpointState, error)
	Load(ctx context.Context, cpt *types.CheckpointState) error
	AwaitingRestore() bool
	ValidateCheckpoint(cpt *types.CheckpointState) error
}

type SpamEngine interface {
	EndOfBlock(blockHeight uint64)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
}

type Snapshot interface {
	Start() error
	Info() ([]byte, int64)
	List() ([]*types.Snapshot, error)
	ReceiveSnapshot(snap *types.Snapshot) error
	RejectSnapshot() error
	ApplySnapshotChunk(chunk *types.RawChunk) (bool, error)
	GetMissingChunks() []uint32
	ApplySnapshot(ctx context.Context) error
	LoadSnapshotChunk(height uint64, format, chunk uint32) (*types.RawChunk, error)
	AddProviders(provs ...types.StateProvider)
	Snapshot(ctx context.Context) ([]byte, error)
}

type StateVarEngine interface {
	ProposedValueReceived(ctx context.Context, ID, nodeID, eventID string, bundle *statevar.KeyValueBundle) error
}

type BlockchainClient interface {
	Validators(height *int64) ([]*tmtypesint.Validator, error)
}

type App struct {
	abci              *abci.App
	currentTimestamp  time.Time
	previousTimestamp time.Time
	txTotals          []uint64
	txSizes           []int
	cBlock            string
	chainCtx          context.Context // use this to have access to chain ID
	blockCtx          context.Context // use this to have access to block hash + height in commit call
	reloadCP          bool
	version           string
	blockchainClient  BlockchainClient

	vegaPaths paths.Paths
	cfg       Config
	log       *logging.Logger
	cancelFn  func()
	rates     *ratelimit.Rates

	// service injection
	assets          Assets
	banking         Banking
	broker          Broker
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
	rewards         RewardEngine
	limits          Limits
	stake           StakeVerifier
	stakingAccounts StakingAccounts
	checkpoint      Checkpoint
	spam            SpamEngine
	epoch           EpochService
	snapshot        Snapshot
	stateVar        StateVarEngine
	cpt             abci.Tx
}

func NewApp(
	log *logging.Logger,
	vegaPaths paths.Paths,
	config Config,
	cancelFn func(),
	assets Assets,
	banking Banking,
	broker Broker,
	witness Witness,
	evtfwd EvtForwarder,
	exec ExecutionEngine,
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
	checkpoint Checkpoint,
	spam SpamEngine,
	stakingAccounts StakingAccounts,
	rewards RewardEngine,
	snapshot Snapshot,
	stateVarEngine StateVarEngine,
	blockchainClient BlockchainClient,
	version string, // we need the version for snapshot reload
) *App {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	app := &App{
		abci: abci.New(&codec{}),

		log:       log,
		vegaPaths: vegaPaths,
		cfg:       config,
		cancelFn:  cancelFn,
		rates: ratelimit.New(
			config.Ratelimit.Requests,
			config.Ratelimit.PerNBlocks,
		),
		reloadCP:         checkpoint.AwaitingRestore(),
		assets:           assets,
		banking:          banking,
		broker:           broker,
		witness:          witness,
		evtfwd:           evtfwd,
		exec:             exec,
		ghandler:         ghandler,
		gov:              gov,
		notary:           notary,
		stats:            stats,
		time:             time,
		top:              top,
		netp:             netp,
		oracles:          oracles,
		delegation:       delegation,
		limits:           limits,
		stake:            stake,
		checkpoint:       checkpoint,
		spam:             spam,
		stakingAccounts:  stakingAccounts,
		epoch:            epoch,
		rewards:          rewards,
		snapshot:         snapshot,
		stateVar:         stateVarEngine,
		version:          version,
		blockchainClient: blockchainClient,
	}

	// register replay protection if needed:
	app.abci.RegisterSnapshot(snapshot)
	// setup handlers
	app.abci.OnInitChain = app.OnInitChain
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnEndBlock = app.OnEndBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnCheckTx = app.OnCheckTx
	app.abci.OnDeliverTx = app.OnDeliverTx
	app.abci.OnInfo = app.Info
	// snapshot specific handlers.
	app.abci.OnListSnapshots = app.ListSnapshots
	app.abci.OnOfferSnapshot = app.OfferSnapshot
	app.abci.OnApplySnapshotChunk = app.ApplySnapshotChunk
	app.abci.OnLoadSnapshotChunk = app.LoadSnapshotChunk

	app.abci.
		HandleCheckTx(txn.NodeSignatureCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.NodeVoteCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.ChainEventCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.SubmitOracleDataCommand, app.CheckSubmitOracleData).
		HandleCheckTx(txn.KeyRotateSubmissionCommand, app.RequireValidatorMasterPubKey).
		HandleCheckTx(txn.StateVariableProposalCommand, app.RequireValidatorPubKey)

	app.abci.
		HandleDeliverTx(txn.TransferFundsCommand,
			app.SendEventOnError(addDeterministicID(app.DeliverTransferFunds))).
		HandleDeliverTx(txn.CancelTransferFundsCommand,
			app.SendEventOnError(app.DeliverCancelTransferFunds)).
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
		HandleDeliverTx(txn.CancelLiquidityProvisionCommand,
			app.SendEventOnError(app.DeliverCancelLiquidityProvision)).
		HandleDeliverTx(txn.AmendLiquidityProvisionCommand,
			app.SendEventOnError(app.DeliverAmendLiquidityProvision)).
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
			app.SendEventOnError(app.DeliverReloadCheckpoint)).
		HandleDeliverTx(txn.KeyRotateSubmissionCommand,
			app.RequireValidatorMasterPubKeyW(app.DeliverKeyRotateSubmission)).
		HandleDeliverTx(txn.StateVariableProposalCommand,
			app.RequireValidatorPubKeyW(app.DeliverStateVarProposal))

	app.time.NotifyOnTick(app.onTick)

	return app
}

// addDeterministicID will build the command id and .
// the command id is built using the signature of the proposer of the command
// the signature is then hashed with sha3_256
// the hash is the hex string encoded.
func addDeterministicID(
	f func(context.Context, abci.Tx, string) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		return f(ctx, tx, hex.EncodeToString(vgcrypto.Hash(tx.Signature())))
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

func (app *App) RequireValidatorMasterPubKeyW(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := app.RequireValidatorMasterPubKey(ctx, tx); err != nil {
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

// ReloadConf updates the internal configuration.
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

func (app *App) Info(_ tmtypes.RequestInfo) tmtypes.ResponseInfo {
	hash, height := app.snapshot.Info()
	app.log.Debug("ABCI service INFO requested", logging.Int64("height", height), logging.String("hash", hex.EncodeToString(hash)))
	return tmtypes.ResponseInfo{
		AppVersion:       0, // application protocol version TBD.
		Version:          app.version,
		LastBlockHeight:  height,
		LastBlockAppHash: hash,
	}
}

func (app *App) ListSnapshots(_ tmtypes.RequestListSnapshots) tmtypes.ResponseListSnapshots {
	app.log.Debug("ABCI service ListSnapshots requested")
	snapshots, err := app.snapshot.List()
	resp := tmtypes.ResponseListSnapshots{}
	if err != nil {
		app.log.Error("Could not list snapshots", logging.Error(err))
		return resp
	}
	resp.Snapshots = make([]*tmtypes.Snapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		resp.Snapshots = append(resp.Snapshots, snap.ToTM())
	}
	return resp
}

func (app *App) OfferSnapshot(req tmtypes.RequestOfferSnapshot) tmtypes.ResponseOfferSnapshot {
	app.log.Debug("ABCI service OfferSnapshot start")
	snap, err := types.SnapshotFromTM(req.Snapshot)
	// invalid hash?
	if err != nil {
		// sender provided an invalid snapshot, that's not exactly something we can trust
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT_SENDER,
		}
	}
	// @TODO this is a placeholder for the actual check
	// if this node produced the wrong hash, don't accept... earlier snapshots may still be valid
	if !bytes.Equal(snap.Hash, req.AppHash) {
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT,
		}
	}
	if err := app.snapshot.ReceiveSnapshot(snap); err != nil {
		ret := tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT,
		}
		if err == types.ErrSnapshotMetaMismatch {
			// hashes match, but the meta doesn't, do not trust
			ret.Result = tmtypes.ResponseOfferSnapshot_REJECT_SENDER
		}
		return ret
	}
	// @TODO initialise snapshot engine to restore snapshot?
	return tmtypes.ResponseOfferSnapshot{
		Result: tmtypes.ResponseOfferSnapshot_ACCEPT,
	}
}

func (app *App) ApplySnapshotChunk(ctx context.Context, req tmtypes.RequestApplySnapshotChunk) tmtypes.ResponseApplySnapshotChunk {
	app.log.Debug("ABCI service ApplySnapshotChunk start")
	chunk := types.RawChunk{
		Nr:   req.Index,
		Data: req.Chunk,
	}
	resp := tmtypes.ResponseApplySnapshotChunk{}
	ready, err := app.snapshot.ApplySnapshotChunk(&chunk)
	if err != nil {
		switch err {
		case types.ErrUnknownSnapshot:
			resp.Result = tmtypes.ResponseApplySnapshotChunk_RETRY_SNAPSHOT // we weren't ready?
		case types.ErrChunkOutOfRange:
			resp.Result = tmtypes.ResponseApplySnapshotChunk_REJECT_SNAPSHOT // try another snapshot
		case types.ErrSnapshotMetaMismatch:
			// refetch the chunk from someone else
			resp.Result = tmtypes.ResponseApplySnapshotChunk_RETRY
			resp.RejectSenders = []string{req.Sender}
		case types.ErrMissingChunks:
			resp.Result = tmtypes.ResponseApplySnapshotChunk_RETRY
			resp.RefetchChunks = app.snapshot.GetMissingChunks()
		default:
			resp.Result = tmtypes.ResponseApplySnapshotChunk_ABORT
			// @TODO panic?
		}
		if resp.Result == tmtypes.ResponseApplySnapshotChunk_RETRY || resp.Result == tmtypes.ResponseApplySnapshotChunk_REJECT_SNAPSHOT {
			if err := app.snapshot.RejectSnapshot(); err == types.ErrSnapshotRetryLimit {
				app.log.Error("Applying snapshot chunk has reaching the retry limit, aborting")
				resp.Result = tmtypes.ResponseApplySnapshotChunk_ABORT
				defer app.log.Panic("Failed to load snapshot, max retry limit reached", logging.Error(err))
			}
		}
		return resp
	}
	resp.Result = tmtypes.ResponseApplySnapshotChunk_ACCEPT
	if ready {
		_ = app.snapshot.ApplySnapshot(ctx)
	}
	return resp
}

func (app *App) LoadSnapshotChunk(req tmtypes.RequestLoadSnapshotChunk) tmtypes.ResponseLoadSnapshotChunk {
	app.log.Debug("ABCI service LoadSnapshotChunk start")
	raw, err := app.snapshot.LoadSnapshotChunk(req.Height, req.Format, req.Chunk)
	if err != nil {
		return tmtypes.ResponseLoadSnapshotChunk{}
	}
	return tmtypes.ResponseLoadSnapshotChunk{
		Chunk: raw.Data,
	}
}

func (app *App) OnInitChain(req tmtypes.RequestInitChain) tmtypes.ResponseInitChain {
	app.log.Debug("ABCI service InitChain start")
	hash := hex.EncodeToString(vgcrypto.Hash(req.AppStateBytes))
	// let's assume genesis block is block 0
	app.chainCtx = vgcontext.WithChainID(context.Background(), req.ChainId)
	ctx := vgcontext.WithBlockHeight(app.chainCtx, 0)
	ctx = vgcontext.WithTraceID(ctx, hash)
	app.blockCtx = ctx

	app.abci.RegisterSnapshot(app.snapshot)

	// Start the snapshot engine letting it clear any pre-existing state so that if we are
	// replaying a chain from block 0 we won't recreate snapshots for blocks as we revisit
	// them
	if err := app.snapshot.Start(); err != nil {
		app.cancel()
		app.log.Fatal("couldn't start snapshot engine", logging.Error(err))
	}

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
		app.log.Fatal("couldn't initialise vega with the genesis block", logging.Error(err))
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

	app.epoch.OnBlockEnd(ctx)

	if app.spam != nil {
		app.spam.EndOfBlock(uint64(req.Height))
	}

	// check if we need to update the voting power of the validators
	if validatorUpdates := app.rewards.EndOfBlock(req.Height); validatorUpdates != nil {
		vUpdates := make([]tmtypes.ValidatorUpdate, 0, len(validatorUpdates))
		for _, v := range validatorUpdates {
			// using the default for key type which is ed25519 which seems fine for now because all validators in genesis use this key type
			// TODO use the proper key type of the validators
			pubkey, err := base64.StdEncoding.DecodeString(v.TmPubKey)
			if err != nil {
				continue
			}
			update := tmtypes.UpdateValidator(pubkey, v.VotingPower, "")
			vUpdates = append(vUpdates, update)
			app.log.Info("Updated voting power of validator", logging.String("tmKey", v.TmPubKey), logging.Int64("votingPower", v.VotingPower))
		}

		resp = tmtypes.ResponseEndBlock{
			ValidatorUpdates: vUpdates,
		}
	}
	return ctx, resp
}

// OnBeginBlock updates the internal lastBlockTime value with each new block.
func (app *App) OnBeginBlock(req tmtypes.RequestBeginBlock) (ctx context.Context, resp tmtypes.ResponseBeginBlock) {
	hash := hex.EncodeToString(req.Hash)
	app.cBlock = hash

	// Set chainID, if we have loaded from a snapshot we will not have called InitChain
	// TODO: we may be able to better if we store the chainID in the appstate's snapshot
	app.chainCtx = vgcontext.WithChainID(context.Background(), req.Header.GetChainID())

	ctx = vgcontext.WithBlockHeight(vgcontext.WithTraceID(app.chainCtx, hash), req.Header.Height)
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

	// will be true only the first time we get out of the bootstrap period
	if app.cpt != nil && app.limits.BootstrapFinished() {
		app.log.Info("restoring a scheduled checkpoint")
		if err := app.DeliverReloadCheckpoint(ctx, app.cpt); err != nil {
			app.log.Error("could not restore scheduled checkpoint",
				logging.Error(err))
		}
		app.cpt = nil
	}

	// read the state of validator set from the previous end of block
	var vd []*tmtypesint.Validator
	if app.blockchainClient != nil && req.Header.Height > 0 {
		h := req.Header.Height - 1
		vd, _ = app.blockchainClient.Validators(&h)
	}
	app.top.BeginBlock(ctx, req, vd)

	return ctx, resp
}

func (app *App) OnCommit() (resp tmtypes.ResponseCommit) {
	app.log.Debug("Processor COMMIT starting")
	defer app.log.Debug("Processor COMMIT completed")

	// call checkpoint _first_ so the snapshot contains the correct checkpoint state.
	cpt, _ := app.checkpoint.Checkpoint(app.blockCtx, app.currentTimestamp)
	snapHash, err := app.snapshot.Snapshot(app.blockCtx)
	if err != nil {
		app.log.Panic("Failed to create snapshot",
			logging.Error(err))
	}
	resp.Data = snapHash
	if len(snapHash) == 0 {
		resp.Data = app.exec.Hash()
		resp.Data = append(resp.Data, app.delegation.Hash()...)
		resp.Data = append(resp.Data, app.gov.Hash()...)
		resp.Data = append(resp.Data, app.stakingAccounts.Hash()...)
	}

	if cpt != nil {
		if len(snapHash) == 0 {
			// only append to commit hash if we aren't using the snapshot hash
			// otherwise restoring a checkpoint would restore an incomplete/wrong hash
			resp.Data = append(resp.Data, cpt.Hash...)
		}
		_ = app.handleCheckpoint(cpt)
	}
	// Compute the AppHash and update the response

	app.updateStats()
	app.setBatchStats()

	return resp
}

func (app *App) handleCheckpoint(cpt *types.CheckpointState) error {
	now := app.currentTimestamp
	height, _ := vgcontext.BlockHeightFromContext(app.blockCtx)
	cpFileName := fmt.Sprintf("%s-%d-%s.cp", now.Format("20060102150405"), height, hex.EncodeToString(cpt.Hash))
	cpFilePath, err := app.vegaPaths.CreateStatePathFor(paths.StatePath(filepath.Join(paths.CheckpointStateHome.String(), cpFileName)))
	if err != nil {
		return fmt.Errorf("couldn't get path for checkpoint file: %w", err)
	}
	if err := vgfs.WriteFile(cpFilePath, cpt.State); err != nil {
		return fmt.Errorf("couldn't write checkpoint file at %s: %w", cpFilePath, err)
	}
	// emit the event indicating a new checkpoint was created
	// this function is called both for interval checkpoints and withdrawal checkpoints
	event := events.NewCheckpointEvent(app.blockCtx, cpt)
	app.broker.Send(event)
	return nil
}

// OnCheckTx performs soft validations.
func (app *App) OnCheckTx(ctx context.Context, _ tmtypes.RequestCheckTx, tx abci.Tx) (context.Context, tmtypes.ResponseCheckTx) {
	resp := tmtypes.ResponseCheckTx{}
	if app.spam != nil {
		if _, err := app.spam.PreBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = abci.AbciSpamError
			resp.Data = []byte(err.Error())
			return ctx, resp
		}
	}

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

	return ctx, resp
}

// limitPubkey returns whether a request should be rate limited or not.
func (app *App) limitPubkey(pk string) (limit bool, isValidator bool) {
	// Do not rate limit validators nodes.
	if app.top.IsValidatorVegaPubKey(pk) {
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

func (app *App) validateScheduleCheckpointRestore(tx abci.Tx) (cpt *types.CheckpointState, err error) {
	defer func() {
		if err != nil {
			app.log.Error("could not schedule checkpoint to be loaded at network bootstraping end", logging.Error(err))
		}
	}()

	if app.cpt != nil {
		// an valid checkpoint is already schedule to be loaded
		// at end of bootstrap, skip
		return nil, errors.New("a valid checkpoint is already scheduled to be loaded at the end of the boostraping period")
	}

	cmd := &commandspb.RestoreSnapshot{}
	if err := tx.Unmarshal(cmd); err != nil {
		return nil, fmt.Errorf("invalid restore checkpoint command: %w", err)
	}

	// convert to checkpoint type:
	cpt = &types.CheckpointState{}
	if err := cpt.SetState(cmd.Data); err != nil {
		return nil, fmt.Errorf("invalid restore checkpoint command: %w", err)
	}

	// now we have a valid checkpoint.
	if err := app.checkpoint.ValidateCheckpoint(cpt); err != nil {
		return nil, err
	}

	return cpt, nil
}

func (app *App) scheduleCheckpointRestore(tx abci.Tx) (err error) {
	cpt, err := app.validateScheduleCheckpointRestore(tx)
	if err != nil {
		return err
	}

	app.cpt = tx
	app.log.Info("new checkpoint scheduled to be loaded after boostraping ends",
		logging.String("hash", hex.EncodeToString(cpt.Hash)))

	return nil
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
		// validator can be identified as with Vega public key with IsValidatorVegaPubKey function
		// or with Vega master publick key with IsValidatorNodeID function.
		if !(app.top.IsValidatorVegaPubKey(party) || app.top.IsValidatorNodeID(party)) {
			return ErrNoTransactionAllowedDuringBootstrap
		}
		cmd := tx.Command()
		// make sure this is a validator command and not a checkpoint.
		// checkpoints are only allow when the bootstrap period is done.
		if !cmd.IsValidatorCommand() {
			return ErrNonValidatorTransactionDisabledDuringBootstrap
		}
		if cmd == txn.CheckpointRestoreCommand {
			_, err := app.validateScheduleCheckpointRestore(tx)
			return err
		}
	}

	switch tx.Command() {
	case txn.CheckpointRestoreCommand:
		// do not get the transaction in the chain if we are not expecting a reload
		if !app.checkpoint.AwaitingRestore() {
			return errors.New("no checkpoint expected to be restored")
		}
	case txn.WithdrawCommand:
		if app.reloadCP {
			// we haven't reloaded the collateral data, withdrawals are going to fail
			return ErrAwaitingCheckpointRestore
		}
	case txn.SubmitOrderCommand, txn.AmendOrderCommand, txn.CancelOrderCommand, txn.LiquidityProvisionCommand, txn.AmendLiquidityProvisionCommand, txn.CancelLiquidityProvisionCommand:
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

	if app.spam != nil {
		if _, err := app.spam.PostBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = abci.AbciSpamError
			resp.Data = []byte(err.Error())
			return ctx, resp
		}
	}

	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = abci.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
	}

	// we don't need to set trace ID on context, it's been handled with OnBeginBlock

	return ctx, resp
}

func (app *App) RequireValidatorPubKey(ctx context.Context, tx abci.Tx) error {
	if !app.top.IsValidatorVegaPubKey(tx.PubKeyHex()) {
		return ErrNodeSignatureFromNonValidator
	}
	return nil
}

func (app *App) RequireValidatorMasterPubKey(ctx context.Context, tx abci.Tx) error {
	if !app.top.IsValidatorNodeID(tx.PubKeyHex()) {
		return ErrNodeSignatureWithNonValidatorMasterKey
	}
	return nil
}

func (app *App) DeliverTransferFunds(ctx context.Context, tx abci.Tx, id string) error {
	tfr := &commandspb.Transfer{}
	if err := tx.Unmarshal(tfr); err != nil {
		return err
	}

	party := tx.Party()
	transferFunds, err := types.NewTransferFromProto(id, party, tfr)
	if err != nil {
		return err
	}

	return app.banking.TransferFunds(ctx, transferFunds)
}

func (app *App) DeliverCancelTransferFunds(ctx context.Context, tx abci.Tx) error {
	cancel := &commandspb.CancelTransfer{}
	if err := tx.Unmarshal(cancel); err != nil {
		return err
	}

	return app.banking.CancelTransferFunds(ctx, types.NewCancelTransferFromProto(tx.Party(), cancel))
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
		lpid := hex.EncodeToString(vgcrypto.Hash([]byte(nm.Market().ID)))
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
	return app.notary.RegisterSignature(ctx, tx.PubKeyHex(), *ns)
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

func (app *App) DeliverCancelLiquidityProvision(ctx context.Context, tx abci.Tx) error {
	cancel := &commandspb.LiquidityProvisionCancellation{}
	if err := tx.Unmarshal(cancel); err != nil {
		return err
	}

	app.log.Debug("Blockchain service received a CANCEL Liquidity Provision request", logging.String("liquidity-provision-market-id", cancel.MarketId))

	lpc, err := types.LiquidityProvisionCancellationFromProto(cancel)
	if err != nil {
		if app.log.GetLevel() <= logging.DebugLevel {
			app.log.Debug("Unable to convert LiquidityProvisionCancellation protobuf message to domain type",
				logging.LiquidityProvisionCancellationProto(cancel), logging.Error(err))
		}
		return err
	}

	err = app.exec.CancelLiquidityProvision(ctx, lpc, tx.Party())
	if err != nil {
		app.log.Error("error on cancelling order", logging.String("liquidity-provision-market-id", lpc.MarketID), logging.Error(err))
		return err
	}
	if app.cfg.LogOrderCancelDebug {
		app.log.Debug("Liquidity provision cancelled", logging.LiquidityProvisionCancellation(*lpc))
	}

	return nil
}

func (app *App) DeliverAmendLiquidityProvision(ctx context.Context, tx abci.Tx) error {
	lp := &commandspb.LiquidityProvisionAmendment{}
	if err := tx.Unmarshal(lp); err != nil {
		return err
	}

	app.log.Debug("Blockchain service received a AMEND Liquidity Provision request", logging.String("liquidity-provision-market-id", lp.MarketId))

	// Convert protobuf into local domain type
	lpa, err := types.LiquidityProvisionAmendmentFromProto(lp)
	if err != nil {
		return err
	}

	// Submit the amend liquidity provision request to the Vega trading core
	err = app.exec.AmendLiquidityProvision(ctx, lpa, tx.Party())
	if err != nil {
		app.log.Error("error on amending Liquidity Provision", logging.String("liquidity-provision-market-id", lpa.MarketID), logging.Error(err))
		return err
	}
	if app.cfg.LogOrderAmendDebug {
		app.log.Debug("Liquidity Provision amended", logging.LiquidityProvisionAmendment(*lpa))
	}

	return nil
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

	pubKey := crypto.NewPublicKey(tx.PubKeyHex(), tx.PubKey())
	oracleData, err := app.oracles.Adaptors.Normalise(pubKey, *data)
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

	pubKey := crypto.NewPublicKey(tx.PubKeyHex(), tx.PubKey())
	_, err := app.oracles.Adaptors.Normalise(pubKey, *data)

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
		case toEnact.IsFreeform():
			app.enactFreeform(ctx, prop)
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

	var signature []byte
	// only build a signature if we are a validator
	if app.top.IsValidator() {
		switch {
		case asset.IsERC20():
			asset, _ := asset.ERC20()
			_, signature, err = asset.SignBridgeListing()
		}
		if err != nil {
			// this cannot happen, if we have an issue, this means
			// the node is not configured properly as a validator
			app.log.Panic("unable to sign allowlisting transaction",
				logging.String("asset-id", prop.ID),
				logging.Error(err))
		}
	}

	// then instruct the notary to start getting signature from validators
	app.notary.StartAggregate(prop.ID, types.NodeSignatureKindAssetNew, signature)
}

func (app *App) enactMarket(_ context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted

	// TODO: add checks for end of auction in here
}

func (app *App) enactFreeform(_ context.Context, prop *types.Proposal) {
	// There is nothing to enact in a freeform proposal so we just set the state
	prop.State = types.ProposalStateEnacted
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
		amount, overflowed := num.UintFromString(ce.Amount, 10)
		if overflowed {
			return errors.New("amount is not a valid base 10 number")
		}
		return app.delegation.UndelegateNow(ctx, tx.Party(), ce.NodeId, amount)
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

func (app *App) DeliverReloadCheckpoint(ctx context.Context, tx abci.Tx) (err error) {
	if !app.limits.BootstrapFinished() {
		// bootstrap is not finished, we eventually schedule a reload of the
		// checkpoint, if valid
		return app.scheduleCheckpointRestore(tx)
	}

	cmd := &commandspb.RestoreSnapshot{}
	defer func() {
		if err != nil {
			app.log.Error("Restoring checkpoint failed",
				logging.Error(err),
			)
			return
		}
		app.log.Info("Checkpoint restored!")
	}()

	if err := tx.Unmarshal(cmd); err != nil {
		return err
	}

	// convert to checkpoint type:
	cpt := &types.CheckpointState{}
	if err := cpt.SetState(cmd.Data); err != nil {
		return err
	}
	bh, err := cpt.GetBlockHeight()
	if err != nil {
		app.log.Panic("Failed to get blockheight from checkpoint", logging.Error(err))
	}
	// ensure block height and chain id are set
	cid, err := vgcontext.ChainIDFromContext(app.chainCtx)
	if err != nil {
		app.log.Panic("Failed to get chain id", logging.Error(err))
	}

	ctx = vgcontext.WithBlockHeight(ctx, bh)
	ctx = vgcontext.WithChainID(ctx, cid)
	app.blockCtx = ctx
	err = app.checkpoint.Load(ctx, cpt)
	if err != nil && err != types.ErrCheckpointStateInvalid && err != types.ErrCheckpointHashIncorrect && !errors.Is(err, checkpoint.ErrNoCheckpointExpectedToBeRestored) && !errors.Is(err, checkpoint.ErrIncompatibleHashes) {
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

func (app *App) DeliverKeyRotateSubmission(ctx context.Context, tx abci.Tx) error {
	if app.reloadCP {
		app.log.Debug("Skipping transaction while waiting for checkpoint restore")
		return nil
	}
	kr := &commandspb.KeyRotateSubmission{}
	if err := tx.Unmarshal(kr); err != nil {
		return err
	}

	currentBlockHeight, _ := vgcontext.BlockHeightFromContext(ctx)

	return app.top.AddKeyRotate(
		ctx,
		tx.PubKeyHex(),
		uint64(currentBlockHeight),
		kr,
	)
}

func (app *App) DeliverStateVarProposal(ctx context.Context, tx abci.Tx) error {
	proposal := &commandspb.StateVariableProposal{}
	if err := tx.Unmarshal(proposal); err != nil {
		app.log.Error("failed to unmarshal StateVariableProposal", logging.Error(err), logging.String("pub-key", tx.PubKeyHex()))
		return err
	}

	stateVarID := proposal.Proposal.StateVarId
	node := tx.PubKeyHex()
	eventID := proposal.Proposal.EventId
	bundle, err := statevar.KeyValueBundleFromProto(proposal.Proposal.Kvb)
	if err != nil {
		app.log.Error("failed to propose value", logging.Error(err))
		return err
	}
	return app.stateVar.ProposedValueReceived(ctx, stateVarID, node, eventID, bundle)
}
