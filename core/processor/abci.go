// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/api"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/pow"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/core/vegatime"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	signatures "code.vegaprotocol.io/vega/libs/crypto/signature"
	verrors "code.vegaprotocol.io/vega/libs/errors"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	proto "code.vegaprotocol.io/vega/protos/vega"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	tmtypes "github.com/cometbft/cometbft/abci/types"
	tmtypes1 "github.com/cometbft/cometbft/proto/tendermint/types"
	types1 "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypesint "github.com/cometbft/cometbft/types"
	"go.uber.org/zap"
)

const AppVersion = 1

type TxWrapper struct {
	tx        abci.Tx
	timeIndex int // this is an indicator of insertion order
	raw       []byte
	priority  uint64
	gasWanted uint64
}

var (
	ErrUnexpectedTxPubKey          = errors.New("no one listens to the public keys that signed this oracle data")
	ErrTradingDisabled             = errors.New("trading disabled")
	ErrMarketProposalDisabled      = errors.New("market proposal disabled")
	ErrAssetProposalDisabled       = errors.New("asset proposal disabled")
	ErrEthOraclesDisabled          = errors.New("ethereum oracles disabled")
	ErrOracleNoSubscribers         = errors.New("there are no subscribes to the oracle data")
	ErrSpotMarketProposalDisabled  = errors.New("spot market proposal disabled")
	ErrPerpsMarketProposalDisabled = errors.New("perps market proposal disabled")
	ErrAMMPoolDisabled             = errors.New("amm pool disabled")
	ErrOracleDataNormalization     = func(err error) error {
		return fmt.Errorf("error normalizing incoming oracle data: %w", err)
	}
)

// Codec interface is here for mocking/testing.
type Codec interface {
	abci.Codec
}

type Checkpoint interface {
	BalanceCheckpoint(ctx context.Context) (*types.CheckpointState, error)
	Checkpoint(ctx context.Context, now time.Time) (*types.CheckpointState, error)
}

type SpamEngine interface {
	BeginBlock(txs []abci.Tx)
	EndPrepareProposal()
	PreBlockAccept(tx abci.Tx) error
	ProcessProposal(txs []abci.Tx) bool
	CheckBlockTx(tx abci.Tx) error
}

type PoWEngine interface {
	api.ProofOfWorkParams
	BeginBlock(blockHeight uint64, blockHash string, txs []abci.Tx)
	CheckBlockTx(tx abci.Tx) (pow.ValidationResult, *uint)
	ProcessProposal(txs []abci.Tx) bool
	EndPrepareProposal([]pow.ValidationEntry)
	CheckTx(tx abci.Tx) error
	GetSpamStatistics(partyID string) *protoapi.PoWStatistic
	OnCommit()
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

	// debug snapshot issues/hash mismatch problems
	SnapshotDump(ctx context.Context, path string) ([]byte, error)
}

type StateVarEngine interface {
	ProposedValueReceived(ctx context.Context, ID, nodeID, eventID string, bundle *statevar.KeyValueBundle) error
	OnBlockEnd(ctx context.Context)
}

type TeamsEngine interface {
	TeamExists(team types.TeamID) bool
	CreateTeam(context.Context, types.PartyID, types.TeamID, *commandspb.CreateReferralSet_Team) error
	UpdateTeam(context.Context, types.PartyID, types.TeamID, *commandspb.UpdateReferralSet_Team) error
	JoinTeam(context.Context, types.PartyID, *commandspb.JoinTeam) error
}

type PartiesEngine interface {
	UpdateProfile(context.Context, types.PartyID, *commandspb.UpdatePartyProfile) error
	CheckSufficientBalanceToUpdateProfile(party types.PartyID, balance *num.Uint) error
}

type ReferralProgram interface {
	UpdateProgram(program *types.ReferralProgram)
	PartyOwnsReferralSet(types.PartyID, types.ReferralSetID) error
	CreateReferralSet(context.Context, types.PartyID, types.ReferralSetID) error
	ApplyReferralCode(context.Context, types.PartyID, types.ReferralSetID) error
	CheckSufficientBalanceForApplyReferralCode(types.PartyID, *num.Uint) error
	CheckSufficientBalanceForCreateOrUpdateReferralSet(types.PartyID, *num.Uint) error
}

type VolumeDiscountProgram interface {
	UpdateProgram(program *types.VolumeDiscountProgram)
}

type VolumeRebateProgram interface {
	UpdateProgram(program *types.VolumeRebateProgram)
}

