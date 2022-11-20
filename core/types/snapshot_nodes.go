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

package types

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type Snapshot struct {
	Format SnapshotFormat
	Height uint64    // the block-height of the snapshot
	Hash   []byte    // the hash of the snapshot (the root hash of the AVL tree)
	Meta   *Metadata // the AVL tree metadata
	// Metadata []byte     // the above metadata serialised
	Nodes []*Payload // the snapshot payloads in the tree (always leaf nodes)

	// Chunk stuff
	Chunks     uint32
	DataChunks []*Chunk
	ByteChunks [][]byte
	ChunksSeen uint32
	byteLen    int
}

type Metadata struct {
	Version     int64       // the version of the AVL tree
	NodeHashes  []*NodeHash // he nodes of the AVL tree ordered such that it can be imported with identical shape
	ChunkHashes []string
}

// NodeHash contains the data for a node in the IAVL, without its value, but the values hash instead.
type NodeHash struct {
	Key     string // the node's key eg. epoch.all
	Hash    string // the hash of the noes value
	IsLeaf  bool   // whether or not the node contains a payload
	Height  int32  // the height of the node in the tree
	Version int64  // the version of that node i.e how many times its value has changed
}

type Chunk struct {
	Data   []*Payload
	Nr, Of int64
}

type Payload struct {
	Data    isPayload
	raw     []byte // access to the raw data for chunking
	treeKey string
}

type isPayload interface {
	isPayload()
	plToProto() interface{}
	Namespace() SnapshotNamespace
	Key() string
}

type PayloadProofOfWork struct {
	BlockHeight   []uint64
	BlockHash     []string
	HeightToTx    map[uint64][]string
	HeightToTid   map[uint64][]string
	BannedParties map[string]uint64
	ActiveParams  []*snapshot.ProofOfWorkParams
}

type PayloadActiveAssets struct {
	ActiveAssets *ActiveAssets
}

type PayloadPendingAssets struct {
	PendingAssets *PendingAssets
}

type PayloadPendingAssetUpdates struct {
	PendingAssetUpdates *PendingAssetUpdates
}

type PayloadBankingBridgeState struct {
	BankingBridgeState *BankingBridgeState
}

type PayloadBankingWithdrawals struct {
	BankingWithdrawals *BankingWithdrawals
}

type PayloadBankingDeposits struct {
	BankingDeposits *BankingDeposits
}

type PayloadBankingSeen struct {
	BankingSeen *BankingSeen
}

type PayloadBankingAssetActions struct {
	BankingAssetActions *BankingAssetActions
}

type PayloadBankingRecurringTransfers struct {
	BankingRecurringTransfers *checkpointpb.RecurringTransfers
}

type PayloadBankingScheduledTransfers struct {
	BankingScheduledTransfers []*checkpointpb.ScheduledTransferAtTime
}

type PayloadCheckpoint struct {
	Checkpoint *CPState
}

type PayloadCollateralAccounts struct {
	CollateralAccounts *CollateralAccounts
}

type PayloadCollateralAssets struct {
	CollateralAssets *CollateralAssets
}

type PayloadAppState struct {
	AppState *AppState
}

type PayloadNetParams struct {
	NetParams *NetParams
}

type PayloadDelegationActive struct {
	DelegationActive *DelegationActive
}

type PayloadDelegationPending struct {
	DelegationPending *DelegationPending
}

type PayloadDelegationAuto struct {
	DelegationAuto *DelegationAuto
}

type PayloadDelegationLastReconTime struct {
	LastReconcilicationTime time.Time
}

type PayloadGovernanceActive struct {
	GovernanceActive *GovernanceActive
}

type PayloadGovernanceEnacted struct {
	GovernanceEnacted *GovernanceEnacted
}

type PayloadGovernanceNode struct {
	GovernanceNode *GovernanceNode
}

type PayloadMarketPositions struct {
	MarketPositions *MarketPositions
}

type PayloadMatchingBook struct {
	MatchingBook *MatchingBook
}

type PayloadExecutionMarkets struct {
	ExecutionMarkets *ExecutionMarkets
}

type PayloadStakingAccounts struct {
	StakingAccounts *StakingAccounts
}

type PayloadStakeVerifierDeposited struct {
	StakeVerifierDeposited []*StakeDeposited
}

type PayloadStakeVerifierRemoved struct {
	StakeVerifierRemoved []*StakeRemoved
}

type PayloadEpoch struct {
	EpochState *EpochState
}

type PayloadLimitState struct {
	LimitState *LimitState
}

type PayloadNotary struct {
	Notary *Notary
}

type PayloadWitness struct {
	Witness *Witness
}

type PayloadTopology struct {
	Topology *Topology
}

type Topology struct {
	ValidatorData                  []*snapshot.ValidatorState
	ChainValidators                []string
	PendingPubKeyRotations         []*snapshot.PendingKeyRotation
	PendingEthereumKeyRotations    []*snapshot.PendingEthereumKeyRotation
	UnresolvedEthereumKeyRotations []*snapshot.PendingEthereumKeyRotation
	Signatures                     *snapshot.ToplogySignatures
	ValidatorPerformance           *snapshot.ValidatorPerformance
}

type PayloadFloatingPointConsensus struct {
	ConsensusData               []*snapshot.NextTimeTrigger
	StateVariablesInternalState []*snapshot.StateVarInternalState
}

type PayloadMarketActivityTracker struct {
	MarketActivityData *snapshot.MarketTracker
}

type Witness struct {
	Resources []*Resource
}

type Resource struct {
	ID         string
	CheckUntil time.Time
	Votes      []string
}

type PayloadEventForwarder struct {
	Keys []string
}

type PayloadERC20MultiSigTopologyVerified struct {
	Verified *snapshot.ERC20MultiSigTopologyVerified
}

type PayloadERC20MultiSigTopologyPending struct {
	Pending *snapshot.ERC20MultiSigTopologyPending
}

type PayloadLiquidityParameters struct {
	Parameters *snapshot.LiquidityParameters
}

type PayloadLiquidityPendingProvisions struct {
	PendingProvisions *snapshot.LiquidityPendingProvisions
}

type PayloadLiquidityProvisions struct {
	Provisions *snapshot.LiquidityProvisions
}

type PayloadLiquidityTarget struct {
	Target *snapshot.LiquidityTarget
}

type PayloadProtocolUpgradeProposals struct {
	Proposals *snapshot.ProtocolUpgradeProposals
}

type MatchingBook struct {
	MarketID        string
	Buy             []*Order
	Sell            []*Order
	LastTradedPrice *num.Uint
	Auction         bool
	BatchID         uint64
}

type ExecutionMarkets struct {
	Markets []*ExecMarket
}

type ExecMarket struct {
	Market                     *Market
	PriceMonitor               *PriceMonitor
	AuctionState               *AuctionState
	PeggedOrders               *PeggedOrdersState
	ExpiringOrders             []*Order
	LastBestBid                *num.Uint
	LastBestAsk                *num.Uint
	LastMidBid                 *num.Uint
	LastMidAsk                 *num.Uint
	LastMarketValueProxy       num.Decimal
	LastEquityShareDistributed int64
	EquityShare                *EquityShare
	CurrentMarkPrice           *num.Uint
	ShortRiskFactor            num.Decimal
	LongRiskFactor             num.Decimal
	RiskFactorConsensusReached bool
	FeeSplitter                *FeeSplitter
	SettlementData             *num.Uint
	NextMTM                    int64
}

type PriceMonitor struct {
	Initialised                 bool
	FPHorizons                  []*KeyDecimalPair
	Now                         time.Time
	Update                      time.Time
	Bounds                      []*PriceBound
	PriceRangeCache             []*PriceRangeCache
	PriceRangeCacheTime         time.Time
	PricesNow                   []*CurrentPrice
	PricesPast                  []*PastPrice
	RefPriceCache               []*KeyDecimalPair
	RefPriceCacheTime           time.Time
	PriceBoundsConsensusReached bool
}

type CurrentPrice struct {
	Price  *num.Uint
	Volume uint64
}

type PastPrice struct {
	Time                time.Time
	VolumeWeightedPrice num.Decimal
}

type PriceBound struct {
	Active     bool
	UpFactor   num.Decimal
	DownFactor num.Decimal
	Trigger    *PriceMonitoringTrigger
}

type PriceRangeCache struct {
	Bound *PriceBound
	Range *PriceRange
}

type PriceRange struct {
	Min num.Decimal
	Max num.Decimal
	Ref num.Decimal
}

type KeyDecimalPair struct {
	Key int64
	Val num.Decimal
}

type PeggedOrdersState struct {
	Parked []*Order
}

type AuctionState struct {
	Mode               MarketTradingMode
	DefaultMode        MarketTradingMode
	Trigger            AuctionTrigger
	Begin              time.Time
	End                *AuctionDuration
	Start              bool
	Stop               bool
	Extension          AuctionTrigger
	ExtensionEventSent bool
}

type FeeSplitter struct {
	TimeWindowStart time.Time
	TradeValue      *num.Uint
	Avg             num.Decimal
	Window          uint64
}

type EpochState struct {
	Seq                  uint64
	StartTime            time.Time
	ExpireTime           time.Time
	ReadyToStartNewEpoch bool
	ReadyToEndEpoch      bool
}

type LimitState struct {
	BlockCount               uint32
	CanProposeMarket         bool
	CanProposeAsset          bool
	GenesisLoaded            bool
	ProposeMarketEnabled     bool
	ProposeAssetEnabled      bool
	ProposeMarketEnabledFrom time.Time
	ProposeAssetEnabledFrom  time.Time
}

type EquityShare struct {
	Mvp                 num.Decimal
	PMvp                num.Decimal
	R                   num.Decimal
	OpeningAuctionEnded bool
	Lps                 []*EquityShareLP
}

type EquityShareLP struct {
	ID     string
	Stake  num.Decimal
	Share  num.Decimal
	Avg    num.Decimal
	VStake num.Decimal
}

type ActiveAssets struct {
	Assets []*Asset
}

type PendingAssets struct {
	Assets []*Asset
}

type PendingAssetUpdates struct {
	Assets []*Asset
}

type BankingBridgeState struct {
	Active      bool
	BlockHeight uint64
	LogIndex    uint64
}

type BankingWithdrawals struct {
	Withdrawals []*RWithdrawal
}

type RWithdrawal struct {
	Ref        string
	Withdrawal *Withdrawal
}

type BankingDeposits struct {
	Deposit []*BDeposit
}

type BDeposit struct {
	ID      string
	Deposit *Deposit
}

type BankingSeen struct {
	Refs []string
}

type BankingAssetActions struct {
	AssetAction []*AssetAction
}

type AssetAction struct {
	ID                      string
	State                   uint32
	Asset                   string
	BlockNumber             uint64
	TxIndex                 uint64
	Hash                    string
	BuiltinD                *BuiltinAssetDeposit
	Erc20D                  *ERC20Deposit
	Erc20AL                 *ERC20AssetList
	ERC20AssetLimitsUpdated *ERC20AssetLimitsUpdated
	BridgeStopped           bool
	BridgeResume            bool
}

type CPState struct {
	NextCp int64
}

type CollateralAccounts struct {
	Accounts []*Account
}

type CollateralAssets struct {
	Assets []*Asset
}

type AppState struct {
	Height  uint64
	Block   string
	Time    int64
	ChainID string
}

type NetParams struct {
	Params []*NetworkParameter
}

type DelegationActive struct {
	Delegations []*Delegation
}

type DelegationPending struct {
	Delegations  []*Delegation
	Undelegation []*Delegation
}

type DelegationAuto struct {
	Parties []string
}

type GovernanceActive struct {
	Proposals []*ProposalData
}

type GovernanceEnacted struct {
	Proposals []*ProposalData
}

type GovernanceNode struct {
	Proposals []*Proposal
}

type ProposalData struct {
	Proposal *Proposal
	Yes      []*Vote
	No       []*Vote
	Invalid  []*Vote
}

type MarketPositions struct {
	MarketID  string
	Positions []*MarketPosition
}

type MarketPosition struct {
	PartyID         string
	Size, Buy, Sell int64
	Price           *num.Uint
	VwBuy, VwSell   *num.Uint
}

type StakingAccounts struct {
	Accounts                []*StakingAccount
	StakingAssetTotalSupply *num.Uint
}

type StakingAccount struct {
	Party   string
	Balance *num.Uint
	Events  []*StakeLinking
}

type NotarySigs struct {
	ID   string
	Kind int32
	Node string
	Sig  string
}

type Notary struct {
	Sigs []*NotarySigs
}

type PayloadLiquiditySupplied struct {
	LiquiditySupplied *snapshot.LiquiditySupplied
}

func (s Snapshot) GetRawChunk(idx uint32) (*RawChunk, error) {
	if s.Chunks < idx {
		return nil, ErrUnknownSnapshotChunkHeight
	}
	i := int(idx)
	return &RawChunk{
		Nr:   idx,
		Data: s.ByteChunks[i],
	}, nil
}

func MetadataFromProto(m *snapshot.Metadata) (*Metadata, error) {
	nh := make([]*NodeHash, 0, len(m.NodeHashes))
	for _, h := range m.NodeHashes {
		hh, err := NodeHashFromProto(h)
		if err != nil {
			return nil, err
		}
		nh = append(nh, hh)
	}
	return &Metadata{
		Version:     m.Version,
		ChunkHashes: m.ChunkHashes[:],
		NodeHashes:  nh,
	}, nil
}

func (m Metadata) IntoProto() *snapshot.Metadata {
	nh := make([]*snapshot.NodeHash, 0, len(m.NodeHashes))
	for _, h := range m.NodeHashes {
		nh = append(nh, h.IntoProto())
	}
	return &snapshot.Metadata{
		Version:     m.Version,
		ChunkHashes: m.ChunkHashes[:],
		NodeHashes:  nh,
	}
}

func NodeHashFromProto(nh *snapshot.NodeHash) (*NodeHash, error) {
	return &NodeHash{
		Key:     nh.Key,
		Hash:    nh.Hash,
		Version: nh.Version,
		Height:  nh.Height,
		IsLeaf:  nh.IsLeaf,
	}, nil
}

func (n NodeHash) IntoProto() *snapshot.NodeHash {
	return &snapshot.NodeHash{
		Key:     n.Key,
		Hash:    n.Hash,
		Height:  n.Height,
		Version: n.Version,
		IsLeaf:  n.IsLeaf,
	}
}

func ChunkFromProto(c *snapshot.Chunk) *Chunk {
	data := make([]*Payload, 0, len(c.Data))
	for _, p := range c.Data {
		data = append(data, PayloadFromProto(p))
	}
	return &Chunk{
		Data: data,
		Nr:   c.Nr,
		Of:   c.Of,
	}
}

func (c Chunk) IntoProto() *snapshot.Chunk {
	data := make([]*snapshot.Payload, 0, len(c.Data))
	for _, p := range c.Data {
		data = append(data, p.IntoProto())
	}
	return &snapshot.Chunk{
		Data: data,
		Nr:   c.Nr,
		Of:   c.Of,
	}
}

