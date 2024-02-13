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

package gql

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
)

var (
	// ErrMissingIDOrReference is returned when neither id nor reference has been supplied in the query.
	ErrMissingIDOrReference = errors.New("missing id or reference")
	// ErrMissingNodeID is returned when no node id has been supplied in the query.
	ErrMissingNodeID = errors.New("missing node id")
	// ErrInvalidVotesSubscription is returned if neither proposal ID nor party ID is specified.
	ErrInvalidVotesSubscription = errors.New("invalid subscription, either proposal or party ID required")
	// ErrInvalidProposal is returned when invalid governance data is received by proposal resolver.
	ErrInvalidProposal = errors.New("invalid proposal")
	// ErrInvalidStopOrder is returned when an invalid stop order is received by the stop order resolver.
	ErrInvalidStopOrder = errors.New("invalid stop order")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/datanode/gateway/graphql CoreProxyServiceClient,TradingDataServiceClientV2

// CoreProxyServiceClient ...
type CoreProxyServiceClient interface {
	vegaprotoapi.CoreServiceClient
}

type TradingDataServiceClientV2 interface {
	v2.TradingDataServiceClient
}

// VegaResolverRoot is the root resolver for all graphql types.
type VegaResolverRoot struct {
	gateway.Config

	log                 *logging.Logger
	tradingProxyClient  CoreProxyServiceClient
	tradingDataClientV2 TradingDataServiceClientV2
	r                   allResolver
}

// NewResolverRoot instantiate a graphql root resolver.
func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient CoreProxyServiceClient,
	tradingDataClientV2 TradingDataServiceClientV2,
) *VegaResolverRoot {
	return &VegaResolverRoot{
		log:                 log,
		Config:              config,
		tradingProxyClient:  tradingClient,
		tradingDataClientV2: tradingDataClientV2,
		r:                   allResolver{log, tradingDataClientV2},
	}
}

// Query returns the query resolver.
func (r *VegaResolverRoot) Query() QueryResolver {
	return (*myQueryResolver)(r)
}

// Candle returns the candles resolver.
func (r *VegaResolverRoot) Candle() CandleResolver {
	return (*myCandleResolver)(r)
}

func (r *VegaResolverRoot) DataSourceDefinition() DataSourceDefinitionResolver {
	return (*myDataSourceDefinitionResolver)(r)
}

func (r *VegaResolverRoot) DataSourceDefinitionExternal() DataSourceDefinitionExternalResolver {
	return (*myDataSourceDefinitionExternalResolver)(r)
}

func (r *VegaResolverRoot) DataSourceDefinitionInternal() DataSourceDefinitionInternalResolver {
	return (*myDataSourceDefinitionInternalResolver)(r)
}

func (r *VegaResolverRoot) DataSourceSpecConfiguration() DataSourceSpecConfigurationResolver {
	return (*myDataSourceSpecConfigurationResolver)(r)
}

// MarginLevels returns the market levels resolver.
func (r *VegaResolverRoot) MarginLevels() MarginLevelsResolver {
	return (*myMarginLevelsResolver)(r)
}

// MarginLevels returns the market levels resolver.
func (r *VegaResolverRoot) MarginEstimate() MarginEstimateResolver {
	return (*myMarginEstimateResolver)(r)
}

// MarginLevelsUpdate returns the market levels resolver.
func (r *VegaResolverRoot) MarginLevelsUpdate() MarginLevelsUpdateResolver {
	return (*myMarginLevelsUpdateResolver)(r)
}

func (r *VegaResolverRoot) OrderInfo() OrderInfoResolver {
	return (*myOrderInfoResolver)(r)
}

type myOrderInfoResolver VegaResolverRoot

func (r *myOrderInfoResolver) Remaining(ctx context.Context, obj *v2.OrderInfo, data string) error {
	remaining, err := strconv.ParseUint(data, 10, 64)
	if err == nil {
		obj.Remaining = remaining
	}
	return err
}

// PriceLevel returns the price levels resolver.
func (r *VegaResolverRoot) PriceLevel() PriceLevelResolver {
	return (*myPriceLevelResolver)(r)
}

// Market returns the markets resolver.
func (r *VegaResolverRoot) Market() MarketResolver {
	return (*myMarketResolver)(r)
}

// Order returns the order resolver.
func (r *VegaResolverRoot) Order() OrderResolver {
	return (*myOrderResolver)(r)
}

// OrderUpdate returns the order resolver.
func (r *VegaResolverRoot) OrderUpdate() OrderUpdateResolver {
	return (*myOrderUpdateResolver)(r)
}

// Trade returns the trades resolver.
func (r *VegaResolverRoot) Trade() TradeResolver {
	return (*myTradeResolver)(r)
}

// Position returns the positions resolver.
func (r *VegaResolverRoot) Position() PositionResolver {
	return (*myPositionResolver)(r)
}

// PositionUpdate returns the positionUpdate resolver.
func (r *VegaResolverRoot) PositionUpdate() PositionUpdateResolver {
	return (*positionUpdateResolver)(r)
}

// Party returns the parties resolver.
func (r *VegaResolverRoot) Party() PartyResolver {
	return (*myPartyResolver)(r)
}

// Subscription returns the subscriptions resolver.
func (r *VegaResolverRoot) Subscription() SubscriptionResolver {
	return (*mySubscriptionResolver)(r)
}

// Account returns the accounts resolver.
func (r *VegaResolverRoot) AccountEvent() AccountEventResolver {
	return (*myAccountEventResolver)(r)
}

// Account returns the accounts resolver.
func (r *VegaResolverRoot) AccountBalance() AccountBalanceResolver {
	return (*myAccountResolver)(r)
}

// Account returns the accounts resolver.
func (r *VegaResolverRoot) AccountDetails() AccountDetailsResolver {
	return (*myAccountDetailsResolver)(r)
}

// Proposal returns the proposal resolver.
func (r *VegaResolverRoot) Proposal() ProposalResolver {
	return (*proposalResolver)(r)
}

// Proposal returns the proposal resolver.
func (r *VegaResolverRoot) ProposalEdge() ProposalEdgeResolver {
	return (*proposalEdgeResolver)(r)
}

// ProposalDetail returns the Proposal detail resolver.
func (r *VegaResolverRoot) ProposalDetail() ProposalDetailResolver {
	return (*proposalDetailResolver)(r)
}

// NodeSignature ...
func (r *VegaResolverRoot) NodeSignature() NodeSignatureResolver {
	return (*myNodeSignatureResolver)(r)
}

// Asset ...
func (r *VegaResolverRoot) Asset() AssetResolver {
	return (*myAssetResolver)(r)
}

// Deposit ...
func (r *VegaResolverRoot) Deposit() DepositResolver {
	return (*myDepositResolver)(r)
}

// Withdrawal ...
func (r *VegaResolverRoot) Withdrawal() WithdrawalResolver {
	return (*myWithdrawalResolver)(r)
}

func (r *VegaResolverRoot) DataSourceSpecConfigurationTime() DataSourceSpecConfigurationTimeResolver {
	return (*myDataSourceSpecConfigurationTimeResolver)(r)
}

func (r *VegaResolverRoot) DataSourceSpecConfigurationTimeTrigger() DataSourceSpecConfigurationTimeTriggerResolver {
	return (*myDataSourceSpecConfigurationTimeTriggerResolver)(r)
}

func (r *VegaResolverRoot) LiquidityOrderReference() LiquidityOrderReferenceResolver {
	return (*myLiquidityOrderReferenceResolver)(r)
}

func (r *VegaResolverRoot) LiquidityProvision() LiquidityProvisionResolver {
	return (*myLiquidityProvisionResolver)(r)
}

func (r *VegaResolverRoot) Future() FutureResolver {
	return (*myFutureResolver)(r)
}

func (r *VegaResolverRoot) FutureProduct() FutureProductResolver {
	return (*myFutureProductResolver)(r)
}

func (r *VegaResolverRoot) Instrument() InstrumentResolver {
	return (*myInstrumentResolver)(r)
}

func (r *VegaResolverRoot) InstrumentConfiguration() InstrumentConfigurationResolver {
	return (*myInstrumentConfigurationResolver)(r)
}

func (r *VegaResolverRoot) TradableInstrument() TradableInstrumentResolver {
	return (*myTradableInstrumentResolver)(r)
}

func (r *VegaResolverRoot) NewAsset() NewAssetResolver {
	return (*newAssetResolver)(r)
}

func (r *VegaResolverRoot) UpdateAsset() UpdateAssetResolver {
	return (*updateAssetResolver)(r)
}

func (r *VegaResolverRoot) NewMarket() NewMarketResolver {
	return (*newMarketResolver)(r)
}

func (r *VegaResolverRoot) ProposalTerms() ProposalTermsResolver {
	return (*proposalTermsResolver)(r)
}

func (r *VegaResolverRoot) BatchProposalTerms() BatchProposalTermsResolver {
	return (*batchProposalTermsResolver)(r)
}

func (r *VegaResolverRoot) BatchProposalTermsChange() BatchProposalTermsChangeResolver {
	return (*batchProposalTermsChangeResolver)(r)
}

func (r *VegaResolverRoot) UpdateMarket() UpdateMarketResolver {
	return (*updateMarketResolver)(r)
}

func (r *VegaResolverRoot) UpdateNetworkParameter() UpdateNetworkParameterResolver {
	return (*updateNetworkParameterResolver)(r)
}

func (r *VegaResolverRoot) NewFreeform() NewFreeformResolver {
	return (*newFreeformResolver)(r)
}

func (r *VegaResolverRoot) NewTransfer() NewTransferResolver {
	return (*newTransferResolver)(r)
}

func (r *VegaResolverRoot) CancelTransfer() CancelTransferResolver {
	return (*cancelTransferResolver)(r)
}

func (r *VegaResolverRoot) UpdateMarketState() UpdateMarketStateResolver {
	return (*updateMarketStateResolver)(r)
}

func (r *VegaResolverRoot) OracleSpec() OracleSpecResolver {
	return (*oracleSpecResolver)(r)
}

func (r *VegaResolverRoot) OracleData() OracleDataResolver {
	return (*oracleDataResolver)(r)
}

func (r *VegaResolverRoot) AuctionEvent() AuctionEventResolver {
	return (*auctionEventResolver)(r)
}

func (r *VegaResolverRoot) Vote() VoteResolver {
	return (*voteResolver)(r)
}

func (r *VegaResolverRoot) NodeData() NodeDataResolver {
	return (*nodeDataResolver)(r)
}

func (r *VegaResolverRoot) Node() NodeResolver {
	return (*nodeResolver)(r)
}

func (r *VegaResolverRoot) RankingScore() RankingScoreResolver {
	return (*rankingScoreResolver)(r)
}

func (r *VegaResolverRoot) KeyRotation() KeyRotationResolver {
	return (*keyRotationResolver)(r)
}

func (r *VegaResolverRoot) EthereumKeyRotation() EthereumKeyRotationResolver {
	return (*ethereumKeyRotationResolver)(r)
}

func (r *VegaResolverRoot) Delegation() DelegationResolver {
	return (*delegationResolver)(r)
}

func (r *VegaResolverRoot) DateRange() DateRangeResolver {
	return (*dateRangeResolver)(r)
}

func (r *VegaResolverRoot) Epoch() EpochResolver {
	return (*epochResolver)(r)
}

func (r *VegaResolverRoot) EpochTimestamps() EpochTimestampsResolver {
	return (*epochTimestampsResolver)(r)
}

func (r *VegaResolverRoot) EthCallSpec() EthCallSpecResolver {
	return (*ethCallSpecResolver)(r)
}

func (r *VegaResolverRoot) Reward() RewardResolver {
	return (*rewardResolver)(r)
}

func (r *VegaResolverRoot) RewardSummary() RewardSummaryResolver {
	return (*rewardSummaryResolver)(r)
}

func (r *VegaResolverRoot) StakeLinking() StakeLinkingResolver {
	return (*stakeLinkingResolver)(r)
}

func (r *VegaResolverRoot) PartyStake() PartyStakeResolver {
	return (*partyStakeResolver)(r)
}

func (r *VegaResolverRoot) Statistics() StatisticsResolver {
	return (*statisticsResolver)(r)
}

func (r *VegaResolverRoot) Transfer() TransferResolver {
	return (*transferResolver)(r)
}

func (r *VegaResolverRoot) RecurringTransfer() RecurringTransferResolver {
	return (*recurringTransferResolver)(r)
}

func (r *VegaResolverRoot) RecurringGovernanceTransfer() RecurringGovernanceTransferResolver {
	return (*recurringGovernanceTransferResolver)(r)
}

func (r *VegaResolverRoot) UpdateMarketConfiguration() UpdateMarketConfigurationResolver {
	return (*updateMarketConfigurationResolver)(r)
}

func (r *VegaResolverRoot) AccountUpdate() AccountUpdateResolver {
	return (*accountUpdateResolver)(r)
}

func (r *VegaResolverRoot) TradeUpdate() TradeUpdateResolver {
	return (*tradeUpdateResolver)(r)
}

func (r *VegaResolverRoot) LiquidityProvisionUpdate() LiquidityProvisionUpdateResolver {
	return (*liquidityProvisionUpdateResolver)(r)
}

func (r *VegaResolverRoot) TransactionResult() TransactionResultResolver {
	return (*transactionResultResolver)(r)
}

func (r *VegaResolverRoot) ProtocolUpgradeProposal() ProtocolUpgradeProposalResolver {
	return (*protocolUpgradeProposalResolver)(r)
}

func (r *VegaResolverRoot) CoreSnapshotData() CoreSnapshotDataResolver {
	return (*coreDataSnapshotResolver)(r)
}

func (r *VegaResolverRoot) EpochRewardSummary() EpochRewardSummaryResolver {
	return (*epochRewardSummaryResolver)(r)
}

func (r *VegaResolverRoot) OrderFilter() OrderFilterResolver {
	return (*orderFilterResolver)(r)
}

func (r *VegaResolverRoot) LedgerEntry() LedgerEntryResolver {
	return (*ledgerEntryResolver)(r)
}

func (r *VegaResolverRoot) ERC20MultiSigSignerAddedBundle() ERC20MultiSigSignerAddedBundleResolver {
	return (*erc20MultiSigSignerAddedBundleResolver)(r)
}

