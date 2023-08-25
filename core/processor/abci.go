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

package processor

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/snapshot"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"go.uber.org/zap"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/api"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/processor/ratelimit"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/core/vegatime"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	signatures "code.vegaprotocol.io/vega/libs/crypto/signature"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	tmtypes "github.com/tendermint/tendermint/abci/types"
	tmtypes1 "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypesint "github.com/tendermint/tendermint/types"
)

const AppVersion = 1

var (
	ErrUnexpectedTxPubKey          = errors.New("no one listens to the public keys that signed this oracle data")
	ErrTradingDisabled             = errors.New("trading disabled")
	ErrMarketProposalDisabled      = errors.New("market proposal disabled")
	ErrAssetProposalDisabled       = errors.New("asset proposal disabled")
	ErrEthOraclesDisabled          = errors.New("ethereum oracles disabled")
	ErrOracleNoSubscribers         = errors.New("there are no subscribes to the oracle data")
	ErrSpotMarketProposalDisabled  = errors.New("spot market proposal disabled")
	ErrPerpsMarketProposalDisabled = errors.New("perps market proposal disabled")
	ErrOracleDataNormalization     = func(err error) error {
		return fmt.Errorf("error normalizing incoming oracle data: %w", err)
	}
)

type Checkpoint interface {
	BalanceCheckpoint(ctx context.Context) (*types.CheckpointState, error)
	Checkpoint(ctx context.Context, now time.Time) (*types.CheckpointState, error)
}

type SpamEngine interface {
	EndOfBlock(blockHeight uint64, now time.Time)
	PreBlockAccept(tx abci.Tx) (bool, error)
	PostBlockAccept(tx abci.Tx) (bool, error)
}

type PoWEngine interface {
	api.ProofOfWorkParams
	BeginBlock(blockHeight uint64, blockHash string)
	EndOfBlock()
	CheckTx(tx abci.Tx) error
	DeliverTx(tx abci.Tx) error
	Commit()
	GetSpamStatistics(partyID string) *protoapi.PoWStatistic
}

//nolint:interfacebloat
type SnapshotEngine interface {
	Info() ([]byte, int64, string)
	Snapshot(context.Context) ([]byte, snapshot.DoneCh, error)
	SnapshotNow(context.Context) ([]byte, error)
	AddProviders(...types.StateProvider)
	HasSnapshots() (bool, error)

	// Calls related to state-sync

	ListLatestSnapshots() ([]*tmtypes.Snapshot, error)
	ReceiveSnapshot(*types.Snapshot) tmtypes.ResponseOfferSnapshot
	ReceiveSnapshotChunk(context.Context, *types.RawChunk, string) tmtypes.ResponseApplySnapshotChunk
	RetrieveSnapshotChunk(uint64, uint32, uint32) (*types.RawChunk, error)
	HasRestoredStateAlready() bool
}

type StateVarEngine interface {
	ProposedValueReceived(ctx context.Context, ID, nodeID, eventID string, bundle *statevar.KeyValueBundle) error
	OnBlockEnd(ctx context.Context)
}

type TeamsEngine interface {
	TeamExists(team types.TeamID) bool
	CreateTeam(context.Context, types.PartyID, types.TeamID, *commandspb.CreateReferralSet_Team) error
	UpdateTeam(context.Context, types.PartyID, types.TeamID, *commandspb.UpdateReferralSet_Team) error
	JoinTeam(context.Context, types.PartyID, *commandspb.ApplyReferralCode) error
}

type ReferralProgram interface {
	UpdateProgram(program *types.ReferralProgram)
	SetExists(types.ReferralSetID) bool
	CreateReferralSet(context.Context, types.PartyID, types.ReferralSetID) error
	ApplyReferralCode(context.Context, types.PartyID, types.ReferralSetID) error
}

type BlockchainClient interface {
	Validators(height *int64) ([]*tmtypesint.Validator, error)
}

type ProtocolUpgradeService interface {
	BeginBlock(ctx context.Context, blockHeight uint64)
	UpgradeProposal(ctx context.Context, pk string, upgradeBlockHeight uint64, vegaReleaseTag string) error
	TimeForUpgrade() bool
	GetUpgradeStatus() types.UpgradeStatus
	SetReadyForUpgrade()
	CoreReadyForUpgrade() bool
	SetCoreReadyForUpgrade()
	Cleanup(ctx context.Context)
	IsValidProposal(ctx context.Context, pk string, upgradeBlockHeight uint64, vegaReleaseTag string) error
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
	lastBlockAppHash  []byte
	version           string
	blockchainClient  BlockchainClient

	vegaPaths      paths.Paths
	cfg            Config
	log            *logging.Logger
	cancelFn       func()
	stopBlockchain func() error
	rates          *ratelimit.Rates

	// service injection
	assets                 Assets
	banking                Banking
	broker                 Broker
	witness                Witness
	evtfwd                 EvtForwarder
	exec                   ExecutionEngine
	ghandler               *genesis.Handler
	gov                    GovernanceEngine
	notary                 Notary
	stats                  Stats
	time                   TimeService
	top                    ValidatorTopology
	netp                   NetworkParameters
	oracles                *Oracle
	delegation             DelegationEngine
	limits                 Limits
	stake                  StakeVerifier
	stakingAccounts        StakingAccounts
	checkpoint             Checkpoint
	spam                   SpamEngine
	pow                    PoWEngine
	epoch                  EpochService
	snapshotEngine         SnapshotEngine
	stateVar               StateVarEngine
	teamsEngine            TeamsEngine
	referralProgram        ReferralProgram
	protocolUpgradeService ProtocolUpgradeService
	erc20MultiSigTopology  ERC20MultiSigTopology
	gastimator             *Gastimator

	nilPow  bool
	nilSpam bool

	maxBatchSize atomic.Uint64
}