func PayloadFromProto(p *snapshot.Payload) *Payload {
	ret := &Payload{}
	switch dt := p.Data.(type) {
	case *snapshot.Payload_AppState:
		ret.Data = PayloadAppStateFromProto(dt)
	case *snapshot.Payload_ActiveAssets:
		ret.Data = PayloadActiveAssetsFromProto(dt)
	case *snapshot.Payload_PendingAssets:
		ret.Data = PayloadPendingAssetsFromProto(dt)
	case *snapshot.Payload_PendingAssetUpdates:
		ret.Data = PayloadPendingAssetUpdatesFromProto(dt)
	case *snapshot.Payload_BankingBridgeState:
		ret.Data = PayloadBankingBridgeStateFromProto(dt)
	case *snapshot.Payload_BankingWithdrawals:
		ret.Data = PayloadBankingWithdrawalsFromProto(dt)
	case *snapshot.Payload_BankingDeposits:
		ret.Data = PayloadBankingDepositsFromProto(dt)
	case *snapshot.Payload_BankingSeen:
		ret.Data = PayloadBankingSeenFromProto(dt)
	case *snapshot.Payload_BankingAssetActions:
		ret.Data = PayloadBankingAssetActionsFromProto(dt)
	case *snapshot.Payload_Checkpoint:
		ret.Data = PayloadCheckpointFromProto(dt)
	case *snapshot.Payload_CollateralAssets:
		ret.Data = PayloadCollateralAssetsFromProto(dt)
	case *snapshot.Payload_CollateralAccounts:
		ret.Data = PayloadCollateralAccountsFromProto(dt)
	case *snapshot.Payload_NetworkParameters:
		ret.Data = PayloadNetParamsFromProto(dt)
	case *snapshot.Payload_DelegationActive:
		ret.Data = PayloadDelegationActiveFromProto(dt)
	case *snapshot.Payload_DelegationPending:
		ret.Data = PayloadDelegationPendingFromProto(dt)
	case *snapshot.Payload_GovernanceActive:
		ret.Data = PayloadGovernanceActiveFromProto(dt)
	case *snapshot.Payload_GovernanceEnacted:
		ret.Data = PayloadGovernanceEnactedFromProto(dt)
	case *snapshot.Payload_GovernanceNode:
		ret.Data = PayloadGovernanceNodeFromProto(dt)
	case *snapshot.Payload_MarketPositions:
		ret.Data = PayloadMarketPositionsFromProto(dt)
	case *snapshot.Payload_MatchingBook:
		ret.Data = PayloadMatchingBookFromProto(dt)
	case *snapshot.Payload_ExecutionMarkets:
		ret.Data = PayloadExecutionMarketsFromProto(dt)
	case *snapshot.Payload_Epoch:
		ret.Data = PayloadEpochFromProto(dt)
	case *snapshot.Payload_StakingAccounts:
		ret.Data = PayloadStakingAccountsFromProto(dt)
	case *snapshot.Payload_DelegationAuto:
		ret.Data = PayloadDelegationAutoFromProto(dt)
	case *snapshot.Payload_LimitState:
		ret.Data = PayloadLimitStateFromProto(dt)
	case *snapshot.Payload_RewardsPendingPayouts:
		ret.Data = PayloadRewardPayoutFromProto(dt)
	case *snapshot.Payload_VoteSpamPolicy:
		ret.Data = PayloadVoteSpamPolicyFromProto(dt)
	case *snapshot.Payload_SimpleSpamPolicy:
		ret.Data = PayloadSimpleSpamPolicyFromProto(dt)
	case *snapshot.Payload_Notary:
		ret.Data = PayloadNotaryFromProto(dt)
	case *snapshot.Payload_EventForwarder:
		ret.Data = PayloadEventForwarderFromProto(dt)
	case *snapshot.Payload_Witness:
		ret.Data = PayloadWitnessFromProto(dt)
	case *snapshot.Payload_DelegationLastReconciliationTime:
		ret.Data = PayloadDelegationLastReconTimeFromProto(dt)
	case *snapshot.Payload_StakeVerifierDeposited:
		ret.Data = PayloadStakeVerifierDepositedFromProto(dt)
	case *snapshot.Payload_StakeVerifierRemoved:
		ret.Data = PayloadStakeVerifierRemovedFromProto(dt)
	case *snapshot.Payload_Topology:
		ret.Data = PayloadTopologyFromProto(dt)
	case *snapshot.Payload_LiquidityParameters:
		ret.Data = PayloadLiquidityParametersFromProto(dt)
	case *snapshot.Payload_LiquidityPendingProvisions:
		ret.Data = PayloadLiquidityPendingProvisionsFromProto(dt)
	case *snapshot.Payload_LiquidityProvisions:
		ret.Data = PayloadLiquidityProvisionsFromProto(dt)
	case *snapshot.Payload_LiquiditySupplied:
		ret.Data = PayloadLiquiditySuppliedFromProto(dt)
	case *snapshot.Payload_LiquidityTarget:
		ret.Data = PayloadLiquidityTargetFromProto(dt)
	case *snapshot.Payload_FloatingPointConsensus:
		ret.Data = PayloadFloatingPointConsensusFromProto(dt)
	case *snapshot.Payload_MarketTracker:
		ret.Data = PayloadMarketActivityTrackerFromProto(dt)
	case *snapshot.Payload_BankingRecurringTransfers:
		ret.Data = PayloadBankingRecurringTransfersFromProto(dt)
	case *snapshot.Payload_BankingScheduledTransfers:
		ret.Data = PayloadBankingScheduledTransfersFromProto(dt)
	case *snapshot.Payload_Erc20MultisigTopologyPending:
		ret.Data = PayloadERC20MultiSigTopologyPendingFromProto(dt)
	case *snapshot.Payload_Erc20MultisigTopologyVerified:
		ret.Data = PayloadERC20MultiSigTopologyVerifiedFromProto(dt)
	case *snapshot.Payload_ProofOfWork:
		ret.Data = PayloadProofOfWorkFromProto(dt)
	case *snapshot.Payload_ProtocolUpgradeProposals:
		ret.Data = PayloadProtocolUpgradeProposalFromProto(dt)
	}

	return ret
}

func (p Payload) Namespace() SnapshotNamespace {
	if p.Data == nil {
		return undefinedSnapshot
	}
	return p.Data.Namespace()
}

func (p Payload) Key() string {
	if p.Data == nil {
		return ""
	}
	return p.Data.Key()
}

func (p *Payload) GetTreeKey() string {
	if len(p.treeKey) == 0 {
		p.treeKey = KeyFromPayload(p.Data)
	}
	return p.treeKey
}

func (p Payload) IntoProto() *snapshot.Payload {
	ret := snapshot.Payload{}
	if p.Data == nil {
		return &ret
	}
	d := p.Data.plToProto()
	switch dt := d.(type) {
	case *snapshot.Payload_AppState:
		ret.Data = dt
	case *snapshot.Payload_ActiveAssets:
		ret.Data = dt
	case *snapshot.Payload_PendingAssets:
		ret.Data = dt
	case *snapshot.Payload_PendingAssetUpdates:
		ret.Data = dt
	case *snapshot.Payload_BankingSeen:
		ret.Data = dt
	case *snapshot.Payload_BankingDeposits:
		ret.Data = dt
	case *snapshot.Payload_BankingBridgeState:
		ret.Data = dt
	case *snapshot.Payload_BankingWithdrawals:
		ret.Data = dt
	case *snapshot.Payload_BankingAssetActions:
		ret.Data = dt
	case *snapshot.Payload_CollateralAssets:
		ret.Data = dt
	case *snapshot.Payload_CollateralAccounts:
		ret.Data = dt
	case *snapshot.Payload_StakingAccounts:
		ret.Data = dt
	case *snapshot.Payload_ExecutionMarkets:
		ret.Data = dt
	case *snapshot.Payload_MatchingBook:
		ret.Data = dt
	case *snapshot.Payload_MarketPositions:
		ret.Data = dt
	case *snapshot.Payload_DelegationActive:
		ret.Data = dt
	case *snapshot.Payload_DelegationPending:
		ret.Data = dt
	case *snapshot.Payload_GovernanceActive:
		ret.Data = dt
	case *snapshot.Payload_GovernanceEnacted:
		ret.Data = dt
	case *snapshot.Payload_GovernanceNode:
		ret.Data = dt
	case *snapshot.Payload_Checkpoint:
		ret.Data = dt
	case *snapshot.Payload_Epoch:
		ret.Data = dt
	case *snapshot.Payload_DelegationAuto:
		ret.Data = dt
	case *snapshot.Payload_LimitState:
		ret.Data = dt
	case *snapshot.Payload_RewardsPendingPayouts:
		ret.Data = dt
	case *snapshot.Payload_VoteSpamPolicy:
		ret.Data = dt
	case *snapshot.Payload_SimpleSpamPolicy:
		ret.Data = dt
	case *snapshot.Payload_Notary:
		ret.Data = dt
	case *snapshot.Payload_EventForwarder:
		ret.Data = dt
	case *snapshot.Payload_Witness:
		ret.Data = dt
	case *snapshot.Payload_DelegationLastReconciliationTime:
		ret.Data = dt
	case *snapshot.Payload_StakeVerifierDeposited:
		ret.Data = dt
	case *snapshot.Payload_StakeVerifierRemoved:
		ret.Data = dt
	case *snapshot.Payload_Topology:
		ret.Data = dt
	case *snapshot.Payload_LiquidityParameters:
		ret.Data = dt
	case *snapshot.Payload_LiquidityPendingProvisions:
		ret.Data = dt
	case *snapshot.Payload_LiquidityPartiesLiquidityOrders:
		ret.Data = dt
	case *snapshot.Payload_LiquidityPartiesOrders:
		ret.Data = dt
	case *snapshot.Payload_LiquidityProvisions:
		ret.Data = dt
	case *snapshot.Payload_LiquiditySupplied:
		ret.Data = dt
	case *snapshot.Payload_NetworkParameters:
		ret.Data = dt
	case *snapshot.Payload_LiquidityTarget:
		ret.Data = dt
	case *snapshot.Payload_FloatingPointConsensus:
		ret.Data = dt
	case *snapshot.Payload_MarketTracker:
		ret.Data = dt
	case *snapshot.Payload_BankingRecurringTransfers:
		ret.Data = dt
	case *snapshot.Payload_BankingScheduledTransfers:
		ret.Data = dt
	case *snapshot.Payload_Erc20MultisigTopologyPending:
		ret.Data = dt
	case *snapshot.Payload_Erc20MultisigTopologyVerified:
		ret.Data = dt
	case *snapshot.Payload_ProofOfWork:
		ret.Data = dt
	case *snapshot.Payload_ProtocolUpgradeProposals:
		ret.Data = dt
	}
	return &ret
}

func (p Payload) GetAppState() *PayloadAppState {
	if p.Namespace() == AppSnapshot {
		pas := p.Data.(*PayloadAppState)
		return pas
	}
	return nil
}

func PayloadERC20MultiSigTopologyVerifiedFromProto(
	s *snapshot.Payload_Erc20MultisigTopologyVerified,
) *PayloadERC20MultiSigTopologyVerified {
	return &PayloadERC20MultiSigTopologyVerified{
		Verified: s.Erc20MultisigTopologyVerified,
	}
}

func (*PayloadERC20MultiSigTopologyVerified) isPayload() {}

func (p *PayloadERC20MultiSigTopologyVerified) plToProto() interface{} {
	return &snapshot.Payload_Erc20MultisigTopologyVerified{
		Erc20MultisigTopologyVerified: p.Verified,
	}
}

func (*PayloadERC20MultiSigTopologyVerified) Namespace() SnapshotNamespace {
	return ERC20MultiSigTopologySnapshot
}

func (p *PayloadERC20MultiSigTopologyVerified) Key() string {
	return "verified"
}

func PayloadERC20MultiSigTopologyPendingFromProto(
	s *snapshot.Payload_Erc20MultisigTopologyPending,
) *PayloadERC20MultiSigTopologyPending {
	return &PayloadERC20MultiSigTopologyPending{
		Pending: s.Erc20MultisigTopologyPending,
	}
}

func (*PayloadERC20MultiSigTopologyPending) isPayload() {}

func (p *PayloadERC20MultiSigTopologyPending) plToProto() interface{} {
	return &snapshot.Payload_Erc20MultisigTopologyPending{
		Erc20MultisigTopologyPending: p.Pending,
	}
}

func (*PayloadERC20MultiSigTopologyPending) Namespace() SnapshotNamespace {
	return ERC20MultiSigTopologySnapshot
}

func (p *PayloadERC20MultiSigTopologyPending) Key() string {
	return "pending"
}

func PayloadLiquidityParametersFromProto(s *snapshot.Payload_LiquidityParameters) *PayloadLiquidityParameters {
	return &PayloadLiquidityParameters{
		Parameters: s.LiquidityParameters,
	}
}

func (*PayloadLiquidityParameters) isPayload() {}

func (p *PayloadLiquidityParameters) plToProto() interface{} {
	return &snapshot.Payload_LiquidityParameters{
		LiquidityParameters: p.Parameters,
	}
}

func (*PayloadLiquidityParameters) Namespace() SnapshotNamespace {
	return LiquiditySnapshot
}

func (p *PayloadLiquidityParameters) Key() string {
	return fmt.Sprintf("parameters:%v", p.Parameters.MarketId)
}

func PayloadLiquidityPendingProvisionsFromProto(s *snapshot.Payload_LiquidityPendingProvisions) *PayloadLiquidityPendingProvisions {
	return &PayloadLiquidityPendingProvisions{
		PendingProvisions: s.LiquidityPendingProvisions,
	}
}

func (*PayloadLiquidityPendingProvisions) isPayload() {}

func (p *PayloadLiquidityPendingProvisions) plToProto() interface{} {
	return &snapshot.Payload_LiquidityPendingProvisions{
		LiquidityPendingProvisions: p.PendingProvisions,
	}
}

func (*PayloadLiquidityPendingProvisions) Namespace() SnapshotNamespace {
	return LiquiditySnapshot
}

func (p *PayloadLiquidityPendingProvisions) Key() string {
	return fmt.Sprintf("pendingProvisions:%v", p.PendingProvisions.MarketId)
}

func PayloadLiquidityProvisionsFromProto(s *snapshot.Payload_LiquidityProvisions) *PayloadLiquidityProvisions {
	return &PayloadLiquidityProvisions{
		Provisions: s.LiquidityProvisions,
	}
}

func (*PayloadLiquidityProvisions) isPayload() {}

func (p *PayloadLiquidityProvisions) plToProto() interface{} {
	return &snapshot.Payload_LiquidityProvisions{
		LiquidityProvisions: p.Provisions,
	}
}

func (*PayloadLiquidityProvisions) Namespace() SnapshotNamespace {
	return LiquiditySnapshot
}

func (p *PayloadLiquidityProvisions) Key() string {
	return fmt.Sprintf("provisions:%v", p.Provisions.MarketId)
}

func PayloadLiquidityTargetFromProto(s *snapshot.Payload_LiquidityTarget) *PayloadLiquidityTarget {
	return &PayloadLiquidityTarget{
		Target: s.LiquidityTarget,
	}
}

func (*PayloadLiquidityTarget) isPayload() {}

func (p *PayloadLiquidityTarget) plToProto() interface{} {
	return &snapshot.Payload_LiquidityTarget{
		LiquidityTarget: p.Target,
	}
}

func (*PayloadLiquidityTarget) Namespace() SnapshotNamespace {
	return LiquidityTargetSnapshot
}

func (p *PayloadLiquidityTarget) Key() string {
	return fmt.Sprintf("target:%v", p.Target.MarketId)
}

func PayloadActiveAssetsFromProto(paa *snapshot.Payload_ActiveAssets) *PayloadActiveAssets {
	return &PayloadActiveAssets{
		ActiveAssets: ActiveAssetsFromProto(paa.ActiveAssets),
	}
}