func (r *VegaResolverRoot) ERC20MultiSigSignerRemovedBundle() ERC20MultiSigSignerRemovedBundleResolver {
	return (*erc20MultiSigSignerRemovedBundleResolver)(r)
}

func (r *VegaResolverRoot) IcebergOrder() IcebergOrderResolver {
	return (*icebergOrderResolver)(r)
}

func (r *VegaResolverRoot) OrderSubmission() OrderSubmissionResolver {
	return (*orderSubmissionResolver)(r)
}

func (r *VegaResolverRoot) StopOrder() StopOrderResolver {
	return (*stopOrderResolver)(r)
}

func (r *VegaResolverRoot) StopOrderFilter() StopOrderFilterResolver {
	return (*stopOrderFilterResolver)(r)
}

// RewardSummaryFilter returns RewardSummaryFilterResolver implementation.
func (r *VegaResolverRoot) RewardSummaryFilter() RewardSummaryFilterResolver {
	return (*rewardSummaryFilterResolver)(r)
}

func (r *VegaResolverRoot) LiquidityProvider() LiquidityProviderResolver {
	return (*liquidityProviderResolver)(r)
}

func (r *VegaResolverRoot) FundingPeriod() FundingPeriodResolver {
	return (*fundingPeriodResolver)(r)
}

func (r *VegaResolverRoot) FundingPeriodDataPoint() FundingPeriodDataPointResolver {
	return (*fundingPeriodDataPointResolver)(r)
}

func (r *VegaResolverRoot) Perpetual() PerpetualResolver {
	return (*perpetualResolver)(r)
}

func (r *VegaResolverRoot) PerpetualProduct() PerpetualProductResolver {
	return (*perpetualProductResolver)(r)
}

func (r *VegaResolverRoot) Spot() SpotResolver {
	return (*spotResolver)(r)
}

func (r *VegaResolverRoot) SpotProduct() SpotProductResolver {
	return (*spotProductResolver)(r)
}

func (r *VegaResolverRoot) LiquiditySLAParameters() LiquiditySLAParametersResolver {
	return (*liquiditySLAParametersResolver)(r)
}

func (r *VegaResolverRoot) LiquidationStrategy() LiquidationStrategyResolver {
	return (*liquidationStrategyResolver)(r)
}

func (r *VegaResolverRoot) CompositePriceConfiguration() CompositePriceConfigurationResolver {
	return (*compositePriceConfigurationResolver)(r)
}

func (r *VegaResolverRoot) NewSpotMarket() NewSpotMarketResolver {
	return (*newSpotMarketResolver)(r)
}

func (r *VegaResolverRoot) UpdateSpotMarket() UpdateSpotMarketResolver {
	return (*updateSpotMarketResolver)(r)
}

func (r *VegaResolverRoot) UpdateSpotMarketConfiguration() UpdateSpotMarketConfigurationResolver {
	return (*updateSpotMarketConfigurationResolver)(r)
}

func (r *VegaResolverRoot) PartyActivityStreak() PartyActivityStreakResolver {
	return (*partyActivityStreakResolver)(r)
}

func (r *VegaResolverRoot) ReferralProgram() ReferralProgramResolver {
	return (*referralProgramResolver)(r)
}

func (r *VegaResolverRoot) CurrentReferralProgram() CurrentReferralProgramResolver {
	return (*currentReferralProgramResolver)(r)
}

func (r *VegaResolverRoot) ReferralSet() ReferralSetResolver {
	return (*referralSetResolver)(r)
}

func (r *VegaResolverRoot) ReferralSetReferee() ReferralSetRefereeResolver {
	return (*referralSetRefereeResolver)(r)
}

func (r *VegaResolverRoot) ReferralSetStats() ReferralSetStatsResolver {
	return (*referralSetStatsResolver)(r)
}

func (r *VegaResolverRoot) BenefitTier() BenefitTierResolver {
	return (*benefitTierResolver)(r)
}

func (r *VegaResolverRoot) Team() TeamResolver {
	return (*teamResolver)(r)
}

func (r *VegaResolverRoot) TeamStatistics() TeamStatisticsResolver {
	return (*teamStatsResolver)(r)
}

func (r *VegaResolverRoot) QuantumRewardsPerEpoch() QuantumRewardsPerEpochResolver {
	return (*quantumRewardsPerEpochResolver)(r)
}

func (r *VegaResolverRoot) QuantumVolumesPerEpoch() QuantumVolumesPerEpochResolver {
	return (*quantumVolumesPerEpochResolver)(r)
}

func (r *VegaResolverRoot) TeamMemberStatistics() TeamMemberStatisticsResolver {
	return (*teamMemberStatsResolver)(r)
}

func (r *VegaResolverRoot) TeamReferee() TeamRefereeResolver {
	return (*teamRefereeResolver)(r)
}

func (r *VegaResolverRoot) TeamRefereeHistory() TeamRefereeHistoryResolver {
	return (*teamRefereeHistoryResolver)(r)
}

func (r *VegaResolverRoot) PartyAmount() PartyAmountResolver {
	return (*partyAmountResolver)(r)
}

func (r *VegaResolverRoot) FeesStats() FeesStatsResolver {
	return (*feesStatsResolver)(r)
}

func (r *VegaResolverRoot) ReferrerRewardsGenerated() ReferrerRewardsGeneratedResolver {
	return (*referrerRewardsGeneratedResolver)(r)
}

func (r *VegaResolverRoot) FundingPayment() FundingPaymentResolver {
	return (*fundingPaymentResolver)(r)
}

func (r *VegaResolverRoot) VolumeDiscountProgram() VolumeDiscountProgramResolver {
	return (*volumeDiscountProgramResolver)(r)
}

func (r *VegaResolverRoot) VolumeDiscountStats() VolumeDiscountStatsResolver {
	return (*volumeDiscountStatsResolver)(r)
}

func (r *VegaResolverRoot) UpdateVolumeDiscountProgram() UpdateVolumeDiscountProgramResolver {
	return (*updateVolumeDiscountProgramResolver)(r)
}

func (r *VegaResolverRoot) UpdateReferralProgram() UpdateReferralProgramResolver {
	return (*updateReferralProgramResolver)(r)
}

func (r *VegaResolverRoot) PaidLiquidityFees() PaidLiquidityFeesResolver {
	return (*paidLiquidityFeesResolver)(r)
}

func (r *VegaResolverRoot) PartyLockedBalance() PartyLockedBalanceResolver {
	return (*partyLockedBalanceResolver)(r)
}

func (r *VegaResolverRoot) PartyVestingBalance() PartyVestingBalanceResolver {
	return (*partyVestingBalanceResolver)(r)
}

func (r *VegaResolverRoot) PartyVestingBalancesSummary() PartyVestingBalancesSummaryResolver {
	return (*partyVestingBalancesSummary)(r)
}

func (r *VegaResolverRoot) TransferNode() TransferNodeResolver {
	return (*transferNodeResolver)(r)
}

func (r *VegaResolverRoot) PartyVestingStats() PartyVestingStatsResolver {
	return (*partyVestingStatsResolver)(r)
}

func (r *VegaResolverRoot) DispatchStrategy() DispatchStrategyResolver {
	return (*dispatchStrategyResolver)(r)
}

func (r *VegaResolverRoot) Game() GameResolver {
	return (*gameResolver)(r)
}

func (r *VegaResolverRoot) PartyMarginMode() PartyMarginModeResolver {
	return (*marginModeResolver)(r)
}

func (r *VegaResolverRoot) PerpetualData() PerpetualDataResolver {
	return (*perpetualDataResolver)(r)
}

type protocolUpgradeProposalResolver VegaResolverRoot

func (r *protocolUpgradeProposalResolver) UpgradeBlockHeight(_ context.Context, obj *eventspb.ProtocolUpgradeEvent) (string, error) {
	return fmt.Sprintf("%d", obj.UpgradeBlockHeight), nil
}

type coreDataSnapshotResolver VegaResolverRoot

func (r *coreDataSnapshotResolver) BlockHeight(_ context.Context, obj *eventspb.CoreSnapshotData) (string, error) {
	return fmt.Sprintf("%d", obj.BlockHeight), nil
}

func (r *coreDataSnapshotResolver) VegaCoreVersion(ctx context.Context, obj *eventspb.CoreSnapshotData) (string, error) {
	return obj.CoreVersion, nil
}

type epochRewardSummaryResolver VegaResolverRoot

func (r *epochRewardSummaryResolver) RewardType(_ context.Context, obj *vega.EpochRewardSummary) (vega.AccountType, error) {
	accountType, ok := vega.AccountType_value[obj.RewardType]
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("unknown account type %v", obj.RewardType)
	}

	return vega.AccountType(accountType), nil
}

func (r *epochRewardSummaryResolver) Epoch(_ context.Context, obj *vega.EpochRewardSummary) (int, error) {
	return int(obj.Epoch), nil
}

type transactionResultResolver VegaResolverRoot

func (r *transactionResultResolver) Error(ctx context.Context, tr *eventspb.TransactionResult) (*string, error) {
	if tr == nil || tr.Status {
		return nil, nil
	}

	return &tr.GetFailure().Error, nil
}

type accountUpdateResolver VegaResolverRoot

func (r *accountUpdateResolver) AssetID(_ context.Context, obj *v2.AccountBalance) (string, error) {
	return obj.Asset, nil
}

func (r *accountUpdateResolver) PartyID(_ context.Context, obj *v2.AccountBalance) (string, error) {
	return obj.Owner, nil
}

// AggregatedLedgerEntriesResolver resolver.
type aggregatedLedgerEntriesResolver VegaResolverRoot

func (r *VegaResolverRoot) AggregatedLedgerEntry() AggregatedLedgerEntryResolver {
	return (*aggregatedLedgerEntriesResolver)(r)
}

func (r *aggregatedLedgerEntriesResolver) VegaTime(_ context.Context, obj *v2.AggregatedLedgerEntry) (int64, error) {
	return obj.Timestamp, nil
}

// LiquidityOrderReference resolver.

type myLiquidityOrderReferenceResolver VegaResolverRoot

func (r *myLiquidityOrderReferenceResolver) Order(ctx context.Context, obj *vegapb.LiquidityOrderReference) (*vegapb.Order, error) {
	if len(obj.OrderId) <= 0 {
		return nil, nil
	}
	return r.r.getOrderByID(ctx, obj.OrderId, nil)
}

// deposit resolver

type myDepositResolver VegaResolverRoot

func (r *myDepositResolver) Asset(ctx context.Context, obj *vegapb.Deposit) (*vegapb.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *myDepositResolver) Party(_ context.Context, obj *vegapb.Deposit) (*vegapb.Party, error) {
	if len(obj.PartyId) <= 0 {
		return nil, errors.New("missing party ID")
	}
	return &vegapb.Party{Id: obj.PartyId}, nil
}

func (r *myDepositResolver) CreatedTimestamp(_ context.Context, obj *vegapb.Deposit) (string, error) {
	if obj.CreatedTimestamp == 0 {
		return "", errors.New("invalid timestamp")
	}
	return vegatime.Format(vegatime.UnixNano(obj.CreatedTimestamp)), nil
}

func (r *myDepositResolver) CreditedTimestamp(_ context.Context, obj *vegapb.Deposit) (*string, error) {
	if obj.CreditedTimestamp == 0 {
		return nil, nil
	}
	t := vegatime.Format(vegatime.UnixNano(obj.CreditedTimestamp))
	return &t, nil
}

// BEGIN: Query Resolver

type myQueryResolver VegaResolverRoot