func NewApp(
	log *logging.Logger,
	vegaPaths paths.Paths,
	config Config,
	cancelFn func(),
	stopBlockchain func() error,
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
	pow PoWEngine,
	stakingAccounts StakingAccounts,
	snapshot SnapshotEngine,
	stateVarEngine StateVarEngine,
	teamsEngine TeamsEngine,
	referralProgram ReferralProgram,
	blockchainClient BlockchainClient,
	erc20MultiSigTopology ERC20MultiSigTopology,
	version string, // we need the version for snapshot reload
	protocolUpgradeService ProtocolUpgradeService,
	codec abci.Codec,
	gastimator *Gastimator,
) *App {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	app := &App{
		abci: abci.New(codec),

		log:            log,
		vegaPaths:      vegaPaths,
		cfg:            config,
		cancelFn:       cancelFn,
		stopBlockchain: stopBlockchain,
		rates: ratelimit.New(
			config.Ratelimit.Requests,
			config.Ratelimit.PerNBlocks,
		),
		assets:                 assets,
		banking:                banking,
		broker:                 broker,
		witness:                witness,
		evtfwd:                 evtfwd,
		exec:                   exec,
		ghandler:               ghandler,
		gov:                    gov,
		notary:                 notary,
		stats:                  stats,
		time:                   time,
		top:                    top,
		netp:                   netp,
		oracles:                oracles,
		delegation:             delegation,
		limits:                 limits,
		stake:                  stake,
		checkpoint:             checkpoint,
		spam:                   spam,
		pow:                    pow,
		stakingAccounts:        stakingAccounts,
		epoch:                  epoch,
		snapshotEngine:         snapshot,
		stateVar:               stateVarEngine,
		teamsEngine:            teamsEngine,
		referralProgram:        referralProgram,
		version:                version,
		blockchainClient:       blockchainClient,
		erc20MultiSigTopology:  erc20MultiSigTopology,
		protocolUpgradeService: protocolUpgradeService,
		gastimator:             gastimator,
	}

	// setup handlers
	app.abci.OnInitChain = app.OnInitChain
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnEndBlock = app.OnEndBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnCheckTx = app.OnCheckTx
	app.abci.OnCheckTxSpam = app.OnCheckTxSpam
	app.abci.OnDeliverTx = app.OnDeliverTx
	app.abci.OnDeliverTxSpam = app.OnDeliverTXSpam
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
		HandleCheckTx(txn.RotateKeySubmissionCommand, app.RequireValidatorMasterPubKey).
		HandleCheckTx(txn.StateVariableProposalCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.ValidatorHeartbeatCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.RotateEthereumKeySubmissionCommand, app.RequireValidatorPubKey).
		HandleCheckTx(txn.ProtocolUpgradeCommand, app.CheckProtocolUpgradeProposal).
		HandleCheckTx(txn.BatchMarketInstructions, app.CheckBatchMarketInstructions).
		HandleCheckTx(txn.ProposeCommand, app.CheckPropose).
		HandleCheckTx(txn.TransferFundsCommand, app.CheckTransferCommand)

	app.abci.
		// node commands
		HandleDeliverTx(txn.NodeSignatureCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeSignature)).
		HandleDeliverTx(txn.NodeVoteCommand,
			app.RequireValidatorPubKeyW(app.DeliverNodeVote)).
		HandleDeliverTx(txn.ChainEventCommand,
			app.RequireValidatorPubKeyW(addDeterministicID(app.DeliverChainEvent))).
		HandleDeliverTx(txn.StateVariableProposalCommand,
			app.RequireValidatorPubKeyW(app.DeliverStateVarProposal)).
		HandleDeliverTx(txn.ValidatorHeartbeatCommand,
			app.DeliverValidatorHeartbeat).
		// validators commands
		HandleDeliverTx(txn.IssueSignatures,
			app.SendTransactionResult(app.DeliverIssueSignatures)).
		HandleDeliverTx(txn.ProtocolUpgradeCommand,
			app.SendTransactionResult(app.DeliverProtocolUpgradeCommand)).
		HandleDeliverTx(txn.RotateKeySubmissionCommand,
			app.SendTransactionResult(
				app.RequireValidatorMasterPubKeyW(app.DeliverKeyRotateSubmission),
			),
		).
		HandleDeliverTx(txn.RotateEthereumKeySubmissionCommand,
			app.SendTransactionResult(
				app.RequireValidatorPubKeyW(app.DeliverEthereumKeyRotateSubmission),
			),
		).
		// user commands
		HandleDeliverTx(txn.AnnounceNodeCommand,
			app.SendTransactionResult(app.DeliverAnnounceNode),
		).
		HandleDeliverTx(txn.CancelTransferFundsCommand,
			app.SendTransactionResult(app.DeliverCancelTransferFunds),
		).
		HandleDeliverTx(txn.TransferFundsCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverTransferFunds),
			),
		).
		HandleDeliverTx(txn.SubmitOrderCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverSubmitOrder),
			),
		).
		HandleDeliverTx(txn.StopOrdersSubmissionCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverStopOrdersSubmission),
			),
		).
		HandleDeliverTx(txn.StopOrdersCancellationCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverStopOrdersCancellation),
			),
		).
		HandleDeliverTx(txn.CancelOrderCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverCancelOrder),
			),
		).
		HandleDeliverTx(txn.AmendOrderCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverAmendOrder),
			),
		).
		HandleDeliverTx(txn.WithdrawCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverWithdraw))).
		HandleDeliverTx(txn.ProposeCommand,
			app.SendTransactionResult(
				app.CheckProposeW(
					addDeterministicID(app.DeliverPropose),
				),
			),
		).
		HandleDeliverTx(txn.VoteCommand,
			app.SendTransactionResult(app.DeliverVote),
		).
		HandleDeliverTx(txn.LiquidityProvisionCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverLiquidityProvision),
			),
		).
		HandleDeliverTx(txn.CancelLiquidityProvisionCommand,
			app.SendTransactionResult(app.DeliverCancelLiquidityProvision),
		).
		HandleDeliverTx(txn.AmendLiquidityProvisionCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverAmendLiquidityProvision),
			),
		).
		HandleDeliverTx(txn.SubmitOracleDataCommand,
			app.SendTransactionResult(app.DeliverSubmitOracleData),
		).
		HandleDeliverTx(txn.DelegateCommand,
			app.SendTransactionResult(app.DeliverDelegate),
		).
		HandleDeliverTx(txn.UndelegateCommand,
			app.SendTransactionResult(app.DeliverUndelegate),
		).
		HandleDeliverTx(txn.BatchMarketInstructions,
			app.SendTransactionResult(
				app.CheckBatchMarketInstructionsW(
					addDeterministicID(app.DeliverBatchMarketInstructions),
				),
			),
		).
		HandleDeliverTx(txn.CreateReferralSetCommand,
			app.SendTransactionResult(addDeterministicID(app.CreateReferralSet)),
		).
		HandleDeliverTx(txn.UpdateReferralSetCommand,
			app.SendTransactionResult(app.UpdateReferralSet),
		).
		HandleDeliverTx(txn.ApplyReferralCodeCommand,
			app.SendTransactionResult(app.ApplyReferralCode),
		)

	app.time.NotifyOnTick(app.onTick)

	app.nilPow = app.pow == nil || reflect.ValueOf(app.pow).IsNil()
	app.nilSpam = app.spam == nil || reflect.ValueOf(app.spam).IsNil()
	app.ensureConfig()
	return app
}

func (app *App) OnSpamProtectionMaxBatchSizeUpdate(_ context.Context, u *num.Uint) error {
	app.maxBatchSize.Store(u.Uint64())
	return nil
}

// addDeterministicID will build the command ID
// the command ID is built using the signature of the proposer of the command
// the signature is then hashed with sha3_256
// the hash is the hex string encoded.
func addDeterministicID(
	f func(context.Context, abci.Tx, string) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		return f(ctx, tx, hex.EncodeToString(vgcrypto.Hash(tx.Signature())))
	}
}

func (app *App) CheckProposeW(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := app.CheckPropose(ctx, tx); err != nil {
			return err
		}
		return f(ctx, tx)
	}
}

func (app *App) CheckBatchMarketInstructionsW(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := app.CheckBatchMarketInstructions(ctx, tx); err != nil {
			return err
		}
		return f(ctx, tx)
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

func (app *App) SendTransactionResult(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := f(ctx, tx); err != nil {
			// Send and error event
			app.broker.Send(events.NewTransactionResultEventFailure(
				ctx, hex.EncodeToString(tx.Hash()), tx.Party(), err, tx.GetCmd(),
			))

			// FIXME(j): remove this once anyone have stopped using the event
			app.broker.Send(events.NewTxErrEvent(ctx, err, tx.Party(), tx.GetCmd(), tx.Command().String()))

			return err
		}

		// Send and error event
		app.broker.Send(events.NewTransactionResultEventSuccess(
			ctx, hex.EncodeToString(tx.Hash()), tx.Party(), tx.GetCmd(),
		))

		return nil
	}
}