func (p PayloadActiveAssets) IntoProto() *snapshot.Payload_ActiveAssets {
	return &snapshot.Payload_ActiveAssets{
		ActiveAssets: p.ActiveAssets.IntoProto(),
	}
}

func (*PayloadActiveAssets) isPayload() {}

func (p *PayloadActiveAssets) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadActiveAssets) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func (*PayloadActiveAssets) Key() string {
	return "active"
}

func PayloadPendingAssetsFromProto(ppa *snapshot.Payload_PendingAssets) *PayloadPendingAssets {
	return &PayloadPendingAssets{
		PendingAssets: PendingAssetsFromProto(ppa.PendingAssets),
	}
}

func (p PayloadPendingAssets) IntoProto() *snapshot.Payload_PendingAssets {
	return &snapshot.Payload_PendingAssets{
		PendingAssets: p.PendingAssets.IntoProto(),
	}
}

func (*PayloadPendingAssets) isPayload() {}

func (p *PayloadPendingAssets) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadPendingAssets) Key() string {
	return "pending"
}

func (*PayloadPendingAssets) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func PayloadPendingAssetUpdatesFromProto(ppa *snapshot.Payload_PendingAssetUpdates) *PayloadPendingAssetUpdates {
	return &PayloadPendingAssetUpdates{
		PendingAssetUpdates: PendingAssetUpdatesFromProto(ppa.PendingAssetUpdates),
	}
}

func (p PayloadPendingAssetUpdates) IntoProto() *snapshot.Payload_PendingAssetUpdates {
	return &snapshot.Payload_PendingAssetUpdates{
		PendingAssetUpdates: p.PendingAssetUpdates.IntoProto(),
	}
}

func (*PayloadPendingAssetUpdates) isPayload() {}

func (p *PayloadPendingAssetUpdates) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadPendingAssetUpdates) Key() string {
	return "pending_updates"
}

func (*PayloadPendingAssetUpdates) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func (a PendingAssetUpdates) IntoProto() *snapshot.PendingAssetUpdates {
	ret := &snapshot.PendingAssetUpdates{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

func PendingAssetUpdatesFromProto(aa *snapshot.PendingAssetUpdates) *PendingAssetUpdates {
	ret := PendingAssetUpdates{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		pa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, pa)
	}
	return &ret
}

func PayloadBankingBridgeStateFromProto(pbbs *snapshot.Payload_BankingBridgeState) *PayloadBankingBridgeState {
	return &PayloadBankingBridgeState{
		BankingBridgeState: &BankingBridgeState{
			Active:      pbbs.BankingBridgeState.BridgeState.Active,
			BlockHeight: pbbs.BankingBridgeState.BridgeState.BlockHeight,
			LogIndex:    pbbs.BankingBridgeState.BridgeState.LogIndex,
		},
	}
}

func (p PayloadBankingBridgeState) IntoProto() *snapshot.Payload_BankingBridgeState {
	return &snapshot.Payload_BankingBridgeState{
		BankingBridgeState: &snapshot.BankingBridgeState{
			BridgeState: &checkpointpb.BridgeState{
				Active:      p.BankingBridgeState.Active,
				BlockHeight: p.BankingBridgeState.BlockHeight,
				LogIndex:    p.BankingBridgeState.LogIndex,
			},
		},
	}
}

func (*PayloadBankingBridgeState) isPayload() {}

func (p *PayloadBankingBridgeState) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingBridgeState) Key() string {
	return "bridgeState"
}

func (*PayloadBankingBridgeState) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingWithdrawalsFromProto(pbw *snapshot.Payload_BankingWithdrawals) *PayloadBankingWithdrawals {
	return &PayloadBankingWithdrawals{
		BankingWithdrawals: BankingWithdrawalsFromProto(pbw.BankingWithdrawals),
	}
}

func (p PayloadBankingWithdrawals) IntoProto() *snapshot.Payload_BankingWithdrawals {
	return &snapshot.Payload_BankingWithdrawals{
		BankingWithdrawals: p.BankingWithdrawals.IntoProto(),
	}
}

func (*PayloadBankingWithdrawals) isPayload() {}

func (p *PayloadBankingWithdrawals) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingWithdrawals) Key() string {
	return "withdrawals"
}

func (*PayloadBankingWithdrawals) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingDepositsFromProto(pbd *snapshot.Payload_BankingDeposits) *PayloadBankingDeposits {
	return &PayloadBankingDeposits{
		BankingDeposits: BankingDepositsFromProto(pbd.BankingDeposits),
	}
}

func (p PayloadBankingDeposits) IntoProto() *snapshot.Payload_BankingDeposits {
	return &snapshot.Payload_BankingDeposits{
		BankingDeposits: p.BankingDeposits.IntoProto(),
	}
}

func (*PayloadBankingDeposits) isPayload() {}

func (p *PayloadBankingDeposits) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingDeposits) Key() string {
	return "deposits"
}

func (*PayloadBankingDeposits) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingRecurringTransfersFromProto(pbd *snapshot.Payload_BankingRecurringTransfers) *PayloadBankingRecurringTransfers {
	return &PayloadBankingRecurringTransfers{
		BankingRecurringTransfers: pbd.BankingRecurringTransfers.RecurringTransfers,
	}
}

func (p PayloadBankingRecurringTransfers) IntoProto() *snapshot.Payload_BankingRecurringTransfers {
	return &snapshot.Payload_BankingRecurringTransfers{
		BankingRecurringTransfers: &snapshot.BankingRecurringTransfers{
			RecurringTransfers: p.BankingRecurringTransfers,
		},
	}
}

func (*PayloadBankingRecurringTransfers) isPayload() {}

func (p *PayloadBankingRecurringTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingRecurringTransfers) Key() string {
	return "recurringTransfers"
}

func (*PayloadBankingRecurringTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingScheduledTransfersFromProto(pbd *snapshot.Payload_BankingScheduledTransfers) *PayloadBankingScheduledTransfers {
	return &PayloadBankingScheduledTransfers{
		BankingScheduledTransfers: pbd.BankingScheduledTransfers.TransfersAtTime,
	}
}

func (p PayloadBankingScheduledTransfers) IntoProto() *snapshot.Payload_BankingScheduledTransfers {
	return &snapshot.Payload_BankingScheduledTransfers{
		BankingScheduledTransfers: &snapshot.BankingScheduledTransfers{
			TransfersAtTime: p.BankingScheduledTransfers,
		},
	}
}

func (*PayloadBankingScheduledTransfers) isPayload() {}

func (p *PayloadBankingScheduledTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingScheduledTransfers) Key() string {
	return "scheduledTransfers"
}

func (*PayloadBankingScheduledTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingSeenFromProto(pbs *snapshot.Payload_BankingSeen) *PayloadBankingSeen {
	return &PayloadBankingSeen{
		BankingSeen: BankingSeenFromProto(pbs.BankingSeen),
	}
}

func (p PayloadBankingSeen) IntoProto() *snapshot.Payload_BankingSeen {
	return &snapshot.Payload_BankingSeen{
		BankingSeen: p.BankingSeen.IntoProto(),
	}
}

func (*PayloadBankingSeen) isPayload() {}

func (p *PayloadBankingSeen) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingSeen) Key() string {
	return "seen"
}

func (*PayloadBankingSeen) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingAssetActionsFromProto(pbs *snapshot.Payload_BankingAssetActions) *PayloadBankingAssetActions {
	return &PayloadBankingAssetActions{
		BankingAssetActions: BankingAssetActionsFromProto(pbs.BankingAssetActions),
	}
}

func (p PayloadBankingAssetActions) IntoProto() *snapshot.Payload_BankingAssetActions {
	return &snapshot.Payload_BankingAssetActions{
		BankingAssetActions: p.BankingAssetActions.IntoProto(),
	}
}

func (*PayloadBankingAssetActions) isPayload() {}

func (p *PayloadBankingAssetActions) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingAssetActions) Key() string {
	return "assetActions"
}

func (*PayloadBankingAssetActions) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadCheckpointFromProto(pc *snapshot.Payload_Checkpoint) *PayloadCheckpoint {
	return &PayloadCheckpoint{
		Checkpoint: CheckpointFromProto(pc.Checkpoint),
	}
}

func (p PayloadCheckpoint) IntoProto() *snapshot.Payload_Checkpoint {
	return &snapshot.Payload_Checkpoint{
		Checkpoint: p.Checkpoint.IntoProto(),
	}
}

func (*PayloadCheckpoint) isPayload() {}

func (p *PayloadCheckpoint) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadCheckpoint) Key() string {
	return "all"
}

func (*PayloadCheckpoint) Namespace() SnapshotNamespace {
	return CheckpointSnapshot
}

func PayloadCollateralAccountsFromProto(pca *snapshot.Payload_CollateralAccounts) *PayloadCollateralAccounts {
	return &PayloadCollateralAccounts{
		CollateralAccounts: CollateralAccountsFromProto(pca.CollateralAccounts),
	}
}

func (p PayloadCollateralAccounts) IntoProto() *snapshot.Payload_CollateralAccounts {
	return &snapshot.Payload_CollateralAccounts{
		CollateralAccounts: p.CollateralAccounts.IntoProto(),
	}
}

func (*PayloadCollateralAccounts) isPayload() {}

func (p *PayloadCollateralAccounts) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadCollateralAccounts) Key() string {
	return "accounts"
}

func (*PayloadCollateralAccounts) Namespace() SnapshotNamespace {
	return CollateralSnapshot
}

func PayloadCollateralAssetsFromProto(pca *snapshot.Payload_CollateralAssets) *PayloadCollateralAssets {
	return &PayloadCollateralAssets{
		CollateralAssets: CollateralAssetsFromProto(pca.CollateralAssets),
	}
}

func (p PayloadCollateralAssets) IntoProto() *snapshot.Payload_CollateralAssets {
	return &snapshot.Payload_CollateralAssets{
		CollateralAssets: p.CollateralAssets.IntoProto(),
	}
}

func (*PayloadCollateralAssets) isPayload() {}

func (p *PayloadCollateralAssets) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadCollateralAssets) Key() string {
	return "assets"
}

func (*PayloadCollateralAssets) Namespace() SnapshotNamespace {
	return CollateralSnapshot
}

func PayloadAppStateFromProto(pas *snapshot.Payload_AppState) *PayloadAppState {
	return &PayloadAppState{
		AppState: AppStateFromProto(pas.AppState),
	}
}

func (p PayloadAppState) IntoProto() *snapshot.Payload_AppState {
	return &snapshot.Payload_AppState{
		AppState: p.AppState.IntoProto(),
	}
}

func (*PayloadAppState) isPayload() {}

func (p *PayloadAppState) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadAppState) Key() string {
	return "all"
}

func (*PayloadAppState) Namespace() SnapshotNamespace {
	return AppSnapshot
}

func PayloadNetParamsFromProto(pnp *snapshot.Payload_NetworkParameters) *PayloadNetParams {
	return &PayloadNetParams{
		NetParams: NetParamsFromProto(pnp.NetworkParameters),
	}
}

func (p PayloadNetParams) IntoProto() *snapshot.Payload_NetworkParameters {
	return &snapshot.Payload_NetworkParameters{
		NetworkParameters: p.NetParams.IntoProto(),
	}
}

func (*PayloadNetParams) isPayload() {}

func (p *PayloadNetParams) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadNetParams) Key() string {
	return "all"
}

func (*PayloadNetParams) Namespace() SnapshotNamespace {
	return NetParamsSnapshot
}

func PayloadDelegationLastReconTimeFromProto(dl *snapshot.Payload_DelegationLastReconciliationTime) *PayloadDelegationLastReconTime {
	return &PayloadDelegationLastReconTime{
		LastReconcilicationTime: time.Unix(0, dl.DelegationLastReconciliationTime.LastReconciliationTime).UTC(),
	}
}

func (p PayloadDelegationLastReconTime) IntoProto() *snapshot.Payload_DelegationLastReconciliationTime {
	return &snapshot.Payload_DelegationLastReconciliationTime{
		DelegationLastReconciliationTime: &snapshot.DelegationLastReconciliationTime{
			LastReconciliationTime: p.LastReconcilicationTime.UnixNano(),
		},
	}
}

func (*PayloadDelegationLastReconTime) isPayload() {}

func (p *PayloadDelegationLastReconTime) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadDelegationLastReconTime) Key() string {
	return "lastReconTime"
}

func (*PayloadDelegationLastReconTime) Namespace() SnapshotNamespace {
	return DelegationSnapshot
}

func PayloadDelegationAutoFromProto(da *snapshot.Payload_DelegationAuto) *PayloadDelegationAuto {
	return &PayloadDelegationAuto{
		DelegationAuto: DelegationAutoFromProto(da.DelegationAuto),
	}
}

func (p PayloadDelegationAuto) IntoProto() *snapshot.Payload_DelegationAuto {
	return &snapshot.Payload_DelegationAuto{
		DelegationAuto: p.DelegationAuto.IntoProto(),
	}
}

func (*PayloadDelegationAuto) isPayload() {}

func (p *PayloadDelegationAuto) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadDelegationAuto) Key() string {
	return "auto"
}

func (*PayloadDelegationAuto) Namespace() SnapshotNamespace {
	return DelegationSnapshot
}

func PayloadDelegationActiveFromProto(da *snapshot.Payload_DelegationActive) *PayloadDelegationActive {
	return &PayloadDelegationActive{
		DelegationActive: DelegationActiveFromProto(da.DelegationActive),
	}
}

func (p PayloadDelegationActive) IntoProto() *snapshot.Payload_DelegationActive {
	return &snapshot.Payload_DelegationActive{
		DelegationActive: p.DelegationActive.IntoProto(),
	}
}

func (*PayloadDelegationActive) isPayload() {}

func (p *PayloadDelegationActive) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadDelegationActive) Key() string {
	return "active"
}

func (*PayloadDelegationActive) Namespace() SnapshotNamespace {
	return DelegationSnapshot
}

func PayloadDelegationPendingFromProto(da *snapshot.Payload_DelegationPending) *PayloadDelegationPending {
	return &PayloadDelegationPending{
		DelegationPending: DelegationPendingFromProto(da.DelegationPending),
	}
}

func (p PayloadDelegationPending) IntoProto() *snapshot.Payload_DelegationPending {
	return &snapshot.Payload_DelegationPending{
		DelegationPending: p.DelegationPending.IntoProto(),
	}
}

func (*PayloadDelegationPending) isPayload() {}

func (p *PayloadDelegationPending) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadDelegationPending) Key() string {
	return "pending"
}

func (*PayloadDelegationPending) Namespace() SnapshotNamespace {
	return DelegationSnapshot
}

func PayloadGovernanceActiveFromProto(ga *snapshot.Payload_GovernanceActive) *PayloadGovernanceActive {
	return &PayloadGovernanceActive{
		GovernanceActive: GovernanceActiveFromProto(ga.GovernanceActive),
	}
}

func (p PayloadGovernanceActive) IntoProto() *snapshot.Payload_GovernanceActive {
	return &snapshot.Payload_GovernanceActive{
		GovernanceActive: p.GovernanceActive.IntoProto(),
	}
}

func (*PayloadGovernanceActive) isPayload() {}

func (p *PayloadGovernanceActive) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadGovernanceActive) Key() string {
	return "active"
}

func (*PayloadGovernanceActive) Namespace() SnapshotNamespace {
	return GovernanceSnapshot
}

