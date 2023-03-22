// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	data "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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

func (r *VegaResolverRoot) DataSourceSpecConfiguration() DataSourceSpecConfigurationResolver {
	return (*myDataSourceSpecConfigurationResolver)(r)
}

// MarginLevels returns the market levels resolver.
func (r *VegaResolverRoot) MarginLevels() MarginLevelsResolver {
	return (*myMarginLevelsResolver)(r)
}

// MarginLevelsUpdate returns the market levels resolver.
func (r *VegaResolverRoot) MarginLevelsUpdate() MarginLevelsUpdateResolver {
	return (*myMarginLevelsUpdateResolver)(r)
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

func (r *VegaResolverRoot) PropertyKey() PropertyKeyResolver {
	return (*myPropertyKeyResolver)(r)
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

func (r *VegaResolverRoot) UpdateFutureProduct() UpdateFutureProductResolver {
	return (*updateFutureProductResolver)(r)
}

func (r *VegaResolverRoot) NewMarket() NewMarketResolver {
	return (*newMarketResolver)(r)
}

func (r *VegaResolverRoot) ProposalTerms() ProposalTermsResolver {
	return (*proposalTermsResolver)(r)
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

// RewardSummaryFilter returns RewardSummaryFilterResolver implementation.
func (r *VegaResolverRoot) RewardSummaryFilter() RewardSummaryFilterResolver {
	return (*rewardSummaryFilterResolver)(r)
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
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("Unknown account type %v", obj.RewardType)
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

func (r *myLiquidityOrderReferenceResolver) Order(ctx context.Context, obj *types.LiquidityOrderReference) (*types.Order, error) {
	if len(obj.OrderId) <= 0 {
		return nil, nil
	}
	return r.r.getOrderByID(ctx, obj.OrderId, nil)
}

// deposit resolver

type myDepositResolver VegaResolverRoot

func (r *myDepositResolver) Asset(ctx context.Context, obj *types.Deposit) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *myDepositResolver) Party(_ context.Context, obj *types.Deposit) (*types.Party, error) {
	if len(obj.PartyId) <= 0 {
		return nil, errors.New("missing party ID")
	}
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myDepositResolver) CreatedTimestamp(_ context.Context, obj *types.Deposit) (string, error) {
	if obj.CreatedTimestamp == 0 {
		return "", errors.New("invalid timestamp")
	}
	return vegatime.Format(vegatime.UnixNano(obj.CreatedTimestamp)), nil
}

func (r *myDepositResolver) CreditedTimestamp(_ context.Context, obj *types.Deposit) (*string, error) {
	if obj.CreditedTimestamp == 0 {
		return nil, nil
	}
	t := vegatime.Format(vegatime.UnixNano(obj.CreditedTimestamp))
	return &t, nil
}

// BEGIN: Query Resolver

type myQueryResolver VegaResolverRoot

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

func (r *myQueryResolver) TransfersConnection(ctx context.Context, partyID *string, direction *TransferDirection,
	pagination *v2.Pagination,
) (*v2.TransferConnection, error) {
	return r.r.transfersConnection(ctx, partyID, direction, pagination)
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

func (r *myQueryResolver) OracleSpec(ctx context.Context, id string) (*types.OracleSpec, error) {
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

func (r *myQueryResolver) NetworkParameter(ctx context.Context, key string) (*types.NetworkParameter, error) {
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
			Node: &ERC20MultiSigSignerAddedBundle{
				NewSigner:  edge.Node.NewSigner,
				Submitter:  edge.Node.Submitter,
				Nonce:      edge.Node.Nonce,
				Timestamp:  fmt.Sprint(edge.Node.Timestamp),
				Signatures: edge.Node.Signatures,
				EpochSeq:   edge.Node.EpochSeq,
			},
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
			Node: &ERC20MultiSigSignerRemovedBundle{
				OldSigner:  edge.Node.OldSigner,
				Submitter:  edge.Node.Submitter,
				Nonce:      edge.Node.Nonce,
				Timestamp:  fmt.Sprint(edge.Node.Timestamp),
				Signatures: edge.Node.Signatures,
				EpochSeq:   edge.Node.EpochSeq,
			},
			Cursor: edge.Cursor,
		})
	}

	return &ERC20MultiSigSignerRemovedConnection{
		Edges:    edges,
		PageInfo: res.Bundles.PageInfo,
	}, nil
}

func (r *myQueryResolver) Withdrawal(ctx context.Context, wid string) (*types.Withdrawal, error) {
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

func (r *myQueryResolver) Deposit(ctx context.Context, did string) (*types.Deposit, error) {
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
	order := &types.Order{}

	var err error

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
	if order.TimeInForce == types.Order_TIME_IN_FORCE_GTT && expiration != nil {
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

func (r *myQueryResolver) Asset(ctx context.Context, id string) (*types.Asset, error) {
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

func (r *myQueryResolver) Market(ctx context.Context, id string) (*types.Market, error) {
	return r.r.getMarketByID(ctx, id)
}

func (r *myQueryResolver) Party(ctx context.Context, name string) (*types.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, name)
}

func (r *myQueryResolver) OrderByID(ctx context.Context, orderID string, version *int) (*types.Order, error) {
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

func (r *myQueryResolver) OrderByReference(ctx context.Context, reference string) (*types.Order, error) {
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

func (r *myQueryResolver) Proposal(ctx context.Context, id *string, reference *string) (*types.GovernanceData, error) {
	if id != nil {
		resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
			ProposalId: id,
		})
		if err != nil {
			return nil, err
		}
		return resp.Data, nil
	} else if reference != nil {
		resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
			Reference: reference,
		})
		if err != nil {
			return nil, err
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

func (r *myQueryResolver) NodeData(ctx context.Context) (*types.NodeData, error) {
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

func (r *myQueryResolver) Node(ctx context.Context, id string) (*types.Node, error) {
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

func (r *myQueryResolver) Epoch(ctx context.Context, id *string) (*types.Epoch, error) {
	var epochID *uint64
	if id != nil {
		parsedID, err := strconv.ParseUint(*id, 10, 64)
		if err != nil {
			return nil, err
		}

		epochID = &parsedID
	}

	resp, err := r.tradingDataClientV2.GetEpoch(ctx, &v2.GetEpochRequest{Id: epochID})
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

func (r *myQueryResolver) NetworkLimits(ctx context.Context) (*types.NetworkLimits, error) {
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

// END: Root Resolver

type myNodeSignatureResolver VegaResolverRoot

func (r *myNodeSignatureResolver) Signature(_ context.Context, obj *commandspb.NodeSignature) (*string, error) {
	sig := base64.StdEncoding.EncodeToString(obj.Sig)
	return &sig, nil
}

// BEGIN: Party Resolver

type myPartyResolver VegaResolverRoot

func (r *myPartyResolver) TransfersConnection(
	ctx context.Context,
	party *types.Party,
	direction *TransferDirection,
	pagination *v2.Pagination,
) (*v2.TransferConnection, error) {
	return r.r.transfersConnection(ctx, &party.Id, direction, pagination)
}

func (r *myPartyResolver) RewardsConnection(ctx context.Context, party *types.Party, assetID *string, pagination *v2.Pagination, fromEpoch *int, toEpoch *int) (*v2.RewardsConnection, error) {
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
	}
	resp, err := r.tradingDataClientV2.ListRewards(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rewards information: %w", err)
	}

	return resp.Rewards, nil
}

func (r *myPartyResolver) RewardSummaries(
	ctx context.Context,
	party *types.Party,
	asset *string,
) ([]*types.RewardSummary, error) {
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

func (r *myPartyResolver) StakingSummary(ctx context.Context, party *types.Party, pagination *v2.Pagination) (*StakingSummary, error) {
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
	party *types.Party,
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

func (r *myPartyResolver) MarginsConnection(ctx context.Context, party *types.Party, marketID *string,
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

func (r *myPartyResolver) OrdersConnection(ctx context.Context, party *types.Party, pagination *v2.Pagination, filter *OrderByMarketIdsFilter) (*v2.OrderConnection, error) {
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

func (r *myPartyResolver) TradesConnection(ctx context.Context, party *types.Party, market *string, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	req := v2.ListTradesRequest{
		PartyId:    &party.Id,
		MarketId:   market,
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

func (r *myPartyResolver) PositionsConnection(ctx context.Context, party *types.Party, market *string, pagination *v2.Pagination) (*v2.PositionConnection, error) {
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

func (r *myPartyResolver) AccountsConnection(ctx context.Context, party *types.Party, marketID *string, asset *string, accType *types.AccountType, pagination *v2.Pagination) (*v2.AccountsConnection, error) {
	if party == nil {
		return nil, errors.New("a party must be specified when querying accounts")
	}
	var (
		marketIDs    = []string{}
		mktID        = ""
		asst         = ""
		accountTypes = []types.AccountType{}
		accTy        = types.AccountType_ACCOUNT_TYPE_UNSPECIFIED
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
		if accTy != types.AccountType_ACCOUNT_TYPE_GENERAL &&
			accTy != types.AccountType_ACCOUNT_TYPE_MARGIN &&
			accTy != types.AccountType_ACCOUNT_TYPE_BOND {
			return nil, fmt.Errorf("invalid account type for party %v", accType)
		}
		accountTypes = []types.AccountType{accTy}
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

func (r *myPartyResolver) ProposalsConnection(ctx context.Context, party *types.Party, proposalType *v2.ListGovernanceDataRequest_Type, inState *vega.Proposal_State,
	pagination *v2.Pagination,
) (*v2.GovernanceDataConnection, error) {
	return handleProposalsRequest(ctx, r.tradingDataClientV2, party, nil, proposalType, inState, pagination)
}

func (r *myPartyResolver) WithdrawalsConnection(ctx context.Context, party *types.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.WithdrawalsConnection, error) {
	return handleWithdrawalsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

func (r *myPartyResolver) DepositsConnection(ctx context.Context, party *types.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.DepositsConnection, error) {
	return handleDepositsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

func (r *myPartyResolver) VotesConnection(ctx context.Context, party *types.Party, pagination *v2.Pagination) (*ProposalVoteConnection, error) {
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

func (r *myPartyResolver) DelegationsConnection(ctx context.Context, party *types.Party, nodeID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
	var partyID *string
	if party != nil {
		partyID = &party.Id
	}

	return handleDelegationConnectionRequest(ctx, r.tradingDataClientV2, partyID, nodeID, nil, pagination)
}

// END: Party Resolver

type myMarginLevelsUpdateResolver VegaResolverRoot

func (r *myMarginLevelsUpdateResolver) InitialLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.InitialMargin, nil
}

func (r *myMarginLevelsUpdateResolver) SearchLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.SearchLevel, nil
}

func (r *myMarginLevelsUpdateResolver) MaintenanceLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.MaintenanceMargin, nil
}

// BEGIN: MarginLevels Resolver

type myMarginLevelsResolver VegaResolverRoot

func (r *myMarginLevelsResolver) Market(ctx context.Context, m *types.MarginLevels) (*types.Market, error) {
	return r.r.getMarketByID(ctx, m.MarketId)
}

func (r *myMarginLevelsResolver) Party(ctx context.Context, m *types.MarginLevels) (*types.Party, error) {
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

func (r *myMarginLevelsResolver) Asset(ctx context.Context, m *types.MarginLevels) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, m.Asset)
}

func (r *myMarginLevelsResolver) CollateralReleaseLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.CollateralReleaseLevel, nil
}

func (r *myMarginLevelsResolver) InitialLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.InitialMargin, nil
}

func (r *myMarginLevelsResolver) SearchLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.SearchLevel, nil
}

func (r *myMarginLevelsResolver) MaintenanceLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.MaintenanceMargin, nil
}

// END: MarginLevels Resolver

type myOrderUpdateResolver VegaResolverRoot

func (r *myOrderUpdateResolver) Price(_ context.Context, obj *types.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderUpdateResolver) Size(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderUpdateResolver) Remaining(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderUpdateResolver) CreatedAt(_ context.Context, obj *types.Order) (int64, error) {
	return obj.CreatedAt, nil
}

func (r *myOrderUpdateResolver) UpdatedAt(_ context.Context, obj *types.Order) (*int64, error) {
	var updatedAt *int64
	if obj.UpdatedAt > 0 {
		t := obj.UpdatedAt
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderUpdateResolver) Version(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderUpdateResolver) ExpiresAt(_ context.Context, obj *types.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}

func (r *myOrderUpdateResolver) RejectionReason(_ context.Context, o *types.Order) (*vega.OrderError, error) {
	return o.Reason, nil
}

// BEGIN: Order Resolver

type myOrderResolver VegaResolverRoot

func (r *myOrderResolver) RejectionReason(_ context.Context, o *types.Order) (*vega.OrderError, error) {
	return o.Reason, nil
}

func (r *myOrderResolver) Price(_ context.Context, obj *types.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderResolver) Market(ctx context.Context, obj *types.Order) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myOrderResolver) Size(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderResolver) Remaining(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderResolver) CreatedAt(_ context.Context, obj *types.Order) (int64, error) {
	return obj.CreatedAt, nil
}

func (r *myOrderResolver) UpdatedAt(_ context.Context, obj *types.Order) (*int64, error) {
	var updatedAt *int64
	if obj.UpdatedAt > 0 {
		t := obj.UpdatedAt
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderResolver) Version(_ context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderResolver) ExpiresAt(_ context.Context, obj *types.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}

func (r *myOrderResolver) TradesConnection(ctx context.Context, ord *types.Order, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	req := v2.ListTradesRequest{OrderId: &ord.Id, Pagination: pagination, DateRange: dateRange}
	res, err := r.tradingDataClientV2.ListTrades(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Trades, nil
}

func (r *myOrderResolver) Party(_ context.Context, order *types.Order) (*types.Party, error) {
	if order == nil {
		return nil, errors.New("nil order")
	}
	if len(order.PartyId) == 0 {
		return nil, errors.New("invalid party")
	}
	return &types.Party{Id: order.PartyId}, nil
}

func (r *myOrderResolver) PeggedOrder(_ context.Context, order *types.Order) (*types.PeggedOrder, error) {
	return order.PeggedOrder, nil
}

func (r *myOrderResolver) LiquidityProvision(ctx context.Context, obj *types.Order) (*types.LiquidityProvision, error) {
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

func (r *myTradeResolver) Market(ctx context.Context, obj *types.Trade) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myTradeResolver) Price(_ context.Context, obj *types.Trade) (string, error) {
	return obj.Price, nil
}

func (r *myTradeResolver) Size(_ context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myTradeResolver) CreatedAt(_ context.Context, obj *types.Trade) (int64, error) {
	return obj.Timestamp, nil
}

func (r *myTradeResolver) Buyer(ctx context.Context, obj *types.Trade) (*types.Party, error) {
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

func (r *myTradeResolver) Seller(ctx context.Context, obj *types.Trade) (*types.Party, error) {
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

func (r *myTradeResolver) BuyerAuctionBatch(_ context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.BuyerAuctionBatch)
	return &i, nil
}

func (r *myTradeResolver) BuyerFee(_ context.Context, obj *types.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.BuyerFee != nil {
		fee.MakerFee = obj.BuyerFee.MakerFee
		fee.InfrastructureFee = obj.BuyerFee.InfrastructureFee
		fee.LiquidityFee = obj.BuyerFee.LiquidityFee
	}
	return &fee, nil
}

func (r *myTradeResolver) SellerAuctionBatch(_ context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.SellerAuctionBatch)
	return &i, nil
}

func (r *myTradeResolver) SellerFee(_ context.Context, obj *types.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.SellerFee != nil {
		fee.MakerFee = obj.SellerFee.MakerFee
		fee.InfrastructureFee = obj.SellerFee.InfrastructureFee
		fee.LiquidityFee = obj.SellerFee.LiquidityFee
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

// END: Candle Resolver

// BEGIN: DataSourceSpecConfiguration Resolver.
type myDataSourceSpecConfigurationResolver VegaResolverRoot

func (m *myDataSourceSpecConfigurationResolver) Signers(_ context.Context, obj *types.DataSourceSpecConfiguration) ([]*Signer, error) {
	return resolveSigners(obj.Signers), nil
}

// END: DataSourceSpecConfiguration Resolver

// BEGIN: Price Level Resolver

type myPriceLevelResolver VegaResolverRoot

func (r *myPriceLevelResolver) Price(_ context.Context, obj *types.PriceLevel) (string, error) {
	return obj.Price, nil
}

func (r *myPriceLevelResolver) Volume(_ context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myPriceLevelResolver) NumberOfOrders(_ context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.NumberOfOrders, 10), nil
}

// END: Price Level Resolver

type positionUpdateResolver VegaResolverRoot

func (r *positionUpdateResolver) OpenVolume(_ context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *positionUpdateResolver) UpdatedAt(_ context.Context, obj *types.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *positionUpdateResolver) LossSocializationAmount(_ context.Context, obj *types.Position) (string, error) {
	return obj.LossSocialisationAmount, nil
}

// BEGIN: Position Resolver

type myPositionResolver VegaResolverRoot

func (r *myPositionResolver) Market(ctx context.Context, obj *types.Position) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myPositionResolver) UpdatedAt(_ context.Context, obj *types.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myPositionResolver) OpenVolume(_ context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *myPositionResolver) RealisedPnl(_ context.Context, obj *types.Position) (string, error) {
	return obj.RealisedPnl, nil
}

func (r *myPositionResolver) UnrealisedPnl(_ context.Context, obj *types.Position) (string, error) {
	return obj.UnrealisedPnl, nil
}

func (r *myPositionResolver) AverageEntryPrice(_ context.Context, obj *types.Position) (string, error) {
	return obj.AverageEntryPrice, nil
}

func (r *myPositionResolver) LossSocializationAmount(_ context.Context, obj *types.Position) (string, error) {
	return obj.LossSocialisationAmount, nil
}

func (r *myPositionResolver) Party(ctx context.Context, obj *types.Position) (*types.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, obj.PartyId)
}

func (r *myPositionResolver) MarginsConnection(ctx context.Context, pos *types.Position, pagination *v2.Pagination) (*v2.MarginConnection, error) {
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

func (r *mySubscriptionResolver) Delegations(ctx context.Context, party, nodeID *string) (<-chan *types.Delegation, error) {
	req := &v2.ObserveDelegationsRequest{
		PartyId: party,
		NodeId:  nodeID,
	}
	stream, err := r.tradingDataClientV2.ObserveDelegations(ctx, req)
	if err != nil {
		return nil, err
	}

	sCtx := stream.Context()
	ch := make(chan *types.Delegation)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("delegations: stream closed", logging.Error(err))
			}
			close(ch)
		}()
		for {
			dl, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("delegations: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("delegations levls: stream closed", logging.Error(err))
				break
			}
			select {
			case ch <- dl.Delegation:
				r.log.Debug("delegations: data sent")
			case <-ctx.Done():
				r.log.Error("delegations: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("delegations: stream closed by server")
				break
			}
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) Rewards(ctx context.Context, assetID, party *string) (<-chan *types.Reward, error) {
	req := &v2.ObserveRewardsRequest{
		AssetId: assetID,
		PartyId: party,
	}
	stream, err := r.tradingDataClientV2.ObserveRewards(ctx, req)
	if err != nil {
		return nil, err
	}

	sCtx := stream.Context()
	ch := make(chan *types.Reward)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				r.log.Error("rewards: stream closed", logging.Error(err))
			}
			close(ch)
		}()
		for {
			rd, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("reward details: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("reward details: stream closed", logging.Error(err))
				break
			}
			select {
			case ch <- rd.Reward:
				r.log.Debug("rewards: data sent")
			case <-ctx.Done():
				r.log.Error("rewards: stream closed")
				break
			case <-sCtx.Done():
				r.log.Error("rewards: stream closed by server")
				break
			}
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) Margins(ctx context.Context, partyID string, marketID *string) (<-chan *types.MarginLevels, error) {
	req := &v2.ObserveMarginLevelsRequest{
		MarketId: marketID,
		PartyId:  partyID,
	}
	stream, err := r.tradingDataClientV2.ObserveMarginLevels(ctx, req)
	if err != nil {
		return nil, err
	}

	sCtx := stream.Context()
	ch := make(chan *types.MarginLevels)
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

func (r *mySubscriptionResolver) Accounts(ctx context.Context, marketID *string, partyID *string, asset *string, typeArg *types.AccountType) (<-chan []*v2.AccountBalance, error) {
	var (
		mkt, pty, ast string
		ty            types.AccountType
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

func (r *mySubscriptionResolver) Orders(ctx context.Context, filter *OrderByMarketAndPartyIdsFilter) (<-chan []*types.Order, error) {
	req := &v2.ObserveOrdersRequest{}
	if filter != nil {
		req.MarketIds = filter.MarketIds
		req.PartyIds = filter.PartyIds
	}

	stream, err := r.tradingDataClientV2.ObserveOrders(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*types.Order)
	var orders []*types.Order
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
			orders = orders[:0]
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

func (r *mySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []*types.Trade, error) {
	req := &v2.ObserveTradesRequest{
		MarketId: market,
		PartyId:  party,
	}
	stream, err := r.tradingDataClientV2.ObserveTrades(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*types.Trade)
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

func (r *mySubscriptionResolver) Positions(ctx context.Context, party, market *string) (<-chan []*types.Position, error) {
	req := &v2.ObservePositionsRequest{
		PartyId:  party,
		MarketId: market,
	}
	stream, err := r.tradingDataClientV2.ObservePositions(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*types.Position)
	var positions []*types.Position
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
	var candleInterval types.Interval
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

func (r *mySubscriptionResolver) subscribeAllProposals(ctx context.Context) (<-chan *types.GovernanceData, error) {
	stream, err := r.tradingDataClientV2.ObserveGovernance(ctx, &v2.ObserveGovernanceRequest{})
	if err != nil {
		return nil, err
	}
	output := make(chan *types.GovernanceData)
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

func (r *mySubscriptionResolver) subscribePartyProposals(ctx context.Context, partyID string) (<-chan *types.GovernanceData, error) {
	stream, err := r.tradingDataClientV2.ObserveGovernance(ctx, &v2.ObserveGovernanceRequest{
		PartyId: &partyID,
	})
	if err != nil {
		return nil, err
	}
	sCtx := stream.Context()
	output := make(chan *types.GovernanceData)
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

func (r *mySubscriptionResolver) Proposals(ctx context.Context, partyID *string) (<-chan *types.GovernanceData, error) {
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

func (r *mySubscriptionResolver) LiquidityProvisions(ctx context.Context, partyID *string, marketID *string) (<-chan []*types.LiquidityProvision, error) {
	req := &v2.ObserveLiquidityProvisionsRequest{
		MarketId: marketID,
		PartyId:  partyID,
	}
	stream, err := r.tradingDataClientV2.ObserveLiquidityProvisions(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []*types.LiquidityProvision)
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

func (r *myAccountDetailsResolver) PartyID(ctx context.Context, acc *types.AccountDetails) (*string, error) {
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

func (r *myAccountResolver) Market(ctx context.Context, acc *v2.AccountBalance) (*types.Market, error) {
	if acc.MarketId == "" {
		return nil, nil
	}
	return r.r.getMarketByID(ctx, acc.MarketId)
}

func (r *myAccountResolver) Party(ctx context.Context, acc *v2.AccountBalance) (*types.Party, error) {
	if acc.Owner == "" {
		return nil, nil
	}
	return getParty(ctx, r.log, r.r.clt2, acc.Owner)
}

func (r *myAccountResolver) Asset(ctx context.Context, obj *v2.AccountBalance) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

// START: Account Resolver

type myAccountEventResolver VegaResolverRoot

func (r *myAccountEventResolver) Balance(ctx context.Context, acc *vega.Account) (string, error) {
	return acc.Balance, nil
}

func (r *myAccountEventResolver) Market(ctx context.Context, acc *vega.Account) (*types.Market, error) {
	if acc.MarketId == "" {
		return nil, nil
	}
	return r.r.getMarketByID(ctx, acc.MarketId)
}

func (r *myAccountEventResolver) Party(ctx context.Context, acc *vega.Account) (*types.Party, error) {
	if acc.Owner == "" {
		return nil, nil
	}
	return getParty(ctx, r.log, r.r.clt2, acc.Owner)
}

func (r *myAccountEventResolver) Asset(ctx context.Context, obj *vega.Account) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

// END: Account Resolver

func getParty(ctx context.Context, _ *logging.Logger, client TradingDataServiceClientV2, id string) (*types.Party, error) {
	if len(id) == 0 {
		return nil, nil
	}
	res, err := client.GetParty(ctx, &v2.GetPartyRequest{PartyId: id})
	if err != nil {
		return nil, err
	}
	return res.Party, nil
}

// Market Data Resolvers.
type myPropertyKeyResolver VegaResolverRoot

func (r *myPropertyKeyResolver) NumberDecimalPlaces(ctx context.Context, obj *data.PropertyKey) (*int, error) {
	ndp := obj.NumberDecimalPlaces
	if ndp == nil {
		return nil, nil
	}
	indp := new(int)
	*indp = int(*ndp)
	return indp, nil
}

// GetMarketDataHistoryByID returns all the market data information for a given market between the dates specified.
func (r *myQueryResolver) GetMarketDataHistoryByID(ctx context.Context, id string, start, end *int64, skip, first, last *int) ([]*types.MarketData, error) {
	pagination := makeAPIV2Pagination(skip, first, last)

	return r.getMarketDataHistoryByID(ctx, id, start, end, pagination)
}

func makeAPIV2Pagination(skip, first, last *int) *v2.OffsetPagination {
	var (
		offset, limit uint64
		descending    bool
	)
	if skip != nil {
		offset = uint64(*skip)
	}
	if last != nil {
		limit = uint64(*last)
		descending = true
	} else if first != nil {
		limit = uint64(*first)
	}
	return &v2.OffsetPagination{
		Skip:       offset,
		Limit:      limit,
		Descending: descending,
	}
}

func (r *myQueryResolver) getMarketData(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) ([]*types.MarketData, error) {
	resp, err := r.tradingDataClientV2.GetMarketDataHistoryByID(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.MarketData == nil {
		return nil, errors.New("no market data not found")
	}

	results := make([]*types.MarketData, 0, len(resp.MarketData.Edges))

	for _, edge := range resp.MarketData.Edges {
		results = append(results, edge.Node)
	}

	return results, nil
}

func (r *myQueryResolver) getMarketDataHistoryByID(ctx context.Context, id string, start, end *int64, pagination *v2.OffsetPagination) ([]*types.MarketData, error) {
	var startTime, endTime *int64

	if start != nil {
		s := time.Unix(*start, 0).UnixNano()
		startTime = &s
	}

	if end != nil {
		e := time.Unix(*end, 0).UnixNano()
		endTime = &e
	}

	req := v2.GetMarketDataHistoryByIDRequest{
		MarketId:         id,
		StartTimestamp:   startTime,
		EndTimestamp:     endTime,
		OffsetPagination: pagination,
	}

	return r.getMarketData(ctx, &req)
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