func (r *myQueryResolver) PartiesProfilesConnection(ctx context.Context, ids []string, pagination *v2.Pagination) (*v2.PartiesProfilesConnection, error) {
	req := v2.ListPartiesProfilesRequest{
		Parties:    ids,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListPartiesProfiles(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.Profiles, nil
}

func (r *myQueryResolver) PartyMarginModes(ctx context.Context, marketID *string, partyID *string, pagination *v2.Pagination) (*v2.PartyMarginModesConnection, error) {
	req := v2.ListPartyMarginModesRequest{
		MarketId:   marketID,
		PartyId:    partyID,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListPartyMarginModes(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.PartyMarginModes, nil
}

func (r *myQueryResolver) Games(ctx context.Context, gameID *string, epochFrom *int, epochTo *int, entityScope *vega.EntityScope, teamID *string, partyID *string, pagination *v2.Pagination) (*v2.GamesConnection, error) {
	var from *uint64
	var to *uint64

	if epochFrom != nil {
		from = ptr.From(uint64(*epochFrom))
	}

	if epochTo != nil {
		to = ptr.From(uint64(*epochTo))
	}

	req := v2.ListGamesRequest{
		GameId:      gameID,
		EpochFrom:   from,
		EpochTo:     to,
		EntityScope: entityScope,
		Pagination:  pagination,
		TeamId:      teamID,
		PartyId:     partyID,
	}
	res, err := r.tradingDataClientV2.ListGames(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.Games, nil
}

func (r *myQueryResolver) FundingPayments(
	ctx context.Context,
	partyID string,
	marketID *string,
	pagination *v2.Pagination,
) (*v2.FundingPaymentConnection, error) {
	req := v2.ListFundingPaymentsRequest{
		PartyId:    partyID,
		MarketId:   marketID,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListFundingPayments(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.FundingPayments, nil
}

func (r *myQueryResolver) FundingPeriods(ctx context.Context, marketID string, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.FundingPeriodConnection, error) {
	req := v2.ListFundingPeriodsRequest{
		MarketId:   marketID,
		DateRange:  dateRange,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListFundingPeriods(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.FundingPeriods, nil
}

func (r *myQueryResolver) FundingPeriodDataPoints(ctx context.Context, marketID string, dateRange *v2.DateRange, source *v1.FundingPeriodDataPoint_Source, pagination *v2.Pagination) (*v2.FundingPeriodDataPointConnection, error) {
	req := &v2.ListFundingPeriodDataPointsRequest{
		MarketId:   marketID,
		DateRange:  dateRange,
		Source:     source,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListFundingPeriodDataPoints(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.FundingPeriodDataPoints, nil
}

func (r *myQueryResolver) Trades(ctx context.Context, filter *TradesFilter, pagination *v2.Pagination, dateRange *v2.DateRange) (*v2.TradeConnection, error) {
	if filter == nil {
		filter = &TradesFilter{}
	}

	resp, err := r.tradingDataClientV2.ListTrades(ctx, &v2.ListTradesRequest{
		MarketIds:  filter.MarketIds,
		OrderIds:   filter.OrderIds,
		PartyIds:   filter.PartyIds,
		Pagination: pagination,
		DateRange:  dateRange,
	})
	if err != nil {
		return nil, err
	}

	return resp.Trades, nil
}

func (r *myQueryResolver) Positions(ctx context.Context, filter *v2.PositionsFilter, pagination *v2.Pagination) (*v2.PositionConnection, error) {
	resp, err := r.tradingDataClientV2.ListAllPositions(ctx, &v2.ListAllPositionsRequest{
		Filter:     filter,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Positions, nil
}

func (r *myQueryResolver) TransfersConnection(
	ctx context.Context,
	partyID *string,
	direction *TransferDirection,
	pagination *v2.Pagination,
	isReward *bool,
	fromEpoch *int,
	toEpoch *int,
	status *eventspb.Transfer_Status,
	scope *v2.ListTransfersRequest_Scope,
	gameID *string,
) (*v2.TransferConnection, error) {
	return r.r.transfersConnection(ctx, partyID, direction, pagination, isReward, fromEpoch, toEpoch, status, scope, gameID)
}

func (r *myQueryResolver) Transfer(ctx context.Context, id string) (*v2.TransferNode, error) {
	req := v2.GetTransferRequest{
		TransferId: id,
	}
	resp, err := r.tradingDataClientV2.GetTransfer(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp.TransferNode, nil
}

func (r *myQueryResolver) LastBlockHeight(ctx context.Context) (string, error) {
	resp, err := r.tradingProxyClient.LastBlockHeight(ctx, &vegaprotoapi.LastBlockHeightRequest{})
	if err != nil {
		return "0", err
	}

	return strconv.FormatUint(resp.Height, 10), nil
}

func (r *myQueryResolver) OracleSpecsConnection(ctx context.Context, pagination *v2.Pagination) (*v2.OracleSpecsConnection, error) {
	req := v2.ListOracleSpecsRequest{
		Pagination: pagination,
	}
	res, err := r.tradingDataClientV2.ListOracleSpecs(ctx, &req)
	if err != nil {
		return nil, err
	}

	return res.OracleSpecs, nil
}

func (r *myQueryResolver) OracleSpec(ctx context.Context, id string) (*vegapb.OracleSpec, error) {
	res, err := r.tradingDataClientV2.GetOracleSpec(
		ctx, &v2.GetOracleSpecRequest{OracleSpecId: id},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleSpec, nil
}

func (r *myQueryResolver) OracleDataBySpecConnection(ctx context.Context, oracleSpecID string,
	pagination *v2.Pagination,
) (*v2.OracleDataConnection, error) {
	var specID *string
	if oracleSpecID != "" {
		specID = &oracleSpecID
	}
	req := v2.ListOracleDataRequest{
		OracleSpecId: specID,
		Pagination:   pagination,
	}

	resp, err := r.tradingDataClientV2.ListOracleData(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.OracleData, nil
}

func (r *myQueryResolver) OracleDataConnection(ctx context.Context, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
	req := v2.ListOracleDataRequest{
		Pagination: pagination,
	}

	resp, err := r.tradingDataClientV2.ListOracleData(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.OracleData, nil
}

func (r *myQueryResolver) NetworkParametersConnection(ctx context.Context, pagination *v2.Pagination) (*v2.NetworkParameterConnection, error) {
	res, err := r.tradingDataClientV2.ListNetworkParameters(ctx, &v2.ListNetworkParametersRequest{
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}
	return res.NetworkParameters, nil
}

func (r *myQueryResolver) NetworkParameter(ctx context.Context, key string) (*vegapb.NetworkParameter, error) {
	res, err := r.tradingDataClientV2.GetNetworkParameter(
		ctx, &v2.GetNetworkParameterRequest{Key: key},
	)
	if err != nil {
		return nil, err
	}

	return res.NetworkParameter, nil
}

func (r *myQueryResolver) Erc20WithdrawalApproval(ctx context.Context, wid string) (*Erc20WithdrawalApproval, error) {
	res, err := r.tradingDataClientV2.GetERC20WithdrawalApproval(
		ctx, &v2.GetERC20WithdrawalApprovalRequest{WithdrawalId: wid},
	)
	if err != nil {
		return nil, err
	}

	return &Erc20WithdrawalApproval{
		AssetSource:   res.AssetSource,
		Amount:        res.Amount,
		Nonce:         res.Nonce,
		Signatures:    res.Signatures,
		TargetAddress: res.TargetAddress,
		Creation:      fmt.Sprintf("%d", res.Creation),
	}, nil
}

func (r *myQueryResolver) Erc20ListAssetBundle(ctx context.Context, assetID string) (*Erc20ListAssetBundle, error) {
	res, err := r.tradingDataClientV2.GetERC20ListAssetBundle(
		ctx, &v2.GetERC20ListAssetBundleRequest{AssetId: assetID})
	if err != nil {
		return nil, err
	}

	return &Erc20ListAssetBundle{
		AssetSource: res.AssetSource,
		VegaAssetID: res.VegaAssetId,
		Nonce:       res.Nonce,
		Signatures:  res.Signatures,
	}, nil
}

func (r *myQueryResolver) Erc20SetAssetLimitsBundle(ctx context.Context, proposalID string) (*ERC20SetAssetLimitsBundle, error) {
	res, err := r.tradingDataClientV2.GetERC20SetAssetLimitsBundle(
		ctx, &v2.GetERC20SetAssetLimitsBundleRequest{ProposalId: proposalID})
	if err != nil {
		return nil, err
	}

	return &ERC20SetAssetLimitsBundle{
		AssetSource:   res.AssetSource,
		VegaAssetID:   res.VegaAssetId,
		Nonce:         res.Nonce,
		LifetimeLimit: res.LifetimeLimit,
		Threshold:     res.Threshold,
		Signatures:    res.Signatures,
	}, nil
}

func (r *myQueryResolver) Erc20MultiSigSignerAddedBundles(ctx context.Context, nodeID string, submitter, epochSeq *string, pagination *v2.Pagination) (*ERC20MultiSigSignerAddedConnection, error) {
	res, err := r.tradingDataClientV2.ListERC20MultiSigSignerAddedBundles(
		ctx, &v2.ListERC20MultiSigSignerAddedBundlesRequest{
			NodeId:     nodeID,
			Submitter:  ptr.UnBox(submitter),
			EpochSeq:   ptr.UnBox(epochSeq),
			Pagination: pagination,
		})
	if err != nil {
		return nil, err
	}

	edges := make([]*ERC20MultiSigSignerAddedBundleEdge, 0, len(res.Bundles.Edges))

	for _, edge := range res.Bundles.Edges {
		edges = append(edges, &ERC20MultiSigSignerAddedBundleEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
	}

	return &ERC20MultiSigSignerAddedConnection{
		Edges:    edges,
		PageInfo: res.Bundles.PageInfo,
	}, nil
}

func (r *myQueryResolver) Erc20MultiSigSignerRemovedBundles(ctx context.Context, nodeID string, submitter, epochSeq *string, pagination *v2.Pagination) (*ERC20MultiSigSignerRemovedConnection, error) {
	res, err := r.tradingDataClientV2.ListERC20MultiSigSignerRemovedBundles(
		ctx, &v2.ListERC20MultiSigSignerRemovedBundlesRequest{
			NodeId:     nodeID,
			Submitter:  ptr.UnBox(submitter),
			EpochSeq:   ptr.UnBox(epochSeq),
			Pagination: pagination,
		})
	if err != nil {
		return nil, err
	}

	edges := make([]*ERC20MultiSigSignerRemovedBundleEdge, 0, len(res.Bundles.Edges))

	for _, edge := range res.Bundles.Edges {
		edges = append(edges, &ERC20MultiSigSignerRemovedBundleEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
	}

	return &ERC20MultiSigSignerRemovedConnection{
		Edges:    edges,
		PageInfo: res.Bundles.PageInfo,
	}, nil
}

func (r *myQueryResolver) Withdrawal(ctx context.Context, wid string) (*vegapb.Withdrawal, error) {
	res, err := r.tradingDataClientV2.GetWithdrawal(
		ctx, &v2.GetWithdrawalRequest{Id: wid},
	)
	if err != nil {
		return nil, err
	}

	return res.Withdrawal, nil
}

func (r *myQueryResolver) Withdrawals(ctx context.Context, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.WithdrawalsConnection, error) {
	res, err := r.tradingDataClientV2.ListWithdrawals(
		ctx, &v2.ListWithdrawalsRequest{
			DateRange:  dateRange,
			Pagination: pagination,
		},
	)
	if err != nil {
		return nil, err
	}

	return res.Withdrawals, nil
}

func (r *myQueryResolver) Deposit(ctx context.Context, did string) (*vegapb.Deposit, error) {
	res, err := r.tradingDataClientV2.GetDeposit(
		ctx, &v2.GetDepositRequest{Id: did},
	)
	if err != nil {
		return nil, err
	}

	return res.Deposit, nil
}

func (r *myQueryResolver) Deposits(ctx context.Context, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.DepositsConnection, error) {
	res, err := r.tradingDataClientV2.ListDeposits(
		ctx, &v2.ListDepositsRequest{DateRange: dateRange, Pagination: pagination},
	)
	if err != nil {
		return nil, err
	}

	return res.Deposits, nil
}

func (r *myQueryResolver) EstimateOrder(
	ctx context.Context,
	market, party string,
	price *string,
	size string,
	side vega.Side,
	timeInForce vega.Order_TimeInForce,
	expiration *int64,
	ty vega.Order_Type,
) (*OrderEstimate, error) {
	order := &vegapb.Order{}
	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	if price != nil {
		order.Price = *price
	}
	s, err := safeStringUint64(size)
	if err != nil {
		return nil, err
	}
	order.Size = s
	if len(market) <= 0 {
		return nil, errors.New("market missing or empty")
	}
	order.MarketId = market
	if len(party) <= 0 {
		return nil, errors.New("party missing or empty")
	}

	order.PartyId = party
	order.TimeInForce = timeInForce
	order.Side = side
	order.Type = ty

	// GTT must have an expiration value
	if order.TimeInForce == vegapb.Order_TIME_IN_FORCE_GTT && expiration != nil {
		order.ExpiresAt = vegatime.UnixNano(*expiration).UnixNano()
	}

	req := v2.EstimateFeeRequest{
		MarketId: order.MarketId,
		Price:    order.Price,
		Size:     order.Size,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	resp, err := r.tradingDataClientV2.EstimateFee(ctx, &req)
	if err != nil {
		r.log.Error("Failed to get fee estimates using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	// calclate the fee total amount
	var mfee, ifee, lfee num.Decimal
	// errors doesn't matter here, they just give us zero values anyway for the decimals
	if len(resp.Fee.MakerFee) > 0 {
		mfee, _ = num.DecimalFromString(resp.Fee.MakerFee)
	}
	if len(resp.Fee.InfrastructureFee) > 0 {
		ifee, _ = num.DecimalFromString(resp.Fee.InfrastructureFee)
	}
	if len(resp.Fee.LiquidityFee) > 0 {
		lfee, _ = num.DecimalFromString(resp.Fee.LiquidityFee)
	}

	fee := TradeFee{
		MakerFee:          resp.Fee.MakerFee,
		InfrastructureFee: resp.Fee.InfrastructureFee,
		LiquidityFee:      resp.Fee.LiquidityFee,
	}

	// now we calculate the margins
	reqm := v2.EstimateMarginRequest{
		MarketId: order.MarketId,
		PartyId:  order.PartyId,
		Price:    order.Price,
		Size:     order.Size,
		Side:     order.Side,
		Type:     order.Type,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	respm, err := r.tradingDataClientV2.EstimateMargin(ctx, &reqm)
	if err != nil {
		r.log.Error("Failed to get margin estimates using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	return &OrderEstimate{
		Fee:            &fee,
		TotalFeeAmount: decimal.Sum(mfee, ifee, lfee).String(),
		MarginLevels:   respm.MarginLevels,
	}, nil
}

func (r *myQueryResolver) EstimateFees(
	ctx context.Context,
	market, party string,
	price *string,
	size string,
	side vega.Side,
	timeInForce vega.Order_TimeInForce,
	expiration *int64,
	ty vega.Order_Type,
) (*FeeEstimate, error) {
	order := &vegapb.Order{}
	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	if price != nil {
		order.Price = *price
	}
	s, err := safeStringUint64(size)
	if err != nil {
		return nil, err
	}
	order.Size = s
	if len(market) <= 0 {
		return nil, errors.New("market missing or empty")
	}
	order.MarketId = market
	if len(party) <= 0 {
		return nil, errors.New("party missing or empty")
	}

	order.PartyId = party
	order.TimeInForce = timeInForce
	order.Side = side
	order.Type = ty

	// GTT must have an expiration value
	if order.TimeInForce == vegapb.Order_TIME_IN_FORCE_GTT && expiration != nil {
		order.ExpiresAt = vegatime.UnixNano(*expiration).UnixNano()
	}

	req := v2.EstimateFeeRequest{
		MarketId: order.MarketId,
		Price:    order.Price,
		Size:     order.Size,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	resp, err := r.tradingDataClientV2.EstimateFee(ctx, &req)
	if err != nil {
		r.log.Error("Failed to get fee estimates using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	// calclate the fee total amount
	var mfee, ifee, lfee num.Decimal
	// errors doesn't matter here, they just give us zero values anyway for the decimals
	if len(resp.Fee.MakerFee) > 0 {
		mfee, _ = num.DecimalFromString(resp.Fee.MakerFee)
	}
	if len(resp.Fee.InfrastructureFee) > 0 {
		ifee, _ = num.DecimalFromString(resp.Fee.InfrastructureFee)
	}
	if len(resp.Fee.LiquidityFee) > 0 {
		lfee, _ = num.DecimalFromString(resp.Fee.LiquidityFee)
	}

	fees := &TradeFee{
		MakerFee:          resp.Fee.MakerFee,
		InfrastructureFee: resp.Fee.InfrastructureFee,
		LiquidityFee:      resp.Fee.LiquidityFee,
	}

	return &FeeEstimate{
		Fees:           fees,
		TotalFeeAmount: decimal.Sum(mfee, ifee, lfee).String(),
	}, nil
}

func (r *myQueryResolver) EstimatePosition(
	ctx context.Context,
	marketId string,
	openVolume string,
	averageEntryPrice string,
	orders []*v2.OrderInfo,
	marginAccountBalance string,
	generalAccountBalance string,
	orderMarginAccountBalance string,
	marginMode vega.MarginMode,
	marginFactor *string,
	incluedRequiredPositionMarginInAvailableCollateral *bool,
	scaleLiquidationPriceToMarketDecimals *bool,
) (*PositionEstimate, error) {
	ov, err := safeStringInt64(openVolume)
	if err != nil {
		return nil, err
	}

	req := &v2.EstimatePositionRequest{
		MarketId:                  marketId,
		OpenVolume:                ov,
		AverageEntryPrice:         averageEntryPrice,
		Orders:                    orders,
		MarginAccountBalance:      marginAccountBalance,
		GeneralAccountBalance:     generalAccountBalance,
		OrderMarginAccountBalance: orderMarginAccountBalance,
		MarginMode:                vegapb.MarginMode(vega.MarginMode_value[marginMode.String()]),
		MarginFactor:              marginFactor,
		IncludeRequiredPositionMarginInAvailableCollateral: incluedRequiredPositionMarginInAvailableCollateral,
		ScaleLiquidationPriceToMarketDecimals:              scaleLiquidationPriceToMarketDecimals,
	}

	resp, err := r.tradingDataClientV2.EstimatePosition(ctx, req)
	if err != nil {
		return nil, err
	}

	return &PositionEstimate{
		Margin:                     resp.Margin,
		CollateralIncreaseEstimate: resp.CollateralIncreaseEstimate,
		Liquidation:                resp.Liquidation,
	}, nil
}

func (r *myQueryResolver) Asset(ctx context.Context, id string) (*vegapb.Asset, error) {
	return r.r.getAssetByID(ctx, id)
}

func (r *myQueryResolver) AssetsConnection(ctx context.Context, id *string, pagination *v2.Pagination) (*v2.AssetsConnection, error) {
	req := &v2.ListAssetsRequest{
		AssetId:    id,
		Pagination: pagination,
	}
	resp, err := r.tradingDataClientV2.ListAssets(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Assets, nil
}

func (r *myQueryResolver) NodeSignaturesConnection(ctx context.Context, resourceID string, pagination *v2.Pagination) (*v2.NodeSignaturesConnection, error) {
	if len(resourceID) <= 0 {
		return nil, ErrMissingIDOrReference
	}

	req := &v2.ListNodeSignaturesRequest{
		Id:         resourceID,
		Pagination: pagination,
	}
	res, err := r.tradingDataClientV2.ListNodeSignatures(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Signatures, nil
}

func (r *myQueryResolver) Market(ctx context.Context, id string) (*vegapb.Market, error) {
	return r.r.getMarketByID(ctx, id)
}

func (r *myQueryResolver) Party(ctx context.Context, name string) (*vegapb.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, name)
}

func (r *myQueryResolver) OrderByID(ctx context.Context, orderID string, version *int) (*vegapb.Order, error) {
	return r.r.getOrderByID(ctx, orderID, version)
}

func (r *myQueryResolver) OrderVersionsConnection(ctx context.Context, orderID *string, pagination *v2.Pagination) (*v2.OrderConnection, error) {
	if orderID == nil {
		return nil, ErrMissingIDOrReference
	}
	req := &v2.ListOrderVersionsRequest{
		OrderId:    *orderID,
		Pagination: pagination,
	}

	resp, err := r.tradingDataClientV2.ListOrderVersions(ctx, req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return resp.Orders, nil
}

func (r *myQueryResolver) OrderByReference(ctx context.Context, reference string) (*vegapb.Order, error) {
	req := &v2.ListOrdersRequest{
		Filter: &v2.OrderFilter{
			Reference: &reference,
		},
	}
	res, err := r.tradingDataClientV2.ListOrders(ctx, req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	if len(res.Orders.Edges) == 0 {
		return nil, fmt.Errorf("order reference not found: %s", reference)
	}

	return res.Orders.Edges[0].Node, nil
}

func (r *myQueryResolver) ProposalsConnection(ctx context.Context, proposalType *v2.ListGovernanceDataRequest_Type, inState *vega.Proposal_State,
	pagination *v2.Pagination,
) (*v2.GovernanceDataConnection, error) {
	return handleProposalsRequest(ctx, r.tradingDataClientV2, nil, nil, proposalType, inState, pagination)
}

func (r *myQueryResolver) Proposal(ctx context.Context, id *string, reference *string) (ProposalNode, error) {
	if id != nil {
		resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
			ProposalId: id,
		})
		if err != nil {
			return nil, err
		}

		resolver := (*proposalEdgeResolver)(r)
		if resp.GetData().ProposalType == vega.GovernanceData_TYPE_BATCH {
			return resolver.BatchProposal(ctx, resp.GetData())
		}

		return resp.Data, nil
	} else if reference != nil {
		resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
			Reference: reference,
		})
		if err != nil {
			return nil, err
		}

		resolver := (*proposalEdgeResolver)(r)
		if resp.GetData().ProposalType == vega.GovernanceData_TYPE_BATCH {
			return resolver.BatchProposal(ctx, resp.GetData())
		}

		return resp.Data, nil
	}

	return nil, ErrMissingIDOrReference
}

func (r *myQueryResolver) ProtocolUpgradeStatus(ctx context.Context) (*ProtocolUpgradeStatus, error) {
	status, err := r.tradingDataClientV2.GetProtocolUpgradeStatus(ctx, &v2.GetProtocolUpgradeStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &ProtocolUpgradeStatus{
		Ready: status.Ready,
	}, nil
}

func (r *myQueryResolver) CoreSnapshots(ctx context.Context, pagination *v2.Pagination) (*v2.CoreSnapshotConnection, error) {
	req := v2.ListCoreSnapshotsRequest{Pagination: pagination}
	resp, err := r.tradingDataClientV2.ListCoreSnapshots(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.CoreSnapshots, nil
}

func (r *myQueryResolver) EpochRewardSummaries(
	ctx context.Context,
	filter *v2.RewardSummaryFilter,
	pagination *v2.Pagination,
) (*v2.EpochRewardSummaryConnection, error) {
	req := v2.ListEpochRewardSummariesRequest{
		Filter:     filter,
		Pagination: pagination,
	}
	resp, err := r.tradingDataClientV2.ListEpochRewardSummaries(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp.Summaries, nil
}

func (r *myQueryResolver) ProtocolUpgradeProposals(
	ctx context.Context,
	inState *eventspb.ProtocolUpgradeProposalStatus,
	approvedBy *string,
	pagination *v2.Pagination,
) (
	*v2.ProtocolUpgradeProposalConnection, error,
) {
	req := v2.ListProtocolUpgradeProposalsRequest{Status: inState, ApprovedBy: approvedBy, Pagination: pagination}
	resp, err := r.tradingDataClientV2.ListProtocolUpgradeProposals(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.ProtocolUpgradeProposals, nil
}

func (r *myQueryResolver) NodeData(ctx context.Context) (*vegapb.NodeData, error) {
	resp, err := r.tradingDataClientV2.GetNetworkData(ctx, &v2.GetNetworkDataRequest{})
	if err != nil {
		return nil, err
	}

	return resp.NodeData, nil
}

func (r *myQueryResolver) NodesConnection(ctx context.Context, pagination *v2.Pagination) (*v2.NodesConnection, error) {
	req := &v2.ListNodesRequest{
		Pagination: pagination,
	}
	resp, err := r.tradingDataClientV2.ListNodes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Nodes, nil
}

func (r *myQueryResolver) Node(ctx context.Context, id string) (*vegapb.Node, error) {
	resp, err := r.tradingDataClientV2.GetNode(ctx, &v2.GetNodeRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return resp.Node, nil
}

func (r *myQueryResolver) KeyRotationsConnection(ctx context.Context, id *string, pagination *v2.Pagination) (*v2.KeyRotationConnection, error) {
	resp, err := r.tradingDataClientV2.ListKeyRotations(ctx, &v2.ListKeyRotationsRequest{NodeId: id, Pagination: pagination})
	if err != nil {
		return nil, err
	}

	return resp.Rotations, nil
}

func (r *myQueryResolver) EthereumKeyRotations(ctx context.Context, nodeID *string) (*v2.EthereumKeyRotationsConnection, error) {
	resp, err := r.tradingDataClientV2.ListEthereumKeyRotations(ctx, &v2.ListEthereumKeyRotationsRequest{NodeId: nodeID})
	if err != nil {
		return nil, err
	}

	return resp.KeyRotations, nil
}

func (r *myQueryResolver) Epoch(ctx context.Context, id *string, block *string) (*vegapb.Epoch, error) {
	var (
		epochID, blockHeight *uint64
		err                  error
	)
	if id != nil {
		parsedID, err := strconv.ParseUint(*id, 10, 64)
		if err != nil {
			return nil, err
		}

		epochID = &parsedID
	}

	if block != nil {
		parsedHeight, err := strconv.ParseUint(*block, 10, 64)
		if err != nil {
			return nil, err
		}
		blockHeight = &parsedHeight
	}

	req := &v2.GetEpochRequest{
		Id:    epochID,
		Block: blockHeight,
	}
	resp, err := r.tradingDataClientV2.GetEpoch(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Epoch, nil
}

func (r *myQueryResolver) Statistics(ctx context.Context) (*vegaprotoapi.Statistics, error) {
	req := &vegaprotoapi.StatisticsRequest{}
	resp, err := r.tradingProxyClient.Statistics(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetStatistics(), nil
}

func (r *myQueryResolver) BalanceChanges(
	ctx context.Context,
	filter *v2.AccountFilter,
	dateRange *v2.DateRange,
	pagination *v2.Pagination,
) (*v2.AggregatedBalanceConnection, error) {
	req := &v2.ListBalanceChangesRequest{
		Filter:     filter,
		DateRange:  dateRange,
		Pagination: pagination,
	}

	resp, err := r.tradingDataClientV2.ListBalanceChanges(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetBalances(), nil
}

func (r *myQueryResolver) LedgerEntries(
	ctx context.Context,
	filter *v2.LedgerEntryFilter,
	dateRange *v2.DateRange,
	pagination *v2.Pagination,
) (*v2.AggregatedLedgerEntriesConnection, error) {
	req := &v2.ListLedgerEntriesRequest{}
	req.Filter = filter

	req.DateRange = dateRange
	req.Pagination = pagination

	resp, err := r.tradingDataClientV2.ListLedgerEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetLedgerEntries(), nil
}

func (r *myQueryResolver) NetworkLimits(ctx context.Context) (*vegapb.NetworkLimits, error) {
	req := &v2.GetNetworkLimitsRequest{}
	resp, err := r.tradingDataClientV2.GetNetworkLimits(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetLimits(), nil
}

func (r *myQueryResolver) MostRecentHistorySegment(ctx context.Context) (*v2.HistorySegment, error) {
	req := &v2.GetMostRecentNetworkHistorySegmentRequest{}

	resp, err := r.tradingDataClientV2.GetMostRecentNetworkHistorySegment(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetSegment(), nil
}

func (r *myQueryResolver) SuccessorMarkets(ctx context.Context, marketID string, fullHistory *bool, pagination *v2.Pagination) (*v2.SuccessorMarketConnection, error) {
	getAll := false

	if fullHistory != nil {
		getAll = *fullHistory
	}

	req := &v2.ListSuccessorMarketsRequest{
		MarketId:           marketID,
		IncludeFullHistory: getAll,
		Pagination:         pagination,
	}

	resp, err := r.tradingDataClientV2.ListSuccessorMarkets(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetSuccessorMarkets(), nil
}

func (r *myQueryResolver) StopOrder(ctx context.Context, id string) (*eventspb.StopOrderEvent, error) {
	req := &v2.GetStopOrderRequest{
		OrderId: id,
	}

	resp, err := r.tradingDataClientV2.GetStopOrder(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Order, nil
}

func (r *myQueryResolver) StopOrders(ctx context.Context, filter *v2.StopOrderFilter, pagination *v2.Pagination) (*v2.StopOrderConnection, error) {
	req := &v2.ListStopOrdersRequest{
		Filter:     filter,
		Pagination: pagination,
	}

	resp, err := r.tradingDataClientV2.ListStopOrders(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Orders, nil
}

func (r *myQueryResolver) LiquidityProviders(ctx context.Context, partyID, marketID *string, pagination *v2.Pagination) (*v2.LiquidityProviderConnection, error) {
	providers, err := r.tradingDataClientV2.ListLiquidityProviders(ctx, &v2.ListLiquidityProvidersRequest{
		MarketId:   marketID,
		PartyId:    partyID,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return providers.LiquidityProviders, nil
}

func (r *myQueryResolver) CurrentReferralProgram(ctx context.Context) (*v2.ReferralProgram, error) {
	resp, err := r.tradingDataClientV2.GetCurrentReferralProgram(ctx, &v2.GetCurrentReferralProgramRequest{})
	if err != nil {
		return &v2.ReferralProgram{}, err
	}

	return resp.CurrentReferralProgram, nil
}

func (r *myQueryResolver) CurrentVolumeDiscountProgram(ctx context.Context) (*v2.VolumeDiscountProgram, error) {
	resp, err := r.tradingDataClientV2.GetCurrentVolumeDiscountProgram(ctx, &v2.GetCurrentVolumeDiscountProgramRequest{})
	if err != nil {
		return &v2.VolumeDiscountProgram{}, err
	}

	return resp.CurrentVolumeDiscountProgram, nil
}

func (r *myQueryResolver) VolumeDiscountStats(ctx context.Context, epoch *int, partyID *string, pagination *v2.Pagination) (*v2.VolumeDiscountStatsConnection, error) {
	var epochU64Ptr *uint64
	if epoch != nil {
		epochU64 := uint64(*epoch)
		epochU64Ptr = &epochU64
	}

	resp, err := r.tradingDataClientV2.GetVolumeDiscountStats(ctx, &v2.GetVolumeDiscountStatsRequest{
		AtEpoch:    epochU64Ptr,
		PartyId:    partyID,
		Pagination: pagination,
	})
	if err != nil {
		return &v2.VolumeDiscountStatsConnection{}, err
	}

	return resp.Stats, nil
}

func (r *myQueryResolver) ReferralSets(ctx context.Context, id, referrer, referee *string, pagination *v2.Pagination) (*v2.ReferralSetConnection, error) {
	req := &v2.ListReferralSetsRequest{
		ReferralSetId: id,
		Pagination:    pagination,
		Referrer:      referrer,
		Referee:       referee,
	}

	resp, err := r.tradingDataClientV2.ListReferralSets(ctx, req)
	if err != nil {
		return &v2.ReferralSetConnection{}, err
	}

	return resp.ReferralSets, nil
}

func (r *myQueryResolver) ReferralSetReferees(ctx context.Context, id, referrer, referee *string, pagination *v2.Pagination, epochsToAggregate *int) (*v2.ReferralSetRefereeConnection, error) {
	var aggregationEpochs uint32 = 30 // default to 30 days

	if epochsToAggregate != nil {
		aggregationEpochs = uint32(*epochsToAggregate)
	}

	req := &v2.ListReferralSetRefereesRequest{
		ReferralSetId:     id,
		Pagination:        pagination,
		Referrer:          referrer,
		Referee:           referee,
		AggregationEpochs: &aggregationEpochs,
	}

	resp, err := r.tradingDataClientV2.ListReferralSetReferees(ctx, req)
	if err != nil {
		return &v2.ReferralSetRefereeConnection{}, err
	}

	return resp.ReferralSetReferees, nil
}

func (r *myQueryResolver) ReferralSetStats(ctx context.Context, setID *string, epoch *int, partyID *string, pagination *v2.Pagination) (*v2.ReferralSetStatsConnection, error) {
	var epochU64Ptr *uint64
	if epoch != nil {
		epochU64 := uint64(*epoch)
		epochU64Ptr = &epochU64
	}

	resp, err := r.tradingDataClientV2.GetReferralSetStats(ctx, &v2.GetReferralSetStatsRequest{
		ReferralSetId: setID,
		AtEpoch:       epochU64Ptr,
		Referee:       partyID,
		Pagination:    pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Stats, nil
}

func (r *myQueryResolver) Teams(ctx context.Context, teamID *string, partyID *string, pagination *v2.Pagination) (*v2.TeamConnection, error) {
	teams, err := r.tradingDataClientV2.ListTeams(ctx, &v2.ListTeamsRequest{
		PartyId:    partyID,
		TeamId:     teamID,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return teams.Teams, nil
}

func (r *myQueryResolver) TeamsStatistics(ctx context.Context, teamID *string, aggregationEpochs *int, pagination *v2.Pagination) (*v2.TeamsStatisticsConnection, error) {
	filters := &v2.ListTeamsStatisticsRequest{
		TeamId:     teamID,
		Pagination: pagination,
	}

	if aggregationEpochs != nil {
		filters.AggregationEpochs = ptr.From(uint64(*aggregationEpochs))
	}

	stats, err := r.tradingDataClientV2.ListTeamsStatistics(ctx, filters)
	if err != nil {
		return nil, err
	}

	return stats.Statistics, nil
}

func (r *myQueryResolver) TeamMembersStatistics(ctx context.Context, teamID string, partyID *string, aggregationEpochs *int, pagination *v2.Pagination) (*v2.TeamMembersStatisticsConnection, error) {
	filters := &v2.ListTeamMembersStatisticsRequest{
		TeamId:     teamID,
		PartyId:    partyID,
		Pagination: pagination,
	}

	if aggregationEpochs != nil {
		filters.AggregationEpochs = ptr.From(uint64(*aggregationEpochs))
	}

	stats, err := r.tradingDataClientV2.ListTeamMembersStatistics(ctx, filters)
	if err != nil {
		return nil, err
	}

	return stats.Statistics, nil
}

func (r *myQueryResolver) TeamReferees(ctx context.Context, teamID string, pagination *v2.Pagination) (*v2.TeamRefereeConnection, error) {
	referees, err := r.tradingDataClientV2.ListTeamReferees(ctx, &v2.ListTeamRefereesRequest{
		TeamId:     teamID,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return referees.TeamReferees, nil
}

func (r *myQueryResolver) TeamRefereeHistory(ctx context.Context, referee string, pagination *v2.Pagination) (*v2.TeamRefereeHistoryConnection, error) {
	history, err := r.tradingDataClientV2.ListTeamRefereeHistory(ctx, &v2.ListTeamRefereeHistoryRequest{
		Referee:    referee,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return history.TeamRefereeHistory, nil
}

func (r *myQueryResolver) FeesStats(ctx context.Context, marketID *string, assetID *string, epoch *int,
	partyID *string,
) (*v1.FeesStats, error) {
	var epochSeq *uint64

	if epoch != nil {
		epochSeq = ptr.From(uint64(*epoch))
	}

	req := &v2.GetFeesStatsRequest{
		MarketId: marketID,
		AssetId:  assetID,
		EpochSeq: epochSeq,
		PartyId:  partyID,
	}

	resp, err := r.tradingDataClientV2.GetFeesStats(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.FeesStats, nil
}

func (r *myQueryResolver) FeesStatsForParty(ctx context.Context, partyID string, assetID *string, fromEpoch, toEpoch *int) ([]*v2.FeesStatsForParty, error) {
	var fromEpochU, toEpochU *uint64

	if fromEpoch != nil {
		fromEpochU = ptr.From(uint64(*fromEpoch))
	}

	if toEpoch != nil {
		toEpochU = ptr.From(uint64(*toEpoch))
	}

	req := &v2.GetFeesStatsForPartyRequest{
		PartyId:   partyID,
		AssetId:   assetID,
		FromEpoch: fromEpochU,
		ToEpoch:   toEpochU,
	}

	resp, err := r.tradingDataClientV2.GetFeesStatsForParty(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.FeesStatsForParty, nil
}

func (r *myQueryResolver) PaidLiquidityFees(
	ctx context.Context,
	marketID *string,
	assetID *string,
	epoch *int,
	partyIDs []string,
) (*v2.PaidLiquidityFeesConnection, error) {
	var epochSeq *uint64

	if epoch != nil {
		epochSeq = ptr.From(uint64(*epoch))
	}

	req := &v2.ListPaidLiquidityFeesRequest{
		MarketId: marketID,
		AssetId:  assetID,
		EpochSeq: epochSeq,
		PartyIds: partyIDs,
	}

	resp, err := r.tradingDataClientV2.ListPaidLiquidityFees(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.PaidLiquidityFees, nil
}

func (r *myQueryResolver) TotalTransferFeeDiscount(
	ctx context.Context,
	partyId string,
	assetId string,
) (*v2.GetTotalTransferFeeDiscountResponse, error) {
	resp, err := r.tradingDataClientV2.GetTotalTransferFeeDiscount(ctx, &v2.GetTotalTransferFeeDiscountRequest{
		PartyId: partyId,
		AssetId: assetId,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *myQueryResolver) EstimateTransferFee(
	ctx context.Context,
	fromAccount string,
	fromAccountType vega.AccountType,
	toAccount string,
	amount string,
	assetId string,
) (*v2.EstimateTransferFeeResponse, error) {
	resp, err := r.tradingDataClientV2.EstimateTransferFee(ctx, &v2.EstimateTransferFeeRequest{
		FromAccount:     fromAccount,
		FromAccountType: fromAccountType,
		ToAccount:       toAccount,
		Amount:          amount,
		AssetId:         assetId,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// END: Root Resolver

type myNodeSignatureResolver VegaResolverRoot

func (r *myNodeSignatureResolver) Signature(_ context.Context, obj *commandspb.NodeSignature) (*string, error) {
	sig := base64.StdEncoding.EncodeToString(obj.Sig)
	return &sig, nil
}

// BEGIN: Party Resolver

type myPartyResolver VegaResolverRoot

func (r *myPartyResolver) VestingStats(
	ctx context.Context,
	obj *vega.Party,
) (*v2.GetPartyVestingStatsResponse, error) {
	res, err := r.r.clt2.GetPartyVestingStats(ctx, &v2.GetPartyVestingStatsRequest{
		PartyId: obj.Id,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *myPartyResolver) VestingBalancesSummary(
	ctx context.Context,
	obj *vega.Party,
	assetID *string,
) (*v2.GetVestingBalancesSummaryResponse, error) {
	res, err := r.r.clt2.GetVestingBalancesSummary(ctx, &v2.GetVestingBalancesSummaryRequest{
		PartyId: obj.Id,
		AssetId: assetID,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *myPartyResolver) ActivityStreak(
	ctx context.Context,
	obj *vega.Party,
	epoch *int,
) (*eventspb.PartyActivityStreak, error) {
	var epoc *uint64
	if epoch != nil {
		if *epoch <= 0 {
			return nil, errors.New("epoch must be positive")
		}
		epoc = ptr.From(uint64(*epoch))
	}
	res, err := r.r.clt2.GetPartyActivityStreak(ctx, &v2.GetPartyActivityStreakRequest{
		PartyId: obj.Id,
		Epoch:   epoc,
	})
	if err != nil {
		return nil, err
	}

	return res.ActivityStreak, nil
}

func (r *myPartyResolver) TransfersConnection(
	ctx context.Context,
	party *vegapb.Party,
	direction *TransferDirection,
	pagination *v2.Pagination,
	isReward *bool,
	fromEpoch, toEpoch *int,
	status *eventspb.Transfer_Status,
	scope *v2.ListTransfersRequest_Scope,
	gameID *string,
) (*v2.TransferConnection, error) {
	return r.r.transfersConnection(ctx, &party.Id, direction, pagination, isReward, fromEpoch, toEpoch, status, scope, gameID)
}

func (r *myPartyResolver) RewardsConnection(ctx context.Context, party *vegapb.Party, assetID *string, pagination *v2.Pagination,
	fromEpoch *int, toEpoch *int, teamID, gameID *string,
) (*v2.RewardsConnection, error) {
	var from, to *uint64

	if fromEpoch != nil {
		from = new(uint64)
		if *fromEpoch < 0 {
			return nil, errors.New("invalid fromEpoch for reward query - must be positive")
		}
		*from = uint64(*fromEpoch)
	}
	if toEpoch != nil {
		to = new(uint64)
		if *toEpoch < 0 {
			return nil, errors.New("invalid toEpoch for reward query - must be positive")
		}
		*to = uint64(*toEpoch)
	}

	req := v2.ListRewardsRequest{
		PartyId:    party.Id,
		AssetId:    assetID,
		Pagination: pagination,
		FromEpoch:  from,
		ToEpoch:    to,
		TeamId:     teamID,
		GameId:     gameID,
	}
	resp, err := r.tradingDataClientV2.ListRewards(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rewards information: %w", err)
	}

	return resp.Rewards, nil
}

func (r *myPartyResolver) RewardSummaries(
	ctx context.Context,
	party *vegapb.Party,
	asset *string,
) ([]*vegapb.RewardSummary, error) {
	var assetID string
	if asset != nil {
		assetID = *asset
	}

	req := &v2.ListRewardSummariesRequest{
		PartyId: &party.Id,
		AssetId: &assetID,
	}

	resp, err := r.tradingDataClientV2.ListRewardSummaries(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Summaries, err
}

func (r *myPartyResolver) StakingSummary(ctx context.Context, party *vegapb.Party, pagination *v2.Pagination) (*StakingSummary, error) {
	if party == nil {
		return nil, errors.New("party must not be nil")
	}

	req := &v2.GetStakeRequest{
		PartyId:    party.Id,
		Pagination: pagination,
	}

	resp, err := r.tradingDataClientV2.GetStake(ctx, req)
	if err != nil {
		return nil, err
	}

	return &StakingSummary{
		CurrentStakeAvailable: resp.CurrentStakeAvailable,
		Linkings:              resp.StakeLinkings,
	}, nil
}

func (r *myPartyResolver) LiquidityProvisionsConnection(
	ctx context.Context,
	party *vegapb.Party,
	market, ref *string,
	live *bool,
	pagination *v2.Pagination,
) (*v2.LiquidityProvisionsConnection, error) {
	var partyID string
	if party != nil {
		partyID = party.Id
	}
	var mid string
	if market != nil {
		mid = *market
	}

	var refID string
	if ref != nil {
		refID = *ref
	}

	var l bool
	if live != nil {
		l = *live
	}

	req := v2.ListLiquidityProvisionsRequest{
		PartyId:    &partyID,
		MarketId:   &mid,
		Reference:  &refID,
		Live:       &l,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListLiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.LiquidityProvisions, nil
}

func (r *myPartyResolver) LiquidityProvisions(ctx context.Context, party *vega.Party, market, ref *string,
	live *bool, pagination *v2.Pagination,
) (*v2.LiquidityProvisionsWithPendingConnection, error) {
	var partyID string
	if party != nil {
		partyID = party.Id
	}
	var mid string
	if market != nil {
		mid = *market
	}

	var refID string
	if ref != nil {
		refID = *ref
	}

	var l bool
	if live != nil {
		l = *live
	}

	req := v2.ListAllLiquidityProvisionsRequest{
		PartyId:    &partyID,
		MarketId:   &mid,
		Reference:  &refID,
		Live:       &l,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListAllLiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.LiquidityProvisions, nil
}

func (r *myPartyResolver) MarginsConnection(ctx context.Context, party *vegapb.Party, marketID *string,
	pagination *v2.Pagination,
) (*v2.MarginConnection, error) {
	if party == nil {
		return nil, errors.New("party is nil")
	}

	market := ""

	if marketID != nil {
		market = *marketID
	}

	req := v2.ListMarginLevelsRequest{
		PartyId:    party.Id,
		MarketId:   market,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListMarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.MarginLevels, nil
}

func (r *myPartyResolver) OrdersConnection(ctx context.Context, party *vegapb.Party, pagination *v2.Pagination, filter *OrderByMarketIdsFilter) (*v2.OrderConnection, error) {
	req := v2.ListOrdersRequest{
		Pagination: pagination,
		Filter: &v2.OrderFilter{
			PartyIds: []string{party.Id},
		},
	}

	if filter != nil {
		req.Filter.MarketIds = filter.MarketIds
		if filter.Order != nil {
			req.Filter.Statuses = filter.Order.Statuses
			req.Filter.Types = filter.Order.Types
			req.Filter.TimeInForces = filter.Order.TimeInForces
			req.Filter.ExcludeLiquidity = filter.Order.ExcludeLiquidity
			req.Filter.Reference = filter.Order.Reference
			req.Filter.DateRange = filter.Order.DateRange
			req.Filter.LiveOnly = filter.Order.LiveOnly
		}
	}

	res, err := r.tradingDataClientV2.ListOrders(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Orders, nil
}

func (r *myPartyResolver) TradesConnection(ctx context.Context, party *vegapb.Party, market *string, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	mkts := []string{}
	if market != nil {
		mkts = []string{*market}
	}
	req := v2.ListTradesRequest{
		PartyIds:   []string{party.Id},
		MarketIds:  mkts,
		Pagination: pagination,
		DateRange:  dateRange,
	}

	res, err := r.tradingDataClientV2.ListTrades(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Trades, nil
}

func (r *myPartyResolver) PositionsConnection(ctx context.Context, party *vegapb.Party, market *string, pagination *v2.Pagination) (*v2.PositionConnection, error) {
	partyID := ""
	if party != nil {
		partyID = party.Id
	}

	marketID := ""
	if market != nil {
		marketID = *market
	}

	req := v2.ListPositionsRequest{
		PartyId:    partyID,
		MarketId:   marketID,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListPositions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.Positions, nil
}

func (r *myPartyResolver) AccountsConnection(ctx context.Context, party *vegapb.Party, marketID *string, asset *string, accType *vegapb.AccountType, pagination *v2.Pagination) (*v2.AccountsConnection, error) {
	if party == nil {
		return nil, errors.New("a party must be specified when querying accounts")
	}
	var (
		marketIDs    = []string{}
		mktID        = ""
		asst         = ""
		accountTypes = []vegapb.AccountType{}
		accTy        = vegapb.AccountType_ACCOUNT_TYPE_UNSPECIFIED
		err          error
	)

	if marketID != nil {
		marketIDs = []string{*marketID}
		mktID = *marketID
	}

	if asset != nil {
		asst = *asset
	}
	if accType != nil {
		accTy = *accType
		if accTy != vegapb.AccountType_ACCOUNT_TYPE_GENERAL &&
			accTy != vegapb.AccountType_ACCOUNT_TYPE_MARGIN &&
			accTy != vegapb.AccountType_ACCOUNT_TYPE_ORDER_MARGIN &&
			accTy != vegapb.AccountType_ACCOUNT_TYPE_BOND &&
			accTy != vega.AccountType_ACCOUNT_TYPE_VESTED_REWARDS &&
			accTy != vega.AccountType_ACCOUNT_TYPE_VESTING_REWARDS {
			return nil, fmt.Errorf("invalid account type for party %v", accType)
		}
		accountTypes = []vegapb.AccountType{accTy}
	}

	filter := v2.AccountFilter{
		AssetId:      asst,
		PartyIds:     []string{party.Id},
		MarketIds:    marketIDs,
		AccountTypes: accountTypes,
	}

	req := v2.ListAccountsRequest{Filter: &filter, Pagination: pagination}
	res, err := r.tradingDataClientV2.ListAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get Party account",
			logging.Error(err),
			logging.String("party-id", party.Id),
			logging.String("market-id", mktID),
			logging.String("asset", asst),
			logging.String("type", accTy.String()))
		return nil, err
	}

	return res.Accounts, nil
}

func (r *myPartyResolver) ProposalsConnection(ctx context.Context, party *vegapb.Party, proposalType *v2.ListGovernanceDataRequest_Type, inState *vega.Proposal_State,
	pagination *v2.Pagination,
) (*v2.GovernanceDataConnection, error) {
	return handleProposalsRequest(ctx, r.tradingDataClientV2, party, nil, proposalType, inState, pagination)
}

func (r *myPartyResolver) WithdrawalsConnection(ctx context.Context, party *vegapb.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.WithdrawalsConnection, error) {
	return handleWithdrawalsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

func (r *myPartyResolver) DepositsConnection(ctx context.Context, party *vegapb.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.DepositsConnection, error) {
	return handleDepositsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

func (r *myPartyResolver) VotesConnection(ctx context.Context, party *vegapb.Party, pagination *v2.Pagination) (*ProposalVoteConnection, error) {
	req := v2.ListVotesRequest{
		PartyId:    &party.Id,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListVotes(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	edges := make([]*ProposalVoteEdge, 0, len(res.Votes.Edges))

	for _, edge := range res.Votes.Edges {
		edges = append(edges, &ProposalVoteEdge{
			Cursor: &edge.Cursor,
			Node:   ProposalVoteFromProto(edge.Node),
		})
	}

	connection := &ProposalVoteConnection{
		Edges:    edges,
		PageInfo: res.Votes.PageInfo,
	}

	return connection, nil
}

func (r *myPartyResolver) DelegationsConnection(ctx context.Context, party *vegapb.Party, nodeID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
	var partyID *string
	if party != nil {
		partyID = &party.Id
	}

	return handleDelegationConnectionRequest(ctx, r.tradingDataClientV2, partyID, nodeID, nil, pagination)
}

// END: Party Resolver

type myMarginLevelsUpdateResolver VegaResolverRoot

func (r *myMarginLevelsUpdateResolver) InitialLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.InitialMargin, nil
}

func (r *myMarginLevelsUpdateResolver) SearchLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.SearchLevel, nil
}

func (r *myMarginLevelsUpdateResolver) MaintenanceLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.MaintenanceMargin, nil
}

func (r *myMarginLevelsUpdateResolver) OrderMarginLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.OrderMargin, nil
}

func (r *myMarginLevelsUpdateResolver) MarginFactor(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.MarginFactor, nil
}

// BEGIN: MarginLevels Resolver

type myMarginLevelsResolver VegaResolverRoot

func (r *myMarginLevelsResolver) Market(ctx context.Context, m *vegapb.MarginLevels) (*vegapb.Market, error) {
	return r.r.getMarketByID(ctx, m.MarketId)
}

func (r *myMarginLevelsResolver) Party(ctx context.Context, m *vegapb.MarginLevels) (*vegapb.Party, error) {
	if m == nil {
		return nil, errors.New("nil order")
	}
	if len(m.PartyId) == 0 {
		return nil, errors.New("invalid party")
	}
	req := v2.GetPartyRequest{PartyId: m.PartyId}
	res, err := r.tradingDataClientV2.GetParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Party, nil
}

func (r *myMarginLevelsResolver) Asset(ctx context.Context, m *vegapb.MarginLevels) (*vegapb.Asset, error) {
	return r.r.getAssetByID(ctx, m.Asset)
}

func (r *myMarginLevelsResolver) CollateralReleaseLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.CollateralReleaseLevel, nil
}

func (r *myMarginLevelsResolver) InitialLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.InitialMargin, nil
}

func (r *myMarginLevelsResolver) SearchLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.SearchLevel, nil
}

func (r *myMarginLevelsResolver) MaintenanceLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.MaintenanceMargin, nil
}

func (r *myMarginLevelsResolver) OrderMarginLevel(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.OrderMargin, nil
}

func (r *myMarginLevelsResolver) MarginFactor(_ context.Context, m *vegapb.MarginLevels) (string, error) {
	return m.MarginFactor, nil
}

// END: MarginLevels Resolver

type myOrderUpdateResolver VegaResolverRoot

func (r *myOrderUpdateResolver) Price(_ context.Context, obj *vegapb.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderUpdateResolver) Size(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderUpdateResolver) Remaining(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderUpdateResolver) CreatedAt(_ context.Context, obj *vegapb.Order) (int64, error) {
	return obj.CreatedAt, nil
}

func (r *myOrderUpdateResolver) UpdatedAt(_ context.Context, obj *vegapb.Order) (*int64, error) {
	var updatedAt *int64
	if obj.UpdatedAt > 0 {
		t := obj.UpdatedAt
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderUpdateResolver) Version(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderUpdateResolver) ExpiresAt(_ context.Context, obj *vegapb.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}

func (r *myOrderUpdateResolver) RejectionReason(_ context.Context, o *vegapb.Order) (*vega.OrderError, error) {
	return o.Reason, nil
}

// BEGIN: Order Resolver

type myOrderResolver VegaResolverRoot

func (r *myOrderResolver) RejectionReason(_ context.Context, o *vegapb.Order) (*vega.OrderError, error) {
	return o.Reason, nil
}

func (r *myOrderResolver) Price(_ context.Context, obj *vegapb.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderResolver) Market(ctx context.Context, obj *vegapb.Order) (*vegapb.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myOrderResolver) Size(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderResolver) Remaining(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderResolver) CreatedAt(_ context.Context, obj *vegapb.Order) (int64, error) {
	return obj.CreatedAt, nil
}

func (r *myOrderResolver) UpdatedAt(_ context.Context, obj *vegapb.Order) (*int64, error) {
	var updatedAt *int64
	if obj.UpdatedAt > 0 {
		t := obj.UpdatedAt
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderResolver) Version(_ context.Context, obj *vegapb.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderResolver) ExpiresAt(_ context.Context, obj *vegapb.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}

func (r *myOrderResolver) TradesConnection(ctx context.Context, ord *vegapb.Order, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	req := v2.ListTradesRequest{
		OrderIds:   []string{ord.Id},
		Pagination: pagination,
		DateRange:  dateRange,
	}
	res, err := r.tradingDataClientV2.ListTrades(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Trades, nil
}

func (r *myOrderResolver) Party(_ context.Context, order *vegapb.Order) (*vegapb.Party, error) {
	if order == nil {
		return nil, errors.New("nil order")
	}
	if len(order.PartyId) == 0 {
		return nil, errors.New("invalid party")
	}
	return &vegapb.Party{Id: order.PartyId}, nil
}

func (r *myOrderResolver) PeggedOrder(_ context.Context, order *vegapb.Order) (*vegapb.PeggedOrder, error) {
	return order.PeggedOrder, nil
}

func (r *myOrderResolver) IcebergOrder(_ context.Context, order *vegapb.Order) IcebergOrderResolver {
	return (*icebergOrderResolver)(r)
}

func (r *myOrderResolver) LiquidityProvision(ctx context.Context, obj *vegapb.Order) (*vegapb.LiquidityProvision, error) {
	if obj == nil || len(obj.LiquidityProvisionId) <= 0 {
		return nil, nil
	}

	req := v2.ListLiquidityProvisionsRequest{
		PartyId:  &obj.PartyId,
		MarketId: &obj.MarketId,
	}
	res, err := r.tradingDataClientV2.ListLiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	if len(res.LiquidityProvisions.Edges) <= 0 {
		return nil, nil
	}

	return res.LiquidityProvisions.Edges[0].Node, nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type myTradeResolver VegaResolverRoot

func (r *myTradeResolver) Market(ctx context.Context, obj *vegapb.Trade) (*vegapb.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myTradeResolver) Price(_ context.Context, obj *vegapb.Trade) (string, error) {
	return obj.Price, nil
}

func (r *myTradeResolver) Size(_ context.Context, obj *vegapb.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myTradeResolver) CreatedAt(_ context.Context, obj *vegapb.Trade) (int64, error) {
	return obj.Timestamp, nil
}

func (r *myTradeResolver) Buyer(ctx context.Context, obj *vegapb.Trade) (*vegapb.Party, error) {
	if obj == nil {
		return nil, errors.New("invalid trade")
	}
	if len(obj.Buyer) == 0 {
		return nil, errors.New("invalid buyer")
	}
	req := v2.GetPartyRequest{PartyId: obj.Buyer}
	res, err := r.tradingDataClientV2.GetParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Party, nil
}

func (r *myTradeResolver) Seller(ctx context.Context, obj *vegapb.Trade) (*vegapb.Party, error) {
	if obj == nil {
		return nil, errors.New("invalid trade")
	}
	if len(obj.Seller) == 0 {
		return nil, errors.New("invalid seller")
	}
	req := v2.GetPartyRequest{PartyId: obj.Seller}
	res, err := r.tradingDataClientV2.GetParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Party, nil
}

func (r *myTradeResolver) BuyerAuctionBatch(_ context.Context, obj *vegapb.Trade) (*int, error) {
	i := int(obj.BuyerAuctionBatch)
	return &i, nil
}

func setIfExists(v string) *string {
	if len(v) <= 0 {
		return nil
	}

	return ptr.From(v)
}

func (r *myTradeResolver) BuyerFee(_ context.Context, obj *vegapb.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.BuyerFee != nil {
		fee.MakerFee = obj.BuyerFee.MakerFee
		fee.MakerFeeReferralDiscount = setIfExists(obj.BuyerFee.MakerFeeReferrerDiscount)
		fee.MakerFeeVolumeDiscount = setIfExists(obj.BuyerFee.MakerFeeVolumeDiscount)
		fee.InfrastructureFee = obj.BuyerFee.InfrastructureFee
		fee.InfrastructureFeeReferralDiscount = setIfExists(obj.BuyerFee.InfrastructureFeeReferrerDiscount)
		fee.InfrastructureFeeVolumeDiscount = setIfExists(obj.BuyerFee.InfrastructureFeeVolumeDiscount)
		fee.LiquidityFee = obj.BuyerFee.LiquidityFee
		fee.LiquidityFeeReferralDiscount = setIfExists(obj.BuyerFee.LiquidityFeeReferrerDiscount)
		fee.LiquidityFeeVolumeDiscount = setIfExists(obj.BuyerFee.LiquidityFeeVolumeDiscount)
	}
	return &fee, nil
}

func (r *myTradeResolver) SellerAuctionBatch(_ context.Context, obj *vegapb.Trade) (*int, error) {
	i := int(obj.SellerAuctionBatch)
	return &i, nil
}

func (r *myTradeResolver) SellerFee(_ context.Context, obj *vegapb.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.SellerFee != nil {
		fee.MakerFee = obj.SellerFee.MakerFee
		fee.MakerFeeReferralDiscount = setIfExists(obj.SellerFee.MakerFeeReferrerDiscount)
		fee.MakerFeeVolumeDiscount = setIfExists(obj.SellerFee.MakerFeeVolumeDiscount)
		fee.InfrastructureFee = obj.SellerFee.InfrastructureFee
		fee.InfrastructureFeeReferralDiscount = setIfExists(obj.SellerFee.InfrastructureFeeReferrerDiscount)
		fee.InfrastructureFeeVolumeDiscount = setIfExists(obj.SellerFee.InfrastructureFeeVolumeDiscount)
		fee.LiquidityFee = obj.SellerFee.LiquidityFee
		fee.LiquidityFeeReferralDiscount = setIfExists(obj.SellerFee.LiquidityFeeReferrerDiscount)
		fee.LiquidityFeeVolumeDiscount = setIfExists(obj.SellerFee.LiquidityFeeVolumeDiscount)
	}

	return &fee, nil
}

// END: Trade Resolver

// BEGIN: Candle Resolver

type myCandleResolver VegaResolverRoot

func (r *myCandleResolver) PeriodStart(_ context.Context, obj *v2.Candle) (int64, error) {
	return obj.Start, nil
}

func (r *myCandleResolver) LastUpdateInPeriod(_ context.Context, obj *v2.Candle) (int64, error) {
	return obj.LastUpdate, nil
}

func (r *myCandleResolver) Volume(_ context.Context, obj *v2.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myCandleResolver) Notional(_ context.Context, obj *v2.Candle) (string, error) {
	return strconv.FormatUint(obj.Notional, 10), nil
}

// END: Candle Resolver

// BEGIN: DataSourceSpecConfiguration Resolver.
type myDataSourceSpecConfigurationResolver VegaResolverRoot

func (m *myDataSourceSpecConfigurationResolver) Signers(_ context.Context, obj *vegapb.DataSourceSpecConfiguration) ([]*Signer, error) {
	return resolveSigners(obj.Signers), nil
}

func (m *myDataSourceSpecConfigurationResolver) Filters(ctx context.Context, obj *vega.DataSourceSpecConfiguration) ([]*Filter, error) {
	if obj != nil {
		return resolveFilters(obj.Filters)
	}

	return nil, errors.New("dataSourceSpecConfiguration object is empty")
}

// END: DataSourceSpecConfiguration Resolver.

// BEGIN: DataSourceSpecConfigurationTime Resolver.
type myDataSourceSpecConfigurationTimeResolver VegaResolverRoot

func (m *myDataSourceSpecConfigurationTimeResolver) Conditions(ctx context.Context, obj *vega.DataSourceSpecConfigurationTime) ([]*Condition, error) {
	if obj != nil {
		if obj.Conditions != nil {
			return resolveConditions(obj.Conditions), nil
		}
		return nil, errors.New("conditions in internal data source time object are empty")
	}
	return nil, errors.New("internal data source time object is empty")
}

// END: DataSourceSpecConfigurationTime Resolver.

// BEGIN: DataSourceSpecConfigurationTimeTriggerResolver.
type myDataSourceSpecConfigurationTimeTriggerResolver VegaResolverRoot

func (m *myDataSourceSpecConfigurationTimeTriggerResolver) Conditions(ctx context.Context, obj *vega.DataSourceSpecConfigurationTimeTrigger) ([]*Condition, error) {
	if obj != nil {
		if obj.Conditions != nil {
			return resolveConditions(obj.Conditions), nil
		}
		return nil, errors.New("conditions in internal data source time trigger object are empty")
	}
	return nil, errors.New("internal data source time trigger object is empty")
}

// END: DataSourceSpecConfigurationTimeTriggerResolver.

// BEGIN: EthCallSpec Resolver.
type ethCallSpecResolver VegaResolverRoot

func (m *ethCallSpecResolver) SourceChainID(ctx context.Context, obj *vegapb.EthCallSpec) (int, error) {
	if obj != nil {
		return int(obj.SourceChainId), nil
	}

	return 0, nil
}

func (m *ethCallSpecResolver) Abi(ctx context.Context, obj *vegapb.EthCallSpec) ([]string, error) {
	if obj != nil {
		if len(obj.Abi) > 0 {
			return []string{obj.Abi}, nil
		}
	}
	return nil, errors.New("abi in ethereum spec object is empty")
}

func (m *ethCallSpecResolver) Args(ctx context.Context, obj *vegapb.EthCallSpec) ([]string, error) {
	if obj != nil {
		jsonArgs := []string{}
		for _, arg := range obj.Args {
			jsonArg, err := arg.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("unable to marshal args: %s", arg)
			}
			jsonArgs = append(jsonArgs, string(jsonArg))
		}
		return jsonArgs, nil
	}
	return nil, errors.New("ethereum spec object is empty")
}

func (m *ethCallSpecResolver) Trigger(_ context.Context, obj *vegapb.EthCallSpec) (*EthCallTrigger, error) {
	if obj != nil {
		if obj.Trigger != nil {
			return &EthCallTrigger{
				Trigger: resolveTrigger(obj.Trigger.Trigger),
			}, nil
		}
		return nil, errors.New("trigger in ethereum spec object is empty")
	}

	return nil, errors.New("ethereum spec object is empty")
}

func (m *ethCallSpecResolver) RequiredConfirmations(ctx context.Context, obj *vega.EthCallSpec) (int, error) {
	if obj != nil {
		return int(obj.RequiredConfirmations), nil
	}

	return int(0), errors.New("ethereum spec object is empty")
}

func (m *ethCallSpecResolver) Normalisers(ctx context.Context, obj *vegapb.EthCallSpec) ([]*Normaliser, error) {
	if obj != nil {
		if obj.Normalisers != nil {
			return resolveNormalisers(obj.Normalisers), nil
		}

		return nil, errors.New("normalisers in ethereum spec object are empty")
	}

	return nil, errors.New("ethereum spec object is empty")
}

func (m *ethCallSpecResolver) Filters(ctx context.Context, obj *vegapb.EthCallSpec) ([]*Filter, error) {
	if obj != nil {
		if obj.Filters != nil {
			return resolveFilters(obj.Filters)
		}

		return nil, errors.New("filters in ethereum spec object are empty")
	}

	return nil, errors.New("ethereum spec object is empty")
}

// END: EthCallSpec resolver.

// BEGIN: Price Level Resolver

type myPriceLevelResolver VegaResolverRoot

func (r *myPriceLevelResolver) Price(_ context.Context, obj *vegapb.PriceLevel) (string, error) {
	return obj.Price, nil
}

func (r *myPriceLevelResolver) Volume(_ context.Context, obj *vegapb.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myPriceLevelResolver) NumberOfOrders(_ context.Context, obj *vegapb.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.NumberOfOrders, 10), nil
}

// END: Price Level Resolver

type positionUpdateResolver VegaResolverRoot

func (r *positionUpdateResolver) OpenVolume(_ context.Context, obj *vegapb.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *positionUpdateResolver) UpdatedAt(_ context.Context, obj *vegapb.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *positionUpdateResolver) LossSocializationAmount(_ context.Context, obj *vegapb.Position) (string, error) {
	return obj.LossSocialisationAmount, nil
}

// BEGIN: Position Resolver

type myPositionResolver VegaResolverRoot

func (r *myPositionResolver) Market(ctx context.Context, obj *vegapb.Position) (*vegapb.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myPositionResolver) UpdatedAt(_ context.Context, obj *vegapb.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myPositionResolver) OpenVolume(_ context.Context, obj *vegapb.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *myPositionResolver) RealisedPnl(_ context.Context, obj *vegapb.Position) (string, error) {
	return obj.RealisedPnl, nil
}

func (r *myPositionResolver) UnrealisedPnl(_ context.Context, obj *vegapb.Position) (string, error) {
	return obj.UnrealisedPnl, nil
}

func (r *myPositionResolver) AverageEntryPrice(_ context.Context, obj *vegapb.Position) (string, error) {
	return obj.AverageEntryPrice, nil
}

func (r *myPositionResolver) LossSocializationAmount(_ context.Context, obj *vegapb.Position) (string, error) {
	return obj.LossSocialisationAmount, nil
}

func (r *myPositionResolver) Party(ctx context.Context, obj *vegapb.Position) (*vegapb.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, obj.PartyId)
}

func (r *myPositionResolver) MarginsConnection(ctx context.Context, pos *vegapb.Position, pagination *v2.Pagination) (*v2.MarginConnection, error) {
	req := v2.ListMarginLevelsRequest{
		PartyId:    pos.PartyId,
		MarketId:   pos.MarketId,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListMarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.MarginLevels, nil
}

// END: Position Resolver

// BEGIN: Subscription Resolver

type mySubscriptionResolver VegaResolverRoot

func (r *mySubscriptionResolver) Margins(ctx context.Context, partyID string, marketID *string) (<-chan *vegapb.MarginLevels, error) {
	req := &v2.ObserveMarginLevelsRequest{
		MarketId: marketID,
		PartyId:  partyID,
	}
	stream, err := r.tradingDataClientV2.ObserveMarginLevels(ctx, req)
	if err != nil {
		return nil, err
	}

	sCtx := stream.Context()
	ch := make(chan *vegapb.MarginLevels)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("margin levels: stream closed", logging.Error(err))
			}
			close(ch)
		}()
		for {
			m, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("margin levels: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("margin levls: stream closed", logging.Error(err))
				break
			}
			select {
			case ch <- m.MarginLevels:
				r.log.Debug("margin levels: data sent")
			case <-ctx.Done():
				r.log.Error("margin levels: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("margin levels: stream closed by server")
				break
			}
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) Accounts(ctx context.Context, marketID *string, partyID *string, asset *string, typeArg *vegapb.AccountType) (<-chan []*v2.AccountBalance, error) {
	var (
		mkt, pty, ast string
		ty            vegapb.AccountType
	)

	if marketID == nil && partyID == nil && asset == nil && typeArg == nil {
		// Updates on every balance update, on every account, for everyone and shouldn't be allowed for GraphQL.
		return nil, errors.New("at least one query filter must be applied for this subscription")
	}
	if asset != nil {
		ast = *asset
	}
	if marketID != nil {
		mkt = *marketID
	}
	if partyID != nil {
		pty = *partyID
	}
	if typeArg != nil {
		ty = *typeArg
	}

	req := &v2.ObserveAccountsRequest{
		Asset:    ast,
		MarketId: mkt,
		PartyId:  pty,
		Type:     ty,
	}
	stream, err := r.tradingDataClientV2.ObserveAccounts(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*v2.AccountBalance)
	var accounts []*v2.AccountBalance
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("accounts: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			a, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("accounts: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("accounts: stream closed", logging.Error(err))
				break
			}

			// empty slice, but preserve cap to avoid excessive reallocation
			accounts = accounts[:0]
			if snapshot := a.GetSnapshot(); snapshot != nil {
				accounts = append(accounts, snapshot.Accounts...)
			}

			if updates := a.GetUpdates(); updates != nil {
				accounts = append(accounts, updates.Accounts...)
			}
			select {
			case c <- accounts:
				r.log.Debug("accounts: data sent")
			case <-ctx.Done():
				r.log.Error("accounts: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("accounts: stream closed by server")
				break
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Orders(ctx context.Context, filter *OrderByMarketAndPartyIdsFilter) (<-chan []*vegapb.Order, error) {
	req := &v2.ObserveOrdersRequest{}
	if filter != nil {
		req.MarketIds = filter.MarketIds
		req.PartyIds = filter.PartyIds
	}

	stream, err := r.tradingDataClientV2.ObserveOrders(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*vegapb.Order)
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			o, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			orders := []*vegapb.Order{}
			if snapshot := o.GetSnapshot(); snapshot != nil {
				orders = append(orders, snapshot.Orders...)
			}
			if updates := o.GetUpdates(); updates != nil {
				orders = append(orders, updates.Orders...)
			}
			select {
			case c <- orders:
				r.log.Debug("orders: data sent")
			case <-ctx.Done():
				r.log.Error("orders: stream closed")
				return
			case <-sCtx.Done():
				r.log.Error("orders: stream closed by server")
				return
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []*vegapb.Trade, error) {
	markets := []string{}
	parties := []string{}
	if market != nil {
		markets = append(markets, *market)
	}

	if party != nil {
		parties = append(parties, *party)
	}

	req := &v2.ObserveTradesRequest{
		MarketIds: markets,
		PartyIds:  parties,
	}

	stream, err := r.tradingDataClientV2.ObserveTrades(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*vegapb.Trade)
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("trades: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("trades: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("trades: stream closed", logging.Error(err))
				break
			}
			select {
			case c <- t.Trades:
				r.log.Debug("trades: data sent")
			case <-ctx.Done():
				r.log.Error("trades: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("trades: stream closed by server")
				break
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) TradesStream(ctx context.Context, filter TradesSubscriptionFilter) (<-chan []*vegapb.Trade, error) {
	req := &v2.ObserveTradesRequest{
		MarketIds: filter.MarketIds,
		PartyIds:  filter.PartyIds,
	}
	stream, err := r.tradingDataClientV2.ObserveTrades(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*vegapb.Trade)
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("trades: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("trades: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("trades: stream closed", logging.Error(err))
				break
			}
			select {
			case c <- t.Trades:
				r.log.Debug("trades: data sent")
			case <-ctx.Done():
				r.log.Error("trades: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("trades: stream closed by server")
				break
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Positions(ctx context.Context, party, market *string) (<-chan []*vegapb.Position, error) {
	req := &v2.ObservePositionsRequest{
		PartyId:  party,
		MarketId: market,
	}
	stream, err := r.tradingDataClientV2.ObservePositions(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*vegapb.Position)
	var positions []*vegapb.Position
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("positions: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("positions: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("positions: stream closed", logging.Error(err))
				break
			}
			positions = positions[:0]
			if snapshot := t.GetSnapshot(); snapshot != nil {
				positions = append(positions, snapshot.Positions...)
			}

			if updates := t.GetUpdates(); updates != nil {
				positions = append(positions, updates.Positions...)
			}
			select {
			case c <- positions:
				r.log.Debug("positions: data sent")
			case <-ctx.Done():
				r.log.Error("positions: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("positions: stream closed by server")
				break
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Candles(ctx context.Context, market string, interval vega.Interval) (<-chan *v2.Candle, error) {
	intervalToCandleIDs, err := r.tradingDataClientV2.ListCandleIntervals(ctx, &v2.ListCandleIntervalsRequest{
		MarketId: market,
	})
	if err != nil {
		return nil, err
	}

	candleID := ""
	var candleInterval vegapb.Interval
	for _, ic := range intervalToCandleIDs.IntervalToCandleId {
		candleInterval, err = convertDataNodeIntervalToProto(ic.Interval)
		if err != nil {
			r.log.Errorf("convert interval to candle id failed: %v", err)
			continue
		}
		if candleInterval == interval {
			candleID = ic.CandleId
			break
		}
	}

	if candleID == "" {
		return nil, fmt.Errorf("candle information not found for market: %s, interval: %s", market, interval)
	}

	req := &v2.ObserveCandleDataRequest{
		CandleId: candleID,
	}
	stream, err := r.tradingDataClientV2.ObserveCandleData(ctx, req)
	if err != nil {
		return nil, err
	}

	sCtx := stream.Context()
	c := make(chan *v2.Candle)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("candles: stream closed", logging.Error(err))
			}
			close(c)
		}()
		for {
			cdl, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("candles: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("candles: stream closed", logging.Error(err))
				break
			}

			select {
			case c <- cdl.Candle:
				r.log.Debug("candles: data sent")
			case <-ctx.Done():
				r.log.Error("candles: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("candles: stream closed by server")
				break
			}
		}
	}()
	return c, nil
}

func isStreamClosed(err error, log *logging.Logger) bool {
	if err == io.EOF {
		log.Error("stream closed by server", logging.Error(err))
		return true
	}
	if err != nil {
		log.Error("stream closed", logging.Error(err))
		return true
	}
	return false
}

func (r *mySubscriptionResolver) subscribeAllProposals(ctx context.Context) (<-chan *vegapb.GovernanceData, error) {
	stream, err := r.tradingDataClientV2.ObserveGovernance(ctx, &v2.ObserveGovernanceRequest{})
	if err != nil {
		return nil, err
	}
	output := make(chan *vegapb.GovernanceData)
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("governance (all): failed to close stream", logging.Error(err))
			}
			close(output)
		}()
		for proposals, err := stream.Recv(); !isStreamClosed(err, r.log); proposals, err = stream.Recv() {
			select {
			case output <- proposals.Data:
				r.log.Debug("governance (all): data sent")
			case <-ctx.Done():
				r.log.Error("governance (all): stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("governance (all): stream closed by server")
				break
			}
		}
	}()
	return output, nil
}

func (r *mySubscriptionResolver) subscribePartyProposals(ctx context.Context, partyID string) (<-chan *vegapb.GovernanceData, error) {
	stream, err := r.tradingDataClientV2.ObserveGovernance(ctx, &v2.ObserveGovernanceRequest{
		PartyId: &partyID,
	})
	if err != nil {
		return nil, err
	}
	sCtx := stream.Context()
	output := make(chan *vegapb.GovernanceData)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("governance (party): stream close failed", logging.Error(err))
			}
			close(output)
		}()
		for proposals, err := stream.Recv(); !isStreamClosed(err, r.log); proposals, err = stream.Recv() {
			select {
			case output <- proposals.Data:
				r.log.Debug("governance (party): data sent")
			case <-ctx.Done():
				r.log.Error("governance (party): stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("governance (party): stream closed by server")
				break
			}
		}
	}()
	return output, nil
}

func (r *mySubscriptionResolver) Proposals(ctx context.Context, partyID *string) (<-chan *vegapb.GovernanceData, error) {
	if partyID != nil && len(*partyID) > 0 {
		return r.subscribePartyProposals(ctx, *partyID)
	}
	return r.subscribeAllProposals(ctx)
}

func (r *mySubscriptionResolver) subscribeProposalVotes(ctx context.Context, proposalID string) (<-chan *ProposalVote, error) {
	output := make(chan *ProposalVote)
	stream, err := r.tradingDataClientV2.ObserveVotes(ctx, &v2.ObserveVotesRequest{
		ProposalId: &proposalID,
	})
	if err != nil {
		return nil, err
	}
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("votes (proposal): stream close failed", logging.Error(err))
			}
			close(output)
		}()
		for {
			votes, err := stream.Recv()
			if isStreamClosed(err, r.log) {
				break
			}
			select {
			case output <- ProposalVoteFromProto(votes.Vote):
				r.log.Debug("votes (proposal): data sent")
			case <-ctx.Done():
				r.log.Error("votes (proposal): stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("votes (proposal): stream closed by server")
				break
			}
		}
	}()
	return output, nil
}

func (r *mySubscriptionResolver) subscribePartyVotes(ctx context.Context, partyID string) (<-chan *ProposalVote, error) {
	output := make(chan *ProposalVote)
	stream, err := r.tradingDataClientV2.ObserveVotes(ctx, &v2.ObserveVotesRequest{
		PartyId: &partyID,
	})
	if err != nil {
		return nil, err
	}
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("votes (party): failed to close stream", logging.Error(err))
			}
			close(output)
		}()
		for {
			votes, err := stream.Recv()
			if isStreamClosed(err, r.log) {
				break
			}
			select {
			case output <- ProposalVoteFromProto(votes.Vote):
				r.log.Debug("votes (party): data sent")
			case <-ctx.Done():
				r.log.Error("votes (party): stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("votes (party): stream closed by server")
				break
			}
		}
	}()
	return output, nil
}

func (r *mySubscriptionResolver) Votes(ctx context.Context, proposalID *string, partyID *string) (<-chan *ProposalVote, error) {
	if proposalID != nil && len(*proposalID) != 0 {
		return r.subscribeProposalVotes(ctx, *proposalID)
	} else if partyID != nil && len(*partyID) != 0 {
		return r.subscribePartyVotes(ctx, *partyID)
	}
	return nil, ErrInvalidVotesSubscription
}

func (r *mySubscriptionResolver) BusEvents(ctx context.Context, types []BusEventType, marketID, partyID *string, batchSize int) (<-chan []*BusEvent, error) {
	if len(types) > 1 {
		return nil, errors.New("busEvents subscription support streaming 1 event at a time for now")
	}
	if len(types) <= 0 {
		return nil, errors.New("busEvents subscription requires 1 event type")
	}
	t := eventTypeToProto(types...)
	req := v2.ObserveEventBusRequest{
		Type:      t,
		BatchSize: int64(batchSize),
	}
	if req.BatchSize == 0 {
		// req.BatchSize = -1 // sending this with -1 to indicate to underlying gRPC call this is a special case: GQL
		batchSize = 0
	}
	if marketID != nil {
		req.MarketId = *marketID
	}
	if partyID != nil {
		req.PartyId = *partyID
	}
	mb := 10
	// about 10MB message size allowed
	msgSize := grpc.MaxCallRecvMsgSize(mb * 10e6)

	// build the bidirectional stream connection
	stream, err := r.tradingDataClientV2.ObserveEventBus(ctx, msgSize)
	if err != nil {
		return nil, err
	}

	// send our initial message to initialize the connection
	if err := stream.Send(&req); err != nil {
		return nil, err
	}

	// we no longer buffer this channel. Client receives batch, then we request the next batch
	out := make(chan []*BusEvent)

	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("Event bus stream close error", logging.Error(err))
			}
			close(out)
		}()

		if batchSize == 0 {
			r.busEvents(ctx, stream, out)
		} else {
			r.busEventsWithBatch(ctx, int64(batchSize), stream, out)
		}
	}()

	return out, nil
}

func (r *mySubscriptionResolver) busEvents(ctx context.Context, stream v2.TradingDataService_ObserveEventBusClient, out chan []*BusEvent) {
	sCtx := stream.Context()
	for {
		// receive batch
		batch, err := stream.Recv()
		if isStreamClosed(err, r.log) {
			return
		}
		if err != nil {
			r.log.Error("Event bus stream error", logging.Error(err))
			return
		}
		select {
		case out <- busEventFromProto(batch.Events...):
			r.log.Debug("bus events: data sent")
		case <-ctx.Done():
			r.log.Debug("bus events: stream closed")
			return
		case <-sCtx.Done():
			r.log.Debug("bus events: stream closed by server")
			return
		}
	}
}

func (r *mySubscriptionResolver) busEventsWithBatch(ctx context.Context, batchSize int64, stream v2.TradingDataService_ObserveEventBusClient, out chan []*BusEvent) {
	sCtx := stream.Context()
	poll := &v2.ObserveEventBusRequest{
		BatchSize: batchSize,
	}
	for {
		// receive batch
		batch, err := stream.Recv()
		if isStreamClosed(err, r.log) {
			return
		}
		if err != nil {
			r.log.Error("Event bus stream error", logging.Error(err))
			return
		}
		select {
		case out <- busEventFromProto(batch.Events...):
			r.log.Debug("bus events: data sent")
		case <-ctx.Done():
			r.log.Debug("bus events: stream closed")
			return
		case <-sCtx.Done():
			r.log.Debug("bus events: stream closed by server")
			return
		}
		// send request for the next batch
		if err := stream.SendMsg(poll); err != nil {
			r.log.Error("Failed to poll next event batch", logging.Error(err))
			return
		}
	}
}

func (r *mySubscriptionResolver) LiquidityProvisions(ctx context.Context, partyID *string, marketID *string) (<-chan []*vegapb.LiquidityProvision, error) {
	req := &v2.ObserveLiquidityProvisionsRequest{
		MarketId: marketID,
		PartyId:  partyID,
	}
	stream, err := r.tradingDataClientV2.ObserveLiquidityProvisions(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*vegapb.LiquidityProvision)
	sCtx := stream.Context()
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("liquidity provisions: failed to close stream", logging.Error(err))
			}
			close(c)
		}()
		for {
			received, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			lps := received.LiquidityProvisions
			if len(lps) == 0 {
				continue
			}
			select {
			case c <- lps:
				r.log.Debug("liquidity provisions: data sent")
			case <-sCtx.Done():
				r.log.Debug("liquidity provisions: stream closed by server")
				break
			case <-ctx.Done():
				r.log.Debug("liquidity provisions: stream closed")
				break
			}
		}
	}()

	return c, nil
}

type myAccountDetailsResolver VegaResolverRoot

func (r *myAccountDetailsResolver) PartyID(ctx context.Context, acc *vegapb.AccountDetails) (*string, error) {
	if acc.Owner != nil {
		return acc.Owner, nil
	}
	return nil, nil
}

// START: Account Resolver

type myAccountResolver VegaResolverRoot

func (r *myAccountResolver) Balance(ctx context.Context, acc *v2.AccountBalance) (string, error) {
	return acc.Balance, nil
}

func (r *myAccountResolver) Market(ctx context.Context, acc *v2.AccountBalance) (*vegapb.Market, error) {
	if acc.MarketId == "" {
		return nil, nil
	}
	return r.r.getMarketByID(ctx, acc.MarketId)
}

func (r *myAccountResolver) Party(ctx context.Context, acc *v2.AccountBalance) (*vegapb.Party, error) {
	if acc.Owner == "" {
		return nil, nil
	}
	return getParty(ctx, r.log, r.r.clt2, acc.Owner)
}

func (r *myAccountResolver) Asset(ctx context.Context, obj *v2.AccountBalance) (*vegapb.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

// START: Account Resolver

type myAccountEventResolver VegaResolverRoot

func (r *myAccountEventResolver) Balance(ctx context.Context, acc *vega.Account) (string, error) {
	return acc.Balance, nil
}

func (r *myAccountEventResolver) Market(ctx context.Context, acc *vega.Account) (*vegapb.Market, error) {
	if acc.MarketId == "" {
		return nil, nil
	}
	return r.r.getMarketByID(ctx, acc.MarketId)
}

func (r *myAccountEventResolver) Party(ctx context.Context, acc *vega.Account) (*vegapb.Party, error) {
	if acc.Owner == "" {
		return nil, nil
	}
	return getParty(ctx, r.log, r.r.clt2, acc.Owner)
}

func (r *myAccountEventResolver) Asset(ctx context.Context, obj *vega.Account) (*vegapb.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

// END: Account Resolver

func getParty(ctx context.Context, _ *logging.Logger, client TradingDataServiceClientV2, id string) (*vegapb.Party, error) {
	if len(id) == 0 {
		return nil, nil
	}
	res, err := client.GetParty(ctx, &v2.GetPartyRequest{PartyId: id})
	if err != nil {
		return nil, err
	}
	return res.Party, nil
}

func (r *myQueryResolver) GetMarketDataHistoryConnectionByID(ctx context.Context, marketID string, start, end *int64, pagination *v2.Pagination) (*v2.MarketDataConnection, error) {
	req := v2.GetMarketDataHistoryByIDRequest{
		MarketId:       marketID,
		StartTimestamp: start,
		EndTimestamp:   end,
		Pagination:     pagination,
	}

	resp, err := r.tradingDataClientV2.GetMarketDataHistoryByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return resp.GetMarketData(), nil
}

func (r *myQueryResolver) MarketsConnection(ctx context.Context, id *string, pagination *v2.Pagination, includeSettled *bool) (*v2.MarketConnection, error) {
	var marketID string

	if id != nil {
		marketID = *id

		resp, err := r.tradingDataClientV2.GetMarket(ctx, &v2.GetMarketRequest{MarketId: marketID})
		if err != nil {
			return nil, err
		}

		connection := &v2.MarketConnection{
			Edges: []*v2.MarketEdge{
				{
					Node:   resp.Market,
					Cursor: "",
				},
			},
			PageInfo: &v2.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     "",
				EndCursor:       "",
			},
		}

		return connection, nil
	}

	resp, err := r.tradingDataClientV2.ListMarkets(ctx, &v2.ListMarketsRequest{
		Pagination:     pagination,
		IncludeSettled: includeSettled,
	})
	if err != nil {
		return nil, err
	}

	return resp.Markets, nil
}

func (r *myQueryResolver) Entities(ctx context.Context, txHash string) (*v2.ListEntitiesResponse, error) {
	resp, err := r.tradingDataClientV2.ListEntities(ctx, &v2.ListEntitiesRequest{
		TransactionHash: txHash,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *myQueryResolver) PartiesConnection(ctx context.Context, partyID *string, pagination *v2.Pagination) (*v2.PartyConnection, error) {
	resp, err := r.tradingDataClientV2.ListParties(ctx, &v2.ListPartiesRequest{
		PartyId:    ptr.UnBox(partyID),
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Parties, nil
}