func PayloadGovernanceNodeFromProto(gn *snapshot.Payload_GovernanceNode) *PayloadGovernanceNode {
	return &PayloadGovernanceNode{
		GovernanceNode: GovernanceNodeFromProto(gn.GovernanceNode),
	}
}

func (p PayloadGovernanceNode) IntoProto() *snapshot.Payload_GovernanceNode {
	return &snapshot.Payload_GovernanceNode{
		GovernanceNode: p.GovernanceNode.IntoProto(),
	}
}

func (*PayloadGovernanceNode) isPayload() {}

func (p *PayloadGovernanceNode) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadGovernanceNode) Key() string {
	return "node"
}

func (*PayloadGovernanceNode) Namespace() SnapshotNamespace {
	return GovernanceSnapshot
}

func PayloadGovernanceEnactedFromProto(ga *snapshot.Payload_GovernanceEnacted) *PayloadGovernanceEnacted {
	return &PayloadGovernanceEnacted{
		GovernanceEnacted: GovernanceEnactedFromProto(ga.GovernanceEnacted),
	}
}

func (p PayloadGovernanceEnacted) IntoProto() *snapshot.Payload_GovernanceEnacted {
	return &snapshot.Payload_GovernanceEnacted{
		GovernanceEnacted: p.GovernanceEnacted.IntoProto(),
	}
}

func (*PayloadGovernanceEnacted) isPayload() {}

func (p *PayloadGovernanceEnacted) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadGovernanceEnacted) Key() string {
	return "enacted"
}

func (*PayloadGovernanceEnacted) Namespace() SnapshotNamespace {
	return GovernanceSnapshot
}

func PayloadMarketPositionsFromProto(mp *snapshot.Payload_MarketPositions) *PayloadMarketPositions {
	return &PayloadMarketPositions{
		MarketPositions: MarketPositionsFromProto(mp.MarketPositions),
	}
}

func (p PayloadMarketPositions) IntoProto() *snapshot.Payload_MarketPositions {
	return &snapshot.Payload_MarketPositions{
		MarketPositions: p.MarketPositions.IntoProto(),
	}
}

func (*PayloadMarketPositions) isPayload() {}

func (p *PayloadMarketPositions) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadMarketPositions) Key() string {
	return p.MarketPositions.MarketID
}

func (*PayloadMarketPositions) Namespace() SnapshotNamespace {
	return PositionsSnapshot
}

func PayloadMatchingBookFromProto(pmb *snapshot.Payload_MatchingBook) *PayloadMatchingBook {
	return &PayloadMatchingBook{
		MatchingBook: MatchingBookFromProto(pmb.MatchingBook),
	}
}

func (p PayloadMatchingBook) IntoProto() *snapshot.Payload_MatchingBook {
	return &snapshot.Payload_MatchingBook{
		MatchingBook: p.MatchingBook.IntoProto(),
	}
}

func (*PayloadMatchingBook) isPayload() {}

func (p *PayloadMatchingBook) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadMatchingBook) Key() string {
	return p.MatchingBook.MarketID
}

func (*PayloadMatchingBook) Namespace() SnapshotNamespace {
	return MatchingSnapshot
}

func PayloadExecutionMarketsFromProto(pem *snapshot.Payload_ExecutionMarkets) *PayloadExecutionMarkets {
	return &PayloadExecutionMarkets{
		ExecutionMarkets: ExecutionMarketsFromProto(pem.ExecutionMarkets),
	}
}

func (p PayloadExecutionMarkets) IntoProto() *snapshot.Payload_ExecutionMarkets {
	return &snapshot.Payload_ExecutionMarkets{
		ExecutionMarkets: p.ExecutionMarkets.IntoProto(),
	}
}

func (*PayloadExecutionMarkets) isPayload() {}

func (p *PayloadExecutionMarkets) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadExecutionMarkets) Key() string {
	return "markets"
}

func (*PayloadExecutionMarkets) Namespace() SnapshotNamespace {
	return ExecutionSnapshot
}

func PayloadEpochFromProto(e *snapshot.Payload_Epoch) *PayloadEpoch {
	return &PayloadEpoch{
		EpochState: EpochFromProto(e.Epoch),
	}
}

func (p PayloadEpoch) IntoProto() *snapshot.Payload_Epoch {
	return &snapshot.Payload_Epoch{
		Epoch: p.EpochState.IntoProto(),
	}
}

func EpochFromProto(e *snapshot.EpochState) *EpochState {
	return &EpochState{
		Seq:                  e.Seq,
		StartTime:            time.Unix(0, e.StartTime).UTC(),
		ExpireTime:           time.Unix(0, e.ExpireTime).UTC(),
		ReadyToStartNewEpoch: e.ReadyToStartNewEpoch,
		ReadyToEndEpoch:      e.ReadyToEndEpoch,
	}
}

func (*PayloadEpoch) isPayload() {}

func (p *PayloadEpoch) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadEpoch) Key() string {
	return "all"
}

func (*PayloadEpoch) Namespace() SnapshotNamespace {
	return EpochSnapshot
}

func PayloadLimitStateFromProto(l *snapshot.Payload_LimitState) *PayloadLimitState {
	return &PayloadLimitState{
		LimitState: LimitFromProto(l.LimitState),
	}
}

func (p PayloadLimitState) IntoProto() *snapshot.Payload_LimitState {
	return &snapshot.Payload_LimitState{
		LimitState: p.LimitState.IntoProto(),
	}
}

func LimitFromProto(l *snapshot.LimitState) *LimitState {
	state := &LimitState{
		BlockCount:               l.BlockCount,
		CanProposeMarket:         l.CanProposeMarket,
		CanProposeAsset:          l.CanProposeAsset,
		GenesisLoaded:            l.GenesisLoaded,
		ProposeMarketEnabled:     l.ProposeMarketEnabled,
		ProposeAssetEnabled:      l.ProposeAssetEnabled,
		ProposeAssetEnabledFrom:  time.Time{},
		ProposeMarketEnabledFrom: time.Time{},
	}

	if l.ProposeAssetEnabledFrom != -1 {
		state.ProposeAssetEnabledFrom = time.Unix(0, l.ProposeAssetEnabledFrom).UTC()
	}

	if l.ProposeMarketEnabledFrom != -1 {
		state.ProposeMarketEnabledFrom = time.Unix(0, l.ProposeAssetEnabledFrom).UTC()
	}
	return state
}

func (*PayloadLimitState) isPayload() {}

func (p *PayloadLimitState) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadLimitState) Key() string {
	return "all"
}

func (*PayloadLimitState) Namespace() SnapshotNamespace {
	return LimitSnapshot
}

func PayloadStakingAccountsFromProto(sa *snapshot.Payload_StakingAccounts) *PayloadStakingAccounts {
	return &PayloadStakingAccounts{
		StakingAccounts: StakingAccountsFromProto(sa.StakingAccounts),
	}
}

func (p PayloadStakingAccounts) IntoProto() *snapshot.Payload_StakingAccounts {
	return &snapshot.Payload_StakingAccounts{
		StakingAccounts: p.StakingAccounts.IntoProto(),
	}
}

func (*PayloadStakingAccounts) isPayload() {}

func (p *PayloadStakingAccounts) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadStakingAccounts) Key() string {
	return "accounts"
}

func (*PayloadStakingAccounts) Namespace() SnapshotNamespace {
	return StakingSnapshot
}

func ActiveAssetsFromProto(aa *snapshot.ActiveAssets) *ActiveAssets {
	ret := ActiveAssets{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		aa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, aa)
	}
	return &ret
}