func (app *App) ensureConfig() {
	if app.cfg.KeepCheckpointsMax < 1 {
		app.cfg.KeepCheckpointsMax = 1
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
	app.ensureConfig()
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
	if len(app.lastBlockAppHash) != 0 {
		// we must've lost connection to tendermint for a bit, tell it where we got up to
		height, _ := vgcontext.BlockHeightFromContext(app.blockCtx)
		app.log.Info("ABCI service INFO requested after reconnect",
			logging.Int64("height", height),
			logging.String("hash", hex.EncodeToString(app.lastBlockAppHash)),
		)
		return tmtypes.ResponseInfo{
			AppVersion:       AppVersion,
			Version:          app.version,
			LastBlockHeight:  height,
			LastBlockAppHash: app.lastBlockAppHash,
		}
	}

	// returns whether or not we have loaded from a snapshot (and may even do the loading)
	old := app.broker.SetStreaming(false)
	defer app.broker.SetStreaming(old)

	resp := tmtypes.ResponseInfo{
		AppVersion: AppVersion,
		Version:    app.version,
	}

	hasSnapshots, err := app.snapshotEngine.HasSnapshots()
	if err != nil {
		app.log.Panic("Failed to verify if the snapshot engine has stored snapshots", logging.Error(err))
	}

	// If the snapshot engine has snapshots stored, the node can safely advertise
	// its chain info. This comes from the snapshot engine because it is
	// its responsibility to store it.
	if hasSnapshots {
		hash, height, chainID := app.snapshotEngine.Info()
		resp.LastBlockHeight = height
		resp.LastBlockAppHash = hash
		app.abci.SetChainID(chainID)
		app.chainCtx = vgcontext.WithChainID(context.Background(), chainID)
	}

	app.log.Info("ABCI service INFO requested",
		logging.String("version", resp.Version),
		logging.Uint64("app-version", resp.AppVersion),
		logging.Int64("height", resp.LastBlockHeight),
		logging.String("hash", hex.EncodeToString(resp.LastBlockAppHash)),
	)
	return resp
}

func (app *App) ListSnapshots(_ tmtypes.RequestListSnapshots) tmtypes.ResponseListSnapshots {
	app.log.Debug("ABCI service ListSnapshots requested")

	latestSnapshots, err := app.snapshotEngine.ListLatestSnapshots()
	if err != nil {
		app.log.Error("Could not list latest snapshots", logging.Error(err))
		return tmtypes.ResponseListSnapshots{}
	}

	return tmtypes.ResponseListSnapshots{
		Snapshots: latestSnapshots,
	}
}

func (app *App) OfferSnapshot(req tmtypes.RequestOfferSnapshot) tmtypes.ResponseOfferSnapshot {
	app.log.Debug("ABCI service OfferSnapshot start")

	if app.snapshotEngine.HasRestoredStateAlready() {
		app.log.Warn("The snapshot engine aborted the snapshot offer from state-sync since the state has already been restored")
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_ABORT,
		}
	}

	deserializedSnapshot, err := types.SnapshotFromTM(req.Snapshot)
	if err != nil {
		app.log.Error("Could not deserialize snapshot", logging.Error(err))
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT_SENDER,
		}
	}

	// check that our unpacked snapshot's hash matches that which tendermint thinks it sent
	if !bytes.Equal(deserializedSnapshot.Hash, req.AppHash) {
		app.log.Error("The hashes from the request and the deserialized snapshot mismatch",
			logging.String("deserialized-hash", hex.EncodeToString(deserializedSnapshot.Hash)),
			logging.String("request-hash", hex.EncodeToString(req.AppHash)))
		return tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT,
		}
	}

	return app.snapshotEngine.ReceiveSnapshot(deserializedSnapshot)
}

func (app *App) ApplySnapshotChunk(ctx context.Context, req tmtypes.RequestApplySnapshotChunk) tmtypes.ResponseApplySnapshotChunk {
	app.log.Debug("ABCI service ApplySnapshotChunk start")

	if app.snapshotEngine.HasRestoredStateAlready() {
		app.log.Warn("The snapshot engine aborted the snapshot chunk from state-sync since the state has already been restored")
		return tmtypes.ResponseApplySnapshotChunk{
			Result: tmtypes.ResponseApplySnapshotChunk_ABORT,
		}
	}

	chunk := &types.RawChunk{
		Nr:   req.Index,
		Data: req.Chunk,
	}

	return app.snapshotEngine.ReceiveSnapshotChunk(ctx, chunk, req.Sender)
}

func (app *App) LoadSnapshotChunk(req tmtypes.RequestLoadSnapshotChunk) tmtypes.ResponseLoadSnapshotChunk {
	app.log.Debug("ABCI service LoadSnapshotChunk start")

	rawChunk, err := app.snapshotEngine.RetrieveSnapshotChunk(req.Height, req.Format, req.Chunk)
	if err != nil {
		app.log.Error("Could not load a snapshot chunk from snapshot engine",
			logging.Uint64("height", req.Height),
			logging.Error(err),
		)
		return tmtypes.ResponseLoadSnapshotChunk{}
	}

	return tmtypes.ResponseLoadSnapshotChunk{
		Chunk: rawChunk.Data,
	}
}

func (app *App) OnInitChain(req tmtypes.RequestInitChain) tmtypes.ResponseInitChain {
	app.log.Debug("ABCI service InitChain start")
	hash := hex.EncodeToString(vgcrypto.Hash(req.AppStateBytes))
	app.abci.SetChainID(req.ChainId)
	app.chainCtx = vgcontext.WithChainID(context.Background(), req.ChainId)
	ctx := vgcontext.WithBlockHeight(app.chainCtx, req.InitialHeight)
	ctx = vgcontext.WithTraceID(ctx, hash)
	app.blockCtx = ctx

	app.broker.Send(
		events.NewBeginBlock(ctx, eventspb.BeginBlock{
			Height:    uint64(req.InitialHeight),
			Timestamp: req.Time.UnixNano(),
			Hash:      hash,
		}),
	)

	if err := app.ghandler.OnGenesis(ctx, req.Time, req.AppStateBytes); err != nil {
		app.cancel()
		app.log.Fatal("couldn't initialise vega with the genesis block", logging.Error(err))
	}

	app.broker.Send(
		events.NewEndBlock(ctx, eventspb.EndBlock{
			Height: uint64(req.InitialHeight),
		}),
	)

	return tmtypes.ResponseInitChain{
		Validators: app.top.GetValidatorPowerUpdates(),
	}
}

func (app *App) OnEndBlock(req tmtypes.RequestEndBlock) (ctx context.Context, resp tmtypes.ResponseEndBlock) {
	app.log.Debug("entering end block", logging.Time("at", time.Now()))
	defer func() { app.log.Debug("leaving end block", logging.Time("at", time.Now())) }()

	app.log.Debug("ABCI service END block completed",
		logging.Int64("current-timestamp", app.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", app.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(app.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(app.previousTimestamp)),
	)

	app.epoch.OnBlockEnd(app.blockCtx)
	if !app.nilPow {
		app.pow.EndOfBlock()
	}

	if !app.nilSpam {
		app.spam.EndOfBlock(uint64(req.Height), app.time.GetTimeNow())
	}

	app.stateVar.OnBlockEnd(app.blockCtx)

	powerUpdates := app.top.GetValidatorPowerUpdates()
	resp = tmtypes.ResponseEndBlock{}
	if len(powerUpdates) > 0 {
		resp.ValidatorUpdates = powerUpdates
	}

	// update max gas based on the network parameter
	resp.ConsensusParamUpdates = &tmtypes.ConsensusParams{
		Block: &tmtypes.BlockParams{
			MaxGas:   int64(app.gastimator.OnBlockEnd()),
			MaxBytes: tmtypesint.DefaultBlockParams().MaxBytes,
		},
		Version: &tmtypes1.VersionParams{
			AppVersion: AppVersion,
		},
	}
	app.exec.BlockEnd(app.blockCtx)

	return ctx, resp
}

// OnBeginBlock updates the internal lastBlockTime value with each new block.
func (app *App) OnBeginBlock(
	req tmtypes.RequestBeginBlock,
) (ctx context.Context, resp tmtypes.ResponseBeginBlock) {
	app.log.Debug("entering begin block", logging.Time("at", time.Now()), logging.Uint64("height", uint64(req.Header.Height)))
	defer func() { app.log.Debug("leaving begin block", logging.Time("at", time.Now())) }()

	hash := hex.EncodeToString(req.Hash)
	ctx = vgcontext.WithBlockHeight(vgcontext.WithTraceID(app.chainCtx, hash), req.Header.Height)

	if app.protocolUpgradeService.CoreReadyForUpgrade() {
		app.startProtocolUpgrade(ctx)
	}

	app.broker.Send(
		events.NewBeginBlock(ctx, eventspb.BeginBlock{
			Height:    uint64(req.Header.Height),
			Timestamp: req.Header.Time.UnixNano(),
			Hash:      hash,
		}),
	)

	app.cBlock = hash

	// update pow engine on a new block
	if !app.nilPow {
		app.pow.BeginBlock(uint64(req.Header.Height), hash)
	}

	app.stats.SetHash(hash)
	app.stats.SetHeight(uint64(req.Header.Height))
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
		logging.Int64("height", req.Header.GetHeight()),
	)

	app.protocolUpgradeService.BeginBlock(ctx, uint64(req.Header.Height))
	app.top.BeginBlock(ctx, req)

	return ctx, resp
}