type BlockchainClient interface {
	Validators(height *int64) ([]*tmtypesint.Validator, error)
	MaxMempoolSize() int64
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

type BalanceChecker interface {
	GetPartyBalance(party string) *num.Uint
	BeginBlock(context.Context)
}

type EthCallEngine interface {
	Start()
}

type TxCache interface {
	SetRawTxs(rtx [][]byte, height uint64)
	GetRawTxs(height uint64) [][]byte
	NewDelayedTransaction(ctx context.Context, delayed [][]byte, height uint64) []byte
	IsDelayRequired(marketID string) bool
	IsDelayRequiredAnyMarket() bool
	IsTxInCache(tx []byte) bool
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

	// service injection
	assets                         Assets
	banking                        Banking
	broker                         Broker
	witness                        Witness
	evtForwarder                   EvtForwarder
	evtHeartbeat                   EvtForwarderHeartbeat
	primaryChainID                 uint64
	secondaryChainID               uint64
	exec                           ExecutionEngine
	ghandler                       *genesis.Handler
	gov                            GovernanceEngine
	notary                         Notary
	stats                          Stats
	time                           TimeService
	top                            ValidatorTopology
	netp                           NetworkParameters
	oracles                        *Oracle
	delegation                     DelegationEngine
	limits                         Limits
	stake                          StakeVerifier
	stakingAccounts                StakingAccounts
	checkpoint                     Checkpoint
	spam                           SpamEngine
	pow                            PoWEngine
	epoch                          EpochService
	snapshotEngine                 SnapshotEngine
	stateVar                       StateVarEngine
	teamsEngine                    TeamsEngine
	partiesEngine                  PartiesEngine
	referralProgram                ReferralProgram
	volumeDiscountProgram          VolumeDiscountProgram
	volumeRebateProgram            VolumeRebateProgram
	protocolUpgradeService         ProtocolUpgradeService
	primaryErc20MultiSigTopology   ERC20MultiSigTopology
	secondaryErc20MultiSigTopology ERC20MultiSigTopology
	gastimator                     *Gastimator
	ethCallEngine                  EthCallEngine
	balanceChecker                 BalanceChecker

	nilPow  bool
	nilSpam bool

	maxBatchSize atomic.Uint64
	txCache      TxCache
}

func NewApp(log *logging.Logger,
	vegaPaths paths.Paths,
	config Config,
	cancelFn func(),
	stopBlockchain func() error,
	assets Assets,
	banking Banking,
	broker Broker,
	witness Witness,
	evtForwarder EvtForwarder,
	evtHeartbeat EvtForwarderHeartbeat,
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
	volumeDiscountProgram VolumeDiscountProgram,
	volumeRebateProgram VolumeRebateProgram,
	blockchainClient BlockchainClient,
	primaryMultisig ERC20MultiSigTopology,
	secondaryMultisig ERC20MultiSigTopology,
	version string,
	protocolUpgradeService ProtocolUpgradeService,
	codec abci.Codec,
	gastimator *Gastimator,
	ethCallEngine EthCallEngine,
	balanceChecker BalanceChecker,
	partiesEngine PartiesEngine,
	txCache TxCache,
) *App {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	app := &App{
		abci: abci.New(codec),

		log:                            log,
		vegaPaths:                      vegaPaths,
		cfg:                            config,
		cancelFn:                       cancelFn,
		stopBlockchain:                 stopBlockchain,
		assets:                         assets,
		banking:                        banking,
		broker:                         broker,
		witness:                        witness,
		evtForwarder:                   evtForwarder,
		evtHeartbeat:                   evtHeartbeat,
		exec:                           exec,
		ghandler:                       ghandler,
		gov:                            gov,
		notary:                         notary,
		stats:                          stats,
		time:                           time,
		top:                            top,
		netp:                           netp,
		oracles:                        oracles,
		delegation:                     delegation,
		limits:                         limits,
		stake:                          stake,
		checkpoint:                     checkpoint,
		spam:                           spam,
		pow:                            pow,
		stakingAccounts:                stakingAccounts,
		epoch:                          epoch,
		snapshotEngine:                 snapshot,
		stateVar:                       stateVarEngine,
		teamsEngine:                    teamsEngine,
		referralProgram:                referralProgram,
		volumeDiscountProgram:          volumeDiscountProgram,
		volumeRebateProgram:            volumeRebateProgram,
		version:                        version,
		blockchainClient:               blockchainClient,
		primaryErc20MultiSigTopology:   primaryMultisig,
		secondaryErc20MultiSigTopology: secondaryMultisig,
		protocolUpgradeService:         protocolUpgradeService,
		gastimator:                     gastimator,
		ethCallEngine:                  ethCallEngine,
		balanceChecker:                 balanceChecker,
		partiesEngine:                  partiesEngine,
		txCache:                        txCache,
	}

	// setup handlers
	app.abci.OnPrepareProposal = app.prepareProposal
	app.abci.OnProcessProposal = app.processProposal
	app.abci.OnInitChain = app.OnInitChain
	app.abci.OnBeginBlock = app.OnBeginBlock
	app.abci.OnEndBlock = app.OnEndBlock
	app.abci.OnCommit = app.OnCommit
	app.abci.OnCheckTx = app.OnCheckTx
	app.abci.OnCheckTxSpam = app.OnCheckTxSpam
	app.abci.OnInfo = app.Info
	app.abci.OnFinalize = app.Finalize
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
		HandleCheckTx(txn.BatchProposeCommand, addDeterministicID(app.CheckBatchPropose)).
		HandleCheckTx(txn.TransferFundsCommand, app.CheckTransferCommand).
		HandleCheckTx(txn.ApplyReferralCodeCommand, app.CheckApplyReferralCode).
		HandleCheckTx(txn.CreateReferralSetCommand, app.CheckCreateOrUpdateReferralSet).
		HandleCheckTx(txn.UpdateReferralSetCommand, app.CheckCreateOrUpdateReferralSet).
		HandleCheckTx(txn.SubmitOrderCommand, app.CheckOrderSubmissionForSpam).
		HandleCheckTx(txn.LiquidityProvisionCommand, app.CheckLPSubmissionForSpam).
		HandleCheckTx(txn.AmendLiquidityProvisionCommand, app.CheckLPAmendForSpam).
		HandleCheckTx(txn.SubmitAMMCommand, app.CheckSubmitAmmForSpam).
		HandleCheckTx(txn.AmendAMMCommand, app.CheckAmendAmmForSpam).
		HandleCheckTx(txn.AmendOrderCommand, app.CheckAmendOrderForSpam).
		HandleCheckTx(txn.CancelOrderCommand, app.CheckCancelOrderForSpam).
		HandleCheckTx(txn.CancelAMMCommand, app.CheckCancelAmmForSpam).
		HandleCheckTx(txn.CancelLiquidityProvisionCommand, app.CheckCancelLPForSpam)

	// node commands
	app.abci.HandleDeliverTx(txn.NodeSignatureCommand,
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
		HandleDeliverTx(txn.SubmitAMMCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverSubmitAMM),
			),
		).
		HandleDeliverTx(txn.AmendAMMCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverAmendAMM),
			),
		).
		HandleDeliverTx(txn.CancelAMMCommand,
			app.SendTransactionResult(
				addDeterministicID(app.DeliverCancelAMM),
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
		HandleDeliverTx(txn.BatchProposeCommand,
			app.SendTransactionResult(
				app.CheckBatchProposeW(
					addDeterministicID(app.DeliverBatchPropose),
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
		).
		HandleDeliverTx(txn.UpdateMarginModeCommand,
			app.SendTransactionResult(app.UpdateMarginMode),
		).
		HandleDeliverTx(txn.JoinTeamCommand,
			app.SendTransactionResult(app.JoinTeam),
		).
		HandleDeliverTx(txn.UpdatePartyProfileCommand,
			app.SendTransactionResult(app.UpdatePartyProfile),
		).
		HandleDeliverTx(txn.DelayedTransactionsWrapper,
			app.SendTransactionResult(app.handleDelayedTransactionWrapper))

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

// generateDeterministicID will build the command ID
// the command ID is built using the signature of the proposer of the command
// the signature is then hashed with sha3_256
// the hash is the hex string encoded.
func generateDeterministicID(tx abci.Tx) string {
	return hex.EncodeToString(vgcrypto.Hash(tx.Signature()))
}

// addDeterministicID decorates give function with deterministic ID.
func addDeterministicID(
	f func(context.Context, abci.Tx, string) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		return f(ctx, tx, generateDeterministicID(tx))
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

func (app *App) CheckBatchProposeW(
	f func(context.Context, abci.Tx) error,
) func(context.Context, abci.Tx) error {
	return func(ctx context.Context, tx abci.Tx) error {
		if err := addDeterministicID(app.CheckBatchPropose)(ctx, tx); err != nil {
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

	v := &proto.EthereumConfig{}
	if err := app.netp.GetJSONStruct(netparams.BlockchainsPrimaryEthereumConfig, v); err != nil {
		return
	}
	primaryChainID, err := strconv.ParseUint(v.ChainId, 10, 64)
	if err != nil {
		return
	}
	app.primaryChainID = primaryChainID
	_ = app.gov.OnChainIDUpdate(primaryChainID)
	_ = app.exec.OnChainIDUpdate(primaryChainID)

	bridgeConfigs := &proto.EVMBridgeConfigs{}
	if err := app.netp.GetJSONStruct(netparams.BlockchainsEVMBridgeConfigs, bridgeConfigs); err != nil {
		return
	}

	secondaryChainID, err := strconv.ParseUint(bridgeConfigs.Configs[0].ChainId, 10, 64)
	if err != nil {
		return
	}
	app.secondaryChainID = secondaryChainID
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

func (app *App) Info(_ context.Context, _ *tmtypes.RequestInfo) (*tmtypes.ResponseInfo, error) {
	if len(app.lastBlockAppHash) != 0 {
		// we must've lost connection to tendermint for a bit, tell it where we got up to
		height, _ := vgcontext.BlockHeightFromContext(app.blockCtx)
		app.log.Info("ABCI service INFO requested after reconnect",
			logging.Uint64("height", height),
			logging.String("hash", hex.EncodeToString(app.lastBlockAppHash)),
		)
		return &tmtypes.ResponseInfo{
			AppVersion:       AppVersion,
			Version:          app.version,
			LastBlockHeight:  int64(height),
			LastBlockAppHash: app.lastBlockAppHash,
		}, nil
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
	return &resp, nil
}

func (app *App) ListSnapshots(_ context.Context, _ *tmtypes.RequestListSnapshots) (*tmtypes.ResponseListSnapshots, error) {
	app.log.Debug("ABCI service ListSnapshots requested")
	latestSnapshots, err := app.snapshotEngine.ListLatestSnapshots()
	if err != nil {
		app.log.Error("Could not list latest snapshots", logging.Error(err))
		return &tmtypes.ResponseListSnapshots{}, err
	}
	return &tmtypes.ResponseListSnapshots{
		Snapshots: latestSnapshots,
	}, nil
}

func (app *App) OfferSnapshot(_ context.Context, req *tmtypes.RequestOfferSnapshot) (*tmtypes.ResponseOfferSnapshot, error) {
	app.log.Debug("ABCI service OfferSnapshot start")
	if app.snapshotEngine.HasRestoredStateAlready() {
		app.log.Warn("The snapshot engine aborted the snapshot offer from state-sync since the state has already been restored")
		return &tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_ABORT,
		}, nil
	}

	deserializedSnapshot, err := types.SnapshotFromTM(req.Snapshot)
	if err != nil {
		app.log.Error("Could not deserialize snapshot", logging.Error(err))
		return &tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT_SENDER,
		}, err
	}

	// check that our unpacked snapshot's hash matches that which tendermint thinks it sent
	if !bytes.Equal(deserializedSnapshot.Hash, req.AppHash) {
		app.log.Error("The hashes from the request and the deserialized snapshot mismatch",
			logging.String("deserialized-hash", hex.EncodeToString(deserializedSnapshot.Hash)),
			logging.String("request-hash", hex.EncodeToString(req.AppHash)))
		return &tmtypes.ResponseOfferSnapshot{
			Result: tmtypes.ResponseOfferSnapshot_REJECT,
		}, fmt.Errorf("hash mismatch")
	}

	res := app.snapshotEngine.ReceiveSnapshot(deserializedSnapshot)
	return &res, nil
}

func (app *App) ApplySnapshotChunk(ctx context.Context, req *tmtypes.RequestApplySnapshotChunk) (*tmtypes.ResponseApplySnapshotChunk, error) {
	app.log.Debug("ABCI service ApplySnapshotChunk start")

	if app.snapshotEngine.HasRestoredStateAlready() {
		app.log.Warn("The snapshot engine aborted the snapshot chunk from state-sync since the state has already been restored")
		return &tmtypes.ResponseApplySnapshotChunk{
			Result: tmtypes.ResponseApplySnapshotChunk_ABORT,
		}, nil // ???
	}
	chunk := &types.RawChunk{
		Nr:   req.Index,
		Data: req.Chunk,
	}

	res := app.snapshotEngine.ReceiveSnapshotChunk(ctx, chunk, req.Sender)
	return &res, nil
}

func (app *App) LoadSnapshotChunk(_ context.Context, req *tmtypes.RequestLoadSnapshotChunk) (*tmtypes.ResponseLoadSnapshotChunk, error) {
	app.log.Debug("ABCI service LoadSnapshotChunk start")
	raw, err := app.snapshotEngine.RetrieveSnapshotChunk(req.Height, req.Format, req.Chunk)
	if err != nil {
		app.log.Error("failed to load snapshot chunk", logging.Error(err), logging.Uint64("height", req.Height))
		return &tmtypes.ResponseLoadSnapshotChunk{}, err
	}
	return &tmtypes.ResponseLoadSnapshotChunk{
		Chunk: raw.Data,
	}, nil
}

func (app *App) OnInitChain(req *tmtypes.RequestInitChain) (*tmtypes.ResponseInitChain, error) {
	app.log.Debug("ABCI service InitChain start")
	hash := hex.EncodeToString(vgcrypto.Hash([]byte(req.ChainId)))
	app.abci.SetChainID(req.ChainId)
	app.chainCtx = vgcontext.WithChainID(context.Background(), req.ChainId)
	ctx := vgcontext.WithBlockHeight(app.chainCtx, uint64(req.InitialHeight))
	ctx = vgcontext.WithTraceID(ctx, hash)
	app.blockCtx = ctx

	app.log.Debug("OnInitChain-NewBeginBlock", logging.Uint64("height", uint64(req.InitialHeight)), logging.Time("blockTime", req.Time), logging.String("blockHash", hash))

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

	app.ethCallEngine.Start()

	return &tmtypes.ResponseInitChain{
		Validators: app.top.GetValidatorPowerUpdates(),
	}, nil
}

// prepareProposal takes an ordered slice of transactions and decides which of them go into the next block.
// The logic for selection is as follows:
// 1. mempool transactions are sorted by priority then insertion order (aka time)
// 2. we add *valid* transaction to the block so long as gas and maxBytes limits are not violated
// 3. we never add transactions failing pow checks
// 4. we never add transactions failing spam checks
// therefore a block generated with this method will never contain any transactions that would violate spam/pow constraints that would have previously
// caused the party to get blocked.
func (app *App) prepareProposal(height uint64, txs []abci.Tx, rawTxs [][]byte) [][]byte {
	var totalBytes int64
	validationResults := []pow.ValidationEntry{}

	// internally we use this as max bytes, externally to consensus params we return max ints. This is done so that cometbft always returns to us the full mempool
	// and we can first sort it by priority and then reap by size.
	maxBytes := tmtypesint.DefaultBlockParams().MaxBytes * 4
	app.log.Debug("prepareProposal called with", logging.Int("txs", len(rawTxs)), logging.Int64("max-bytes", maxBytes))

	// as transactions that are wrapped for sending in the next block are not removed from the mempool
	// to avoid adding them both from the mempool and from the cache we need to check
	// they were not in the cache.
	// we still need to check that the transactions from previous block are passing pow and spam requirements.
	addedFromPreviousHash := map[string]struct{}{}
	delayedTxs := [][]byte{}
	for _, txx := range app.txCache.GetRawTxs(height) {
		tx, err := app.abci.GetTx(txx)
		if err != nil {
			continue
		}
		if !app.nilPow {
			vr, d := app.pow.CheckBlockTx(tx)
			validationResults = append(validationResults, pow.ValidationEntry{Tx: tx, Difficulty: d, ValResult: vr})
			if vr != pow.ValidationResultSuccess && vr != pow.ValidationResultValidatorCommand {
				app.log.Debug("pow failure", logging.Int64("validation-result", int64(vr)))
				continue
			}
		}
		if !app.nilSpam {
			err := app.spam.CheckBlockTx(tx)
			if err != nil {
				app.log.Debug("spam error", logging.Error(err))
				continue
			}
		}
		if err := app.canSubmitTx(tx); err != nil {
			continue
		}

		addedFromPreviousHash[hex.EncodeToString(tx.Hash())] = struct{}{}
		delayedTxs = append(delayedTxs, txx)
		totalBytes += int64(len(txx))
	}

	// wrap the transaction with information about gas wanted and priority
	wrappedTxs := make([]*TxWrapper, 0, len(txs))
	for i, v := range txs {
		wtx, error := app.wrapTx(v, rawTxs[i], i)
		if error != nil {
			continue
		}
		if _, ok := addedFromPreviousHash[hex.EncodeToString(wtx.tx.Hash())]; ok {
			app.log.Debug("ignoring mempool transaction corresponding to a delayed transaction from previous block")
			continue
		}
		wrappedTxs = append(wrappedTxs, wtx)
	}

	// sort by priority descending. If priority is equal use the order in the mempol ascending
	sort.Slice(wrappedTxs, func(i, j int) bool {
		if wrappedTxs[i].priority == wrappedTxs[j].priority {
			return wrappedTxs[i].timeIndex < wrappedTxs[j].timeIndex
		}
		return wrappedTxs[i].priority > wrappedTxs[j].priority
	})

	// add transactions to the block as long as we can without breaking size and gas limits in order of priority
	maxGas := app.getMaxGas()
	totalGasWanted := uint64(0)
	cancellations := [][]byte{}
	postOnly := [][]byte{}
	anythingElseFromThisBlock := [][]byte{}
	nextBlockRtx := [][]byte{}

	for _, tx := range wrappedTxs {
		totalBytes += int64(len(tx.raw))
		if totalBytes > maxBytes {
			break
		}
		totalGasWanted += tx.gasWanted
		if totalGasWanted > maxGas {
			break
		}

		if tx.tx.Command() == txn.DelayedTransactionsWrapper {
			app.log.Debug("delayed transaction wrapper should never be submitted into the mempool")
			continue
		}

		if !app.nilPow {
			vr, d := app.pow.CheckBlockTx(tx.tx)
			validationResults = append(validationResults, pow.ValidationEntry{Tx: tx.tx, Difficulty: d, ValResult: vr})
			if vr != pow.ValidationResultSuccess && vr != pow.ValidationResultValidatorCommand {
				app.log.Debug("pow failure", logging.Int64("validation-result", int64(vr)))
				continue
			}
		}

		if !app.nilSpam {
			err := app.spam.CheckBlockTx(tx.tx)
			if err != nil {
				app.log.Debug("spam error", logging.Error(err))
				continue
			}
		}

		if err := app.canSubmitTx(tx.tx); err != nil {
			continue
		}

		switch tx.tx.Command() {
		case txn.CancelOrderCommand:
			s := &commandspb.OrderCancellation{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if len(s.MarketId) > 0 {
				if app.txCache.IsDelayRequired(s.MarketId) {
					cancellations = append(cancellations, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			} else if app.txCache.IsDelayRequiredAnyMarket() {
				cancellations = append(cancellations, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.CancelAMMCommand:
			s := &commandspb.CancelAMM{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if len(s.MarketId) > 0 {
				if app.txCache.IsDelayRequired(s.MarketId) {
					cancellations = append(cancellations, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			} else if app.txCache.IsDelayRequiredAnyMarket() {
				cancellations = append(cancellations, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.StopOrdersCancellationCommand:
			s := &commandspb.StopOrdersCancellation{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if s.MarketId != nil {
				if app.txCache.IsDelayRequired(*s.MarketId) {
					cancellations = append(cancellations, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			} else if app.txCache.IsDelayRequiredAnyMarket() {
				cancellations = append(cancellations, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.SubmitOrderCommand:
			s := &commandspb.OrderSubmission{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if s.PostOnly {
				postOnly = append(postOnly, tx.raw)
			} else if app.txCache.IsDelayRequired(s.MarketId) {
				nextBlockRtx = append(nextBlockRtx, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.AmendOrderCommand:
			s := &commandspb.OrderAmendment{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if app.txCache.IsDelayRequired(s.MarketId) {
				nextBlockRtx = append(nextBlockRtx, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.AmendAMMCommand:
			s := &commandspb.AmendAMM{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if app.txCache.IsDelayRequired(s.MarketId) {
				nextBlockRtx = append(nextBlockRtx, tx.raw)
			} else {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
			}
		case txn.StopOrdersSubmissionCommand:
			s := &commandspb.StopOrdersSubmission{}
			if err := tx.tx.Unmarshal(s); err != nil {
				continue
			}
			if s.RisesAbove != nil && s.FallsBelow == nil {
				if app.txCache.IsDelayRequired(s.RisesAbove.OrderSubmission.MarketId) {
					nextBlockRtx = append(nextBlockRtx, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			} else if s.FallsBelow != nil && s.RisesAbove == nil {
				if app.txCache.IsDelayRequired(s.FallsBelow.OrderSubmission.MarketId) {
					nextBlockRtx = append(nextBlockRtx, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			} else if s.FallsBelow != nil && s.RisesAbove != nil {
				if app.txCache.IsDelayRequired(s.FallsBelow.OrderSubmission.MarketId) || app.txCache.IsDelayRequired(s.RisesAbove.OrderSubmission.MarketId) {
					nextBlockRtx = append(nextBlockRtx, tx.raw)
				} else {
					anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				}
			}
		case txn.BatchMarketInstructions:
			batch := &commandspb.BatchMarketInstructions{}
			if err := tx.tx.Unmarshal(batch); err != nil {
				continue
			}
			someMarketRequiresDelay := false
			for _, s := range batch.Submissions {
				if app.txCache.IsDelayRequired(s.MarketId) {
					someMarketRequiresDelay = true
					break
				}
			}
			if !someMarketRequiresDelay {
				for _, s := range batch.Amendments {
					if app.txCache.IsDelayRequired(s.MarketId) {
						someMarketRequiresDelay = true
						break
					}
				}
			}
			if !someMarketRequiresDelay {
				for _, s := range batch.Cancellations {
					if len(s.MarketId) != 0 && app.txCache.IsDelayRequired(s.MarketId) {
						someMarketRequiresDelay = true
						break
					}
				}
			}
			if !someMarketRequiresDelay {
				for _, s := range batch.StopOrdersSubmission {
					if s.FallsBelow != nil && s.FallsBelow.OrderSubmission != nil && app.txCache.IsDelayRequired(s.FallsBelow.OrderSubmission.MarketId) {
						someMarketRequiresDelay = true
						break
					}
					if !someMarketRequiresDelay {
						if s.RisesAbove != nil && s.RisesAbove.OrderSubmission != nil && app.txCache.IsDelayRequired(s.RisesAbove.OrderSubmission.MarketId) {
							someMarketRequiresDelay = true
							break
						}
					}
				}
			}
			if !someMarketRequiresDelay {
				for _, s := range batch.StopOrdersCancellation {
					if app.txCache.IsDelayRequired(*s.MarketId) {
						someMarketRequiresDelay = true
						break
					}
				}
			}
			if !someMarketRequiresDelay {
				anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
				continue
			}
			// if there are no amends/submissions
			if len(batch.Amendments) == 0 && len(batch.Submissions) == 0 && len(batch.StopOrdersSubmission) == 0 {
				cancellations = append(cancellations, tx.raw)
			} else if len(batch.Amendments) == 0 && len(batch.StopOrdersSubmission) == 0 {
				allPostOnly := true
				for _, sub := range batch.Submissions {
					if !sub.PostOnly {
						allPostOnly = false
						break
					}
				}
				if allPostOnly {
					postOnly = append(postOnly, tx.raw)
				} else {
					nextBlockRtx = append(nextBlockRtx, tx.raw)
				}
			} else {
				nextBlockRtx = append(nextBlockRtx, tx.raw)
			}
		default:
			anythingElseFromThisBlock = append(anythingElseFromThisBlock, tx.raw)
		}
	}
	blockTxs := [][]byte{}
	blockTxs = append(blockTxs, cancellations...) // cancellations go first
	blockTxs = append(blockTxs, postOnly...)      // then post only orders
	if delayedTxs != nil {
		blockTxs = append(blockTxs, delayedTxs...) // then anything from previous block
	}
	blockTxs = append(blockTxs, anythingElseFromThisBlock...) // finally anything else from this block
	if len(nextBlockRtx) > 0 {
		wrapperTx := app.txCache.NewDelayedTransaction(app.blockCtx, nextBlockRtx, height)
		blockTxs = append(blockTxs, wrapperTx)
	}
	if !app.nilPow {
		app.pow.EndPrepareProposal(validationResults)
	}
	if !app.nilSpam {
		app.spam.EndPrepareProposal()
	}
	return blockTxs
}

// processProposal takes a block proposal and verifies that it has no malformed or offending transactions which should never be if the validator is using the prepareProposal
// to generate a block.
// The verifications include:
// 1. no violations of pow and spam
// 2. max gas limit is not exceeded
// 3. (soft) max bytes is not exceeded.
func (app *App) processProposal(height uint64, txs []abci.Tx) bool {
	totalGasWanted := 0
	maxGas := app.gastimator.GetMaxGas()
	maxBytes := tmtypesint.DefaultBlockParams().MaxBytes * 4
	size := int64(0)
	delayedTxCount := 0

	expectedDelayedAtHeight := app.txCache.GetRawTxs(height)
	expectedDelayedTxs := make(map[string]struct{}, len(expectedDelayedAtHeight))
	for _, tx := range expectedDelayedAtHeight {
		txx, err := app.abci.GetTx(tx)
		if err == nil {
			expectedDelayedTxs[hex.EncodeToString(txx.Hash())] = struct{}{}
		}
	}
	foundDelayedTxs := make(map[string]struct{}, len(expectedDelayedAtHeight))

	for _, tx := range txs {
		size += int64(tx.GetLength())
		if size > maxBytes {
			return false
		}
		gw, err := app.getGasWanted(tx)
		if err != nil {
			return false
		}
		totalGasWanted += int(gw)
		if totalGasWanted > int(maxGas) {
			return false
		}
		// allow only one delayed transaction wrapper in one block and its transactions must match what we expect.
		if tx.Command() == txn.DelayedTransactionsWrapper {
			if delayedTxCount > 0 {
				app.log.Debug("more than one DelayedTransactionsWrapper")
				return false
			}
			delayedTxCount += 1
		}
		if _, ok := expectedDelayedTxs[hex.EncodeToString(tx.Hash())]; ok {
			foundDelayedTxs[hex.EncodeToString(tx.Hash())] = struct{}{}
		}
	}

	if len(foundDelayedTxs) != len(expectedDelayedAtHeight) {
		return false
	}

	if !app.nilPow && !app.pow.ProcessProposal(txs) {
		return false
	}

	if !app.nilSpam && !app.spam.ProcessProposal(txs) {
		return false
	}
	return true
}

func (app *App) OnEndBlock(blockHeight uint64) (tmtypes.ValidatorUpdates, types1.ConsensusParams) {
	app.log.Debug("entering end block", logging.Time("at", time.Now()))
	defer func() { app.log.Debug("leaving end block", logging.Time("at", time.Now())) }()

	app.log.Debug("ABCI service END block completed",
		logging.Int64("current-timestamp", app.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", app.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(app.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(app.previousTimestamp)),
	)

	app.epoch.OnBlockEnd(app.blockCtx)
	app.stateVar.OnBlockEnd(app.blockCtx)
	app.banking.OnBlockEnd(app.blockCtx, app.currentTimestamp)

	powerUpdates := app.top.GetValidatorPowerUpdates()
	if len(powerUpdates) == 0 {
		powerUpdates = tmtypes.ValidatorUpdates{}
	}

	// update max gas based on the network parameter
	consensusParamUpdates := types1.ConsensusParams{
		Block: &types1.BlockParams{
			MaxGas:   int64(app.gastimator.OnBlockEnd()),
			MaxBytes: -1, // we tell comet that we always want to get the full mempool
		},
		Version: &tmtypes1.VersionParams{
			App: AppVersion,
		},
	}
	app.exec.BlockEnd(app.blockCtx)

	return powerUpdates, consensusParamUpdates
}

// OnBeginBlock updates the internal lastBlockTime value with each new block.
func (app *App) OnBeginBlock(blockHeight uint64, blockHash string, blockTime time.Time, proposer string, txs []abci.Tx) context.Context {
	app.log.Debug("entering begin block", logging.Time("at", time.Now()), logging.Uint64("height", blockHeight), logging.Time("time", blockTime), logging.String("blockHash", blockHash))
	defer func() { app.log.Debug("leaving begin block", logging.Time("at", time.Now())) }()

	app.txCache.SetRawTxs(nil, blockHeight)

	ctx := vgcontext.WithBlockHeight(vgcontext.WithTraceID(app.chainCtx, blockHash), blockHeight)
	if app.protocolUpgradeService.CoreReadyForUpgrade() {
		app.startProtocolUpgrade(ctx)
	}
	app.log.Info("WWW sending Begin block", logging.Uint64("h", blockHeight))
	app.broker.Send(events.NewBeginBlock(ctx, eventspb.BeginBlock{
		Height:    blockHeight,
		Timestamp: blockTime.UnixNano(),
		Hash:      blockHash,
	}))
	app.cBlock = blockHash

	for _, tx := range txs {
		app.setTxStats(tx.GetLength())
	}

	// update pow engine on a new block
	if !app.nilPow {
		app.pow.BeginBlock(blockHeight, blockHash, txs)
	}

	if !app.nilSpam {
		app.spam.BeginBlock(txs)
	}

	app.stats.SetHash(blockHash)
	app.stats.SetHeight(blockHeight)
	app.blockCtx = ctx
	now := blockTime
	app.time.SetTimeNow(ctx, now)
	app.currentTimestamp = app.time.GetTimeNow()
	app.previousTimestamp = app.time.GetTimeLastBatch()
	app.log.Debug("ABCI service BEGIN completed",
		logging.Int64("current-timestamp", app.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", app.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(app.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(app.previousTimestamp)),
		logging.Uint64("height", blockHeight),
	)

	app.protocolUpgradeService.BeginBlock(ctx, blockHeight)
	app.top.BeginBlock(ctx, blockHeight, proposer)
	app.balanceChecker.BeginBlock(ctx)
	blockDuration := app.currentTimestamp.Sub(app.previousTimestamp)
	app.exec.BeginBlock(ctx, blockDuration)
	return ctx
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
				app.log.Panic("failed to wait for data node to get ready for upgrade", logging.Error(err))
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

// Finalize calculates the app hash for the block ending.
func (app *App) Finalize() []byte {
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
	} else if app.cfg.SnapshotDebug.DevEnabled {
		if height, _ := vgcontext.BlockHeightFromContext(app.blockCtx); height == app.cfg.SnapshotDebug.CrashAtHeight {
			hash, err := app.snapshotEngine.SnapshotDump(app.blockCtx, app.cfg.SnapshotDebug.DebugCrashFile)
			if err != nil {
				app.log.Panic("Failed to dump snapshot file", logging.Error(err), logging.String("snapshot-hash", string(hash)))
			} else {
				app.log.Panic("Dumped snapshot file successfully", logging.String("snapshot-hash", string(hash)), logging.String("dump-file", app.cfg.SnapshotDebug.DebugCrashFile))
			}
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
	appHash := snapHash

	if len(snapHash) == 0 {
		appHash = vgcrypto.Hash([]byte(app.version))
		appHash = append(appHash, app.exec.Hash()...)
		appHash = append(appHash, app.delegation.Hash()...)
		appHash = append(appHash, app.gov.Hash()...)
		appHash = append(appHash, app.stakingAccounts.Hash()...)
	}

	if cpt != nil {
		if len(snapHash) == 0 {
			// only append to commit hash if we aren't using the snapshot hash
			// otherwise restoring a checkpoint would restore an incomplete/wrong hash
			appHash = append(appHash, cpt.Hash...)
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
		appHash = vgcrypto.Hash(appHash)
	} else {
		app.broker.Send(events.NewSnapshotEventEvent(app.blockCtx, app.stats.Height(), app.cBlock, app.protocolUpgradeService.TimeForUpgrade()))
	}

	// Update response and save the apphash incase we lose connection with tendermint and need to verify our
	// current state
	app.log.Debug("apphash calculated", logging.String("response-data", hex.EncodeToString(appHash)))
	return appHash
}

func (app *App) OnCommit() (*tmtypes.ResponseCommit, error) {
	app.log.Debug("entering commit", logging.Time("at", time.Now()), logging.Uint64("height", app.stats.Height()))
	defer func() { app.log.Debug("leaving commit", logging.Time("at", time.Now())) }()
	app.updateStats()
	app.setBatchStats()
	if !app.nilPow {
		app.pow.OnCommit()
	}
	app.log.Info("WWW sending end block", logging.Uint64("h", app.stats.Height()))
	app.broker.Send(
		events.NewEndBlock(app.blockCtx, eventspb.EndBlock{
			Height: app.stats.Height(),
		}),
	)

	return &tmtypes.ResponseCommit{}, nil
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

	if app.txCache.IsTxInCache(tx.Hash()) {
		resp.Code = blockchain.AbciSpamError
		resp.Data = []byte("delayed transaction already included in a block")
		return resp
	}

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
		if err := app.spam.PreBlockAccept(tx); err != nil {
			app.log.Error(err.Error())
			resp.Code = blockchain.AbciSpamError
			resp.Data = []byte(err.Error())
			return resp
		}
	}
	return resp
}

// OnCheckTx performs soft validations.
func (app *App) OnCheckTx(ctx context.Context, _ *tmtypes.RequestCheckTx, tx abci.Tx) (context.Context, *tmtypes.ResponseCheckTx) {
	resp := tmtypes.ResponseCheckTx{}

	if app.log.IsDebug() {
		app.log.Debug("entering checkTx", logging.String("tid", tx.GetPoWTID()), logging.String("command", tx.Command().String()))
	}

	if err := app.canSubmitTx(tx); err != nil {
		resp.Code = blockchain.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
		return ctx, &resp
	}

	gasWanted, err := app.gastimator.CalcGasWantedForTx(tx)
	if err != nil { // this error means the transaction couldn't be parsed
		app.log.Error("error getting gas estimate", logging.Error(err))
		resp.Code = blockchain.AbciTxnValidationFailure
		resp.Data = []byte(err.Error())
		return ctx, &resp
	}

	resp.GasWanted = int64(gasWanted)
	if app.log.IsDebug() {
		app.log.Debug("transaction passed checkTx", logging.String("tid", tx.GetPoWTID()), logging.String("command", tx.Command().String()))
	}

	return ctx, &resp
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
	case txn.SubmitAMMCommand, txn.AmendAMMCommand, txn.CancelAMMCommand:
		if !app.limits.CanUseAMMPool() {
			return ErrAMMPoolDisabled
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
			return validateUseOfEthOracles(p.Terms.Change, app.netp)
		case types.ProposalTermsTypeUpdateMarket:
			return validateUseOfEthOracles(p.Terms.Change, app.netp)

		case types.ProposalTermsTypeNewAsset:
			if !app.limits.CanProposeAsset() {
				return ErrAssetProposalDisabled
			}
		case types.ProposalTermsTypeNewSpotMarket:
			if !app.limits.CanProposeSpotMarket() {
				return ErrSpotMarketProposalDisabled
			}
		}
	case txn.BatchProposeCommand:
		ps := &commandspb.BatchProposalSubmission{}
		if err := tx.Unmarshal(ps); err != nil {
			return fmt.Errorf("could not unmarshal batch proposal submission: %w", err)
		}

		idgen := idgeneration.New(generateDeterministicID(tx))
		ids := make([]string, 0, len(ps.Terms.Changes))

		for i := 0; i < len(ps.Terms.Changes); i++ {
			ids = append(ids, idgen.NextID())
		}

		p, err := types.NewBatchProposalSubmissionFromProto(ps, ids)
		if err != nil {
			return fmt.Errorf("invalid batch proposal submission: %w", err)
		}
		if p.Terms == nil || len(p.Terms.Changes) == 0 {
			return errors.New("invalid batch proposal submission")
		}

		for _, batchChange := range p.Terms.Changes {
			switch c := batchChange.Change.(type) {
			case *types.ProposalTermsNewMarket:
				if !app.limits.CanProposeMarket() {
					return ErrMarketProposalDisabled
				}

				if c.NewMarket.Changes.ProductType() == types.ProductTypePerps && !app.limits.CanProposePerpsMarket() {
					return ErrPerpsMarketProposalDisabled
				}
				return validateUseOfEthOracles(c, app.netp)
			case *types.ProposalTermsUpdateMarket:
				return validateUseOfEthOracles(c, app.netp)

			case *types.ProposalTermsNewSpotMarket:
				if !app.limits.CanProposeSpotMarket() {
					return ErrSpotMarketProposalDisabled
				}
			}
		}
	}
	return nil
}

func validateUseOfEthOracles(change types.ProposalTerm, netp NetworkParameters) error {
	ethOracleEnabled, _ := netp.GetInt(netparams.EthereumOraclesEnabled)

	switch c := change.(type) {
	case *types.ProposalTermsNewMarket:
		m := c.NewMarket

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

	case *types.ProposalTermsUpdateMarket:
		m := c.UpdateMarket

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
	size := uint64(len(bmi.UpdateMarginMode) + len(bmi.Cancellations) + len(bmi.Amendments) + len(bmi.Submissions) + len(bmi.StopOrdersSubmission) + len(bmi.StopOrdersCancellation))
	if size > maxBatchSize {
		return ErrMarketBatchInstructionTooBig(size, maxBatchSize)
	}

	for _, s := range bmi.Submissions {
		if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), s.MarketId); err != nil {
			return err
		}

		os, err := types.NewOrderSubmissionFromProto(s)
		if err != nil {
			return err
		}

		if err := app.exec.CheckOrderSubmissionForSpam(os, tx.Party()); err != nil {
			return err
		}
	}

	for _, s := range bmi.Amendments {
		if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), s.MarketId); err != nil {
			return err
		}
		// TODO add amend checks
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

	return app.top.IssueSignatures(ctx, vgcrypto.EthereumChecksumAddress(is.Submitter), is.ValidatorNodeId, is.ChainId, is.Kind)
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

func (app *App) CheckApplyReferralCode(_ context.Context, tx abci.Tx) error {
	if err := app.referralProgram.CheckSufficientBalanceForApplyReferralCode(types.PartyID(tx.Party()), app.balanceChecker.GetPartyBalance(tx.Party())); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckCreateOrUpdateReferralSet(_ context.Context, tx abci.Tx) error {
	if err := app.referralProgram.CheckSufficientBalanceForCreateOrUpdateReferralSet(types.PartyID(tx.Party()), app.balanceChecker.GetPartyBalance(tx.Party())); err != nil {
		return err
	}
	return nil
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

func (app *App) DeliverWithdraw(ctx context.Context, tx abci.Tx, id string) error {
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

func (app *App) CheckCancelOrderForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.OrderCancellation{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckCancelAmmForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.CancelAMM{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckCancelLPForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.LiquidityProvisionCancellation{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckAmendOrderForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.OrderAmendment{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckAmendAmmForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.AmendAMM{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckSubmitAmmForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.SubmitAMM{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckLPSubmissionForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.LiquidityProvisionSubmission{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckLPAmendForSpam(_ context.Context, tx abci.Tx) error {
	sub := &commandspb.LiquidityProvisionAmendment{}
	if err := tx.Unmarshal(sub); err != nil {
		return err
	}
	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), sub.MarketId); err != nil {
		return err
	}
	return nil
}

func (app *App) CheckOrderSubmissionForSpam(_ context.Context, tx abci.Tx) error {
	s := &commandspb.OrderSubmission{}
	if err := tx.Unmarshal(s); err != nil {
		return err
	}

	if err := app.exec.CheckCanSubmitOrderOrLiquidityCommitment(tx.Party(), s.MarketId); err != nil {
		return err
	}

	// Convert from proto to domain type
	os, err := types.NewOrderSubmissionFromProto(s)
	if err != nil {
		return err
	}

	return app.exec.CheckOrderSubmissionForSpam(os, tx.Party())
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

func (app *App) CheckBatchPropose(_ context.Context, tx abci.Tx, deterministicBatchID string) error {
	p := &commandspb.BatchProposalSubmission{}
	if err := tx.Unmarshal(p); err != nil {
		return err
	}

	idgen := idgeneration.New(deterministicBatchID)
	ids := make([]string, 0, len(p.Terms.Changes))

	for i := 0; i < len(p.Terms.Changes); i++ {
		ids = append(ids, idgen.NextID())
	}

	propSubmission, err := types.NewBatchProposalSubmissionFromProto(p, ids)
	if err != nil {
		return err
	}

	errs := verrors.NewCumulatedErrors()
	for _, change := range propSubmission.Terms.Changes {
		switch term := change.Change.(type) {
		case *types.ProposalTermsUpdateNetworkParameter:
			if err := app.netp.IsUpdateAllowed(term.UpdateNetworkParameter.Changes.Key); err != nil {
				errs.Add(err)
			}
		}
	}

	if errs.HasAny() {
		return errs
	}

	return nil
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

func (app *App) DeliverBatchPropose(ctx context.Context, tx abci.Tx, deterministicBatchID string) (err error) {
	prop := &commandspb.BatchProposalSubmission{}
	if err := tx.Unmarshal(prop); err != nil {
		return err
	}

	party := tx.Party()

	if app.log.GetLevel() <= logging.DebugLevel {
		app.log.Debug("submitting batch proposal",
			logging.ProposalID(deterministicBatchID),
			logging.String("proposal-reference", prop.Reference),
			logging.String("proposal-party", party),
			logging.String("proposal-terms", prop.Terms.String()))
	}

	idgen := idgeneration.New(deterministicBatchID)

	// Burn one so the first proposal doesn't have the same ID as the batch ID
	idgen.NextID()
	ids := make([]string, 0, len(prop.Terms.Changes))

	for i := 0; i < len(prop.Terms.Changes); i++ {
		ids = append(ids, idgen.NextID())
	}

	propSubmission, err := types.NewBatchProposalSubmissionFromProto(prop, ids)
	if err != nil {
		return err
	}
	toSubmits, err := app.gov.SubmitBatchProposal(ctx, *propSubmission, deterministicBatchID, party)
	if err != nil {
		app.log.Debug("could not submit batch proposal",
			logging.ProposalID(deterministicBatchID),
			logging.Error(err))
		return err
	}

	var submittedMarketIDs []string
	defer func() {
		if err == nil {
			return
		}

		// an error happened when submitting the market
		// we should cancel this proposal now
		if err := app.gov.RejectBatchProposal(ctx, deterministicBatchID,
			types.ProposalErrorCouldNotInstantiateMarket, err); err != nil {
			// this should never happen
			app.log.Panic("tried to reject a nonexistent batch proposal",
				logging.String("proposal-id", deterministicBatchID),
				logging.Error(err))
		}

		for _, marketID := range submittedMarketIDs {
			if err := app.exec.RejectMarket(ctx, marketID); err != nil {
				// this should never happen
				app.log.Panic("unable to submit reject submitted market",
					logging.ProposalID(marketID),
					logging.Error(err))
			}
		}
	}()

	for _, toSubmit := range toSubmits {
		if toSubmit.IsNewMarket() {
			// opening auction start
			oos := time.Unix(toSubmit.Proposal().Terms.ClosingTimestamp, 0).Round(time.Second)
			nm := toSubmit.NewMarket()

			// @TODO pass in parent and insurance pool share if required
			if err = app.exec.SubmitMarket(ctx, nm.Market(), party, oos); err != nil {
				app.log.Debug("unable to submit new market with liquidity submission",
					logging.ProposalID(nm.Market().ID),
					logging.Error(err))
				return err
			}

			submittedMarketIDs = append(submittedMarketIDs, nm.Market().ID)
		} else if toSubmit.IsNewSpotMarket() {
			oos := time.Unix(toSubmit.Proposal().Terms.ClosingTimestamp, 0).Round(time.Second)
			nm := toSubmit.NewSpotMarket()
			if err = app.exec.SubmitSpotMarket(ctx, nm.Market(), party, oos); err != nil {
				app.log.Debug("unable to submit new spot market",
					logging.ProposalID(nm.Market().ID),
					logging.Error(err))
				return err
			}

			submittedMarketIDs = append(submittedMarketIDs, nm.Market().ID)
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
		app.log.Debug("error on cancelling order",
			logging.PartyID(tx.Party()),
			logging.String("liquidity-provision-market-id", lpc.MarketID),
			logging.Error(err))
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
		app.log.Debug("error on amending Liquidity Provision",
			logging.String("liquidity-provision-market-id", lpa.MarketID),
			logging.PartyID(tx.Party()),
			logging.Error(err))
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
			prop.State = types.ProposalStateEnacted
			app.referralProgram.UpdateProgram(toEnact.ReferralProgramChanges())
		case toEnact.IsVolumeDiscountProgramUpdate():
			prop.State = types.ProposalStateEnacted
			app.volumeDiscountProgram.UpdateProgram(toEnact.VolumeDiscountProgramUpdate())
		case toEnact.IsVolumeRebateProgramUpdate():
			prop.State = types.ProposalStateEnacted
			app.volumeRebateProgram.UpdateProgram(toEnact.VolumeRebateProgramUpdate())
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
		currentBlockHeight,
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

func (app *App) wrapTx(tx abci.Tx, rawTx []byte, insertionOrder int) (*TxWrapper, error) {
	priority := app.getPriority(tx)
	gasWanted, err := app.getGasWanted(tx)
	if err != nil {
		return nil, err
	}

	return &TxWrapper{
		tx:        tx,
		timeIndex: insertionOrder,
		raw:       rawTx,
		priority:  priority,
		gasWanted: gasWanted,
	}, nil
}

func (app *App) getPriority(tx abci.Tx) uint64 {
	return app.gastimator.GetPriority(tx)
}

func (app *App) getGasWanted(tx abci.Tx) (uint64, error) {
	return app.gastimator.CalcGasWantedForTx(tx)
}

func (app *App) getMaxGas() uint64 {
	return app.gastimator.maxGas
}

func (app *App) UpdateMarginMode(ctx context.Context, tx abci.Tx) error {
	var err error
	params := &commandspb.UpdateMarginMode{}
	if err = tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize UpdateMarginMode command: %w", err)
	}
	marginFactor := num.DecimalZero()
	if params.MarginFactor != nil && len(*params.MarginFactor) > 0 {
		marginFactor, err = num.DecimalFromString(*params.MarginFactor)
		if err != nil {
			return err
		}
	}
	return app.exec.UpdateMarginMode(ctx, tx.Party(), params.MarketId, types.MarginMode(params.Mode), marginFactor)
}

func (app *App) DeliverSubmitAMM(ctx context.Context, tx abci.Tx, deterministicID string) error {
	params := &commandspb.SubmitAMM{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize SubmitAMM command: %w", err)
	}

	submit := types.NewSubmitAMMFromProto(params, tx.Party())
	return app.exec.SubmitAMM(ctx, submit, deterministicID)
}

func (app *App) DeliverAmendAMM(ctx context.Context, tx abci.Tx, deterministicID string) error {
	params := &commandspb.AmendAMM{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize AmendAMM command: %w", err)
	}

	amend := types.NewAmendAMMFromProto(params, tx.Party())
	return app.exec.AmendAMM(ctx, amend, deterministicID)
}

func (app *App) DeliverCancelAMM(ctx context.Context, tx abci.Tx, deterministicID string) error {
	params := &commandspb.CancelAMM{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize CancelAMM command: %w", err)
	}

	cancel := types.NewCancelAMMFromProto(params, tx.Party())
	return app.exec.CancelAMM(ctx, cancel, deterministicID)
}

func (app *App) CreateReferralSet(ctx context.Context, tx abci.Tx, deterministicID string) error {
	params := &commandspb.CreateReferralSet{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize CreateReferralSet command: %w", err)
	}

	if !params.DoNotCreateReferralSet {
		if err := app.referralProgram.CreateReferralSet(ctx, types.PartyID(tx.Party()), types.ReferralSetID(deterministicID)); err != nil {
			return err
		}
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

	// Is this relevant at all now? With anyone able to create a team, this verification should not matter.
	//
	// if err := app.referralProgram.PartyOwnsReferralSet(types.PartyID(tx.Party()), types.ReferralSetID(params.Id)); err != nil {
	//     return fmt.Errorf("cannot update referral set: %w", err)
	// }

	// ultimately this has just become a createOrUpdateTeam.
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
	if err != nil {
		return fmt.Errorf("could not apply the referral code: %w", err)
	}

	if !params.DoNotJoinTeam {
		teamID := types.TeamID(params.Id)
		joinTeam := &commandspb.JoinTeam{
			Id: params.Id,
		}
		err = app.teamsEngine.JoinTeam(ctx, partyID, joinTeam)
		// This is ok as well, as not all referral sets are teams as well.
		if err != nil && err.Error() != teams.ErrNoTeamMatchesID(teamID).Error() {
			return fmt.Errorf("couldn't join team: %w", err)
		}
	}

	return nil
}

func (app *App) JoinTeam(ctx context.Context, tx abci.Tx) error {
	params := &commandspb.JoinTeam{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize JoinTeam command: %w", err)
	}

	partyID := types.PartyID(tx.Party())
	err := app.teamsEngine.JoinTeam(ctx, partyID, params)
	if err != nil {
		return fmt.Errorf("couldn't join team: %w", err)
	}

	return nil
}

func (app *App) handleDelayedTransactionWrapper(ctx context.Context, tx abci.Tx) error {
	txs := &commandspb.DelayedTransactionsWrapper{}
	if err := tx.Unmarshal(txs); err != nil {
		return fmt.Errorf("could not deserialize DelayedTransactionsWrapper command: %w", err)
	}
	app.txCache.SetRawTxs(txs.Transactions, txs.Height)
	return nil
}

func (app *App) UpdatePartyProfile(ctx context.Context, tx abci.Tx) error {
	params := &commandspb.UpdatePartyProfile{}
	if err := tx.Unmarshal(params); err != nil {
		return fmt.Errorf("could not deserialize UpdatePartyProfile command: %w", err)
	}

	err := app.partiesEngine.CheckSufficientBalanceToUpdateProfile(
		types.PartyID(tx.Party()),
		app.balanceChecker.GetPartyBalance(tx.Party()),
	)
	if err != nil {
		return err
	}

	partyID := types.PartyID(tx.Party())
	err = app.partiesEngine.UpdateProfile(ctx, partyID, params)
	if err != nil {
		return fmt.Errorf("couldn't update profile: %w", err)
	}

	return nil
}

func (app *App) OnBlockchainPrimaryEthereumConfigUpdate(_ context.Context, conf any) error {
	cfg, err := types.EthereumConfigFromUntypedProto(conf)
	if err != nil {
		return err
	}
	cID, err := strconv.ParseUint(cfg.ChainID(), 10, 64)
	if err != nil {
		return err
	}
	app.primaryChainID = cID
	_ = app.exec.OnChainIDUpdate(cID)
	return app.gov.OnChainIDUpdate(cID)
}

func (app *App) OnBlockchainEVMChainConfigUpdate(_ context.Context, conf any) error {
	cfg, err := types.EVMChainConfigFromUntypedProto(conf)
	if err != nil {
		return err
	}
	cID, err := strconv.ParseUint(cfg.Configs[0].ChainID(), 10, 64)
	if err != nil {
		return err
	}
	app.secondaryChainID = cID
	return nil
}