func (a ActiveAssets) IntoProto() *snapshot.ActiveAssets {
	ret := &snapshot.ActiveAssets{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

func PendingAssetsFromProto(aa *snapshot.PendingAssets) *PendingAssets {
	ret := PendingAssets{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		pa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, pa)
	}
	return &ret
}

func (a PendingAssets) IntoProto() *snapshot.PendingAssets {
	ret := &snapshot.PendingAssets{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

func BankingWithdrawalsFromProto(bw *snapshot.BankingWithdrawals) *BankingWithdrawals {
	ret := &BankingWithdrawals{
		Withdrawals: make([]*RWithdrawal, 0, len(bw.Withdrawals)),
	}
	for _, w := range bw.Withdrawals {
		ret.Withdrawals = append(ret.Withdrawals, RWithdrawalFromProto(w))
	}
	return ret
}

func (b BankingWithdrawals) IntoProto() *snapshot.BankingWithdrawals {
	ret := snapshot.BankingWithdrawals{
		Withdrawals: make([]*snapshot.Withdrawal, 0, len(b.Withdrawals)),
	}
	for _, w := range b.Withdrawals {
		ret.Withdrawals = append(ret.Withdrawals, w.IntoProto())
	}
	return &ret
}

func RWithdrawalFromProto(rw *snapshot.Withdrawal) *RWithdrawal {
	return &RWithdrawal{
		Ref:        rw.Ref,
		Withdrawal: WithdrawalFromProto(rw.Withdrawal),
	}
}

func (r RWithdrawal) IntoProto() *snapshot.Withdrawal {
	return &snapshot.Withdrawal{
		Ref:        r.Ref,
		Withdrawal: r.Withdrawal.IntoProto(),
	}
}

func BankingDepositsFromProto(bd *snapshot.BankingDeposits) *BankingDeposits {
	ret := &BankingDeposits{
		Deposit: make([]*BDeposit, 0, len(bd.Deposit)),
	}
	for _, d := range bd.Deposit {
		ret.Deposit = append(ret.Deposit, BDepositFromProto(d))
	}
	return ret
}

func (b BankingDeposits) IntoProto() *snapshot.BankingDeposits {
	ret := snapshot.BankingDeposits{
		Deposit: make([]*snapshot.Deposit, 0, len(b.Deposit)),
	}
	for _, d := range b.Deposit {
		ret.Deposit = append(ret.Deposit, d.IntoProto())
	}
	return &ret
}

func BDepositFromProto(d *snapshot.Deposit) *BDeposit {
	return &BDeposit{
		ID:      d.Id,
		Deposit: DepositFromProto(d.Deposit),
	}
}

func (b BDeposit) IntoProto() *snapshot.Deposit {
	return &snapshot.Deposit{
		Id:      b.ID,
		Deposit: b.Deposit.IntoProto(),
	}
}

func BankingSeenFromProto(bs *snapshot.BankingSeen) *BankingSeen {
	ret := BankingSeen{
		Refs: bs.Refs,
	}
	return &ret
}

func (b BankingSeen) IntoProto() *snapshot.BankingSeen {
	ret := snapshot.BankingSeen{
		Refs: b.Refs,
	}
	return &ret
}

func (a *BankingAssetActions) IntoProto() *snapshot.BankingAssetActions {
	ret := snapshot.BankingAssetActions{
		AssetAction: make([]*snapshot.AssetAction, 0, len(a.AssetAction)),
	}
	for _, aa := range a.AssetAction {
		ret.AssetAction = append(ret.AssetAction, aa.IntoProto())
	}
	return &ret
}

func (aa *AssetAction) IntoProto() *snapshot.AssetAction {
	ret := &snapshot.AssetAction{
		Id:                 aa.ID,
		State:              aa.State,
		Asset:              aa.Asset,
		BlockNumber:        aa.BlockNumber,
		TxIndex:            aa.TxIndex,
		Hash:               aa.Hash,
		Erc20BridgeStopped: aa.BridgeStopped,
		Erc20BridgeResumed: aa.BridgeResume,
	}
	if aa.BuiltinD != nil {
		ret.BuiltinDeposit = aa.BuiltinD.IntoProto()
	}
	if aa.Erc20D != nil {
		ret.Erc20Deposit = aa.Erc20D.IntoProto()
	}
	if aa.Erc20AL != nil {
		ret.AssetList = aa.Erc20AL.IntoProto()
	}
	if aa.ERC20AssetLimitsUpdated != nil {
		ret.Erc20AssetLimitsUpdated = aa.ERC20AssetLimitsUpdated.IntoProto()
	}
	return ret
}

func BankingAssetActionsFromProto(aa *snapshot.BankingAssetActions) *BankingAssetActions {
	ret := BankingAssetActions{
		AssetAction: make([]*AssetAction, 0, len(aa.AssetAction)),
	}

	for _, a := range aa.AssetAction {
		ret.AssetAction = append(ret.AssetAction, AssetActionFromProto(a))
	}
	return &ret
}

func AssetActionFromProto(a *snapshot.AssetAction) *AssetAction {
	aa := &AssetAction{
		ID:            a.Id,
		State:         a.State,
		Asset:         a.Asset,
		BlockNumber:   a.BlockNumber,
		TxIndex:       a.TxIndex,
		Hash:          a.Hash,
		BridgeStopped: a.Erc20BridgeStopped,
		BridgeResume:  a.Erc20BridgeResumed,
	}

	if a.Erc20Deposit != nil {
		erc20d, err := NewERC20DepositFromProto(a.Erc20Deposit)
		if err == nil {
			aa.Erc20D = erc20d
		}
	}

	if a.BuiltinDeposit != nil {
		builtind, err := NewBuiltinAssetDepositFromProto(a.BuiltinDeposit)
		if err == nil {
			aa.BuiltinD = builtind
		}
	}

	if a.AssetList != nil {
		aa.Erc20AL = NewERC20AssetListFromProto(a.AssetList)
	}

	if a.Erc20AssetLimitsUpdated != nil {
		aa.ERC20AssetLimitsUpdated = NewERC20AssetLimitsUpdatedFromProto(a.Erc20AssetLimitsUpdated)
	}

	return aa
}

func CheckpointFromProto(c *snapshot.Checkpoint) *CPState {
	return &CPState{
		NextCp: c.NextCp,
	}
}

func (c CPState) IntoProto() *snapshot.Checkpoint {
	return &snapshot.Checkpoint{
		NextCp: c.NextCp,
	}
}

func CollateralAccountsFromProto(ca *snapshot.CollateralAccounts) *CollateralAccounts {
	ret := CollateralAccounts{
		Accounts: make([]*Account, 0, len(ca.Accounts)),
	}
	for _, a := range ca.Accounts {
		ret.Accounts = append(ret.Accounts, AccountFromProto(a))
	}
	return &ret
}

func (c CollateralAccounts) IntoProto() *snapshot.CollateralAccounts {
	accs := Accounts(c.Accounts)
	return &snapshot.CollateralAccounts{
		Accounts: accs.IntoProto(),
	}
}

func CollateralAssetsFromProto(ca *snapshot.CollateralAssets) *CollateralAssets {
	ret := CollateralAssets{
		Assets: make([]*Asset, 0, len(ca.Assets)),
	}
	for _, a := range ca.Assets {
		ca, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}

		ret.Assets = append(ret.Assets, ca)
	}
	return &ret
}

func (c CollateralAssets) IntoProto() *snapshot.CollateralAssets {
	ret := snapshot.CollateralAssets{
		Assets: make([]*vega.Asset, 0, len(c.Assets)),
	}
	for _, a := range c.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return &ret
}

func AppStateFromProto(as *snapshot.AppState) *AppState {
	return &AppState{
		Height:  as.Height,
		Block:   as.Block,
		Time:    as.Time,
		ChainID: as.ChainId,
	}
}

func (a AppState) IntoProto() *snapshot.AppState {
	return &snapshot.AppState{
		Height:  a.Height,
		Block:   a.Block,
		Time:    a.Time,
		ChainId: a.ChainID,
	}
}

func NetParamsFromProto(np *snapshot.NetParams) *NetParams {
	ret := NetParams{
		Params: make([]*NetworkParameter, 0, len(np.Params)),
	}
	for _, p := range np.Params {
		ret.Params = append(ret.Params, NetworkParameterFromProto(p))
	}
	return &ret
}

func (n NetParams) IntoProto() *snapshot.NetParams {
	ret := snapshot.NetParams{
		Params: make([]*vega.NetworkParameter, 0, len(n.Params)),
	}
	for _, p := range n.Params {
		ret.Params = append(ret.Params, p.IntoProto())
	}
	return &ret
}

func DelegationActiveFromProto(da *snapshot.DelegationActive) *DelegationActive {
	ret := DelegationActive{
		Delegations: make([]*Delegation, 0, len(da.Delegations)),
	}
	for _, d := range da.Delegations {
		ret.Delegations = append(ret.Delegations, DelegationFromProto(d))
	}
	return &ret
}

func (d DelegationActive) IntoProto() *snapshot.DelegationActive {
	ret := snapshot.DelegationActive{
		Delegations: make([]*vega.Delegation, 0, len(d.Delegations)),
	}
	for _, a := range d.Delegations {
		ret.Delegations = append(ret.Delegations, a.IntoProto())
	}
	return &ret
}

func DelegationPendingFromProto(dp *snapshot.DelegationPending) *DelegationPending {
	ret := DelegationPending{
		Delegations:  make([]*Delegation, 0, len(dp.Delegations)),
		Undelegation: make([]*Delegation, 0, len(dp.Undelegation)),
	}
	for _, d := range dp.Delegations {
		ret.Delegations = append(ret.Delegations, DelegationFromProto(d))
	}
	for _, d := range dp.Undelegation {
		ret.Undelegation = append(ret.Undelegation, DelegationFromProto(d))
	}
	return &ret
}

func (d DelegationPending) IntoProto() *snapshot.DelegationPending {
	ret := snapshot.DelegationPending{
		Delegations:  make([]*vega.Delegation, 0, len(d.Delegations)),
		Undelegation: make([]*vega.Delegation, 0, len(d.Undelegation)),
	}
	for _, a := range d.Delegations {
		ret.Delegations = append(ret.Delegations, a.IntoProto())
	}
	for _, u := range d.Undelegation {
		ret.Undelegation = append(ret.Undelegation, u.IntoProto())
	}
	return &ret
}

func DelegationAutoFromProto(da *snapshot.DelegationAuto) *DelegationAuto {
	return &DelegationAuto{
		Parties: da.Parties[:],
	}
}

func (d DelegationAuto) IntoProto() *snapshot.DelegationAuto {
	return &snapshot.DelegationAuto{
		Parties: d.Parties[:],
	}
}

func GovernanceEnactedFromProto(ge *snapshot.GovernanceEnacted) *GovernanceEnacted {
	ret := GovernanceEnacted{
		Proposals: make([]*ProposalData, 0, len(ge.Proposals)),
	}
	for _, p := range ge.Proposals {
		ep := ProposalDataFromProto(p)
		ret.Proposals = append(ret.Proposals, ep)
	}
	return &ret
}

func (g GovernanceEnacted) IntoProto() *snapshot.GovernanceEnacted {
	ret := snapshot.GovernanceEnacted{
		Proposals: make([]*snapshot.ProposalData, 0, len(g.Proposals)),
	}
	for _, p := range g.Proposals {
		ret.Proposals = append(ret.Proposals, p.IntoProto())
	}
	return &ret
}

func GovernanceNodeFromProto(ge *snapshot.GovernanceNode) *GovernanceNode {
	ret := GovernanceNode{
		Proposals: make([]*Proposal, 0, len(ge.Proposals)),
	}
	for _, p := range ge.Proposals {
		gn, _ := ProposalFromProto(p)
		ret.Proposals = append(ret.Proposals, gn)
	}
	return &ret
}

func (g GovernanceNode) IntoProto() *snapshot.GovernanceNode {
	ret := snapshot.GovernanceNode{
		Proposals: make([]*vega.Proposal, 0, len(g.Proposals)),
	}
	for _, p := range g.Proposals {
		ret.Proposals = append(ret.Proposals, p.IntoProto())
	}
	return &ret
}

func ProposalDataFromProto(pp *snapshot.ProposalData) *ProposalData {
	p, _ := ProposalFromProto(pp.Proposal)
	ret := ProposalData{
		Proposal: p,
		Yes:      make([]*Vote, 0, len(pp.Yes)),
		No:       make([]*Vote, 0, len(pp.No)),
		Invalid:  make([]*Vote, 0, len(pp.Invalid)),
	}
	for _, v := range pp.Yes {
		// hashes were checked, comes from chain, this shouldn't result in errors
		// the balance and weight probably isn't even set, making errors impossible
		vote, _ := VoteFromProto(v)
		ret.Yes = append(ret.Yes, vote)
	}
	for _, v := range pp.No {
		vote, _ := VoteFromProto(v)
		ret.No = append(ret.No, vote)
	}
	for _, v := range pp.Invalid {
		vote, _ := VoteFromProto(v)
		ret.Invalid = append(ret.Invalid, vote)
	}
	return &ret
}

func (p ProposalData) IntoProto() *snapshot.ProposalData {
	ret := snapshot.ProposalData{
		Proposal: p.Proposal.IntoProto(),
		Yes:      make([]*vega.Vote, 0, len(p.Yes)),
		No:       make([]*vega.Vote, 0, len(p.No)),
		Invalid:  make([]*vega.Vote, 0, len(p.Invalid)),
	}
	for _, v := range p.Yes {
		ret.Yes = append(ret.Yes, v.IntoProto())
	}
	for _, v := range p.No {
		ret.No = append(ret.No, v.IntoProto())
	}
	for _, v := range p.Invalid {
		ret.Invalid = append(ret.Invalid, v.IntoProto())
	}
	return &ret
}

func GovernanceActiveFromProto(ga *snapshot.GovernanceActive) *GovernanceActive {
	ret := GovernanceActive{
		Proposals: make([]*ProposalData, 0, len(ga.Proposals)),
	}
	for _, p := range ga.Proposals {
		ret.Proposals = append(ret.Proposals, ProposalDataFromProto(p))
	}
	return &ret
}

func (g GovernanceActive) IntoProto() *snapshot.GovernanceActive {
	ret := snapshot.GovernanceActive{
		Proposals: make([]*snapshot.ProposalData, 0, len(g.Proposals)),
	}
	for _, p := range g.Proposals {
		ret.Proposals = append(ret.Proposals, p.IntoProto())
	}
	return &ret
}

func MarketPositionFromProto(p *snapshot.Position) *MarketPosition {
	price, _ := num.UintFromString(p.Price, 10)
	vwBuy, _ := num.UintFromString(p.VwBuyPrice, 10)
	vwSell, _ := num.UintFromString(p.VwSellPrice, 10)
	return &MarketPosition{
		PartyID: p.PartyId,
		Size:    p.Size,
		Buy:     p.Buy,
		Sell:    p.Sell,
		Price:   price,
		VwBuy:   vwBuy,
		VwSell:  vwSell,
	}
}

func (p MarketPosition) IntoProto() *snapshot.Position {
	return &snapshot.Position{
		PartyId:     p.PartyID,
		Size:        p.Size,
		Buy:         p.Buy,
		Sell:        p.Sell,
		Price:       p.Price.String(),
		VwBuyPrice:  p.VwBuy.String(),
		VwSellPrice: p.VwSell.String(),
	}
}

func MarketPositionsFromProto(mp *snapshot.MarketPositions) *MarketPositions {
	ret := MarketPositions{
		MarketID:  mp.MarketId,
		Positions: make([]*MarketPosition, 0, len(mp.Positions)),
	}
	for _, p := range mp.Positions {
		ret.Positions = append(ret.Positions, MarketPositionFromProto(p))
	}
	return &ret
}

func (m MarketPositions) IntoProto() *snapshot.MarketPositions {
	ret := snapshot.MarketPositions{
		MarketId:  m.MarketID,
		Positions: make([]*snapshot.Position, 0, len(m.Positions)),
	}
	for _, p := range m.Positions {
		ret.Positions = append(ret.Positions, p.IntoProto())
	}
	return &ret
}

func MatchingBookFromProto(mb *snapshot.MatchingBook) *MatchingBook {
	lastTradedPrice, _ := num.UintFromString(mb.LastTradedPrice, 10)
	ret := MatchingBook{
		MarketID:        mb.MarketId,
		Buy:             make([]*Order, 0, len(mb.Buy)),
		Sell:            make([]*Order, 0, len(mb.Sell)),
		LastTradedPrice: lastTradedPrice,
		Auction:         mb.Auction,
		BatchID:         mb.BatchId,
	}
	for _, o := range mb.Buy {
		or, _ := OrderFromProto(o)
		ret.Buy = append(ret.Buy, or)
	}
	for _, o := range mb.Sell {
		or, _ := OrderFromProto(o)
		ret.Sell = append(ret.Sell, or)
	}
	return &ret
}

func (m MatchingBook) IntoProto() *snapshot.MatchingBook {
	ret := snapshot.MatchingBook{
		MarketId:        m.MarketID,
		Buy:             make([]*vega.Order, 0, len(m.Buy)),
		Sell:            make([]*vega.Order, 0, len(m.Sell)),
		LastTradedPrice: m.LastTradedPrice.String(),
		Auction:         m.Auction,
		BatchId:         m.BatchID,
	}
	for _, o := range m.Buy {
		ret.Buy = append(ret.Buy, o.IntoProto())
	}
	for _, o := range m.Sell {
		ret.Sell = append(ret.Sell, o.IntoProto())
	}
	return &ret
}

func EquityShareFromProto(es *snapshot.EquityShare) *EquityShare {
	var mvp, r, pMvp num.Decimal
	if len(es.Mvp) > 0 {
		mvp, _ = num.DecimalFromString(es.Mvp)
	}
	if len(es.R) > 0 {
		r, _ = num.DecimalFromString(es.R)
	}
	if len(es.PMvp) > 0 {
		pMvp, _ = num.DecimalFromString(es.PMvp)
	}
	ret := EquityShare{
		Mvp:                 mvp,
		PMvp:                pMvp,
		R:                   r,
		OpeningAuctionEnded: es.OpeningAuctionEnded,
		Lps:                 make([]*EquityShareLP, 0, len(es.Lps)),
	}
	for _, s := range es.Lps {
		ret.Lps = append(ret.Lps, EquityShareLPFromProto(s))
	}
	return &ret
}

func (e EquityShare) IntoProto() *snapshot.EquityShare {
	ret := snapshot.EquityShare{
		Mvp:                 e.Mvp.String(),
		PMvp:                e.PMvp.String(),
		R:                   e.R.String(),
		OpeningAuctionEnded: e.OpeningAuctionEnded,
		Lps:                 make([]*snapshot.EquityShareLP, 0, len(e.Lps)),
	}
	for _, s := range e.Lps {
		ret.Lps = append(ret.Lps, s.IntoProto())
	}
	return &ret
}

func EquityShareLPFromProto(esl *snapshot.EquityShareLP) *EquityShareLP {
	var stake, vStake, share, avg num.Decimal
	if len(esl.Stake) > 0 {
		stake, _ = num.DecimalFromString(esl.Stake)
	}
	if len(esl.Vshare) > 0 {
		vStake, _ = num.DecimalFromString(esl.Vshare)
	}
	if len(esl.Share) > 0 {
		share, _ = num.DecimalFromString(esl.Share)
	}
	if len(esl.Avg) > 0 {
		avg, _ = num.DecimalFromString(esl.Avg)
	}
	return &EquityShareLP{
		ID:     esl.Id,
		Stake:  stake,
		Share:  share,
		Avg:    avg,
		VStake: vStake,
	}
}

func (e EquityShareLP) IntoProto() *snapshot.EquityShareLP {
	return &snapshot.EquityShareLP{
		Id:     e.ID,
		Stake:  e.Stake.String(),
		Share:  e.Share.String(),
		Avg:    e.Avg.String(),
		Vshare: e.VStake.String(),
	}
}

func PeggedOrdersStateFromProto(s *snapshot.PeggedOrders) *PeggedOrdersState {
	po := &PeggedOrdersState{
		Parked: make([]*Order, 0, len(s.ParkedOrders)),
	}

	for _, v := range s.ParkedOrders {
		o, _ := OrderFromProto(v)
		po.Parked = append(po.Parked, o)
	}
	return po
}

func (s PeggedOrdersState) IntoProto() *snapshot.PeggedOrders {
	po := &snapshot.PeggedOrders{
		ParkedOrders: make([]*vega.Order, 0, len(s.Parked)),
	}

	for _, v := range s.Parked {
		po.ParkedOrders = append(po.ParkedOrders, v.IntoProto())
	}
	sort.Slice(po.ParkedOrders, func(i, j int) bool { return po.ParkedOrders[i].Id < po.ParkedOrders[j].Id })
	return po
}

func AuctionStateFromProto(as *snapshot.AuctionState) *AuctionState {
	var end *AuctionDuration
	if as.End != nil {
		end = AuctionDurationFromProto(as.End)
	}
	return &AuctionState{
		Mode:               as.Mode,
		DefaultMode:        as.DefaultMode,
		Begin:              time.Unix(0, as.Begin).UTC(),
		Trigger:            as.Trigger,
		End:                end,
		Start:              as.Start,
		Stop:               as.Stop,
		Extension:          as.Extension,
		ExtensionEventSent: as.ExtensionEventSent,
	}
}

func (a AuctionState) IntoProto() *snapshot.AuctionState {
	var end *vega.AuctionDuration
	if a.End != nil {
		end = a.End.IntoProto()
	}
	return &snapshot.AuctionState{
		Mode:               a.Mode,
		DefaultMode:        a.DefaultMode,
		Trigger:            a.Trigger,
		Begin:              a.Begin.UnixNano(),
		End:                end,
		Start:              a.Start,
		Stop:               a.Stop,
		Extension:          a.Extension,
		ExtensionEventSent: a.ExtensionEventSent,
	}
}

func (e *EpochState) IntoProto() *snapshot.EpochState {
	return &snapshot.EpochState{
		Seq:                  e.Seq,
		StartTime:            e.StartTime.UnixNano(),
		ExpireTime:           e.ExpireTime.UnixNano(),
		ReadyToStartNewEpoch: e.ReadyToStartNewEpoch,
		ReadyToEndEpoch:      e.ReadyToEndEpoch,
	}
}

func (l *LimitState) IntoProto() *snapshot.LimitState {
	state := &snapshot.LimitState{
		BlockCount:               l.BlockCount,
		CanProposeMarket:         l.CanProposeMarket,
		CanProposeAsset:          l.CanProposeAsset,
		GenesisLoaded:            l.GenesisLoaded,
		ProposeMarketEnabled:     l.ProposeMarketEnabled,
		ProposeAssetEnabled:      l.ProposeAssetEnabled,
		ProposeMarketEnabledFrom: l.ProposeMarketEnabledFrom.UnixNano(),
		ProposeAssetEnabledFrom:  l.ProposeAssetEnabledFrom.UnixNano(),
	}

	// Use -1 to mean it hasn't been set
	if l.ProposeAssetEnabledFrom.IsZero() {
		state.ProposeAssetEnabledFrom = -1
	}

	if l.ProposeMarketEnabledFrom.IsZero() {
		state.ProposeMarketEnabledFrom = -1
	}
	return state
}

func KeyDecimalPairFromProto(dm *snapshot.DecimalMap) *KeyDecimalPair {
	var v num.Decimal
	if len(dm.Val) > 0 {
		v, _ = num.DecimalFromString(dm.Val)
	}
	return &KeyDecimalPair{
		Key: dm.Key,
		Val: v,
	}
}

func (d KeyDecimalPair) IntoProto() *snapshot.DecimalMap {
	return &snapshot.DecimalMap{
		Key: d.Key,
		Val: d.Val.String(),
	}
}

func PriceBoundFromProto(pb *snapshot.PriceBound) *PriceBound {
	var up, down num.Decimal
	if len(pb.UpFactor) > 0 {
		up, _ = num.DecimalFromString(pb.UpFactor)
	}
	if len(pb.DownFactor) > 0 {
		down, _ = num.DecimalFromString(pb.DownFactor)
	}
	return &PriceBound{
		Active:     pb.Active,
		UpFactor:   up,
		DownFactor: down,
		Trigger:    PriceMonitoringTriggerFromProto(pb.Trigger),
	}
}

func (p PriceBound) IntoProto() *snapshot.PriceBound {
	return &snapshot.PriceBound{
		Active:     p.Active,
		UpFactor:   p.UpFactor.String(),
		DownFactor: p.DownFactor.String(),
		Trigger:    p.Trigger.IntoProto(),
	}
}

func PriceRangeFromProto(pr *snapshot.PriceRange) *PriceRange {
	var min, max, ref num.Decimal
	if len(pr.Min) > 0 {
		min, _ = num.DecimalFromString(pr.Min)
	}
	if len(pr.Max) > 0 {
		max, _ = num.DecimalFromString(pr.Max)
	}
	if len(pr.Ref) > 0 {
		ref, _ = num.DecimalFromString(pr.Ref)
	}
	return &PriceRange{
		Min: min,
		Max: max,
		Ref: ref,
	}
}

func (p PriceRange) IntoProto() *snapshot.PriceRange {
	return &snapshot.PriceRange{
		Min: p.Min.String(),
		Max: p.Max.String(),
		Ref: p.Ref.String(),
	}
}

func PriceRangeCacheFromProto(prc *snapshot.PriceRangeCache) *PriceRangeCache {
	return &PriceRangeCache{
		Bound: PriceBoundFromProto(prc.Bound),
		Range: PriceRangeFromProto(prc.Range),
	}
}

func (p PriceRangeCache) IntoProto() *snapshot.PriceRangeCache {
	return &snapshot.PriceRangeCache{
		Bound: p.Bound.IntoProto(),
		Range: p.Range.IntoProto(),
	}
}

func CurrentPriceFromProto(scp *snapshot.CurrentPrice) *CurrentPrice {
	price, _ := num.UintFromString(scp.Price, 10)
	return &CurrentPrice{
		Price:  price,
		Volume: scp.Volume,
	}
}

func (cp CurrentPrice) IntoProto() *snapshot.CurrentPrice {
	return &snapshot.CurrentPrice{
		Price:  cp.Price.String(),
		Volume: cp.Volume,
	}
}

func PastPriceFromProto(spp *snapshot.PastPrice) *PastPrice {
	vwp, _ := num.DecimalFromString(spp.VolumeWeightedPrice)
	return &PastPrice{
		Time:                time.Unix(0, spp.Time).UTC(),
		VolumeWeightedPrice: vwp,
	}
}

func (pp PastPrice) IntoProto() *snapshot.PastPrice {
	return &snapshot.PastPrice{
		Time:                pp.Time.UnixNano(),
		VolumeWeightedPrice: pp.VolumeWeightedPrice.String(),
	}
}

func PriceMonitorFromProto(pm *snapshot.PriceMonitor) *PriceMonitor {
	ret := PriceMonitor{
		Initialised:                 pm.Initialised,
		FPHorizons:                  make([]*KeyDecimalPair, 0, len(pm.FpHorizons)),
		Now:                         time.Unix(0, pm.Now).UTC(),
		Update:                      time.Unix(0, pm.Update).UTC(),
		Bounds:                      make([]*PriceBound, 0, len(pm.Bounds)),
		PriceRangeCacheTime:         time.Unix(0, pm.PriceRangeCacheTime).UTC(),
		PriceRangeCache:             make([]*PriceRangeCache, 0, len(pm.PriceRangeCache)),
		PricesNow:                   make([]*CurrentPrice, 0, len(pm.PricesNow)),
		PricesPast:                  make([]*PastPrice, 0, len(pm.PricesPast)),
		RefPriceCacheTime:           time.Unix(0, pm.RefPriceCacheTime).UTC(),
		RefPriceCache:               make([]*KeyDecimalPair, 0, len(pm.RefPriceCache)),
		PriceBoundsConsensusReached: pm.ConsensusReached,
	}
	for _, d := range pm.FpHorizons {
		ret.FPHorizons = append(ret.FPHorizons, KeyDecimalPairFromProto(d))
	}
	for _, d := range pm.RefPriceCache {
		ret.RefPriceCache = append(ret.RefPriceCache, KeyDecimalPairFromProto(d))
	}
	for _, b := range pm.Bounds {
		ret.Bounds = append(ret.Bounds, PriceBoundFromProto(b))
	}
	for _, r := range pm.PriceRangeCache {
		ret.PriceRangeCache = append(ret.PriceRangeCache, PriceRangeCacheFromProto(r))
	}
	for _, p := range pm.PricesNow {
		ret.PricesNow = append(ret.PricesNow, CurrentPriceFromProto(p))
	}
	for _, p := range pm.PricesPast {
		ret.PricesPast = append(ret.PricesPast, PastPriceFromProto(p))
	}
	return &ret
}

func (p PriceMonitor) IntoProto() *snapshot.PriceMonitor {
	ret := snapshot.PriceMonitor{
		Initialised:         p.Initialised,
		FpHorizons:          make([]*snapshot.DecimalMap, 0, len(p.FPHorizons)),
		Now:                 p.Now.UnixNano(),
		Update:              p.Update.UnixNano(),
		Bounds:              make([]*snapshot.PriceBound, 0, len(p.Bounds)),
		PriceRangeCacheTime: p.PriceRangeCacheTime.UnixNano(),
		PriceRangeCache:     make([]*snapshot.PriceRangeCache, 0, len(p.PriceRangeCache)),
		PricesNow:           make([]*snapshot.CurrentPrice, 0, len(p.PricesNow)),
		PricesPast:          make([]*snapshot.PastPrice, 0, len(p.PricesPast)),
		RefPriceCacheTime:   p.RefPriceCacheTime.UnixNano(),
		RefPriceCache:       make([]*snapshot.DecimalMap, 0, len(p.RefPriceCache)),
		ConsensusReached:    p.PriceBoundsConsensusReached,
	}
	for _, d := range p.FPHorizons {
		ret.FpHorizons = append(ret.FpHorizons, d.IntoProto())
	}
	for _, d := range p.RefPriceCache {
		ret.RefPriceCache = append(ret.RefPriceCache, d.IntoProto())
	}
	for _, b := range p.Bounds {
		ret.Bounds = append(ret.Bounds, b.IntoProto())
	}
	for _, r := range p.PriceRangeCache {
		ret.PriceRangeCache = append(ret.PriceRangeCache, r.IntoProto())
	}
	for _, r := range p.PricesNow {
		ret.PricesNow = append(ret.PricesNow, r.IntoProto())
	}
	for _, r := range p.PricesPast {
		ret.PricesPast = append(ret.PricesPast, r.IntoProto())
	}

	return &ret
}

func FeeSplitterFromProto(fs *snapshot.FeeSplitter) *FeeSplitter {
	tv, _ := num.UintFromString(fs.TradeValue, 10)
	avg, _ := num.DecimalFromString(fs.Avg)
	ret := FeeSplitter{
		TimeWindowStart: time.Unix(0, fs.TimeWindowStart),
		TradeValue:      tv,
		Avg:             avg,
		Window:          fs.Window,
	}

	return &ret
}

func (f FeeSplitter) IntoProto() *snapshot.FeeSplitter {
	ret := snapshot.FeeSplitter{
		TimeWindowStart: f.TimeWindowStart.UnixNano(),
		TradeValue:      f.TradeValue.String(),
		Avg:             f.Avg.String(),
		Window:          f.Window,
	}

	return &ret
}

func ExecMarketFromProto(em *snapshot.Market) *ExecMarket {
	var (
		lastBB, lastBA, lastMB, lastMA, markPrice *num.Uint
		lastMVP                                   num.Decimal
	)
	lastBB, _ = num.UintFromString(em.LastBestBid, 10)
	lastBA, _ = num.UintFromString(em.LastBestAsk, 10)
	lastMB, _ = num.UintFromString(em.LastMidBid, 10)
	lastMA, _ = num.UintFromString(em.LastMidAsk, 10)
	markPrice, _ = num.UintFromString(em.CurrentMarkPrice, 10)

	shortRF, _ := num.DecimalFromString(em.RiskFactorShort)
	longRF, _ := num.DecimalFromString(em.RiskFactorLong)

	if len(em.LastMarketValueProxy) > 0 {
		lastMVP, _ = num.DecimalFromString(em.LastMarketValueProxy)
	}

	var sp *num.Uint
	if em.SettlementData != "" {
		sp, _ = num.UintFromString(em.SettlementData, 10)
	}

	ret := ExecMarket{
		Market:                     MarketFromProto(em.Market),
		PriceMonitor:               PriceMonitorFromProto(em.PriceMonitor),
		AuctionState:               AuctionStateFromProto(em.AuctionState),
		PeggedOrders:               PeggedOrdersStateFromProto(em.PeggedOrders),
		ExpiringOrders:             make([]*Order, 0, len(em.ExpiringOrders)),
		LastEquityShareDistributed: em.LastEquityShareDistributed,
		EquityShare:                EquityShareFromProto(em.EquityShare),
		LastBestAsk:                lastBA,
		LastBestBid:                lastBB,
		LastMidAsk:                 lastMA,
		LastMidBid:                 lastMB,
		LastMarketValueProxy:       lastMVP,
		CurrentMarkPrice:           markPrice,
		ShortRiskFactor:            shortRF,
		LongRiskFactor:             longRF,
		RiskFactorConsensusReached: em.RiskFactorConsensusReached,
		FeeSplitter:                FeeSplitterFromProto(em.FeeSplitter),
		SettlementData:             sp,
		NextMTM:                    em.NextMarkToMarket,
	}
	for _, o := range em.ExpiringOrders {
		or, _ := OrderFromProto(o)
		ret.ExpiringOrders = append(ret.ExpiringOrders, or)
	}
	return &ret
}

func (e ExecMarket) IntoProto() *snapshot.Market {
	sp := ""
	if e.SettlementData != nil {
		sp = e.SettlementData.String()
	}

	ret := snapshot.Market{
		Market:                     e.Market.IntoProto(),
		PriceMonitor:               e.PriceMonitor.IntoProto(),
		AuctionState:               e.AuctionState.IntoProto(),
		PeggedOrders:               e.PeggedOrders.IntoProto(),
		ExpiringOrders:             make([]*vega.Order, 0, len(e.ExpiringOrders)),
		LastEquityShareDistributed: e.LastEquityShareDistributed,
		EquityShare:                e.EquityShare.IntoProto(),
		LastBestAsk:                e.LastBestAsk.String(),
		LastBestBid:                e.LastBestBid.String(),
		LastMidAsk:                 e.LastMidAsk.String(),
		LastMidBid:                 e.LastMidBid.String(),
		LastMarketValueProxy:       e.LastMarketValueProxy.String(),
		CurrentMarkPrice:           e.CurrentMarkPrice.String(),
		RiskFactorShort:            e.ShortRiskFactor.String(),
		RiskFactorLong:             e.LongRiskFactor.String(),
		RiskFactorConsensusReached: e.RiskFactorConsensusReached,
		FeeSplitter:                e.FeeSplitter.IntoProto(),
		SettlementData:             sp,
		NextMarkToMarket:           e.NextMTM,
	}
	for _, o := range e.ExpiringOrders {
		ret.ExpiringOrders = append(ret.ExpiringOrders, o.IntoProto())
	}
	return &ret
}

func ExecutionMarketsFromProto(em *snapshot.ExecutionMarkets) *ExecutionMarkets {
	mkts := make([]*ExecMarket, 0, len(em.Markets))
	for _, m := range em.Markets {
		mkts = append(mkts, ExecMarketFromProto(m))
	}
	return &ExecutionMarkets{
		Markets: mkts,
	}
}

func (e ExecutionMarkets) IntoProto() *snapshot.ExecutionMarkets {
	mkts := make([]*snapshot.Market, 0, len(e.Markets))
	for _, m := range e.Markets {
		mkts = append(mkts, m.IntoProto())
	}
	return &snapshot.ExecutionMarkets{
		Markets: mkts,
	}
}

func StakingAccountsFromProto(sa *snapshot.StakingAccounts) *StakingAccounts {
	accs := make([]*StakingAccount, 0, len(sa.Accounts))
	for _, a := range sa.Accounts {
		accs = append(accs, StakingAccountFromProto(a))
	}
	bal, _ := num.UintFromString(sa.StakingAssetTotalSupply, 10)
	return &StakingAccounts{
		Accounts:                accs,
		StakingAssetTotalSupply: bal,
	}
}

func (s StakingAccounts) IntoProto() *snapshot.StakingAccounts {
	accs := make([]*snapshot.StakingAccount, 0, len(s.Accounts))
	for _, a := range s.Accounts {
		accs = append(accs, a.IntoProto())
	}
	amount := "0"
	if s.StakingAssetTotalSupply != nil {
		amount = s.StakingAssetTotalSupply.String()
	}
	return &snapshot.StakingAccounts{
		Accounts:                accs,
		StakingAssetTotalSupply: amount,
	}
}

func StakingAccountFromProto(sa *snapshot.StakingAccount) *StakingAccount {
	bal, _ := num.UintFromString(sa.Balance, 10)
	evts := make([]*StakeLinking, 0, len(sa.Events))
	for _, e := range sa.Events {
		evts = append(evts, StakeLinkingFromProto(e))
	}
	return &StakingAccount{
		Party:   sa.Party,
		Balance: bal,
		Events:  evts,
	}
}

func (s StakingAccount) IntoProto() *snapshot.StakingAccount {
	evts := make([]*eventspb.StakeLinking, 0, len(s.Events))
	for _, e := range s.Events {
		evts = append(evts, e.IntoProto())
	}
	return &snapshot.StakingAccount{
		Party:   s.Party,
		Balance: s.Balance.String(),
		Events:  evts,
	}
}

type PartyTokenBalance struct {
	Party   string
	Balance *num.Uint
}

type PartyProposalVoteCount struct {
	Party    string
	Proposal string
	Count    uint64
}

type PartyCount struct {
	Party string
	Count uint64
}

type BannedParty struct {
	Party      string
	UntilEpoch uint64
}

type BlockRejectStats struct {
	Total    uint64
	Rejected uint64
}

type PayloadVoteSpamPolicy struct {
	VoteSpamPolicy *VoteSpamPolicy
}

type PayloadSimpleSpamPolicy struct {
	SimpleSpamPolicy *SimpleSpamPolicy
}

type SimpleSpamPolicy struct {
	PolicyName      string
	PartyToCount    []*PartyCount
	BannedParty     []*BannedParty
	CurrentEpochSeq uint64
}

type VoteSpamPolicy struct {
	PartyProposalVoteCount  []*PartyProposalVoteCount
	BannedParty             []*BannedParty
	RecentBlocksRejectStats []*BlockRejectStats
	CurrentBlockIndex       uint64
	LastIncreaseBlock       uint64
	CurrentEpochSeq         uint64
	MinVotingTokensFactor   *num.Uint
}

func PayloadSimpleSpamPolicyFromProto(ssp *snapshot.Payload_SimpleSpamPolicy) *PayloadSimpleSpamPolicy {
	return &PayloadSimpleSpamPolicy{
		SimpleSpamPolicy: SimpleSpamPolicyFromProto(ssp.SimpleSpamPolicy),
	}
}

func PayloadVoteSpamPolicyFromProto(vsp *snapshot.Payload_VoteSpamPolicy) *PayloadVoteSpamPolicy {
	return &PayloadVoteSpamPolicy{
		VoteSpamPolicy: VoteSpamPolicyFromProto(vsp.VoteSpamPolicy),
	}
}

func SimpleSpamPolicyFromProto(ssp *snapshot.SimpleSpamPolicy) *SimpleSpamPolicy {
	partyCount := make([]*PartyCount, 0, len(ssp.PartyToCount))
	for _, ptv := range ssp.PartyToCount {
		partyCount = append(partyCount, PartyCountFromProto(ptv))
	}

	bannedParties := make([]*BannedParty, 0, len(ssp.BannedParties))
	for _, ban := range ssp.BannedParties {
		bannedParties = append(bannedParties, BannedPartyFromProto(ban))
	}

	return &SimpleSpamPolicy{
		PolicyName:      ssp.PolicyName,
		PartyToCount:    partyCount,
		BannedParty:     bannedParties,
		CurrentEpochSeq: ssp.CurrentEpochSeq,
	}
}

func VoteSpamPolicyFromProto(vsp *snapshot.VoteSpamPolicy) *VoteSpamPolicy {
	partyProposalVoteCount := make([]*PartyProposalVoteCount, 0, len(vsp.PartyToVote))
	for _, ptv := range vsp.PartyToVote {
		partyProposalVoteCount = append(partyProposalVoteCount, PartyProposalVoteCountFromProto(ptv))
	}

	bannedParties := make([]*BannedParty, 0, len(vsp.BannedParties))
	for _, ban := range vsp.BannedParties {
		bannedParties = append(bannedParties, BannedPartyFromProto(ban))
	}

	recentBlocksRejectStats := make([]*BlockRejectStats, 0, len(vsp.RecentBlocksRejectStats))
	for _, rejects := range vsp.RecentBlocksRejectStats {
		recentBlocksRejectStats = append(recentBlocksRejectStats, BlockRejectStatsFromProto(rejects))
	}

	minTokensFactor, _ := num.UintFromString(vsp.MinVotingTokensFactor, 10)

	return &VoteSpamPolicy{
		PartyProposalVoteCount:  partyProposalVoteCount,
		BannedParty:             bannedParties,
		RecentBlocksRejectStats: recentBlocksRejectStats,
		LastIncreaseBlock:       vsp.LastIncreaseBlock,
		CurrentBlockIndex:       vsp.CurrentBlockIndex,
		CurrentEpochSeq:         vsp.CurrentEpochSeq,
		MinVotingTokensFactor:   minTokensFactor,
	}
}

func BlockRejectStatsFromProto(rejects *snapshot.BlockRejectStats) *BlockRejectStats {
	return &BlockRejectStats{
		Total:    rejects.Total,
		Rejected: rejects.Rejected,
	}
}

func (brs *BlockRejectStats) IntoProto() *snapshot.BlockRejectStats {
	return &snapshot.BlockRejectStats{
		Total:    brs.Total,
		Rejected: brs.Rejected,
	}
}

func PartyTokenBalanceFromProto(balance *snapshot.PartyTokenBalance) *PartyTokenBalance {
	b, _ := num.UintFromString(balance.Balance, 10)
	return &PartyTokenBalance{
		Party:   balance.Party,
		Balance: b,
	}
}

func BannedPartyFromProto(ban *snapshot.BannedParty) *BannedParty {
	return &BannedParty{
		Party:      ban.Party,
		UntilEpoch: ban.UntilEpoch,
	}
}

func PartyProposalVoteCountFromProto(ppvc *snapshot.PartyProposalVoteCount) *PartyProposalVoteCount {
	return &PartyProposalVoteCount{
		Party:    ppvc.Party,
		Proposal: ppvc.Proposal,
		Count:    ppvc.Count,
	}
}

func PartyCountFromProto(pc *snapshot.SpamPartyTransactionCount) *PartyCount {
	return &PartyCount{
		Party: pc.Party,
		Count: pc.Count,
	}
}

func (p *PartyProposalVoteCount) IntoProto() *snapshot.PartyProposalVoteCount {
	return &snapshot.PartyProposalVoteCount{
		Party:    p.Party,
		Proposal: p.Proposal,
		Count:    p.Count,
	}
}

func (b *BannedParty) IntoProto() *snapshot.BannedParty {
	return &snapshot.BannedParty{
		Party:      b.Party,
		UntilEpoch: b.UntilEpoch,
	}
}

func (ptc *PartyTokenBalance) IntoProto() *snapshot.PartyTokenBalance {
	return &snapshot.PartyTokenBalance{
		Party:   ptc.Party,
		Balance: ptc.Balance.String(),
	}
}

func (ssp *SimpleSpamPolicy) IntoProto() *snapshot.SimpleSpamPolicy {
	partyToCount := make([]*snapshot.SpamPartyTransactionCount, 0, len(ssp.PartyToCount))
	for _, pc := range ssp.PartyToCount {
		partyToCount = append(partyToCount, &snapshot.SpamPartyTransactionCount{Party: pc.Party, Count: pc.Count})
	}

	bannedParties := make([]*snapshot.BannedParty, 0, len(ssp.BannedParty))
	for _, ban := range ssp.BannedParty {
		bannedParties = append(bannedParties, ban.IntoProto())
	}

	return &snapshot.SimpleSpamPolicy{
		PolicyName:      ssp.PolicyName,
		PartyToCount:    partyToCount,
		BannedParties:   bannedParties,
		CurrentEpochSeq: ssp.CurrentEpochSeq,
	}
}

func (vsp *VoteSpamPolicy) IntoProto() *snapshot.VoteSpamPolicy {
	partyProposalVoteCount := make([]*snapshot.PartyProposalVoteCount, 0, len(vsp.PartyProposalVoteCount))
	for _, ptv := range vsp.PartyProposalVoteCount {
		partyProposalVoteCount = append(partyProposalVoteCount, ptv.IntoProto())
	}

	bannedParties := make([]*snapshot.BannedParty, 0, len(vsp.BannedParty))
	for _, ban := range vsp.BannedParty {
		bannedParties = append(bannedParties, ban.IntoProto())
	}

	recentBlocksRejectStats := make([]*snapshot.BlockRejectStats, 0, len(vsp.RecentBlocksRejectStats))
	for _, rejects := range vsp.RecentBlocksRejectStats {
		recentBlocksRejectStats = append(recentBlocksRejectStats, rejects.IntoProto())
	}
	return &snapshot.VoteSpamPolicy{
		PartyToVote:             partyProposalVoteCount,
		BannedParties:           bannedParties,
		RecentBlocksRejectStats: recentBlocksRejectStats,
		LastIncreaseBlock:       vsp.LastIncreaseBlock,
		CurrentBlockIndex:       vsp.CurrentBlockIndex,
		CurrentEpochSeq:         vsp.CurrentEpochSeq,
		MinVotingTokensFactor:   vsp.MinVotingTokensFactor.String(),
	}
}

func (p *PayloadVoteSpamPolicy) IntoProto() *snapshot.Payload_VoteSpamPolicy {
	return &snapshot.Payload_VoteSpamPolicy{
		VoteSpamPolicy: p.VoteSpamPolicy.IntoProto(),
	}
}

func (*PayloadVoteSpamPolicy) isPayload() {}

func (p *PayloadVoteSpamPolicy) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadVoteSpamPolicy) Key() string {
	return "voteSpamPolicy"
}

func (*PayloadVoteSpamPolicy) Namespace() SnapshotNamespace {
	return SpamSnapshot
}

func (p *PayloadSimpleSpamPolicy) IntoProto() *snapshot.Payload_SimpleSpamPolicy {
	return &snapshot.Payload_SimpleSpamPolicy{
		SimpleSpamPolicy: p.SimpleSpamPolicy.IntoProto(),
	}
}

func (*PayloadSimpleSpamPolicy) isPayload() {}

func (p *PayloadSimpleSpamPolicy) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadSimpleSpamPolicy) Key() string {
	return p.SimpleSpamPolicy.PolicyName
}