func (app *App) startProtocolUpgrade(ctx context.Context) {
	// Stop blockchain server so it doesn't accept transactions and it doesn't times out.
	go func() {
		if err := app.stopBlockchain(); err != nil {
			app.log.Error("an error occurred while stopping the blockchain", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var eventsCh <-chan events.Event
	var errsCh <-chan error
	if app.broker.StreamingEnabled() {
		// wait here for data node send back the confirmation
		eventsCh, errsCh = app.broker.SocketClient().Receive(ctx)
	}

	app.broker.Send(
		events.NewProtocolUpgradeStarted(ctx, eventspb.ProtocolUpgradeStarted{
			LastBlockHeight: app.stats.Height(),
		}),
	)

	if eventsCh != nil {
		app.log.Info("waiting for data node to get ready for upgrade")

	Loop:
		for {
			select {
			case e := <-eventsCh:
				if e.Type() != events.ProtocolUpgradeDataNodeReadyEvent {
					continue
				}
				if e.StreamMessage().GetProtocolUpgradeDataNodeReady().GetLastBlockHeight() == app.stats.Height() {
					cancel()
					break Loop
				}
			case err := <-errsCh:
				app.log.Fatal("failed to wait for data node to get ready for upgrade", logging.Error(err))
			}
		}
	}

	app.protocolUpgradeService.SetReadyForUpgrade()

	// wait until killed
	for {
		time.Sleep(1 * time.Second)
		app.log.Info("application is ready for shutdown")
	}
}

func (app *App) OnCommit() (resp tmtypes.ResponseCommit) {
	app.log.Debug("entering commit", logging.Time("at", time.Now()))
	defer func() { app.log.Debug("leaving commit", logging.Time("at", time.Now())) }()

	if !app.nilPow {
		app.pow.Commit()
	}

	// call checkpoint _first_ so the snapshot contains the correct checkpoint state.
	cpt, _ := app.checkpoint.Checkpoint(app.blockCtx, app.currentTimestamp)

	t0 := time.Now()

	var snapHash []byte
	var err error
	// if there is an approved protocol upgrade proposal and the current block height is later than the proposal's block height then take a snapshot and wait to be killed by the process manager
	if app.protocolUpgradeService.TimeForUpgrade() {
		app.protocolUpgradeService.Cleanup(app.blockCtx)
		snapHash, err = app.snapshotEngine.SnapshotNow(app.blockCtx)
		if err == nil {
			app.protocolUpgradeService.SetCoreReadyForUpgrade()
		}
	} else {
		snapHash, _, err = app.snapshotEngine.Snapshot(app.blockCtx)
	}

	if err != nil {
		app.log.Panic("Failed to create snapshot",
			logging.Error(err))
	}

	t1 := time.Now()
	if len(snapHash) > 0 {
		app.log.Info("State has been snapshotted", logging.Float64("duration", t1.Sub(t0).Seconds()))
	}
	resp.Data = snapHash

	if len(snapHash) == 0 {
		resp.Data = vgcrypto.Hash([]byte(app.version))
		resp.Data = append(resp.Data, app.exec.Hash()...)
		resp.Data = append(resp.Data, app.delegation.Hash()...)
		resp.Data = append(resp.Data, app.gov.Hash()...)
		resp.Data = append(resp.Data, app.stakingAccounts.Hash()...)
	}

	if cpt != nil {
		if len(snapHash) == 0 {
			// only append to commit hash if we aren't using the snapshot hash
			// otherwise restoring a checkpoint would restore an incomplete/wrong hash
			resp.Data = append(resp.Data, cpt.Hash...)
			app.log.Debug("checkpoint hash", logging.String("response-data", hex.EncodeToString(cpt.Hash)))
		}
		_ = app.handleCheckpoint(cpt)
	}

	// the snapshot produce an actual hash, so no need
	// to rehash if we have a snapshot hash.
	// otherwise, it's a concatenation of hash that we get,
	// so we just re-hash to have an output which is actually an
	// hash and is consistent over all calls to Commit
	if len(snapHash) <= 0 {
		resp.Data = vgcrypto.Hash(resp.Data)
	} else {
		app.broker.Send(events.NewSnapshotEventEvent(app.blockCtx, app.stats.Height(), app.cBlock, app.protocolUpgradeService.TimeForUpgrade()))
	}

	// Update response and save the apphash incase we lose connection with tendermint and need to verify our
	// current state
	app.lastBlockAppHash = resp.Data
	app.log.Debug("apphash calculated", logging.String("response-data", hex.EncodeToString(resp.Data)))
	app.updateStats()
	app.setBatchStats()

	app.broker.Send(
		events.NewEndBlock(app.blockCtx, eventspb.EndBlock{
			Height: app.stats.Height(),
		}),
	)

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

	return app.removeOldCheckpoints()
}

func (app *App) removeOldCheckpoints() error {
	cpDirPath, err := app.vegaPaths.CreateStatePathFor(paths.StatePath(paths.CheckpointStateHome.String()))
	if err != nil {
		return fmt.Errorf("couldn't get checkpoints directory: %w", err)
	}

	files, err := ioutil.ReadDir(cpDirPath)
	if err != nil {
		return fmt.Errorf("could not open the checkpoint directory: %w", err)
	}

	// we assume that the files in this directory are only
	// from the checkpoints
	// and always keep the last 20, so return if we have less than that
	if len(files) <= int(app.cfg.KeepCheckpointsMax) {
		return nil
	}

	oldest := app.stats.Height()
	toRemove := ""
	for _, file := range files {
		// checkpoint have the following format:
		// 20230322173929-12140156-d833359cb648eb315b4d3f9ccaa5092bd175b2f72a9d44783377ca5d7a2ec965.cp
		// which is:
		// time-block-hash.cp
		// we split and should have the block in splitted[1]
		splitted := strings.Split(file.Name(), "-")
		if len(splitted) != 3 {
			app.log.Error("weird checkpoint file name", logging.String("checkpoint-file", file.Name()))
			// weird file, keep going
			continue
		}
		block, err := strconv.ParseInt(splitted[1], 10, 64)
		if err != nil {
			app.log.Error("could not parse block number", logging.Error(err), logging.String("checkpoint-file", file.Name()))
			continue
		}

		if uint64(block) < oldest {
			oldest = uint64(block)
			toRemove = file.Name()
		}
	}

	if len(toRemove) > 0 {
		finalPath := filepath.Join(cpDirPath, toRemove)
		if err := os.Remove(finalPath); err != nil {
			app.log.Error("could not remove old checkpoint file",
				logging.Error(err),
				logging.String("checkpoint-file", finalPath),
			)
		}
		// just return an error, not much we can do
		return nil
	}

	return nil
}

// OnCheckTxSpam checks for spam and replay.
func (app *App) OnCheckTxSpam(tx abci.Tx) tmtypes.ResponseCheckTx {
	resp := tmtypes.ResponseCheckTx{}

	// verify proof of work and replay
	if !app.nilPow {
		if err := app.pow.CheckTx(tx); err != nil {
			if app.log.IsDebug() {
				app.log.Debug(err.Error())
			}
			resp.Code = blockchain.AbciSpamError
			resp.Data = []byte(err.Error())
			return resp
		}
	}
	// additional spam checks
	if !app.nilSpam {
		if _, err := app.spam.PreBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = blockchain.AbciSpamError
			resp.Data = []byte(err.Error())
			return resp
		}
	}
	return resp
}

// OnCheckTx performs soft validations.
func (app *App) OnCheckTx(ctx context.Context, _ tmtypes.RequestCheckTx, tx abci.Tx) (context.Context, tmtypes.ResponseCheckTx) {
	resp := tmtypes.ResponseCheckTx{}

	if app.log.IsDebug() {
		app.log.Debug("entering checkTx", logging.String("tid", tx.GetPoWTID()), logging.String("command", tx.Command().String()))
	}

	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = blockchain.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
		return ctx, resp
	}

	// Check ratelimits
	// FIXME(): temporary disable all rate limiting
	_, isval := app.limitPubkey(tx.PubKeyHex())

	gasWanted, err := app.gastimator.CalcGasWantedForTx(tx)
	if err != nil { // this error means the transaction couldn't be parsed
		app.log.Error("error getting gas estimate", logging.Error(err))
		resp.Code = blockchain.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
		return ctx, resp
	}

	resp.GasWanted = int64(gasWanted)
	resp.Priority = int64(app.gastimator.GetPriority(tx))
	if app.log.IsDebug() {
		app.log.Debug("transaction passed checkTx", logging.String("tid", tx.GetPoWTID()), logging.String("command", tx.Command().String()), logging.Int64("priority", resp.Priority), logging.Int64("gas-wanted", resp.GasWanted), logging.Int64("max-gas", int64(app.gastimator.GetMaxGas())))
	}

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

func (app *App) canSubmitTx(tx abci.Tx) (err error) {
	defer func() {
		if err != nil {
			app.log.Error("cannot submit transaction", logging.Error(err))
		}
	}()

	switch tx.Command() {
	case txn.SubmitOrderCommand, txn.AmendOrderCommand, txn.CancelOrderCommand, txn.LiquidityProvisionCommand, txn.AmendLiquidityProvisionCommand, txn.CancelLiquidityProvisionCommand, txn.StopOrdersCancellationCommand, txn.StopOrdersSubmissionCommand:
		if !app.limits.CanTrade() {
			return ErrTradingDisabled
		}
	case txn.ProposeCommand:
		praw := &commandspb.ProposalSubmission{}
		if err := tx.Unmarshal(praw); err != nil {
			return fmt.Errorf("could not unmarshal proposal submission: %w", err)
		}
		p, err := types.NewProposalSubmissionFromProto(praw)
		if err != nil {
			return fmt.Errorf("invalid proposal submission: %w", err)
		}
		if p.Terms == nil {
			return errors.New("invalid proposal submission")
		}
		switch p.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewMarket:
			if !app.limits.CanProposeMarket() {
				return ErrMarketProposalDisabled
			}
			if p.Terms.GetNewMarket().Changes.ProductType() == types.ProductTypePerps && !app.limits.CanProposePerpsMarket() {
				return ErrPerpsMarketProposalDisabled
			}
			return validateUseOfEthOracles(p.Terms, app.netp)
		case types.ProposalTermsTypeUpdateMarket:
			return validateUseOfEthOracles(p.Terms, app.netp)

		case types.ProposalTermsTypeNewAsset:
			if !app.limits.CanProposeAsset() {
				return ErrAssetProposalDisabled
			}
		case types.ProposalTermsTypeNewSpotMarket:
			if !app.limits.CanProposeSpotMarket() {
				return ErrSpotMarketProposalDisabled
			}
		}
	}
	return nil
}

func validateUseOfEthOracles(terms *types.ProposalTerms, netp NetworkParameters) error {
	ethOracleEnabled, _ := netp.GetInt(netparams.EthereumOraclesEnabled)

	switch terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		m := terms.GetNewMarket()
		if m == nil {
			return nil
		}

		if m.Changes == nil {
			return nil
		}

		// Here the instrument is *types.InstrumentConfiguration
		if m.Changes.Instrument == nil {
			return nil
		}

		if m.Changes.Instrument.Product == nil {
			return nil
		}

		switch product := m.Changes.Instrument.Product.(type) {
		case *types.InstrumentConfigurationFuture:
			if product.Future == nil {
				return nil
			}
			terminatedWithEthOracle := !product.Future.DataSourceSpecForTradingTermination.GetEthCallSpec().IsZero()
			settledWithEthOracle := !product.Future.DataSourceSpecForSettlementData.GetEthCallSpec().IsZero()
			if (terminatedWithEthOracle || settledWithEthOracle) && ethOracleEnabled != 1 {
				return ErrEthOraclesDisabled
			}
		}

	case types.ProposalTermsTypeUpdateMarket:
		m := terms.GetUpdateMarket()
		if m == nil {
			return nil
		}

		if m.Changes == nil {
			return nil
		}

		// Here the instrument is *types.UpdateInstrumentConfiguration
		if m.Changes.Instrument == nil {
			return nil
		}

		if m.Changes.Instrument.Product == nil {
			return nil
		}

		switch product := m.Changes.Instrument.Product.(type) {
		case *types.UpdateInstrumentConfigurationFuture:
			if product.Future == nil {
				return nil
			}
			terminatedWithEthOracle := !product.Future.DataSourceSpecForTradingTermination.GetEthCallSpec().IsZero()
			settledWithEthOracle := !product.Future.DataSourceSpecForSettlementData.GetEthCallSpec().IsZero()
			if (terminatedWithEthOracle || settledWithEthOracle) && ethOracleEnabled != 1 {
				return ErrEthOraclesDisabled
			}
		}
	}

	return nil
}

