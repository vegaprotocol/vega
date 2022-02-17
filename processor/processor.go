package processor

import (
	"context"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/pkg/errors"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	ErrInvalidSignature                       = errors.New("invalid signature")
	ErrChainEventFromNonValidator             = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent                  = errors.New("unsupported chain event")
	ErrNodeSignatureFromNonValidator          = errors.New("node signature not sent by validator")
	ErrNodeSignatureWithNonValidatorMasterKey = errors.New("node signature not signed with validator master key")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/processor TimeService
type TimeService interface {
	GetTimeNow() time.Time
	GetTimeLastBatch() time.Time
	NotifyOnTick(f func(context.Context, time.Time))
	SetTimeNow(context.Context, time.Time)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/epoch_service_mock.go -package mocks code.vegaprotocol.io/vega/processor EpochService
type EpochService interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
	OnBlockEnd(ctx context.Context)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor DelegationEngine
type DelegationEngine interface {
	Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateAtEndOfEpoch(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateNow(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor ExecutionEngine
type ExecutionEngine interface {
	// orders stuff
	SubmitOrder(ctx context.Context, orderSubmission *types.OrderSubmission, party string, deterministicId string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation, party string, deterministicId string) ([]*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment, party string, deterministicId string) (*types.OrderConfirmation, error)

	// market stuff
	SubmitMarket(ctx context.Context, marketConfig *types.Market) error
	SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpid, deterministicId string) error
	RejectMarket(ctx context.Context, marketid string) error
	StartOpeningAuction(ctx context.Context, marketid string) error

	// LP stuff
	SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicId string) error
	CancelLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionCancellation, party string) error
	AmendLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionAmendment, party string, deterministicId string) error
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, types.ProposalSubmission, string, string) (*governance.ToSubmit, error)
	FinaliseEnactment(ctx context.Context, prop *types.Proposal)
	AddVote(context.Context, types.VoteSubmission, string) error
	OnChainTimeUpdate(context.Context, time.Time) ([]*governance.ToEnact, []*governance.VoteClosed)
	RejectProposal(context.Context, *types.Proposal, types.ProposalError, error) error
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stats_mock.go -package mocks code.vegaprotocol.io/vega/processor Stats
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
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/processor Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetDetails) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/processor ValidatorTopology
type ValidatorTopology interface {
	Len() int
	IsValidatorVegaPubKey(pk string) bool
	IsValidatorNodeID(nodeID string) bool
	AllVegaPubKeys() []string
	IsValidator() bool
	AddKeyRotate(ctx context.Context, nodeID string, currentBlockHeight uint64, kr *commandspb.KeyRotateSubmission) error
	BeginBlock(ctx context.Context, req abcitypes.RequestBeginBlock, vd []*tmtypes.Validator)
	GetValidatorPowerUpdates() []abcitypes.ValidatorUpdate
	ProcessAnnounceNode(ctx context.Context, nr *commandspb.AnnounceNode) error
	ProcessValidatorHeartbeat(context.Context, *commandspb.ValidatorHeartbeat, func(message, signature, pubkey []byte) error, func(message, signature []byte, hexAddress string) error) error
	AddForwarder(ID string)
}

// Broker - the event bus.
type Broker interface {
	Send(e events.Event)
	SetStreaming(on bool) bool
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/processor Notary
type Notary interface {
	StartAggregate(resID string, kind commandspb.NodeSignatureKind, signature []byte)
	RegisterSignature(ctx context.Context, pubKey string, ns commandspb.NodeSignature) error
	IsSigned(context.Context, string, commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool)
}

// Witness ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/processor Witness
type Witness interface {
	AddNodeCheck(ctx context.Context, nv *commandspb.NodeVote) error
}

// EvtForwarder ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evtforwarder_mock.go -package mocks code.vegaprotocol.io/vega/processor EvtForwarder
type EvtForwarder interface {
	Ack(*commandspb.ChainEvent) bool
}

// Banking ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/banking_mock.go -package mocks code.vegaprotocol.io/vega/processor Banking
type Banking interface {
	EnableBuiltinAsset(context.Context, string) error
	DepositBuiltinAsset(context.Context, *types.BuiltinAssetDeposit, string, uint64) error
	WithdrawBuiltinAsset(context.Context, string, string, string, *num.Uint) error
	EnableERC20(context.Context, *types.ERC20AssetList, string, uint64, uint64, string) error
	DepositERC20(context.Context, *types.ERC20Deposit, string, uint64, uint64, string) error
	WithdrawERC20(context.Context, string, string, string, *num.Uint, *types.Erc20WithdrawExt) error
	ERC20WithdrawalEvent(context.Context, *types.ERC20Withdrawal, uint64, uint64, string) error
	TransferFunds(context.Context, *types.TransferFunds) error
	CancelTransferFunds(context.Context, *types.CancelTransferFunds) error
}

// NetworkParameters ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/network_parameters_mock.go -package mocks code.vegaprotocol.io/vega/processor NetworkParameters
type NetworkParameters interface {
	Update(ctx context.Context, key, value string) error
	DispatchChanges(ctx context.Context)
}

type Oracle struct {
	Engine   OraclesEngine
	Adaptors OracleAdaptors
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracles_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor OraclesEngine
type OraclesEngine interface {
	BroadcastData(context.Context, oracles.OracleData) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_adaptors_mock.go -package mocks code.vegaprotocol.io/vega/processor OracleAdaptors
type OracleAdaptors interface {
	Normalise(crypto.PublicKey, commandspb.OracleDataSubmission) (*oracles.OracleData, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/limits_mock.go -package mocks code.vegaprotocol.io/vega/processor Limits
type Limits interface {
	CanProposeMarket() bool
	CanProposeAsset() bool
	CanTrade() bool
	BootstrapFinished() bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stake_verifier_mock.go -package mocks code.vegaprotocol.io/vega/processor StakeVerifier
type StakeVerifier interface {
	ProcessStakeRemoved(ctx context.Context, event *types.StakeRemoved) error
	ProcessStakeDeposited(ctx context.Context, event *types.StakeDeposited) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/staking_accounts_mock.go -package mocks code.vegaprotocol.io/vega/processor StakingAccounts
type StakingAccounts interface {
	Hash() []byte
	ProcessStakeTotalSupply(ctx context.Context, event *types.StakeTotalSupply) error
}