func (*PayloadSimpleSpamPolicy) Namespace() SnapshotNamespace {
	return SpamSnapshot
}

type PayloadRewardsPayout struct {
	RewardsPendingPayouts *RewardsPendingPayouts
}

type RewardsPendingPayouts struct {
	ScheduledRewardsPayout []*ScheduledRewardsPayout
}

type ScheduledRewardsPayout struct {
	PayoutTime    int64
	RewardsPayout []*RewardsPayout
}

type RewardsPayout struct {
	FromAccount  string
	Asset        string
	PartyAmounts []*RewardsPartyAmount
	TotalReward  *num.Uint
	EpochSeq     string
	Timestamp    int64
}

type RewardsPartyAmount struct {
	Party  string
	Amount *num.Uint
}

func PayloadRewardPayoutFromProto(rpp *snapshot.Payload_RewardsPendingPayouts) *PayloadRewardsPayout {
	return &PayloadRewardsPayout{
		RewardsPendingPayouts: RewardPendingPayoutsFromProto(rpp.RewardsPendingPayouts),
	}
}

func RewardPendingPayoutsFromProto(rpps *snapshot.RewardsPendingPayouts) *RewardsPendingPayouts {
	scheduledPayouts := make([]*ScheduledRewardsPayout, 0, len(rpps.ScheduledRewardsPayout))

	for _, p := range rpps.ScheduledRewardsPayout {
		scheduledPayouts = append(scheduledPayouts, ScheduledRewardsPayoutFromProto(p))
	}

	return &RewardsPendingPayouts{
		ScheduledRewardsPayout: scheduledPayouts,
	}
}

