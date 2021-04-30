package processor

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrInvalidSignature              = errors.New("invalid signature")
	ErrVegaWalletRequired            = errors.New("vega wallet required")
	ErrChainEventFromNonValidator    = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent         = errors.New("unsupported chain event")
	ErrNodeSignatureFromNonValidator = errors.New("node signature not sent by validator")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/processor TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
	NotifyOnTick(f func(context.Context, time.Time))
	SetTimeNow(context.Context, time.Time)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor ExecutionEngine
type ExecutionEngine interface {
	// orders stuff
	SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) ([]*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment) (*types.OrderConfirmation, error)

	// market stuff
	SubmitMarket(ctx context.Context, marketConfig *types.Market) error
	SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpid string) error
	RejectMarket(ctx context.Context, marketid string) error
	StartOpeningAuction(ctx context.Context, marketid string) error

	// LP stuff
	SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, id string) error

	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, types.ProposalSubmission, string, string) (*governance.ToSubmit, error)
	AddVote(context.Context, types.VoteSubmission, string) error
	OnChainTimeUpdate(context.Context, time.Time) ([]*governance.ToEnact, []*governance.VoteClosed)
	RejectProposal(context.Context, *types.Proposal, types.ProposalError) error
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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/processor Wallet
type Wallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/processor Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetSource) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/processor Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/processor ValidatorTopology
type ValidatorTopology interface {
	AddNodeRegistration(nr *types.NodeRegistration) error
	UpdateValidatorSet(keys [][]byte)
	Exists(key []byte) bool
	Len() int
	AllPubKeys() [][]byte
	IsValidator() bool
}

// Broker - the event bus
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/processor Broker
type Broker interface {
	Send(e events.Event)
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/processor Notary
type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind) error
	AddSig(ctx context.Context, pubKey []byte, ns types.NodeSignature) ([]types.NodeSignature, bool, error)
	IsSigned(context.Context, string, types.NodeSignatureKind) ([]types.NodeSignature, bool)
}

// Witness ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/processor Witness
type Witness interface {
	AddNodeCheck(ctx context.Context, nv *types.NodeVote) error
}

// EvtForwarder ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evtforwarder_mock.go -package mocks code.vegaprotocol.io/vega/processor EvtForwarder
type EvtForwarder interface {
	Ack(*types.ChainEvent) bool
}

// Banking ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/banking_mock.go -package mocks code.vegaprotocol.io/vega/processor Banking
type Banking interface {
	EnableBuiltinAsset(context.Context, string) error
	DepositBuiltinAsset(context.Context, *types.BuiltinAssetDeposit, string, uint64) error
	WithdrawalBuiltinAsset(context.Context, string, string, string, uint64) error
	EnableERC20(context.Context, *types.ERC20AssetList, uint64, uint64, string) error
	DepositERC20(context.Context, *types.ERC20Deposit, string, uint64, uint64, string) error
	LockWithdrawalERC20(context.Context, string, string, string, uint64, *types.Erc20WithdrawExt) error
	WithdrawalERC20(context.Context, *types.ERC20Withdrawal, uint64, uint64, string) error
	HasBalance(string) bool
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
	Normalise(types.OracleDataSubmission) (*oracles.OracleData, error)
}
