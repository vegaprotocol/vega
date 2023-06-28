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
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/pkg/errors"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/processor TimeService,EpochService,DelegationEngine,ExecutionEngine,GovernanceEngine,Stats,Assets,ValidatorTopology,Notary,EvtForwarder,Witness,Banking,NetworkParameters,OraclesEngine,OracleAdaptors,Limits,StakeVerifier,StakingAccounts,ERC20MultiSigTopology,Checkpoint

var (
	ErrInvalidSignature                       = errors.New("invalid signature")
	ErrChainEventFromNonValidator             = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent                  = errors.New("unsupported chain event")
	ErrNodeSignatureFromNonValidator          = errors.New("node signature not sent by validator")
	ErrNodeSignatureWithNonValidatorMasterKey = errors.New("node signature not signed with validator master key")
	ErrMarketBatchInstructionTooBig           = func(got, expected uint64) error {
		return fmt.Errorf("market batch instructions too big, got(%d), expected(%d)", got, expected)
	}
)

type TimeService interface {
	GetTimeNow() time.Time
	GetTimeLastBatch() time.Time
	NotifyOnTick(...func(context.Context, time.Time))
	SetTimeNow(context.Context, time.Time)
}

type EpochService interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
	OnBlockEnd(ctx context.Context)
}

type DelegationEngine interface {
	Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateAtEndOfEpoch(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateNow(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	Hash() []byte
}

//nolint:interfacebloat
type ExecutionEngine interface {
	// orders stuff
	SubmitOrder(ctx context.Context, orderSubmission *types.OrderSubmission, party string, idgen common.IDGenerator, orderID string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation, party string, idgen common.IDGenerator) ([]*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment, party string, idgen common.IDGenerator) (*types.OrderConfirmation, error)

	// stop orders stuff
	SubmitStopOrders(ctx context.Context, stopOrdersSubmission *types.StopOrdersSubmission, party string, idgen common.IDGenerator, stopOrderID1, stopOrderID2 *string) (*types.OrderConfirmation, error)
	CancelStopOrders(ctx context.Context, stopOrdersCancellation *types.StopOrdersCancellation, party string, idgen common.IDGenerator) error

	// market stuff
	SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error
	UpdateMarket(ctx context.Context, marketConfig *types.Market) error
	RejectMarket(ctx context.Context, marketid string) ([]int, error)
	StartOpeningAuction(ctx context.Context, marketid string) error
	SucceedMarket(ctx context.Context, successor, parent string) error

	// LP stuff
	SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicID string) error
	CancelLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionCancellation, party string) error
	AmendLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionAmendment, party string, deterministicID string) error
	Hash() []byte

	// End of block
	BlockEnd(ctx context.Context)
}

type GovernanceEngine interface {
	SubmitProposal(context.Context, types.ProposalSubmission, string, string) (*governance.ToSubmit, error)
	FinaliseEnactment(ctx context.Context, prop *types.Proposal)
	AddVote(context.Context, types.VoteSubmission, string) error
	OnTick(context.Context, time.Time) ([]*governance.ToEnact, []*governance.VoteClosed)
	RejectProposal(context.Context, *types.Proposal, types.ProposalError, error) error
	Hash() []byte
}

//nolint:interfacebloat
type Stats interface {
	IncTotalCreateOrder()
	AddCurrentTradesInBatch(i uint64)
	AddTotalTrades(i uint64) uint64
	IncTotalOrders()
	IncCurrentOrdersInBatch()
	IncTotalCancelOrder()
	IncTotalAmendOrder()
	// batch stats
	IncTotalBatches()
	NewBatch()
	TotalOrders() uint64
	TotalBatches() uint64
	SetAverageOrdersPerBatch(i uint64)
	SetBlockDuration(uint64)
	CurrentOrdersInBatch() uint64
	CurrentTradesInBatch() uint64
	SetOrdersPerSecond(i uint64)
	SetTradesPerSecond(i uint64)
	CurrentEventsInBatch() uint64
	SetEventsPerSecond(uint64)
	// blockchain stats
	IncTotalTxCurrentBatch()
	IncHeight()
	Height() uint64
	SetAverageTxPerBatch(i uint64)
	SetAverageTxSizeBytes(i uint64)
	SetTotalTxLastBatch(i uint64)
	SetTotalTxCurrentBatch(i uint64)
	TotalTxCurrentBatch() uint64
	TotalTxLastBatch() uint64
	SetHash(string)
	SetHeight(uint64)
}

type Assets interface {
	NewAsset(ctx context.Context, ref string, assetSrc *types.AssetDetails) (string, error)
	StageAssetUpdate(*types.Asset) error
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
	EnactPendingAsset(assetID string)
}