func ScheduledRewardsPayoutFromProto(srp *snapshot.ScheduledRewardsPayout) *ScheduledRewardsPayout {
	payouts := make([]*RewardsPayout, 0, len(srp.RewardsPayout))
	for _, p := range srp.RewardsPayout {
		payouts = append(payouts, RewardsPayoutFromProto(p))
	}

	return &ScheduledRewardsPayout{
		PayoutTime:    srp.PayoutTime,
		RewardsPayout: payouts,
	}
}

func RewardsPayoutFromProto(p *snapshot.RewardsPayout) *RewardsPayout {
	totalReward, _ := num.UintFromString(p.TotalReward, 10)
	partyAmounts := make([]*RewardsPartyAmount, 0, len(p.RewardPartyAmount))
	for _, pa := range p.RewardPartyAmount {
		amount, _ := num.UintFromString(pa.Amount, 10)
		partyAmounts = append(partyAmounts, &RewardsPartyAmount{Party: pa.Party, Amount: amount})
	}

	return &RewardsPayout{
		FromAccount:  p.FromAccount,
		Asset:        p.Asset,
		TotalReward:  totalReward,
		EpochSeq:     p.EpochSeq,
		Timestamp:    p.Timestamp,
		PartyAmounts: partyAmounts,
	}
}

func (p PayloadRewardsPayout) IntoProto() *snapshot.Payload_RewardsPendingPayouts {
	return &snapshot.Payload_RewardsPendingPayouts{
		RewardsPendingPayouts: p.RewardsPendingPayouts.IntoProto(),
	}
}

func (rpp RewardsPendingPayouts) IntoProto() *snapshot.RewardsPendingPayouts {
	scheduled := make([]*snapshot.ScheduledRewardsPayout, 0, len(rpp.ScheduledRewardsPayout))
	for _, p := range rpp.ScheduledRewardsPayout {
		scheduled = append(scheduled, p.IntoProto())
	}
	return &snapshot.RewardsPendingPayouts{
		ScheduledRewardsPayout: scheduled,
	}
}

func (srp ScheduledRewardsPayout) IntoProto() *snapshot.ScheduledRewardsPayout {
	payouts := make([]*snapshot.RewardsPayout, 0, len(srp.RewardsPayout))
	for _, p := range srp.RewardsPayout {
		payouts = append(payouts, p.IntoProto())
	}

	return &snapshot.ScheduledRewardsPayout{
		PayoutTime:    srp.PayoutTime,
		RewardsPayout: payouts,
	}
}

func (rp *RewardsPayout) IntoProto() *snapshot.RewardsPayout {
	totalReward := rp.TotalReward.String()
	partyAmounts := make([]*snapshot.RewardsPartyAmount, 0, len(rp.PartyAmounts))
	for _, pa := range rp.PartyAmounts {
		amount := pa.Amount.String()
		partyAmounts = append(partyAmounts, &snapshot.RewardsPartyAmount{Party: pa.Party, Amount: amount})
	}

	return &snapshot.RewardsPayout{
		FromAccount:       rp.FromAccount,
		Asset:             rp.Asset,
		TotalReward:       totalReward,
		EpochSeq:          rp.EpochSeq,
		Timestamp:         rp.Timestamp,
		RewardPartyAmount: partyAmounts,
	}
}

func (*PayloadRewardsPayout) isPayload() {}

func (p *PayloadRewardsPayout) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadRewardsPayout) Key() string {
	return "pendingPayout"
}

func (*PayloadRewardsPayout) Namespace() SnapshotNamespace {
	return RewardSnapshot
}

func PayloadNotaryFromProto(n *snapshot.Payload_Notary) *PayloadNotary {
	return &PayloadNotary{
		Notary: NotaryFromProto(n.Notary),
	}
}

func NotaryFromProto(n *snapshot.Notary) *Notary {
	sigKinds := make([]*NotarySigs, 0, len(n.NotarySigs))

	for _, sk := range n.NotarySigs {
		sigKinds = append(sigKinds, NotarySigFromProto(sk))
	}

	return &Notary{
		Sigs: sigKinds,
	}
}

func NotarySigFromProto(sk *snapshot.NotarySigs) *NotarySigs {
	return &NotarySigs{
		ID:   sk.Id,
		Kind: sk.Kind,
		Node: sk.Node,
		Sig:  sk.Sig,
	}
}

func (p PayloadNotary) IntoProto() *snapshot.Payload_Notary {
	return &snapshot.Payload_Notary{
		Notary: p.Notary.IntoProto(),
	}
}

func (n Notary) IntoProto() *snapshot.Notary {
	sigKinds := make([]*snapshot.NotarySigs, 0, len(n.Sigs))
	for _, sk := range n.Sigs {
		sigKinds = append(sigKinds, sk.IntoProto())
	}
	return &snapshot.Notary{
		NotarySigs: sigKinds,
	}
}

func (sk NotarySigs) IntoProto() *snapshot.NotarySigs {
	return &snapshot.NotarySigs{
		Id:   sk.ID,
		Kind: sk.Kind,
		Node: sk.Node,
		Sig:  sk.Sig,
	}
}

func (*PayloadNotary) isPayload() {}

