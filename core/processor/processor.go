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
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/pkg/errors"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrInvalidSignature                       = errors.New("invalid signature")
	ErrChainEventFromNonValidator             = errors.New("chain event emitted from a non-validator node")
	ErrUnsupportedChainEvent                  = errors.New("unsupported chain event")
	ErrNodeSignatureFromNonValidator          = errors.New("node signature not sent by validator")
	ErrNodeSignatureWithNonValidatorMasterKey = errors.New("node signature not signed with validator master key")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/processor TimeService
type TimeService interface {
	GetTimeNow() time.Time
	GetTimeLastBatch() time.Time
	NotifyOnTick(...func(context.Context, time.Time))
	SetTimeNow(context.Context, time.Time)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/epoch_service_mock.go -package mocks code.vegaprotocol.io/vega/core/processor EpochService
type EpochService interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
	OnBlockEnd(ctx context.Context)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_engine_mock.go -package mocks code.vegaprotocol.io/vega/core/processor DelegationEngine
type DelegationEngine interface {
	Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateAtEndOfEpoch(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	UndelegateNow(ctx context.Context, party string, nodeID string, amount *num.Uint) error
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/core/processor ExecutionEngine
type ExecutionEngine interface {
	// orders stuff
	SubmitOrder(ctx context.Context, orderSubmission *types.OrderSubmission, party string, deterministicId string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation, party string, deterministicId string) ([]*types.OrderCancellationConfirmation, error)
	AmendOrder(ctx context.Context, order *types.OrderAmendment, party string, deterministicId string) (*types.OrderConfirmation, error)

	// market stuff
	SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string) error
	UpdateMarket(ctx context.Context, marketConfig *types.Market) error
	SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpid, deterministicId string) error
	RejectMarket(ctx context.Context, marketid string) error
	StartOpeningAuction(ctx context.Context, marketid string) error

	// LP stuff
	SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicId string) error
	CancelLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionCancellation, party string) error
	AmendLiquidityProvision(ctx context.Context, order *types.LiquidityProvisionAmendment, party string, deterministicId string) error
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/core/processor GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, types.ProposalSubmission, string, string) (*governance.ToSubmit, error)
	FinaliseEnactment(ctx context.Context, prop *types.Proposal)
	AddVote(context.Context, types.VoteSubmission, string) error
	OnTick(context.Context, time.Time) ([]*governance.ToEnact, []*governance.VoteClosed)
	RejectProposal(context.Context, *types.Proposal, types.ProposalError, error) error
	Hash() []byte
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stats_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Stats
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
	SetHash(string)
	SetHeight(uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Assets
type Assets interface {
	NewAsset(ctx context.Context, ref string, assetSrc *types.AssetDetails) (string, error)
	StageAssetUpdate(*types.Asset) error
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/core/processor ValidatorTopology
type ValidatorTopology interface {
	Len() int
	IsValidatorVegaPubKey(pk string) bool
	IsValidatorNodeID(nodeID string) bool
	AllVegaPubKeys() []string
	IsValidator() bool
	AddKeyRotate(ctx context.Context, nodeID string, currentBlockHeight uint64, kr *commandspb.KeyRotateSubmission) error
	RotateEthereumKey(ctx context.Context, nodeID string, currentBlockHeight uint64, kr *commandspb.EthereumKeyRotateSubmission) error
	BeginBlock(ctx context.Context, req abcitypes.RequestBeginBlock)
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
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Notary
type Notary interface {
	StartAggregate(resID string, kind commandspb.NodeSignatureKind, signature []byte)
	RegisterSignature(ctx context.Context, pubKey string, ns commandspb.NodeSignature) error
	IsSigned(context.Context, string, commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool)
}

// Witness ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Witness
type Witness interface {
	AddNodeCheck(ctx context.Context, nv *commandspb.NodeVote, key crypto.PublicKey) error
}

// EvtForwarder ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evtforwarder_mock.go -package mocks code.vegaprotocol.io/vega/core/processor EvtForwarder
type EvtForwarder interface {
	Ack(*commandspb.ChainEvent) bool
}

// Banking ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/banking_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Banking
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
}

// NetworkParameters ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/network_parameters_mock.go -package mocks code.vegaprotocol.io/vega/core/processor NetworkParameters
type NetworkParameters interface {
	Update(ctx context.Context, key, value string) error
	DispatchChanges(ctx context.Context)
}

type Oracle struct {
	Engine   OraclesEngine
	Adaptors OracleAdaptors
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracles_engine_mock.go -package mocks code.vegaprotocol.io/vega/core/processor OraclesEngine
type OraclesEngine interface {
	BroadcastData(context.Context, oracles.OracleData) error
	ListensToPubKeys(oracles.OracleData) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_adaptors_mock.go -package mocks code.vegaprotocol.io/vega/core/processor OracleAdaptors
type OracleAdaptors interface {
	Normalise(crypto.PublicKey, commandspb.OracleDataSubmission) (*oracles.OracleData, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/limits_mock.go -package mocks code.vegaprotocol.io/vega/core/processor Limits
type Limits interface {
	CanProposeMarket() bool
	CanProposeAsset() bool
	CanTrade() bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stake_verifier_mock.go -package mocks code.vegaprotocol.io/vega/core/processor StakeVerifier
type StakeVerifier interface {
	ProcessStakeRemoved(ctx context.Context, event *types.StakeRemoved) error
	ProcessStakeDeposited(ctx context.Context, event *types.StakeDeposited) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/staking_accounts_mock.go -package mocks code.vegaprotocol.io/vega/core/processor StakingAccounts
type StakingAccounts interface {
	Hash() []byte
	ProcessStakeTotalSupply(ctx context.Context, event *types.StakeTotalSupply) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/erc20_multisig_topology_mock.go -package mocks code.vegaprotocol.io/vega/core/processor ERC20MultiSigTopology
type ERC20MultiSigTopology interface {
	ProcessSignerEvent(event *types.SignerEvent) error
	ProcessThresholdEvent(event *types.SignerThresholdSetEvent) error
}
