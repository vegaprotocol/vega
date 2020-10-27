package processor

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrUnknownCommand                               = errors.New("unknown command when validating payload")
	ErrInvalidSignature                             = errors.New("invalid signature")
	ErrOrderSubmissionPartyAndPubKeyDoesNotMatch    = errors.New("order submission party and pubkey does not match")
	ErrOrderCancellationPartyAndPubKeyDoesNotMatch  = errors.New("order cancellation party and pubkey does not match")
	ErrOrderAmendmentPartyAndPubKeyDoesNotMatch     = errors.New("order amendment party and pubkey does not match")
	ErrProposalSubmissionPartyAndPubKeyDoesNotMatch = errors.New("proposal submission party and pubkey does not match")
	ErrVoteSubmissionPartyAndPubKeyDoesNotMatch     = errors.New("vote submission party and pubkey does not match")
	ErrWithdrawPartyAndPublKeyDoesNotMatch          = errors.New("withdraw party and pubkey does not match")
	ErrNodeSignatureKeyDoesNotMatch                 = errors.New("node signature pubkey does not match")
	ErrCommandKindUnknown                           = errors.New("unknown command kind when validating payload")
	ErrUnknownNodeKey                               = errors.New("node pubkey unknown")
	ErrUnknownProposal                              = errors.New("proposal unknown")
	ErrNotAnAssetProposal                           = errors.New("proposal is not a new asset proposal")
	ErrNoVegaWalletFound                            = errors.New("node wallet not found")
	ErrAssetProposalReferenceDuplicate              = errors.New("duplicate asset proposal for reference")
	ErrRegisterNodePubKeyDoesNotMatch               = errors.New("node register key does not match")
	ErrProposalValidationTimestampInvalid           = errors.New("asset proposal validation timestamp invalid")
	ErrVegaWalletRequired                           = errors.New("vega wallet required")
	ErrProposalCorrupted                            = errors.New("proposal internal data corrupted")
	ErrChainEventFromNonValidator                   = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent                        = errors.New("unsupprted chain event")
	ErrNotAnAssetListChainEvent                     = errors.New("not an asset list chain event")
	ErrNodeSignatureFromNonValidator                = errors.New("node signature not sent by validator")
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
	SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) ([]*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment) (*types.OrderConfirmation, error)
	SubmitMarket(ctx context.Context, marketConfig *types.Market) error
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/processor GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, types.Proposal) error
	AddVote(context.Context, types.Vote) error
	OnChainTimeUpdate(context.Context, time.Time) []*governance.ToEnact
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
	IsSigned(string, types.NodeSignatureKind) ([]types.NodeSignature, bool)
}

// ExtResChecker ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/erc_mock.go -package mocks code.vegaprotocol.io/vega/processor ExtResChecker
type ExtResChecker interface {
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
	DepositBuiltinAsset(context.Context, *types.BuiltinAssetDeposit, uint64) error
	WithdrawalBuiltinAsset(context.Context, string, string, uint64) error
	EnableERC20(context.Context, *types.ERC20AssetList, uint64, uint64) error
	DepositERC20(context.Context, *types.ERC20Deposit, uint64, uint64) error
	LockWithdrawalERC20(context.Context, string, string, uint64, *types.Erc20WithdrawExt) error
	WithdrawalERC20(*types.ERC20Withdrawal, uint64, uint64) error
}