// OnDeliverTXSpam checks spam and replay.
func (app *App) OnDeliverTXSpam(ctx context.Context, tx abci.Tx) tmtypes.ResponseDeliverTx {
	var resp tmtypes.ResponseDeliverTx
	ctxWithHash := vgcontext.WithTxHash(ctx, hex.EncodeToString(tx.Hash()))

	// verify proof of work
	if !app.nilPow {
		if err := app.pow.DeliverTx(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = blockchain.AbciSpamError
			resp.Data = []byte(err.Error())
			app.broker.Send(events.NewTxErrEvent(ctxWithHash, err, tx.Party(), tx.GetCmd(), tx.Command().String()))
			return resp
		}
	}
	if !app.nilSpam {
		if _, err := app.spam.PostBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = blockchain.AbciSpamError
			resp.Data = []byte(err.Error())
			evt := events.NewTxErrEvent(ctxWithHash, err, tx.Party(), tx.GetCmd(), tx.Command().String())
			app.broker.Send(evt)
			return resp
		}
	}
	return resp
}

// OnDeliverTx increments the internal tx counter and decorates the context with tracing information.
func (app *App) OnDeliverTx(ctx context.Context, req tmtypes.RequestDeliverTx, tx abci.Tx) (context.Context, tmtypes.ResponseDeliverTx) {
	app.setTxStats(len(req.Tx))
	var resp tmtypes.ResponseDeliverTx
	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = blockchain.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
	}

	// we don't need to set trace ID on context, it's been handled with OnBeginBlock

	return ctx, resp
}

func (app *App) CheckProtocolUpgradeProposal(ctx context.Context, tx abci.Tx) error {
	if err := app.RequireValidatorPubKey(ctx, tx); err != nil {
		return err
	}
	pu := &commandspb.ProtocolUpgradeProposal{}
	if err := tx.Unmarshal(pu); err != nil {
		return err
	}
	return app.protocolUpgradeService.IsValidProposal(ctx, tx.PubKeyHex(), pu.UpgradeBlockHeight, pu.VegaReleaseTag)
}

func (app *App) RequireValidatorPubKey(_ context.Context, tx abci.Tx) error {
	if !app.top.IsValidatorVegaPubKey(tx.PubKeyHex()) {
		return ErrNodeSignatureFromNonValidator
	}
	return nil
}