func (p *PayloadNotary) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadNotary) Key() string {
	return "all"
}

func (*PayloadNotary) Namespace() SnapshotNamespace {
	return NotarySnapshot
}

func PayloadStakeVerifierRemovedFromProto(svd *snapshot.Payload_StakeVerifierRemoved) *PayloadStakeVerifierRemoved {
	pending := make([]*StakeRemoved, 0, len(svd.StakeVerifierRemoved.PendingRemoved))

	for _, pr := range svd.StakeVerifierRemoved.PendingRemoved {
		removed := &StakeRemoved{
			EthereumAddress: crypto.EthereumChecksumAddress(pr.EthereumAddress),
			TxID:            pr.TxId,
			LogIndex:        pr.LogIndex,
			BlockNumber:     pr.BlockNumber,
			ID:              pr.Id,
			VegaPubKey:      pr.VegaPublicKey,
			BlockTime:       pr.BlockTime,
			Amount:          num.UintZero(),
		}

		if len(pr.Amount) > 0 {
			removed.Amount, _ = num.UintFromString(pr.Amount, 10)
		}
		pending = append(pending, removed)
	}

	return &PayloadStakeVerifierRemoved{
		StakeVerifierRemoved: pending,
	}
}

func (p *PayloadStakeVerifierRemoved) IntoProto() *snapshot.Payload_StakeVerifierRemoved {
	pending := make([]*snapshot.StakeVerifierPending, 0, len(p.StakeVerifierRemoved))

	for _, p := range p.StakeVerifierRemoved {
		pending = append(pending,
			&snapshot.StakeVerifierPending{
				EthereumAddress: crypto.EthereumChecksumAddress(p.EthereumAddress),
				VegaPublicKey:   p.VegaPubKey,
				Amount:          p.Amount.String(),
				BlockTime:       p.BlockTime,
				BlockNumber:     p.BlockNumber,
				LogIndex:        p.LogIndex,
				TxId:            p.TxID,
				Id:              p.ID,
			})
	}

	return &snapshot.Payload_StakeVerifierRemoved{
		StakeVerifierRemoved: &snapshot.StakeVerifierRemoved{
			PendingRemoved: pending,
		},
	}
}

func (*PayloadStakeVerifierRemoved) isPayload() {}

func (p *PayloadStakeVerifierRemoved) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadStakeVerifierRemoved) Key() string {
	return "removed"
}

func (*PayloadStakeVerifierRemoved) Namespace() SnapshotNamespace {
	return StakeVerifierSnapshot
}

func PayloadStakeVerifierDepositedFromProto(svd *snapshot.Payload_StakeVerifierDeposited) *PayloadStakeVerifierDeposited {
	pending := make([]*StakeDeposited, 0, len(svd.StakeVerifierDeposited.PendingDeposited))

	for _, pd := range svd.StakeVerifierDeposited.PendingDeposited {
		deposit := &StakeDeposited{
			EthereumAddress: crypto.EthereumChecksumAddress(pd.EthereumAddress),
			TxID:            pd.TxId,
			LogIndex:        pd.LogIndex,
			BlockNumber:     pd.BlockNumber,
			ID:              pd.Id,
			VegaPubKey:      pd.VegaPublicKey,
			BlockTime:       pd.BlockTime,
			Amount:          num.UintZero(),
		}

		if len(pd.Amount) > 0 {
			deposit.Amount, _ = num.UintFromString(pd.Amount, 10)
		}
		pending = append(pending, deposit)
	}

	return &PayloadStakeVerifierDeposited{
		StakeVerifierDeposited: pending,
	}
}

func (p *PayloadStakeVerifierDeposited) IntoProto() *snapshot.Payload_StakeVerifierDeposited {
	pending := make([]*snapshot.StakeVerifierPending, 0, len(p.StakeVerifierDeposited))

	for _, p := range p.StakeVerifierDeposited {
		pending = append(pending,
			&snapshot.StakeVerifierPending{
				EthereumAddress: crypto.EthereumChecksumAddress(p.EthereumAddress),
				VegaPublicKey:   p.VegaPubKey,
				Amount:          p.Amount.String(),
				BlockTime:       p.BlockTime,
				BlockNumber:     p.BlockNumber,
				LogIndex:        p.LogIndex,
				TxId:            p.TxID,
				Id:              p.ID,
			})
	}

	return &snapshot.Payload_StakeVerifierDeposited{
		StakeVerifierDeposited: &snapshot.StakeVerifierDeposited{
			PendingDeposited: pending,
		},
	}
}

func (*PayloadStakeVerifierDeposited) isPayload() {}

func (p *PayloadStakeVerifierDeposited) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadStakeVerifierDeposited) Key() string {
	return "deposited"
}

func (*PayloadStakeVerifierDeposited) Namespace() SnapshotNamespace {
	return StakeVerifierSnapshot
}

func PayloadEventForwarderFromProto(ef *snapshot.Payload_EventForwarder) *PayloadEventForwarder {
	return &PayloadEventForwarder{
		Keys: ef.EventForwarder.AckedEvents,
	}
}

func (p *PayloadEventForwarder) IntoProto() *snapshot.Payload_EventForwarder {
	return &snapshot.Payload_EventForwarder{
		EventForwarder: &snapshot.EventForwarder{AckedEvents: p.Keys},
	}
}

func (*PayloadEventForwarder) isPayload() {}

func (p *PayloadEventForwarder) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadEventForwarder) Key() string {
	return "all"
}

func (*PayloadEventForwarder) Namespace() SnapshotNamespace {
	return EventForwarderSnapshot
}

func PayloadWitnessFromProto(w *snapshot.Payload_Witness) *PayloadWitness {
	resources := make([]*Resource, 0, len(w.Witness.Resources))
	for _, r := range w.Witness.Resources {
		resources = append(resources, ResourceFromProto(r))
	}
	return &PayloadWitness{
		Witness: &Witness{
			Resources: resources,
		},
	}
}

func ResourceFromProto(r *snapshot.Resource) *Resource {
	return &Resource{
		ID:         r.Id,
		CheckUntil: time.Unix(0, r.CheckUntil).UTC(),
		Votes:      r.Votes,
	}
}

func (p *PayloadWitness) IntoProto() *snapshot.Payload_Witness {
	resources := make([]*snapshot.Resource, 0, len(p.Witness.Resources))
	for _, r := range p.Witness.Resources {
		resources = append(resources, r.IntoProto())
	}
	return &snapshot.Payload_Witness{
		Witness: &snapshot.Witness{
			Resources: resources,
		},
	}
}

func (r *Resource) IntoProto() *snapshot.Resource {
	return &snapshot.Resource{
		Id:         r.ID,
		CheckUntil: r.CheckUntil.UnixNano(),
		Votes:      r.Votes,
	}
}

func (*PayloadWitness) isPayload() {}

func (p *PayloadWitness) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadWitness) Key() string {
	return "all"
}

func (*PayloadWitness) Namespace() SnapshotNamespace {
	return WitnessSnapshot
}

func (*PayloadFloatingPointConsensus) isPayload() {}

func PayloadFloatingPointConsensusFromProto(t *snapshot.Payload_FloatingPointConsensus) *PayloadFloatingPointConsensus {
	return &PayloadFloatingPointConsensus{
		ConsensusData:               t.FloatingPointConsensus.NextTimeTrigger,
		StateVariablesInternalState: t.FloatingPointConsensus.StateVariables,
	}
}

func (p *PayloadFloatingPointConsensus) IntoProto() *snapshot.Payload_FloatingPointConsensus {
	return &snapshot.Payload_FloatingPointConsensus{
		FloatingPointConsensus: &snapshot.FloatingPointConsensus{
			NextTimeTrigger: p.ConsensusData,
			StateVariables:  p.StateVariablesInternalState,
		},
	}
}

func (p *PayloadFloatingPointConsensus) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadFloatingPointConsensus) Key() string {
	return "floatingPointConsensus"
}

func (*PayloadFloatingPointConsensus) Namespace() SnapshotNamespace {
	return FloatingPointConsensusSnapshot
}

func (*PayloadMarketActivityTracker) isPayload() {}

func PayloadMarketActivityTrackerFromProto(t *snapshot.Payload_MarketTracker) *PayloadMarketActivityTracker {
	return &PayloadMarketActivityTracker{
		MarketActivityData: t.MarketTracker,
	}
}

func (p *PayloadMarketActivityTracker) IntoProto() *snapshot.Payload_MarketTracker {
	return &snapshot.Payload_MarketTracker{
		MarketTracker: p.MarketActivityData,
	}
}

func (p *PayloadMarketActivityTracker) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadMarketActivityTracker) Key() string {
	return "marketActivityTracker"
}

func (*PayloadMarketActivityTracker) Namespace() SnapshotNamespace {
	return MarketActivityTrackerSnapshot
}

func (*PayloadTopology) isPayload() {}

func PayloadTopologyFromProto(t *snapshot.Payload_Topology) *PayloadTopology {
	return &PayloadTopology{
		Topology: &Topology{
			ChainValidators:                t.Topology.ChainKeys,
			ValidatorData:                  t.Topology.ValidatorData,
			PendingPubKeyRotations:         t.Topology.PendingPubKeyRotations,
			PendingEthereumKeyRotations:    t.Topology.PendingEthereumKeyRotations,
			UnresolvedEthereumKeyRotations: t.Topology.UnsolvedEthereumKeyRotations,
			Signatures:                     t.Topology.Signatures,
			ValidatorPerformance:           t.Topology.ValidatorPerformance,
		},
	}
}

func (p *PayloadTopology) IntoProto() *snapshot.Payload_Topology {
	return &snapshot.Payload_Topology{
		Topology: &snapshot.Topology{
			ChainKeys:                    p.Topology.ChainValidators,
			ValidatorData:                p.Topology.ValidatorData,
			PendingPubKeyRotations:       p.Topology.PendingPubKeyRotations,
			PendingEthereumKeyRotations:  p.Topology.PendingEthereumKeyRotations,
			UnsolvedEthereumKeyRotations: p.Topology.UnresolvedEthereumKeyRotations,
			Signatures:                   p.Topology.Signatures,
			ValidatorPerformance:         p.Topology.ValidatorPerformance,
		},
	}
}

func (p *PayloadTopology) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadTopology) Key() string {
	return "all"
}

func (*PayloadTopology) Namespace() SnapshotNamespace {
	return TopologySnapshot
}

func (*PayloadLiquiditySupplied) isPayload() {}

func PayloadLiquiditySuppliedFromProto(ls *snapshot.Payload_LiquiditySupplied) *PayloadLiquiditySupplied {
	return &PayloadLiquiditySupplied{
		LiquiditySupplied: ls.LiquiditySupplied,
	}
}

func (p *PayloadLiquiditySupplied) IntoProto() *snapshot.Payload_LiquiditySupplied {
	return &snapshot.Payload_LiquiditySupplied{
		LiquiditySupplied: p.LiquiditySupplied,
	}
}

func (p *PayloadLiquiditySupplied) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadLiquiditySupplied) Key() string {
	return fmt.Sprintf("liquiditySupplied:%v", p.LiquiditySupplied.MarketId)
}

func (*PayloadLiquiditySupplied) Namespace() SnapshotNamespace {
	return LiquiditySnapshot
}

func (*PayloadProofOfWork) isPayload() {}

func PayloadProofOfWorkFromProto(s *snapshot.Payload_ProofOfWork) *PayloadProofOfWork {
	pow := &PayloadProofOfWork{
		BlockHeight:   s.ProofOfWork.BlockHeight,
		BlockHash:     s.ProofOfWork.BlockHash,
		BannedParties: make(map[string]uint64, len(s.ProofOfWork.Banned)),
		HeightToTx:    make(map[uint64][]string, len(s.ProofOfWork.TxAtHeight)),
		HeightToTid:   make(map[uint64][]string, len(s.ProofOfWork.TidAtHeight)),
		ActiveParams:  s.ProofOfWork.PowParams,
	}

	for _, bp := range s.ProofOfWork.Banned {
		pow.BannedParties[bp.Party] = bp.UntilEpoch
	}
	for _, tah := range s.ProofOfWork.TxAtHeight {
		pow.HeightToTx[tah.Height] = tah.Transactions
	}
	for _, tah := range s.ProofOfWork.TidAtHeight {
		pow.HeightToTid[tah.Height] = tah.Transactions
	}
	return pow
}

func (p *PayloadProofOfWork) IntoProto() *snapshot.Payload_ProofOfWork {
	banned := make([]*snapshot.BannedParty, 0, len(p.BannedParties))
	for k, v := range p.BannedParties {
		banned = append(banned, &snapshot.BannedParty{Party: k, UntilEpoch: v})
	}
	sort.Slice(banned, func(i, j int) bool { return banned[i].Party < banned[j].Party })

	txAtHeight := make([]*snapshot.TransactionsAtHeight, 0, len(p.HeightToTx))
	for k, v := range p.HeightToTx {
		txAtHeight = append(txAtHeight, &snapshot.TransactionsAtHeight{Height: k, Transactions: v})
	}
	sort.Slice(txAtHeight, func(i, j int) bool { return txAtHeight[i].Height < txAtHeight[j].Height })

	tidAtHeight := make([]*snapshot.TransactionsAtHeight, 0, len(p.HeightToTid))
	for k, v := range p.HeightToTid {
		tidAtHeight = append(tidAtHeight, &snapshot.TransactionsAtHeight{Height: k, Transactions: v})
	}
	sort.Slice(tidAtHeight, func(i, j int) bool { return tidAtHeight[i].Height < tidAtHeight[j].Height })
	return &snapshot.Payload_ProofOfWork{
		ProofOfWork: &snapshot.ProofOfWork{
			BlockHeight: p.BlockHeight,
			BlockHash:   p.BlockHash,
			Banned:      banned,
			TxAtHeight:  txAtHeight,
			TidAtHeight: tidAtHeight,
			PowParams:   p.ActiveParams,
		},
	}
}

func (p *PayloadProofOfWork) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadProofOfWork) Key() string {
	return "pow"
}

func (*PayloadProofOfWork) Namespace() SnapshotNamespace {
	return PoWSnapshot
}

func PayloadProtocolUpgradeProposalFromProto(pup *snapshot.Payload_ProtocolUpgradeProposals) *PayloadProtocolUpgradeProposals {
	return &PayloadProtocolUpgradeProposals{Proposals: pup.ProtocolUpgradeProposals}
}

func (p *PayloadProtocolUpgradeProposals) IntoProto() *snapshot.Payload_ProtocolUpgradeProposals {
	return &snapshot.Payload_ProtocolUpgradeProposals{
		ProtocolUpgradeProposals: p.Proposals,
	}
}

func (p *PayloadProtocolUpgradeProposals) plToProto() interface{} {
	return p.IntoProto()
}

func (p *PayloadProtocolUpgradeProposals) Key() string {
	return "protocolUpgradeProposals"
}

func (*PayloadProtocolUpgradeProposals) isPayload() {}

func (*PayloadProtocolUpgradeProposals) Namespace() SnapshotNamespace {
	return ProtocolUpgradeSnapshot
}

// KeyFromPayload is useful in snapshot engine, used by the Payload type, too.
func KeyFromPayload(p isPayload) string {
	return GetNodeKey(p.Namespace(), p.Key())
}

// GetNodeKey is a utility function, we don't want this mess scattered throughout the code.
func GetNodeKey(ns SnapshotNamespace, k string) string {
	return strings.Join([]string{
		ns.String(),
		k,
	}, ".")
}