//nolint:interfacebloat
type ValidatorTopology interface {
	Len() int
	IsValidatorVegaPubKey(pk string) bool
	IsValidatorNodeID(nodeID string) bool
	AllVegaPubKeys() []string
	IsValidator() bool
	AddKeyRotate(ctx context.Context, nodeID string, currentBlockHeight uint64, kr *commandspb.KeyRotateSubmission) error
	ProcessEthereumKeyRotation(ctx context.Context, nodeID string, kr *commandspb.EthereumKeyRotateSubmission, verify func(message, signature []byte, hexAddress string) error) error
	BeginBlock(ctx context.Context, req abcitypes.RequestBeginBlock)
	GetValidatorPowerUpdates() []abcitypes.ValidatorUpdate
	ProcessAnnounceNode(ctx context.Context, nr *commandspb.AnnounceNode) error
	ProcessValidatorHeartbeat(context.Context, *commandspb.ValidatorHeartbeat, func(message, signature, pubkey []byte) error, func(message, signature []byte, hexAddress string) error) error
	AddForwarder(ID string)
	IssueSignatures(ctx context.Context, submitter, nodeID string, kind types.NodeSignatureKind) error
}

// Broker - the event bus.
type Broker interface {
	Send(e events.Event)
	SetStreaming(on bool) bool
	StreamingEnabled() bool
	SocketClient() broker.SocketClient
}

// Notary.
type Notary interface {
	StartAggregate(resID string, kind commandspb.NodeSignatureKind, signature []byte)
	RegisterSignature(ctx context.Context, pubKey string, ns commandspb.NodeSignature) error
	IsSigned(context.Context, string, commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool)
}

// Witness ...
type Witness interface {
	AddNodeCheck(ctx context.Context, nv *commandspb.NodeVote, key crypto.PublicKey) error
}

// EvtForwarder ...
type EvtForwarder interface {
	Ack(*commandspb.ChainEvent) bool
}

// Banking ..
//
//nolint:interfacebloat
type Banking interface {
	EnableBuiltinAsset(context.Context, string) error
	DepositBuiltinAsset(context.Context, *types.BuiltinAssetDeposit, string, uint64) error
	WithdrawBuiltinAsset(context.Context, string, string, string, *num.Uint) error
	EnableERC20(context.Context, *types.ERC20AssetList, string, uint64, uint64, string) error
	UpdateERC20(context.Context, *types.ERC20AssetLimitsUpdated, string, uint64, uint64, string) error
	DepositERC20(context.Context, *types.ERC20Deposit, string, uint64, uint64, string) error
	WithdrawERC20(context.Context, string, string, string, *num.Uint, *types.Erc20WithdrawExt) error
	ERC20WithdrawalEvent(context.Context, *types.ERC20Withdrawal, uint64, uint64, string) error
	TransferFunds(context.Context, *types.TransferFunds) error
	CancelTransferFunds(context.Context, *types.CancelTransferFunds) error
	BridgeStopped(context.Context, bool, string, uint64, uint64, string) error
	BridgeResumed(context.Context, bool, string, uint64, uint64, string) error
	CheckTransfer(t *types.TransferBase) error
	NewGovernanceTransfer(ctx context.Context, ID, reference string, transferConfig *types.NewTransferConfiguration) error
	VerifyGovernanceTransfer(transfer *types.NewTransferConfiguration) error
	VerifyCancelGovernanceTransfer(transferID string) error
	CancelGovTransfer(ctx context.Context, ID string) error
}

// NetworkParameters ...
type NetworkParameters interface {
	Update(ctx context.Context, key, value string) error
	DispatchChanges(ctx context.Context)
	IsUpdateAllowed(key string) error
}

type Oracle struct {
	Engine                  OraclesEngine
	Adaptors                OracleAdaptors
	EthereumOraclesVerifier EthereumOracleVerifier
}

type OraclesEngine interface {
	BroadcastData(context.Context, oracles.OracleData) error
	ListensToSigners(oracles.OracleData) bool
	HasMatch(data oracles.OracleData) (bool, error)
}

type OracleAdaptors interface {
	Normalise(crypto.PublicKey, commandspb.OracleDataSubmission) (*oracles.OracleData, error)
}

type EthereumOracleVerifier interface {
	ProcessEthereumContractCallResult(callEvent types.EthContractCallEvent) error
}

type Limits interface {
	CanProposeMarket() bool
	CanProposeAsset() bool
	CanTrade() bool
}

type StakeVerifier interface {
	ProcessStakeRemoved(ctx context.Context, event *types.StakeRemoved) error
	ProcessStakeDeposited(ctx context.Context, event *types.StakeDeposited) error
}

type StakingAccounts interface {
	Hash() []byte
	ProcessStakeTotalSupply(ctx context.Context, event *types.StakeTotalSupply) error
}

type ERC20MultiSigTopology interface {
	ProcessSignerEvent(event *types.SignerEvent) error
	ProcessThresholdEvent(event *types.SignerThresholdSetEvent) error
}