func (app *App) CheckBatchMarketInstructions(_ context.Context, tx abci.Tx) error {
	bmi := &commandspb.BatchMarketInstructions{}
	if err := tx.Unmarshal(bmi); err != nil {
		return err
	}

	maxBatchSize := app.maxBatchSize.Load()
	size := uint64(len(bmi.Cancellations) + len(bmi.Amendments) + len(bmi.Submissions) + len(bmi.StopOrdersSubmission) + len(bmi.StopOrdersCancellation))
	if size > maxBatchSize {
		return ErrMarketBatchInstructionTooBig(size, maxBatchSize)
	}

	return nil
}

func (app *App) DeliverBatchMarketInstructions(
	ctx context.Context,
	tx abci.Tx,
	deterministicID string,
) error {
	batch := &commandspb.BatchMarketInstructions{}
	if err := tx.Unmarshal(batch); err != nil {
		return err
	}

	return NewBMIProcessor(app.log, app.exec, Validate{}).
		ProcessBatch(ctx, batch, tx.Party(), deterministicID, app.stats)
}

func (app *App) RequireValidatorMasterPubKey(_ context.Context, tx abci.Tx) error {
	if !app.top.IsValidatorNodeID(tx.PubKeyHex()) {
		return ErrNodeSignatureWithNonValidatorMasterKey
	}
	return nil
}

func (app *App) DeliverIssueSignatures(ctx context.Context, tx abci.Tx) error {
	is := &commandspb.IssueSignatures{}
	if err := tx.Unmarshal(is); err != nil {
		return err
	}
	return app.top.IssueSignatures(ctx, vgcrypto.EthereumChecksumAddress(is.Submitter), is.ValidatorNodeId, is.Kind)
}

func (app *App) DeliverProtocolUpgradeCommand(ctx context.Context, tx abci.Tx) error {
	pu := &commandspb.ProtocolUpgradeProposal{}
	if err := tx.Unmarshal(pu); err != nil {
		return err
	}
	return app.protocolUpgradeService.UpgradeProposal(ctx, tx.PubKeyHex(), pu.UpgradeBlockHeight, pu.VegaReleaseTag)
}

func (app *App) DeliverAnnounceNode(ctx context.Context, tx abci.Tx) error {
	an := &commandspb.AnnounceNode{}
	if err := tx.Unmarshal(an); err != nil {
		return err
	}

	return app.top.ProcessAnnounceNode(ctx, an)
}

func (app *App) DeliverValidatorHeartbeat(ctx context.Context, tx abci.Tx) error {
	an := &commandspb.ValidatorHeartbeat{}
	if err := tx.Unmarshal(an); err != nil {
		return err
	}

	return app.top.ProcessValidatorHeartbeat(ctx, an, signatures.VerifyVegaSignature, signatures.VerifyEthereumSignature)
}

