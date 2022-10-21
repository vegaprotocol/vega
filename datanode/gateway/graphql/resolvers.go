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

	"google.golang.org/grpc"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/datanode/gateway/graphql CoreProxyServiceClient,TradingDataServiceClientV2,TradingDataServiceClient

// CoreProxyServiceClient ...
type CoreProxyServiceClient interface {
	vegaprotoapi.CoreServiceClient
}

// TradingDataServiceClient ...
type TradingDataServiceClient interface {
	protoapi.TradingDataServiceClient
}

type TradingDataServiceClientV2 interface {
	v2.TradingDataServiceClient
}

// VegaResolverRoot is the root resolver for all graphql types.
type VegaResolverRoot struct {
	gateway.Config

	log                 *logging.Logger
	tradingProxyClient  CoreProxyServiceClient
	tradingDataClient   TradingDataServiceClient
	tradingDataClientV2 TradingDataServiceClientV2
	r                   allResolver
}

// NewResolverRoot instantiate a graphql root resolver.
func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient CoreProxyServiceClient,
	tradingDataClient TradingDataServiceClient,
	tradingDataClientV2 TradingDataServiceClientV2,
) *VegaResolverRoot {
	return &VegaResolverRoot{
		log:                 log,
		Config:              config,
		tradingProxyClient:  tradingClient,
		tradingDataClient:   tradingDataClient,
		tradingDataClientV2: tradingDataClientV2,
		r:                   allResolver{log, tradingDataClient, tradingDataClientV2},
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

func (r *VegaResolverRoot) AccountEdge() AccountEdgeResolver {
	return (*myAccountEdgeResolver)(r)
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

func (r *VegaResolverRoot) MarketTimestamps() MarketTimestampsResolver {
	return (*marketTimestampsResolver)(r)
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

// TODO: RewardPerAssetDetail is deprecated, remove once front end has caught up.
func (r *VegaResolverRoot) RewardPerAssetDetail() RewardPerAssetDetailResolver {
	return (*rewardPerAssetDetailResolver)(r)
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

func (r *VegaResolverRoot) OneOffTransfer() OneOffTransferResolver {
	return (*oneoffTransferResolver)(r)
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

type transactionResultResolver VegaResolverRoot

func (r *transactionResultResolver) Error(ctx context.Context, tr *eventspb.TransactionResult) (*string, error) {
	if tr == nil || tr.Status {
		return nil, nil
	}

	return &tr.GetFailure().Error, nil
}

type accountUpdateResolver VegaResolverRoot

func (r *accountUpdateResolver) AssetID(ctx context.Context, obj *v2.AccountBalance) (string, error) {
	return obj.Asset, nil
}

// AggregatedLedgerEntriesResolver resolver.
type aggregatedLedgerEntriesResolver VegaResolverRoot

func (r *VegaResolverRoot) AggregatedLedgerEntries() AggregatedLedgerEntriesResolver {
	return (*aggregatedLedgerEntriesResolver)(r)
}

func (r *aggregatedLedgerEntriesResolver) VegaTime(ctx context.Context, obj *v2.AggregatedLedgerEntries) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}

func (r *aggregatedLedgerEntriesResolver) TransferType(ctx context.Context, obj *v2.AggregatedLedgerEntries) (*string, error) {
	tt := obj.TransferType.String()
	return &tt, nil
}

// LedgerEntryFilterResolver resolver.
type ledgerEntryFilterResolver VegaResolverRoot

func (r *VegaResolverRoot) LedgerEntryFilter() LedgerEntryFilterResolver {
	return (*ledgerEntryFilterResolver)(r)
}

func (r *ledgerEntryFilterResolver) TransferTypes(ctx context.Context, obj *v2.LedgerEntryFilter, data []*TransferType) error {
	return nil
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

func (r *myDepositResolver) Party(ctx context.Context, obj *types.Deposit) (*types.Party, error) {
	if len(obj.PartyId) <= 0 {
		return nil, errors.New("missing party ID")
	}
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myDepositResolver) CreatedTimestamp(ctx context.Context, obj *types.Deposit) (string, error) {
	if obj.CreatedTimestamp == 0 {
		return "", errors.New("invalid timestamp")
	}
	return vegatime.Format(vegatime.UnixNano(obj.CreatedTimestamp)), nil
}

func (r *myDepositResolver) CreditedTimestamp(ctx context.Context, obj *types.Deposit) (*string, error) {
	if obj.CreditedTimestamp == 0 {
		return nil, nil
	}
	t := vegatime.Format(vegatime.UnixNano(obj.CreditedTimestamp))
	return &t, nil
}

// BEGIN: Query Resolver

type myQueryResolver VegaResolverRoot

// Deprecated: Use TransfersConnection instead.
func (r *myQueryResolver) Transfers(
	ctx context.Context, pubkey string, isFrom *bool, isTo *bool,
) ([]*eventspb.Transfer, error) {
	from := false
	to := false

	if isFrom != nil {
		from = *isFrom
	}

	if isTo != nil {
		to = *isTo
	}

	response, err := r.tradingDataClient.Transfers(ctx, &protoapi.TransfersRequest{
		Pubkey: pubkey,
		IsFrom: from,
		IsTo:   to,
	})
	if err != nil {
		return nil, err
	}

	return response.Transfers, nil
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

// Deprecated: Use OracleSpecsConnection.
func (r *myQueryResolver) OracleSpecs(ctx context.Context, pagination *OffsetPagination) ([]*oraclespb.OracleSpec, error) {
	paginationProto, err := pagination.ToProto()
	if err != nil {
		return nil, fmt.Errorf("invalid pagination object: %w", err)
	}
	res, err := r.tradingDataClient.OracleSpecs(
		ctx, &protoapi.OracleSpecsRequest{
			Pagination: &paginationProto,
		},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleSpecs, nil
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

func (r *myQueryResolver) OracleSpec(ctx context.Context, id string) (*oraclespb.OracleSpec, error) {
	res, err := r.tradingDataClientV2.GetOracleSpec(
		ctx, &v2.GetOracleSpecRequest{OracleSpecId: id},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleSpec, nil
}

// Deprecated: Use OracleDataBySpecConnection instead.
func (r *myQueryResolver) OracleDataBySpec(ctx context.Context, id string,
	pagination *OffsetPagination,
) ([]*oraclespb.OracleData, error) {
	paginationProto, err := pagination.ToProto()
	if err != nil {
		return nil, fmt.Errorf("invalid pagination object: %w", err)
	}

	res, err := r.tradingDataClient.OracleDataBySpec(
		ctx, &protoapi.OracleDataBySpecRequest{
			Id:         id,
			Pagination: &paginationProto,
		},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleData, nil
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

// Deprecated: Use OracleDataConnection instead.
func (r *myQueryResolver) OracleData(ctx context.Context, pagination *OffsetPagination) ([]*oraclespb.OracleData, error) {
	paginationProto, err := pagination.ToProto()
	if err != nil {
		return nil, fmt.Errorf("invalid pagination object: %w", err)
	}

	res, err := r.tradingDataClient.ListOracleData(
		ctx, &protoapi.ListOracleDataRequest{
			Pagination: &paginationProto,
		},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleData, nil
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

// Deprecated: Use NetworkParametersConnection instead.
func (r *myQueryResolver) NetworkParameters(ctx context.Context) ([]*types.NetworkParameter, error) {
	res, err := r.tradingDataClient.NetworkParameters(
		ctx, &protoapi.NetworkParametersRequest{},
	)
	if err != nil {
		return nil, err
	}

	return res.NetworkParameters, nil
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
		Expiry:        strconv.FormatInt(res.Expiry, 10),
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
			Submitter:  fromPtr(submitter),
			EpochSeq:   fromPtr(epochSeq),
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
			Submitter:  fromPtr(submitter),
			EpochSeq:   fromPtr(epochSeq),
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

func fromPtr[T any](ptr *T) (ret T) {
	if ptr != nil {
		ret = *ptr
	}
	return
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
	expiration *string, ty vega.Order_Type,
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
		var expiresAt time.Time
		expiresAt, err = vegatime.Parse(*expiration)
		if err != nil {
			return nil, fmt.Errorf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration)
		}

		// move to pure timestamps or convert an RFC format shortly
		order.ExpiresAt = expiresAt.UnixNano()
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
		return nil, customErrorFromStatus(err)
	}

	// calclate the fee total amount
	ttf := resp.Fee.MakerFee + resp.Fee.InfrastructureFee + resp.Fee.LiquidityFee

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
		return nil, customErrorFromStatus(err)
	}

	return &OrderEstimate{
		Fee:            &fee,
		TotalFeeAmount: ttf,
		MarginLevels:   respm.MarginLevels,
	}, nil
}

func (r *myQueryResolver) Asset(ctx context.Context, id string) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, id)
}

func (r *myQueryResolver) Assets(ctx context.Context) ([]*types.Asset, error) {
	return r.r.allAssets(ctx)
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

// Deprecated: Use NodeSignaturesConnection instead.
func (r *myQueryResolver) NodeSignatures(ctx context.Context, resourceID string) ([]*commandspb.NodeSignature, error) {
	if len(resourceID) <= 0 {
		return nil, ErrMissingIDOrReference
	}

	req := &protoapi.GetNodeSignaturesAggregateRequest{
		Id: resourceID,
	}
	res, err := r.tradingDataClient.GetNodeSignaturesAggregate(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Signatures, nil
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

// Deprecated: Use MarketsConnection instead.
func (r *myQueryResolver) Markets(ctx context.Context, id *string) ([]*types.Market, error) {
	return r.r.allMarkets(ctx, id)
}

func (r *myQueryResolver) Market(ctx context.Context, id string) (*types.Market, error) {
	return r.r.getMarketByID(ctx, id)
}

// Deprecated: Use PartiesConnection instead.
func (r *myQueryResolver) Parties(ctx context.Context, name *string) ([]*types.Party, error) {
	if name == nil {
		var empty protoapi.PartiesRequest
		resp, err := r.tradingDataClient.Parties(ctx, &empty)
		if err != nil {
			return nil, err
		}
		if resp.Parties == nil {
			return []*types.Party{}, nil
		}
		return resp.Parties, nil
	}
	party, err := r.Party(ctx, *name)
	if err != nil {
		return nil, err
	}

	// if we asked for a single party it may be null
	// so then we return an empty slice
	if party == nil {
		return []*types.Party{}, nil
	}

	return []*types.Party{party}, nil
}

func (r *myQueryResolver) Party(ctx context.Context, name string) (*types.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, name)
}

func (r *myQueryResolver) OrderByID(ctx context.Context, orderID string, version *int) (*types.Order, error) {
	return r.r.getOrderByID(ctx, orderID, version)
}

// Deprecated: Use OrderVersionsConnection instead.
func (r *myQueryResolver) OrderVersions(
	ctx context.Context, orderID string, skip, first, last *int,
) ([]*types.Order, error) {
	p := makePagination(skip, first, last)
	reqest := &protoapi.OrderVersionsByIDRequest{
		OrderId:    orderID,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrderVersionsByID(ctx, reqest)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Orders, nil
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
		return nil, customErrorFromStatus(err)
	}
	return resp.Orders, nil
}

func (r *myQueryResolver) OrderByReference(ctx context.Context, reference string) (*types.Order, error) {
	req := &v2.ListOrdersRequest{
		Reference: &reference,
	}
	res, err := r.tradingDataClientV2.ListOrders(ctx, req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Orders.Edges) == 0 {
		return nil, fmt.Errorf("order reference not found: %s", reference)
	}

	return res.Orders.Edges[0].Node, nil
}

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) Proposals(ctx context.Context, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetProposals(ctx, &protoapi.GetProposalsRequest{
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
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

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) NewMarketProposals(ctx context.Context, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetNewMarketProposals(ctx, &protoapi.GetNewMarketProposalsRequest{
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) UpdateMarketProposals(ctx context.Context, marketID *string, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	var market string
	if marketID != nil {
		market = *marketID
	}
	resp, err := r.tradingDataClient.GetUpdateMarketProposals(ctx, &protoapi.GetUpdateMarketProposalsRequest{
		MarketId:      market,
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) NetworkParametersProposals(ctx context.Context, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetNetworkParametersProposals(ctx, &protoapi.GetNetworkParametersProposalsRequest{
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) NewAssetProposals(ctx context.Context, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetNewAssetProposals(ctx, &protoapi.GetNewAssetProposalsRequest{
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Deprecated: Use ProposalsConnection instead.
func (r *myQueryResolver) NewFreeformProposals(ctx context.Context, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetNewFreeformProposals(ctx, &protoapi.GetNewFreeformProposalsRequest{
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myQueryResolver) NodeData(ctx context.Context) (*types.NodeData, error) {
	resp, err := r.tradingDataClientV2.GetNetworkData(ctx, &v2.GetNetworkDataRequest{})
	if err != nil {
		return nil, err
	}

	return resp.NodeData, nil
}

// Deprecated: Use NodesConnection instead.
func (r *myQueryResolver) Nodes(ctx context.Context) ([]*types.Node, error) {
	resp, err := r.tradingDataClient.GetNodes(ctx, &protoapi.GetNodesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Nodes, nil
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

// Deprecated: Use KeyRotationConnection instead.
func (r *myQueryResolver) KeyRotations(ctx context.Context, id *string) ([]*eventspb.KeyRotation, error) {
	if id != nil {
		resp, err := r.tradingDataClient.GetKeyRotationsByNode(ctx, &protoapi.GetKeyRotationsByNodeRequest{NodeId: *id})
		if err != nil {
			return nil, err
		}

		return dataNodeV1KeyRotationsToVegaEvent(resp.Rotations...), nil
	}

	resp, err := r.tradingDataClient.GetKeyRotations(ctx, &protoapi.GetKeyRotationsRequest{})
	if err != nil {
		return nil, err
	}

	return dataNodeV1KeyRotationsToVegaEvent(resp.Rotations...), nil
}

func dataNodeV1KeyRotationsToVegaEvent(rotations ...*protoapi.KeyRotation) []*eventspb.KeyRotation {
	events := make([]*eventspb.KeyRotation, len(rotations))
	for i, v := range rotations {
		events[i] = &eventspb.KeyRotation{
			NodeId:      v.NodeId,
			OldPubKey:   v.OldPubKey,
			NewPubKey:   v.NewPubKey,
			BlockHeight: v.BlockHeight,
		}
	}
	return events
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
	sumParties, sumMarkets, sumTypes *bool,
	dateRange *v2.DateRange,
	pagination *v2.Pagination,
) (*v2.AggregatedBalanceConnection, error) {
	req := &v2.ListBalanceChangesRequest{
		Filter:           filter,
		SumAcrossParties: ptr.UnBox(sumParties),
		SumAcrossMarkets: ptr.UnBox(sumMarkets),
		SumAcrossTypes:   ptr.UnBox(sumTypes),
		DateRange:        dateRange,
		Pagination:       pagination,
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
	groupOptions *GroupOptions,
	dateRange *v2.DateRange,
	pagination *v2.Pagination,
) (*v2.AggregatedLedgerEntriesConnection, error) {
	req := &v2.ListLedgerEntriesRequest{}
	req.Filter = filter

	groupByAccountField := []v2.AccountField{}
	groupByLedgerEntryField := []v2.LedgerEntryField{}
	if groupOptions != nil {
		if groupOptions.ByAccountField != nil {
			for _, af := range groupOptions.ByAccountField {
				groupByAccountField = append(groupByAccountField, *af)
			}
		}

		if groupOptions.ByLedgerEntryField != nil {
			for _, lf := range groupOptions.ByLedgerEntryField {
				groupByLedgerEntryField = append(groupByLedgerEntryField, *lf)
			}
		}

		req.GroupOptions = &v2.GroupOptions{
			ByAccountField:     groupByAccountField,
			ByLedgerEntryField: groupByLedgerEntryField,
		}
	}

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
	req := &v2.GetMostRecentDeHistorySegmentRequest{}

	resp, err := r.tradingDataClientV2.GetMostRecentDeHistorySegment(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetSegment(), nil
}

// END: Root Resolver

type myNodeSignatureResolver VegaResolverRoot

func (r *myNodeSignatureResolver) Signature(ctx context.Context, obj *commandspb.NodeSignature) (*string, error) {
	sig := base64.StdEncoding.EncodeToString(obj.Sig)
	return &sig, nil
}

// BEGIN: Party Resolver

type myPartyResolver VegaResolverRoot

// func makePagination(skip, first, last *int) *protoapi.Pagination {.
func makePagination(skip, first, last *int) *protoapi.Pagination {
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
	return &protoapi.Pagination{
		Skip:       offset,
		Limit:      limit,
		Descending: descending,
	}
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

// TODO: RewardDetails have been depricated, remove once front end catches up.
func (r *myPartyResolver) RewardDetails(
	ctx context.Context,
	party *types.Party,
) ([]*types.RewardSummary, error) {
	req := &protoapi.GetRewardSummariesRequest{
		PartyId: party.Id,
	}
	resp, err := r.tradingDataClient.GetRewardSummaries(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Summaries, err
}

// Deprecated: Use RewardsConnection instead.
func (r *myPartyResolver) Rewards(
	ctx context.Context,
	party *types.Party,
	asset *string,
	skip, first, last *int,
) ([]*types.Reward, error) {
	var assetID string
	if asset != nil {
		assetID = *asset
	}

	p := makePagination(skip, first, last)

	req := &protoapi.GetRewardsRequest{
		PartyId:    party.Id,
		AssetId:    assetID,
		Pagination: p,
	}
	resp, err := r.tradingDataClient.GetRewards(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Rewards, err
}

func (r *myPartyResolver) TransfersConnection(
	ctx context.Context,
	party *types.Party,
	direction *TransferDirection,
	pagination *v2.Pagination,
) (*v2.TransferConnection, error) {
	return r.r.transfersConnection(ctx, &party.Id, direction, pagination)
}

func (r *myPartyResolver) RewardsConnection(ctx context.Context, party *types.Party, assetID *string, pagination *v2.Pagination) (*v2.RewardsConnection, error) {
	req := v2.ListRewardsRequest{
		PartyId:    party.Id,
		AssetId:    assetID,
		Pagination: pagination,
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

// Deprecated: Use StakingSummary instead.
func (r *myPartyResolver) Stake(
	ctx context.Context,
	party *types.Party,
) (*protoapi.PartyStakeResponse, error) {
	return r.tradingDataClient.PartyStake(
		ctx, &protoapi.PartyStakeRequest{
			Party: party.Id,
		},
	)
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

// Deprecated: Use LiquidityProvisionsConnection instead.
func (r *myPartyResolver) LiquidityProvisions(
	ctx context.Context,
	party *types.Party,
	market, ref *string,
) ([]*types.LiquidityProvision, error) {
	var mid string
	if market != nil {
		mid = *market
	}

	req := protoapi.LiquidityProvisionsRequest{
		Party:  party.Id,
		Market: mid,
	}
	res, err := r.tradingDataClient.LiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	var out []*types.LiquidityProvision
	if ref != nil {
		for _, v := range res.LiquidityProvisions {
			if v.Reference == *ref {
				out = append(out, v)
			}
		}
	} else {
		out = res.LiquidityProvisions
	}

	return out, nil
}

func (r *myPartyResolver) LiquidityProvisionsConnection(
	ctx context.Context,
	party *types.Party,
	market, ref *string,
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

	req := v2.ListLiquidityProvisionsRequest{
		PartyId:    &partyID,
		MarketId:   &mid,
		Reference:  &refID,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListLiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return res.LiquidityProvisions, nil
}

// Deprecated: Use MarginsConnection instead.
func (r *myPartyResolver) Margins(ctx context.Context,
	party *types.Party, marketID *string,
) ([]*types.MarginLevels, error) {
	req := protoapi.MarginLevelsRequest{
		PartyId: party.Id,
	}
	if marketID != nil {
		req.MarketId = *marketID
	}

	res, err := r.tradingDataClient.MarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	out := make([]*types.MarginLevels, 0, len(res.MarginLevels))
	out = append(out, res.MarginLevels...)
	return out, nil
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
		return nil, customErrorFromStatus(err)
	}

	return res.MarginLevels, nil
}

// Deprecated: Use OrdersConnection instead.
func (r *myPartyResolver) Orders(ctx context.Context, party *types.Party,
	skip, first, last *int,
) ([]*types.Order, error) {
	p := makePagination(skip, first, last)
	req := protoapi.OrdersByPartyRequest{
		PartyId:    party.Id,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrdersByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Orders) > 0 {
		return res.Orders, nil
	}
	// mandatory return field in schema
	return []*types.Order{}, nil
}

func (r *myPartyResolver) OrdersConnection(ctx context.Context, party *types.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.OrderConnection, error) {
	if party == nil {
		return nil, errors.New("party is required")
	}
	req := v2.ListOrdersRequest{
		PartyId:    &party.Id,
		Pagination: pagination,
		DateRange:  dateRange,
	}
	res, err := r.tradingDataClientV2.ListOrders(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Orders, nil
}

// Deprecated: Use TradesConnection instead.
func (r *myPartyResolver) Trades(ctx context.Context, party *types.Party,
	market *string, skip, first, last *int,
) ([]*types.Trade, error) {
	var mkt string
	if market != nil {
		mkt = *market
	}

	p := makePagination(skip, first, last)
	req := protoapi.TradesByPartyRequest{
		PartyId:    party.Id,
		MarketId:   mkt,
		Pagination: p,
	}

	res, err := r.tradingDataClient.TradesByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Trades) > 0 {
		return res.Trades, nil
	}
	// mandatory return field in schema
	return []*types.Trade{}, nil
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
		return nil, customErrorFromStatus(err)
	}
	return res.Trades, nil
}

// Deprecated: Use PositionsConnection instead.
func (r *myPartyResolver) Positions(ctx context.Context, party *types.Party) ([]*types.Position, error) {
	if party == nil {
		return nil, errors.New("nil party")
	}
	req := protoapi.PositionsByPartyRequest{PartyId: party.Id}
	res, err := r.tradingDataClient.PositionsByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	if len(res.Positions) > 0 {
		return res.Positions, nil
	}
	// mandatory return field in schema
	return []*types.Position{}, nil
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
		return nil, customErrorFromStatus(err)
	}

	return res.Positions, nil
}

// Deprecated: use accountConnection instead.
func (r *myPartyResolver) Accounts(ctx context.Context, party *types.Party,
	marketID *string, asset *string, accType *types.AccountType,
) ([]*v2.AccountBalance, error) {
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

	req := v2.ListAccountsRequest{Filter: &filter}
	res, err := r.tradingDataClientV2.ListAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get Party account",
			logging.Error(err),
			logging.String("party-id", party.Id),
			logging.String("market-id", mktID),
			logging.String("asset", asst),
			logging.String("type", accTy.String()))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Accounts.Edges) == 0 {
		// mandatory return field in schema
		return []*v2.AccountBalance{}, nil
	}

	accounts := make([]*v2.AccountBalance, len(res.Accounts.Edges))
	for i, edge := range res.Accounts.Edges {
		accounts[i] = edge.Account
	}
	return accounts, nil
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
		return nil, customErrorFromStatus(err)
	}

	return res.Accounts, nil
}

// Deprecated: use ProposalsConnection instead.
func (r *myPartyResolver) Proposals(ctx context.Context, party *types.Party, inState *vega.Proposal_State) ([]*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetProposalsByParty(ctx, &protoapi.GetProposalsByPartyRequest{
		PartyId:       party.Id,
		SelectInState: inState,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myPartyResolver) ProposalsConnection(ctx context.Context, party *types.Party, proposalType *v2.ListGovernanceDataRequest_Type, inState *vega.Proposal_State,
	pagination *v2.Pagination,
) (*v2.GovernanceDataConnection, error) {
	return handleProposalsRequest(ctx, r.tradingDataClientV2, party, nil, proposalType, inState, pagination)
}

// Deprecated: Use WithdrawalsConnection instead.
func (r *myPartyResolver) Withdrawals(ctx context.Context, party *types.Party) ([]*types.Withdrawal, error) {
	res, err := r.tradingDataClient.Withdrawals(
		ctx, &protoapi.WithdrawalsRequest{PartyId: party.Id},
	)
	if err != nil {
		return nil, err
	}

	return res.Withdrawals, nil
}

func (r *myPartyResolver) WithdrawalsConnection(ctx context.Context, party *types.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.WithdrawalsConnection, error) {
	return handleWithdrawalsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

// Deprecated: Use DepositsConnection instead.
func (r *myPartyResolver) Deposits(ctx context.Context, party *types.Party) ([]*types.Deposit, error) {
	res, err := r.tradingDataClient.Deposits(
		ctx, &protoapi.DepositsRequest{PartyId: party.Id},
	)
	if err != nil {
		return nil, err
	}

	return res.Deposits, nil
}

func (r *myPartyResolver) DepositsConnection(ctx context.Context, party *types.Party, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.DepositsConnection, error) {
	return handleDepositsConnectionRequest(ctx, r.tradingDataClientV2, party, dateRange, pagination)
}

// Deprecated: Use VotesConnection instead.
func (r *myPartyResolver) Votes(ctx context.Context, party *types.Party) ([]*ProposalVote, error) {
	resp, err := r.tradingDataClient.GetVotesByParty(ctx, &protoapi.GetVotesByPartyRequest{
		PartyId: party.Id,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*ProposalVote, len(resp.Votes))
	for i, vote := range resp.Votes {
		result[i] = ProposalVoteFromProto(vote)
	}
	return result, nil
}

func (r *myPartyResolver) VotesConnection(ctx context.Context, party *types.Party, pagination *v2.Pagination) (*ProposalVoteConnection, error) {
	req := v2.ListVotesRequest{
		PartyId:    party.Id,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListVotes(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
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

// Deprecated: Use DelegationsConnection instead.
func (r *myPartyResolver) Delegations(
	ctx context.Context,
	obj *types.Party,
	nodeID *string,
	skip, first, last *int,
) ([]*types.Delegation, error) {
	req := &protoapi.DelegationsRequest{
		Party:      obj.Id,
		Pagination: makePagination(skip, first, last),
	}

	if nodeID != nil {
		req.NodeId = *nodeID
	}

	resp, err := r.tradingDataClient.Delegations(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Delegations, nil
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

func (r *myMarginLevelsUpdateResolver) Timestamp(_ context.Context, m *types.MarginLevels) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
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
		return nil, customErrorFromStatus(err)
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

func (r *myMarginLevelsResolver) Timestamp(_ context.Context, m *types.MarginLevels) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

// END: MarginLevels Resolver

type myOrderUpdateResolver VegaResolverRoot

func (r *myOrderUpdateResolver) Price(ctx context.Context, obj *types.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderUpdateResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderUpdateResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderUpdateResolver) CreatedAt(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}

func (r *myOrderUpdateResolver) UpdatedAt(ctx context.Context, obj *types.Order) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderUpdateResolver) Version(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderUpdateResolver) ExpiresAt(ctx context.Context, obj *types.Order) (*string, error) {
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

func (r *myOrderResolver) Price(ctx context.Context, obj *types.Order) (string, error) {
	return obj.Price, nil
}

func (r *myOrderResolver) Market(ctx context.Context, obj *types.Order) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}

func (r *myOrderResolver) CreatedAt(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}

func (r *myOrderResolver) UpdatedAt(ctx context.Context, obj *types.Order) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myOrderResolver) Version(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myOrderResolver) ExpiresAt(ctx context.Context, obj *types.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}

// Deprecated: Use TradesConnection instead.
func (r *myOrderResolver) Trades(ctx context.Context, ord *types.Order) ([]*types.Trade, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	req := protoapi.TradesByOrderRequest{OrderId: ord.Id}
	res, err := r.tradingDataClient.TradesByOrder(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Trades, nil
}

func (r *myOrderResolver) TradesConnection(ctx context.Context, ord *types.Order, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	req := v2.ListTradesRequest{OrderId: &ord.Id, Pagination: pagination, DateRange: dateRange}
	res, err := r.tradingDataClientV2.ListTrades(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Trades, nil
}

func (r *myOrderResolver) Party(ctx context.Context, order *types.Order) (*types.Party, error) {
	if order == nil {
		return nil, errors.New("nil order")
	}
	if len(order.PartyId) == 0 {
		return nil, errors.New("invalid party")
	}
	return &types.Party{Id: order.PartyId}, nil
}

func (r *myOrderResolver) PeggedOrder(ctx context.Context, order *types.Order) (*types.PeggedOrder, error) {
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
		return nil, customErrorFromStatus(err)
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

func (r *myTradeResolver) Price(ctx context.Context, obj *types.Trade) (string, error) {
	return obj.Price, nil
}

func (r *myTradeResolver) Size(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *myTradeResolver) CreatedAt(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

func (r *myTradeResolver) BuyerAuctionBatch(ctx context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.BuyerAuctionBatch)
	return &i, nil
}

func (r *myTradeResolver) BuyerFee(ctx context.Context, obj *types.Trade) (*TradeFee, error) {
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

func (r *myTradeResolver) SellerAuctionBatch(ctx context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.SellerAuctionBatch)
	return &i, nil
}

func (r *myTradeResolver) SellerFee(ctx context.Context, obj *types.Trade) (*TradeFee, error) {
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

func (r *myCandleResolver) PeriodStart(_ context.Context, obj *v2.Candle) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Start)), nil
}

func (r *myCandleResolver) LastUpdateInPeriod(_ context.Context, obj *v2.Candle) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.LastUpdate)), nil
}

func (r *myCandleResolver) Volume(_ context.Context, obj *v2.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

// END: Candle Resolver

// BEGIN: Price Level Resolver

type myPriceLevelResolver VegaResolverRoot

func (r *myPriceLevelResolver) Price(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return obj.Price, nil
}

func (r *myPriceLevelResolver) Volume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.NumberOfOrders, 10), nil
}

// END: Price Level Resolver

type positionUpdateResolver VegaResolverRoot

func (r *positionUpdateResolver) OpenVolume(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *positionUpdateResolver) UpdatedAt(ctx context.Context, obj *types.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

// BEGIN: Position Resolver

type myPositionResolver VegaResolverRoot

func (r *myPositionResolver) Market(ctx context.Context, obj *types.Position) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myPositionResolver) UpdatedAt(ctx context.Context, obj *types.Position) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myPositionResolver) OpenVolume(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *myPositionResolver) RealisedPnl(ctx context.Context, obj *types.Position) (string, error) {
	return obj.RealisedPnl, nil
}

func (r *myPositionResolver) UnrealisedPnl(ctx context.Context, obj *types.Position) (string, error) {
	return obj.UnrealisedPnl, nil
}

func (r *myPositionResolver) AverageEntryPrice(ctx context.Context, obj *types.Position) (string, error) {
	return obj.AverageEntryPrice, nil
}

func (r *myPositionResolver) Party(ctx context.Context, obj *types.Position) (*types.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClientV2, obj.PartyId)
}

// Deprecated: Use MarginsConnection instead.
func (r *myPositionResolver) Margins(ctx context.Context, obj *types.Position) ([]*types.MarginLevels, error) {
	if obj == nil {
		return nil, errors.New("invalid position")
	}
	if len(obj.PartyId) <= 0 {
		return nil, errors.New("missing party id")
	}
	req := protoapi.MarginLevelsRequest{
		PartyId:  obj.PartyId,
		MarketId: obj.MarketId,
	}
	res, err := r.tradingDataClient.MarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.MarginLevels, nil
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.Delegation)
	go func() {
		defer func() {
			stream.CloseSend()
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
			ch <- dl.Delegation
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
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.Reward)
	go func() {
		defer func() {
			stream.CloseSend()
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
			ch <- rd.Reward
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
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.MarginLevels)
	go func() {
		defer func() {
			stream.CloseSend()
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
			ch <- m.MarginLevels
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
		return nil, customErrorFromStatus(err)
	}

	c := make(chan []*v2.AccountBalance)
	go func() {
		defer func() {
			stream.CloseSend()
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

			if snapshot := a.GetSnapshot(); snapshot != nil {
				c <- snapshot.Accounts
			}

			if updates := a.GetUpdates(); updates != nil {
				c <- updates.Accounts
			}
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []*types.Order, error) {
	req := &v2.ObserveOrdersRequest{
		MarketId: market,
		PartyId:  party,
	}
	stream, err := r.tradingDataClientV2.ObserveOrders(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan []*types.Order)
	go func() {
		defer func() {
			stream.CloseSend()
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
			if snapshot := o.GetSnapshot(); snapshot != nil {
				c <- snapshot.Orders
			}
			if updates := o.GetUpdates(); updates != nil {
				c <- updates.Orders
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
		return nil, customErrorFromStatus(err)
	}

	c := make(chan []*types.Trade)
	go func() {
		defer func() {
			stream.CloseSend()
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
			c <- t.Trades
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
		return nil, customErrorFromStatus(err)
	}

	c := make(chan []*types.Position)
	go func() {
		defer func() {
			stream.CloseSend()
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
			if snapshot := t.GetSnapshot(); snapshot != nil {
				c <- snapshot.Positions
			}

			if updates := t.GetUpdates(); updates != nil {
				c <- updates.Positions
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *v2.Candle)
	go func() {
		defer func() {
			stream.CloseSend()
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

			c <- cdl.Candle
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
		return nil, customErrorFromStatus(err)
	}
	output := make(chan *types.GovernanceData)
	go func() {
		defer func() {
			stream.CloseSend()
			close(output)
		}()
		for data, err := stream.Recv(); !isStreamClosed(err, r.log); data, err = stream.Recv() {
			output <- data.Data
		}
	}()
	return output, nil
}

func (r *mySubscriptionResolver) subscribePartyProposals(ctx context.Context, partyID string) (<-chan *types.GovernanceData, error) {
	stream, err := r.tradingDataClientV2.ObserveGovernance(ctx, &v2.ObserveGovernanceRequest{
		PartyId: &partyID,
	})
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	output := make(chan *types.GovernanceData)
	go func() {
		defer func() {
			stream.CloseSend()
			close(output)
		}()
		for data, err := stream.Recv(); !isStreamClosed(err, r.log); data, err = stream.Recv() {
			output <- data.Data
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
		return nil, customErrorFromStatus(err)
	}
	go func() {
		defer func() {
			stream.CloseSend()
			close(output)
		}()
		for {
			data, err := stream.Recv()
			if isStreamClosed(err, r.log) {
				break
			}
			output <- ProposalVoteFromProto(data.Vote)
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
		return nil, customErrorFromStatus(err)
	}
	go func() {
		defer func() {
			stream.CloseSend()
			close(output)
		}()
		for {
			data, err := stream.Recv()
			if isStreamClosed(err, r.log) {
				break
			}
			output <- ProposalVoteFromProto(data.Vote)
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
		return nil, customErrorFromStatus(err)
	}

	// send our initial message to initialize the connection
	if err := stream.Send(&req); err != nil {
		return nil, customErrorFromStatus(err)
	}

	// we no longer buffer this channel. Client receives batch, then we request the next batch
	out := make(chan []*BusEvent)

	go func() {
		defer func() {
			stream.CloseSend()
			close(out)
		}()

		if batchSize == 0 {
			r.busEvents(stream, out)
		} else {
			r.busEventsWithBatch(int64(batchSize), stream, out)
		}
	}()

	return out, nil
}

func (r *mySubscriptionResolver) busEvents(stream v2.TradingDataService_ObserveEventBusClient, out chan []*BusEvent) {
	for {
		// receive batch
		data, err := stream.Recv()
		if isStreamClosed(err, r.log) {
			return
		}
		if err != nil {
			r.log.Error("Event bus stream error", logging.Error(err))
			return
		}
		be := busEventFromProto(data.Events...)
		out <- be
	}
}

func (r *mySubscriptionResolver) busEventsWithBatch(batchSize int64, stream v2.TradingDataService_ObserveEventBusClient, out chan []*BusEvent) {
	poll := &protoapi.ObserveEventBusRequest{
		BatchSize: batchSize,
	}
	for {
		// receive batch
		data, err := stream.Recv()
		if isStreamClosed(err, r.log) {
			return
		}
		if err != nil {
			r.log.Error("Event bus stream error", logging.Error(err))
			return
		}
		be := busEventFromProto(data.Events...)
		out <- be
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
		return nil, customErrorFromStatus(err)
	}

	c := make(chan []*types.LiquidityProvision)
	go func() {
		defer func() {
			stream.CloseSend()
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
			if lps := received.LiquidityProvisions; lps != nil {
				c <- lps
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

func getParty(ctx context.Context, log *logging.Logger, client TradingDataServiceClientV2, id string) (*types.Party, error) {
	if len(id) == 0 {
		return nil, nil
	}
	res, err := client.GetParty(ctx, &v2.GetPartyRequest{PartyId: id})
	if err != nil {
		log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

// Market Data Resolvers

// GetMarketDataHistoryByID returns all the market data information for a given market between the dates specified.
func (r *myQueryResolver) GetMarketDataHistoryByID(ctx context.Context, id string, start, end, skip, first, last *int) ([]*types.MarketData, error) {
	var startTime, endTime *int64

	if start != nil {
		s := int64(*start)
		startTime = &s
	}

	if end != nil {
		e := int64(*end)
		endTime = &e
	}

	pagination := makeAPIV2Pagination(skip, first, last)

	return r.getMarketDataHistoryByID(ctx, id, startTime, endTime, pagination)
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

func (r *myQueryResolver) GetMarketDataHistoryConnectionByID(ctx context.Context, marketID string, start *int, end *int, pagination *v2.Pagination) (*v2.MarketDataConnection, error) {
	var startTime, endTime *int64

	if start != nil {
		s := int64(*start)
		startTime = &s
	}

	if end != nil {
		e := int64(*end)
		endTime = &e
	}

	req := v2.GetMarketDataHistoryByIDRequest{
		MarketId:       marketID,
		StartTimestamp: startTime,
		EndTimestamp:   endTime,
		Pagination:     pagination,
	}

	resp, err := r.tradingDataClientV2.GetMarketDataHistoryByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return resp.GetMarketData(), nil
}

func (r *myQueryResolver) MarketsConnection(ctx context.Context, id *string, pagination *v2.Pagination) (*v2.MarketConnection, error) {
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
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Markets, nil
}

func (r *myQueryResolver) PartiesConnection(ctx context.Context, id *string, pagination *v2.Pagination) (*v2.PartyConnection, error) {
	var partyID string
	if id != nil {
		partyID = *id
	}
	resp, err := r.tradingDataClientV2.ListParties(ctx, &v2.ListPartiesRequest{
		PartyId:    partyID,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Parties, nil
}