func (app *App) CheckTransferCommand(_ context.Context, tx abci.Tx) error {
	tfr := &commandspb.Transfer{}
	if err := tx.Unmarshal(tfr); err != nil {
		return err
	}
	party := tx.Party()
	transfer, err := types.NewTransferFromProto("", party, tfr)
	if err != nil {
		return err
	}
	switch transfer.Kind {
	case types.TransferCommandKindOneOff:
		return app.banking.CheckTransfer(transfer.OneOff.TransferBase)
	case types.TransferCommandKindRecurring:
		return app.banking.CheckTransfer(transfer.Recurring.TransferBase)
	default:
		return errors.New("unsupported transfer kind")
	}
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

func (app *App) DeliverStopOrdersSubmission(ctx context.Context, tx abci.Tx, deterministicID string) error {
	s := &commandspb.StopOrdersSubmission{}
	if err := tx.Unmarshal(s); err != nil {
		return err
	}

	// Convert from proto to domain type
	os, err := types.NewStopOrderSubmissionFromProto(s)
	if err != nil {
		return err
	}

	// Submit the create order request to the execution engine
	idgen := idgeneration.New(deterministicID)
	var fallsBelow, risesAbove *string
	if os.FallsBelow != nil {
		fallsBelow = ptr.From(idgen.NextID())
	}
	if os.RisesAbove != nil {
		risesAbove = ptr.From(idgen.NextID())
	}

	_, err = app.exec.SubmitStopOrders(ctx, os, tx.Party(), idgen, fallsBelow, risesAbove)
	if err != nil {
		app.log.Error("could not submit stop order",
			logging.StopOrderSubmission(os), logging.Error(err))
	}

	return nil
}

func (app *App) DeliverStopOrdersCancellation(ctx context.Context, tx abci.Tx, deterministicID string) error {
	s := &commandspb.StopOrdersCancellation{}
	if err := tx.Unmarshal(s); err != nil {
		return err
	}

	// Convert from proto to domain type
	os := types.NewStopOrderCancellationFromProto(s)

	// Submit the create order request to the execution engine
	idgen := idgeneration.New(deterministicID)
	err := app.exec.CancelStopOrders(ctx, os, tx.Party(), idgen)
	if err != nil {
		app.log.Error("could not submit stop order",
			logging.StopOrderCancellation(os), logging.Error(err))
	}

	return nil
}

func (app *App) DeliverSubmitOrder(ctx context.Context, tx abci.Tx, deterministicID string) error {
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
	idgen := idgeneration.New(deterministicID)
	conf, err := app.exec.SubmitOrder(ctx, os, tx.Party(), idgen, idgen.NextID())
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

	return err
}

func (app *App) DeliverCancelOrder(ctx context.Context, tx abci.Tx, deterministicID string) error {
	porder := &commandspb.OrderCancellation{}
	if err := tx.Unmarshal(porder); err != nil {
		return err
	}

	app.stats.IncTotalCancelOrder()
	app.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", porder.OrderId))

	order := types.OrderCancellationFromProto(porder)
	// Submit the cancel new order request to the Vega trading core
	idgen := idgeneration.New(deterministicID)
	msg, err := app.exec.CancelOrder(ctx, order, tx.Party(), idgen)
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

func (app *App) DeliverAmendOrder(
	ctx context.Context,
	tx abci.Tx,
	deterministicID string,
) (errl error) {
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
	idgen := idgeneration.New(deterministicID)
	msg, err := app.exec.AmendOrder(ctx, oa, tx.Party(), idgen)
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
	ctx context.Context, tx abci.Tx, id string,
) error {
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

func (app *App) CheckPropose(_ context.Context, tx abci.Tx) error {
	p := &commandspb.ProposalSubmission{}
	if err := tx.Unmarshal(p); err != nil {
		return err
	}

	propSubmission, err := types.NewProposalSubmissionFromProto(p)
	if err != nil {
		return err
	}

	terms := propSubmission.Terms
	switch terms.Change.GetTermType() {
	case types.ProposalTermsTypeUpdateNetworkParameter:
		return app.netp.IsUpdateAllowed(terms.GetUpdateNetworkParameter().Changes.Key)
	default:
		return nil
	}
}

func (app *App) DeliverPropose(ctx context.Context, tx abci.Tx, deterministicID string) error {
	prop := &commandspb.ProposalSubmission{}
	if err := tx.Unmarshal(prop); err != nil {
		return err
	}

	party := tx.Party()

	if app.log.GetLevel() <= logging.DebugLevel {
		app.log.Debug("submitting proposal",
			logging.ProposalID(deterministicID),
			logging.String("proposal-reference", prop.Reference),
			logging.String("proposal-party", party),
			logging.String("proposal-terms", prop.Terms.String()))
	}

	propSubmission, err := types.NewProposalSubmissionFromProto(prop)
	if err != nil {
		return err
	}
	toSubmit, err := app.gov.SubmitProposal(ctx, *propSubmission, deterministicID, party)
	if err != nil {
		app.log.Debug("could not submit proposal",
			logging.ProposalID(deterministicID),
			logging.Error(err))
		return err
	}

	if toSubmit.IsNewMarket() {
		// opening auction start
		oos := time.Unix(toSubmit.Proposal().Terms.ClosingTimestamp, 0).Round(time.Second)
		nm := toSubmit.NewMarket()

		// @TODO pass in parent and insurance pool share if required
		if err := app.exec.SubmitMarket(ctx, nm.Market(), party, oos); err != nil {
			app.log.Debug("unable to submit new market with liquidity submission",
				logging.ProposalID(nm.Market().ID),
				logging.Error(err))
			// an error happened when submitting the market
			// we should cancel this proposal now
			if err := app.gov.RejectProposal(ctx, toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, err); err != nil {
				// this should never happen
				app.log.Panic("tried to reject a nonexistent proposal",
					logging.String("proposal-id", toSubmit.Proposal().ID),
					logging.Error(err))
			}
			return err
		}
	} else if toSubmit.IsNewSpotMarket() {
		oos := time.Unix(toSubmit.Proposal().Terms.ClosingTimestamp, 0).Round(time.Second)
		nm := toSubmit.NewSpotMarket()
		if err := app.exec.SubmitSpotMarket(ctx, nm.Market(), party, oos); err != nil {
			app.log.Debug("unable to submit new spot market",
				logging.ProposalID(nm.Market().ID),
				logging.Error(err))
			// an error happened when submitting the market
			// we should cancel this proposal now
			if err := app.gov.RejectProposal(ctx, toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, err); err != nil {
				// this should never happen
				app.log.Panic("tried to reject a nonexistent proposal",
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

func (app *App) DeliverLiquidityProvision(ctx context.Context, tx abci.Tx, deterministicID string) error {
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

	return app.exec.SubmitLiquidityProvision(ctx, lps, tx.Party(), deterministicID)
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

func (app *App) DeliverAmendLiquidityProvision(ctx context.Context, tx abci.Tx, deterministicID string) error {
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
	err = app.exec.AmendLiquidityProvision(ctx, lpa, tx.Party(), deterministicID)
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

	pubKey := vgcrypto.NewPublicKey(tx.PubKeyHex(), tx.PubKey())

	return app.witness.AddNodeCheck(ctx, vote, pubKey)
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

	pubKey := vgcrypto.NewPublicKey(tx.PubKeyHex(), tx.PubKey())
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

	pubKey := vgcrypto.NewPublicKey(tx.PubKeyHex(), tx.PubKey())
	oracleData, err := app.oracles.Adaptors.Normalise(pubKey, *data)
	if err != nil {
		return ErrOracleDataNormalization(err)
	}

	if !app.oracles.Engine.ListensToSigners(*oracleData) {
		return ErrUnexpectedTxPubKey
	}

	hasMatch, err := app.oracles.Engine.HasMatch(*oracleData)
	if err != nil {
		return err
	}
	if !hasMatch {
		return ErrOracleNoSubscribers
	}
	return nil
}

func (app *App) onTick(ctx context.Context, t time.Time) {
	toEnactProposals, voteClosedProposals := app.gov.OnTick(ctx, t)
	for _, voteClosed := range voteClosedProposals {
		prop := voteClosed.Proposal()
		switch {
		case voteClosed.IsNewMarket(): // can be spot or futures new market
			// Here we panic in both case as we should never reach a point
			// where we try to Reject or start the opening auction of a
			// non-existing market or any other error would be quite critical
			// anyway...
			nm := voteClosed.NewMarket()
			if nm.Rejected() {
				// RejectMarket can return an error if the proposed successor market was rejected because its parent
				// was already rejected
				if err := app.exec.RejectMarket(ctx, prop.ID); err != nil && !prop.IsSuccessorMarket() {
					app.log.Panic("unable to reject market",
						logging.String("market-id", prop.ID),
						logging.Error(err))
				}
			} else if nm.StartAuction() {
				err := app.exec.StartOpeningAuction(ctx, prop.ID)
				if err != nil {
					if prop.IsSuccessorMarket() {
						app.log.Warn("parent market was already succeeded, market rejected",
							logging.String("market-id", prop.ID),
						)
						prop.FailWithErr(types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketAlreadySucceeded)
					} else {
						app.log.Panic("unable to start market opening auction",
							logging.String("market-id", prop.ID),
							logging.Error(err))
					}
				}
			}
		}
	}

	for _, toEnact := range toEnactProposals {
		prop := toEnact.Proposal()
		switch {
		case prop.IsSuccessorMarket():
			app.enactSuccessorMarket(ctx, prop)
		case toEnact.IsNewMarket():
			app.enactMarket(ctx, prop)
		case toEnact.IsNewSpotMarket():
			app.enactSpotMarket(ctx, prop)
		case toEnact.IsNewAsset():
			app.enactAsset(ctx, prop, toEnact.NewAsset())
		case toEnact.IsUpdateAsset():
			app.enactAssetUpdate(ctx, prop, toEnact.UpdateAsset())
		case toEnact.IsUpdateMarket():
			app.enactUpdateMarket(ctx, prop, toEnact.UpdateMarket())
		case toEnact.IsUpdateSpotMarket():
			app.enactUpdateSpotMarket(ctx, prop, toEnact.UpdateSpotMarket())
		case toEnact.IsUpdateNetworkParameter():
			app.enactNetworkParameterUpdate(ctx, prop, toEnact.UpdateNetworkParameter())
		case toEnact.IsFreeform():
			app.enactFreeform(ctx, prop)
		case toEnact.IsNewTransfer():
			app.enactNewTransfer(ctx, prop)
		case toEnact.IsCancelTransfer():
			app.enactCancelTransfer(ctx, prop)
		case toEnact.IsMarketStateUpdate():
			app.enactMarketStateUpdate(ctx, prop)
		case toEnact.IsReferralProgramUpdate():
			app.referralProgram.UpdateProgram(toEnact.ReferralProgramUpdate())
		default:
			app.log.Error("unknown proposal cannot be enacted", logging.ProposalID(prop.ID))
			prop.FailUnexpectedly(fmt.Errorf("unknown proposal \"%s\" cannot be enacted", prop.ID))
		}

		app.gov.FinaliseEnactment(ctx, prop)
	}
}

func (app *App) enactAsset(ctx context.Context, prop *types.Proposal, _ *types.Asset) {
	prop.State = types.ProposalStateEnacted
	asset, err := app.assets.Get(prop.ID)
	if err != nil {
		app.log.Panic("couldn't retrieve asset when enacting asset update",
			logging.AssetID(prop.ID),
			logging.Error(err))
	}

	// if this is a builtin asset nothing needs to be done, just start the asset
	// straight away
	if asset.IsBuiltinAsset() {
		err = app.banking.EnableBuiltinAsset(ctx, asset.Type().ID)
		if err != nil {
			app.log.Panic("unable to get builtin asset enabled",
				logging.AssetID(prop.ID),
				logging.Error(err))
		}
		return
	}
	app.assets.EnactPendingAsset(prop.ID)
}

func (app *App) enactAssetUpdate(_ context.Context, prop *types.Proposal, updatedAsset *types.Asset) {
	asset, err := app.assets.Get(updatedAsset.ID)
	if err != nil {
		app.log.Panic("couldn't retrieve asset when enacting asset update",
			logging.AssetID(updatedAsset.ID),
			logging.Error(err))
	}

	var signature []byte
	if app.top.IsValidator() {
		switch {
		case asset.IsERC20():
			// need to remove IDs
			nonce, err := num.UintFromHex("0x" + strings.TrimLeft(prop.ID, "0"))
			if err != nil {
				app.log.Panic("couldn't generate nonce from proposal ID",
					logging.AssetID(updatedAsset.ID),
					logging.ProposalID(prop.ID),
					logging.Error(err),
				)
			}
			asset, _ := asset.ERC20()
			_, signature, err = asset.SignSetAssetLimits(
				nonce,
				updatedAsset.Details.GetERC20().LifetimeLimit.Clone(),
				updatedAsset.Details.GetERC20().WithdrawThreshold.Clone(),
			)
			if err != nil {
				app.log.Panic("couldn't to sign transaction to set asset limits, is the node properly configured as a validator?",
					logging.AssetID(updatedAsset.ID),
					logging.Error(err))
			}
		}
	}

	prop.State = types.ProposalStateEnacted

	if err := app.assets.StageAssetUpdate(updatedAsset); err != nil {
		app.log.Panic("couldn't stage the asset update",
			logging.Error(err),
			logging.AssetID(updatedAsset.ID),
		)
	}

	// then instruct the notary to start getting signature from validators
	app.notary.StartAggregate(prop.ID, types.NodeSignatureKindAssetUpdate, signature)
}

func (app *App) enactSuccessorMarket(ctx context.Context, prop *types.Proposal) {
	// @TODO remove parent market (or flag as ready to be removed)
	// transfer the insurance pool balance and ELS state
	// then finally:
	successor := prop.ID
	nm := prop.NewMarket()
	parent := nm.Changes.Successor.ParentID
	if err := app.exec.SucceedMarket(ctx, successor, parent); err != nil {
		prop.State = types.ProposalStateFailed
		prop.ErrorDetails = err.Error()
		return
	}
	prop.State = types.ProposalStateEnacted
}

func (app *App) enactMarket(_ context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted

	// TODO: add checks for end of auction in here
}

func (app *App) enactSpotMarket(_ context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted
}

func (app *App) enactFreeform(_ context.Context, prop *types.Proposal) {
	// There is nothing to enact in a freeform proposal so we just set the state
	prop.State = types.ProposalStateEnacted
}

func (app *App) enactNewTransfer(ctx context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted
	proposal := prop.Terms.GetNewTransfer().Changes

	if err := app.banking.VerifyGovernanceTransfer(proposal); err != nil {
		app.log.Error("failed to enact governance transfer - invalid transfer", logging.String("proposal", prop.ID), logging.String("error", err.Error()))
		prop.FailWithErr(types.ProporsalErrorInvalidGovernanceTransfer, err)
		return
	}

	_ = app.banking.NewGovernanceTransfer(ctx, prop.ID, prop.Reference, proposal)
}

func (app *App) enactCancelTransfer(ctx context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted
	transferID := prop.Terms.GetCancelTransfer().Changes.TransferID
	if err := app.banking.VerifyCancelGovernanceTransfer(transferID); err != nil {
		app.log.Error("failed to enact governance transfer cancellation - invalid transfer cancellation", logging.String("proposal", prop.ID), logging.String("error", err.Error()))
		prop.FailWithErr(types.ProporsalErrorFailedGovernanceTransferCancel, err)
		return
	}
	if err := app.banking.CancelGovTransfer(ctx, transferID); err != nil {
		app.log.Error("failed to enact governance transfer cancellation", logging.String("proposal", prop.ID), logging.String("error", err.Error()))
		prop.FailWithErr(types.ProporsalErrorFailedGovernanceTransferCancel, err)
		return
	}
}

func (app *App) enactMarketStateUpdate(ctx context.Context, prop *types.Proposal) {
	prop.State = types.ProposalStateEnacted
	changes := prop.Terms.GetMarketStateUpdate().Changes
	if err := app.exec.VerifyUpdateMarketState(changes); err != nil {
		app.log.Error("failed to enact governance market state update", logging.String("proposal", prop.ID), logging.String("error", err.Error()))
		prop.FailWithErr(types.ProposalErrorInvalidStateUpdate, err)
		return
	}
	if err := app.exec.UpdateMarketState(ctx, changes); err != nil {
		app.log.Error("failed to enact governance market state update", logging.String("proposal", prop.ID), logging.String("error", err.Error()))
		prop.FailWithErr(types.ProposalErrorInvalidStateUpdate, err)
	}
}

func (app *App) enactNetworkParameterUpdate(ctx context.Context, prop *types.Proposal, np *types.NetworkParameter) {
	prop.State = types.ProposalStateEnacted
	if err := app.netp.Update(ctx, np.Key, np.Value); err != nil {
		prop.FailUnexpectedly(err)
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

func (app *App) DeliverKeyRotateSubmission(ctx context.Context, tx abci.Tx) error {
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

func (app *App) enactUpdateMarket(ctx context.Context, prop *types.Proposal, market *types.Market) {
	if err := app.exec.UpdateMarket(ctx, market); err != nil {
		prop.FailUnexpectedly(err)
		app.log.Error("failed to update market",
			logging.ProposalID(prop.ID),
			logging.Error(err))
		return
	}
	prop.State = types.ProposalStateEnacted
}

func (app *App) enactUpdateSpotMarket(ctx context.Context, prop *types.Proposal, market *types.Market) {
	if err := app.exec.UpdateSpotMarket(ctx, market); err != nil {
		prop.FailUnexpectedly(err)
		app.log.Error("failed to update spot market",
			logging.ProposalID(prop.ID),
			logging.Error(err))
		return
	}
	prop.State = types.ProposalStateEnacted
}

func (app *App) DeliverEthereumKeyRotateSubmission(ctx context.Context, tx abci.Tx) error {
	kr := &commandspb.EthereumKeyRotateSubmission{}
	if err := tx.Unmarshal(kr); err != nil {
		return err
	}

	return app.top.ProcessEthereumKeyRotation(
		ctx,
		tx.PubKeyHex(),
		kr,
		signatures.VerifyEthereumSignature,
	)
}

func (app *App) CreateReferralSet(ctx context.Context, tx abci.Tx, deterministicID string) error {
	params := &commandspb.CreateReferralSet{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize CreateReferralSet command: %w", err)
	}

	if err := app.referralProgram.CreateReferralSet(ctx, types.PartyID(tx.Party()), types.ReferralSetID(deterministicID)); err != nil {
		return err
	}

	if params.IsTeam {
		return app.teamsEngine.CreateTeam(ctx, types.PartyID(tx.Party()), types.TeamID(deterministicID), params.Team)
	}

	return nil
}

// UpdateReferralSet this is effectively Update team, but also served to create
// a team for an existing referral set...
func (app *App) UpdateReferralSet(ctx context.Context, tx abci.Tx) error {
	params := &commandspb.UpdateReferralSet{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize UpdateReferralSet command: %w", err)
	}

	if !app.referralProgram.SetExists(types.ReferralSetID(params.Id)) {
		return fmt.Errorf("no referral set for ID %q", params.Id)
	}

	if params.IsTeam {
		teamID := types.TeamID(params.Id)
		if app.teamsEngine.TeamExists(teamID) {
			return app.teamsEngine.UpdateTeam(ctx, types.PartyID(tx.Party()), teamID, params.Team)
		}

		return app.teamsEngine.CreateTeam(ctx, types.PartyID(tx.Party()), teamID, &commandspb.CreateReferralSet_Team{
			Name:      ptr.UnBox(params.Team.Name),
			AvatarUrl: params.Team.AvatarUrl,
			TeamUrl:   params.Team.TeamUrl,
			Closed:    ptr.UnBox(params.Team.Closed),
		})
	}

	return nil
}

func (app *App) ApplyReferralCode(ctx context.Context, tx abci.Tx) error {
	params := &commandspb.ApplyReferralCode{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize ApplyReferralCode command: %w", err)
	}

	partyID := types.PartyID(tx.Party())
	err := app.referralProgram.ApplyReferralCode(ctx, partyID, types.ReferralSetID(params.Id))

	// it's OK to switch team if the party was already a referrer / referee
	if !errors.Is(err, referral.ErrIsAlreadyAReferee(partyID)) &&
		!errors.Is(err, referral.ErrIsAlreadyAReferrer(partyID)) {
		return fmt.Errorf("could not apply the referral code: %w", err)
	}

	return app.teamsEngine.JoinTeam(ctx, partyID, params)
}
