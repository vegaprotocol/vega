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
	"google.golang.org/protobuf/types/known/wrapperspb"

	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/vegatime"
)

var (
	// ErrMissingIDOrReference is returned when neither id nor reference has been supplied in the query
	ErrMissingIDOrReference = errors.New("missing id or reference")
	// ErrInvalidVotesSubscription is returned if neither proposal ID nor party ID is specified
	ErrInvalidVotesSubscription = errors.New("invalid subscription, either proposal or party ID required")
	// ErrInvalidProposal is returned when invalid governance data is received by proposal resolver
	ErrInvalidProposal = errors.New("invalid proposal")
)

// TradingServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_service_client_mock.go -package mocks code.vegaprotocol.io/vega/gateway/graphql TradingServiceClient
type TradingServiceClient interface {
	protoapi.TradingServiceClient
}

// TradingDataServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_data_service_client_mock.go -package mocks code.vegaprotocol.io/vega/gateway/graphql TradingDataServiceClient
type TradingDataServiceClient interface {
	protoapi.TradingDataServiceClient
}

// VegaResolverRoot is the root resolver for all graphql types
type VegaResolverRoot struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     TradingServiceClient
	tradingDataClient TradingDataServiceClient
	r                 allResolver
}

// NewResolverRoot instantiate a graphql root resolver
func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient TradingServiceClient,
	tradingDataClient TradingDataServiceClient,
) *VegaResolverRoot {

	return &VegaResolverRoot{
		log:               log,
		Config:            config,
		tradingClient:     tradingClient,
		tradingDataClient: tradingDataClient,
		r:                 allResolver{log, tradingDataClient},
	}
}

// Query returns the query resolver
func (r *VegaResolverRoot) Query() QueryResolver {
	return (*myQueryResolver)(r)
}

// Mutation returns the mutations resolver
func (r *VegaResolverRoot) Mutation() MutationResolver {
	return (*myMutationResolver)(r)
}

// Candle returns the candles resolver
func (r *VegaResolverRoot) Candle() CandleResolver {
	return (*myCandleResolver)(r)
}

// MarketDepth returns the market depth resolver
func (r *VegaResolverRoot) MarketDepth() MarketDepthResolver {
	return (*myMarketDepthResolver)(r)
}

// MarketDepthUpdate returns the market depth update resolver
func (r *VegaResolverRoot) MarketDepthUpdate() MarketDepthUpdateResolver {
	return (*myMarketDepthUpdateResolver)(r)
}

// MarketData returns the market data resolver
func (r *VegaResolverRoot) MarketData() MarketDataResolver {
	return (*myMarketDataResolver)(r)
}

// MarginLevels returns the market levels resolver
func (r *VegaResolverRoot) MarginLevels() MarginLevelsResolver {
	return (*myMarginLevelsResolver)(r)
}

// PriceLevel returns the price levels resolver
func (r *VegaResolverRoot) PriceLevel() PriceLevelResolver {
	return (*myPriceLevelResolver)(r)
}

// Market returns the markets resolver
func (r *VegaResolverRoot) Market() MarketResolver {
	return (*myMarketResolver)(r)
}

// Order returns the order resolver
func (r *VegaResolverRoot) Order() OrderResolver {
	return (*myOrderResolver)(r)
}

// Trade returns the trades resolver
func (r *VegaResolverRoot) Trade() TradeResolver {
	return (*myTradeResolver)(r)
}

// Position returns the positions resolver
func (r *VegaResolverRoot) Position() PositionResolver {
	return (*myPositionResolver)(r)
}

// Party returns the parties resolver
func (r *VegaResolverRoot) Party() PartyResolver {
	return (*myPartyResolver)(r)
}

// Subscription returns the subscriptions resolver
func (r *VegaResolverRoot) Subscription() SubscriptionResolver {
	return (*mySubscriptionResolver)(r)
}

// Account returns the accounts resolver
func (r *VegaResolverRoot) Account() AccountResolver {
	return (*myAccountResolver)(r)
}

// Statistics returns the statistics resolver
func (r *VegaResolverRoot) Statistics() StatisticsResolver {
	return (*myStatisticsResolver)(r)
}

// Proposal returns the proposal resolver
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

func (r *VegaResolverRoot) LiquidityOrder() LiquidityOrderResolver {
	return (*myLiquidityOrderResolver)(r)
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

func (r *VegaResolverRoot) PeggedOrder() PeggedOrderResolver {
	return (*myPeggedOrderResolver)(r)
}

func (r *VegaResolverRoot) OracleSpec() OracleSpecResolver {
	return (*oracleSpecResolver)(r)
}

func (r *VegaResolverRoot) PropertyKey() PropertyKeyResolver {
	return (*propertyKeyResolver)(r)
}

func (r *VegaResolverRoot) Condition() ConditionResolver {
	return (*conditionResolver)(r)
}

func (r *VegaResolverRoot) AuctionEvent() AuctionEventResolver {
	return (*auctionEventResolver)(r)

}
func (r *VegaResolverRoot) Vote() VoteResolver {
	return (*voteResolver)(r)
}

// LiquidityOrder resolver

type myLiquidityOrderResolver VegaResolverRoot

func (r *myLiquidityOrderResolver) Proportion(ctx context.Context, obj *types.LiquidityOrder) (int, error) {
	return int(obj.Proportion), nil
}

func (r *myLiquidityOrderResolver) Reference(ctx context.Context, obj *types.LiquidityOrder) (PeggedReference, error) {
	return convertPeggedReferenceFromProto(obj.Reference)
}

// LiquidityOrderReference resolver

type myLiquidityOrderReferenceResolver VegaResolverRoot

func (r *myLiquidityOrderReferenceResolver) Order(ctx context.Context, obj *types.LiquidityOrderReference) (*types.Order, error) {
	if len(obj.OrderId) <= 0 {
		return nil, nil
	}
	return r.r.getOrderByID(ctx, obj.OrderId, nil)
}

// LiquidityProvision resolver

type myLiquidityProvisionResolver VegaResolverRoot

func (r *myLiquidityProvisionResolver) Party(ctx context.Context, obj *types.LiquidityProvision) (*types.Party, error) {
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myLiquidityProvisionResolver) CreatedAt(ctx context.Context, obj *types.LiquidityProvision) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}
func (r *myLiquidityProvisionResolver) UpdatedAt(ctx context.Context, obj *types.LiquidityProvision) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}
func (r *myLiquidityProvisionResolver) Market(ctx context.Context, obj *types.LiquidityProvision) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}
func (r *myLiquidityProvisionResolver) CommitmentAmount(ctx context.Context, obj *types.LiquidityProvision) (int, error) {
	return int(obj.CommitmentAmount), nil
}

func (r *myLiquidityProvisionResolver) Status(ctx context.Context, obj *types.LiquidityProvision) (LiquidityProvisionStatus, error) {
	return convertLiquidityProvisionStatusFromProto(obj.Status)
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

func (r *myDepositResolver) Status(ctx context.Context, obj *types.Deposit) (DepositStatus, error) {
	return convertDepositStatusFromProto(obj.Status)
}

// BEGIN: Query Resolver

type myQueryResolver VegaResolverRoot

func (r *myQueryResolver) OracleSpecs(ctx context.Context) ([]*oraclespb.OracleSpec, error) {
	res, err := r.tradingDataClient.OracleSpecs(
		ctx, &protoapi.OracleSpecsRequest{},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleSpecs, nil
}

func (r *myQueryResolver) OracleSpec(ctx context.Context, id string) (*oraclespb.OracleSpec, error) {
	res, err := r.tradingDataClient.OracleSpec(
		ctx, &protoapi.OracleSpecRequest{Id: id},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleSpec, nil
}

func (r *myQueryResolver) OracleDataBySpec(ctx context.Context, id string) ([]*oraclespb.OracleData, error) {
	res, err := r.tradingDataClient.OracleDataBySpec(
		ctx, &protoapi.OracleDataBySpecRequest{Id: id},
	)
	if err != nil {
		return nil, err
	}

	return res.OracleData, nil
}

func (r *myQueryResolver) NetworkParameters(ctx context.Context) ([]*types.NetworkParameter, error) {
	res, err := r.tradingDataClient.NetworkParameters(
		ctx, &protoapi.NetworkParametersRequest{},
	)
	if err != nil {
		return nil, err
	}

	return res.NetworkParameters, nil
}

func (r *myQueryResolver) Erc20WithdrawalApproval(ctx context.Context, wid string) (*Erc20WithdrawalApproval, error) {
	res, err := r.tradingDataClient.ERC20WithdrawalApproval(
		ctx, &protoapi.ERC20WithdrawalApprovalRequest{WithdrawalId: wid},
	)
	if err != nil {
		return nil, err
	}

	return &Erc20WithdrawalApproval{
		AssetSource: res.AssetSource,
		Amount:      res.Amount,
		Expiry:      strconv.FormatInt(res.Expiry, 10),
		Nonce:       res.Nonce,
		Signatures:  res.Signatures,
	}, nil
}

func (r *myQueryResolver) Withdrawal(ctx context.Context, wid string) (*types.Withdrawal, error) {
	res, err := r.tradingDataClient.Withdrawal(
		ctx, &protoapi.WithdrawalRequest{Id: wid},
	)
	if err != nil {
		return nil, err
	}

	return res.Withdrawal, nil
}

func (r *myQueryResolver) Deposit(ctx context.Context, did string) (*types.Deposit, error) {
	res, err := r.tradingDataClient.Deposit(
		ctx, &protoapi.DepositRequest{Id: did},
	)
	if err != nil {
		return nil, err
	}

	return res.Deposit, nil
}

func (r *myQueryResolver) EstimateOrder(ctx context.Context, market, party string, price *string, size string, side Side,
	timeInForce OrderTimeInForce, expiration *string, ty OrderType) (*OrderEstimate, error) {
	order := &types.Order{}

	var (
		p   uint64
		err error
	)

	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	if price != nil {
		p, err = safeStringUint64(*price)
		if err != nil {
			return nil, err
		}
	}
	order.Price = p
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
	if order.TimeInForce, err = convertOrderTimeInForceToProto(timeInForce); err != nil {
		return nil, err
	}
	if order.Side, err = convertSideToProto(side); err != nil {
		return nil, err
	}
	if order.Type, err = convertOrderTypeToProto(ty); err != nil {
		return nil, err
	}

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

	req := protoapi.EstimateFeeRequest{
		Order: order,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	resp, err := r.tradingDataClient.EstimateFee(ctx, &req)
	if err != nil {
		r.log.Error("Failed to get fee estimates using rpc client in graphQL resolver", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	// calclate the fee total amount
	ttf := resp.Fee.MakerFee + resp.Fee.InfrastructureFee + resp.Fee.LiquidityFee

	fee := TradeFee{
		MakerFee:          fmt.Sprintf("%d", resp.Fee.MakerFee),
		InfrastructureFee: fmt.Sprintf("%d", resp.Fee.InfrastructureFee),
		LiquidityFee:      fmt.Sprintf("%d", resp.Fee.LiquidityFee),
	}

	// now we calculate the margins
	reqm := protoapi.EstimateMarginRequest{
		Order: order,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	respm, err := r.tradingDataClient.EstimateMargin(ctx, &reqm)
	if err != nil {
		r.log.Error("Failed to get margin estimates using rpc client in graphQL resolver", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return &OrderEstimate{
		Fee:            &fee,
		TotalFeeAmount: fmt.Sprintf("%d", ttf),
		MarginLevels:   respm.MarginLevels,
	}, nil

}

func (r *myQueryResolver) Asset(ctx context.Context, id string) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, id)
}

func (r *myQueryResolver) Assets(ctx context.Context) ([]*types.Asset, error) {
	return r.r.allAssets(ctx)
}

func (r *myQueryResolver) NodeSignatures(ctx context.Context, resourceID string) ([]*types.NodeSignature, error) {
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

func (r *myQueryResolver) Markets(ctx context.Context, id *string) ([]*types.Market, error) {
	return r.r.allMarkets(ctx, id)
}

func (r *myQueryResolver) Market(ctx context.Context, id string) (*types.Market, error) {
	return r.r.getMarketByID(ctx, id)
}

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
	return getParty(ctx, r.log, r.tradingDataClient, name)
}

func (r *myQueryResolver) Statistics(ctx context.Context) (*types.Statistics, error) {
	res, err := r.tradingDataClient.Statistics(ctx, &protoapi.StatisticsRequest{})
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Statistics, nil
}

func (r *myQueryResolver) OrderByID(ctx context.Context, orderID string, version *int) (*types.Order, error) {
	return r.r.getOrderByID(ctx, orderID, version)
}

func (r *myQueryResolver) OrderVersions(
	ctx context.Context, orderID string, skip, first, last *int) ([]*types.Order, error) {

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

func (r *myQueryResolver) OrderByReference(ctx context.Context, reference string) (*types.Order, error) {
	req := &protoapi.OrderByReferenceRequest{
		Reference: reference,
	}
	res, err := r.tradingDataClient.OrderByReference(ctx, req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Order, err
}

func (r *myQueryResolver) Proposals(ctx context.Context, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	resp, err := r.tradingDataClient.GetProposals(ctx, &protoapi.GetProposalsRequest{
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myQueryResolver) Proposal(ctx context.Context, id *string, reference *string) (*types.GovernanceData, error) {
	if id != nil {
		resp, err := r.tradingDataClient.GetProposalByID(ctx, &protoapi.GetProposalByIDRequest{
			ProposalId: *id,
		})
		if err != nil {
			return nil, err
		}
		return resp.Data, nil
	} else if reference != nil {
		resp, err := r.tradingDataClient.GetProposalByReference(ctx, &protoapi.GetProposalByReferenceRequest{
			Reference: *reference,
		})
		if err != nil {
			return nil, err
		}
		return resp.Data, nil
	}

	return nil, ErrMissingIDOrReference
}

func (r *myQueryResolver) NewMarketProposals(ctx context.Context, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	resp, err := r.tradingDataClient.GetNewMarketProposals(ctx, &protoapi.GetNewMarketProposalsRequest{
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myQueryResolver) UpdateMarketProposals(ctx context.Context, marketID *string, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	var market string
	if marketID != nil {
		market = *marketID
	}
	resp, err := r.tradingDataClient.GetUpdateMarketProposals(ctx, &protoapi.GetUpdateMarketProposalsRequest{
		MarketId:      market,
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myQueryResolver) NetworkParametersProposals(ctx context.Context, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	resp, err := r.tradingDataClient.GetNetworkParametersProposals(ctx, &protoapi.GetNetworkParametersProposalsRequest{
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myQueryResolver) NewAssetProposals(ctx context.Context, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	resp, err := r.tradingDataClient.GetNewAssetProposals(ctx, &protoapi.GetNewAssetProposalsRequest{
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// END: Root Resolver

type myNodeSignatureResolver VegaResolverRoot

func (r *myNodeSignatureResolver) Signature(ctx context.Context, obj *types.NodeSignature) (*string, error) {
	sig := base64.StdEncoding.EncodeToString(obj.Sig)
	return &sig, nil
}

func (r *myNodeSignatureResolver) Kind(ctx context.Context, obj *types.NodeSignature) (*NodeSignatureKind, error) {
	kind, err := convertNodeSignatureKindFromProto(obj.Kind)
	if err != nil {
		return nil, err
	}
	return &kind, nil
}

// BEGIN: Party Resolver

type myPartyResolver VegaResolverRoot

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

func (r *myPartyResolver) Margins(ctx context.Context,
	party *types.Party, marketID *string) ([]*types.MarginLevels, error) {

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

func (r *myPartyResolver) Orders(ctx context.Context, party *types.Party,
	skip, first, last *int) ([]*types.Order, error) {

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

func (r *myPartyResolver) Trades(ctx context.Context, party *types.Party,
	market *string, skip, first, last *int) ([]*types.Trade, error) {

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

func (r *myPartyResolver) Accounts(ctx context.Context, party *types.Party,
	marketID *string, asset *string, accType *AccountType) ([]*types.Account, error) {
	if party == nil {
		return nil, errors.New("a party must be specified when querying accounts")
	}
	var (
		mktid = ""
		asst  = ""
		accTy = types.AccountType_ACCOUNT_TYPE_UNSPECIFIED
		err   error
	)

	if marketID != nil {
		mktid = *marketID
	}
	if asset != nil {
		asst = *asset
	}
	if accType != nil {
		accTy, err = convertAccountTypeToProto(*accType)
		if err != nil ||
			(accTy != types.AccountType_ACCOUNT_TYPE_GENERAL &&
				accTy != types.AccountType_ACCOUNT_TYPE_MARGIN &&
				accTy != types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW &&
				accTy != types.AccountType_ACCOUNT_TYPE_BOND) {
			return nil, fmt.Errorf("invalid account type for party %v", accType)
		}
	}
	req := protoapi.PartyAccountsRequest{
		PartyId:  party.Id,
		MarketId: mktid,
		Asset:    asst,
		Type:     accTy,
	}
	res, err := r.tradingDataClient.PartyAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get Party account",
			logging.Error(err),
			logging.String("party-id", party.Id),
			logging.String("market-id", mktid),
			logging.String("asset", asst),
			logging.String("type", accTy.String()))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Accounts) > 0 {
		return res.Accounts, nil
	}
	// mandatory return field in schema
	return []*types.Account{}, nil
}

func (r *myPartyResolver) Proposals(ctx context.Context, party *types.Party, inState *ProposalState) ([]*types.GovernanceData, error) {
	filter, err := inState.ToOptionalProposalState()
	if err != nil {
		return nil, err
	}
	resp, err := r.tradingDataClient.GetProposalsByParty(ctx, &protoapi.GetProposalsByPartyRequest{
		PartyId:       party.Id,
		SelectInState: filter,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (r *myPartyResolver) Withdrawals(ctx context.Context, party *types.Party) ([]*types.Withdrawal, error) {
	res, err := r.tradingDataClient.Withdrawals(
		ctx, &protoapi.WithdrawalsRequest{PartyId: party.Id},
	)
	if err != nil {
		return nil, err
	}

	return res.Withdrawals, nil
}

func (r *myPartyResolver) Deposits(ctx context.Context, party *types.Party) ([]*types.Deposit, error) {
	res, err := r.tradingDataClient.Deposits(
		ctx, &protoapi.DepositsRequest{PartyId: party.Id},
	)
	if err != nil {
		return nil, err
	}

	return res.Deposits, nil
}

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

// END: Party Resolver

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
	req := protoapi.PartyByIDRequest{PartyId: m.PartyId}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
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
	return strconv.FormatUint(m.CollateralReleaseLevel, 10), nil
}

func (r *myMarginLevelsResolver) InitialLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatUint(m.InitialMargin, 10), nil
}

func (r *myMarginLevelsResolver) SearchLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatUint(m.SearchLevel, 10), nil
}

func (r *myMarginLevelsResolver) MaintenanceLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatUint(m.MaintenanceMargin, 10), nil
}

func (r *myMarginLevelsResolver) Timestamp(_ context.Context, m *types.MarginLevels) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

// END: MarginLevels Resolver

// BEGIN: MarketData resolver

type myMarketDataResolver VegaResolverRoot

func (r *myMarketDataResolver) AuctionStart(_ context.Context, m *types.MarketData) (*string, error) {
	if m.AuctionStart <= 0 {
		return nil, nil
	}
	s := vegatime.Format(vegatime.UnixNano(m.AuctionStart))
	return &s, nil
}

func (r *myMarketDataResolver) AuctionEnd(_ context.Context, m *types.MarketData) (*string, error) {
	if m.AuctionEnd <= 0 {
		return nil, nil
	}
	s := vegatime.Format(vegatime.UnixNano(m.AuctionEnd))
	return &s, nil
}

func (r *myMarketDataResolver) MarketTradingMode(_ context.Context, m *types.MarketData) (MarketTradingMode, error) {
	return convertMarketTradingModeFromProto(m.MarketTradingMode)
}

func (r *myMarketDataResolver) IndicativePrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.IndicativePrice, 10), nil
}

func (r *myMarketDataResolver) IndicativeVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.IndicativeVolume, 10), nil
}

func (r *myMarketDataResolver) BestBidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestBidPrice, 10), nil
}

func (r *myMarketDataResolver) BestStaticBidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestStaticBidPrice, 10), nil
}

func (r *myMarketDataResolver) BestStaticBidVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestStaticBidVolume, 10), nil
}

func (r *myMarketDataResolver) OpenInterest(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.OpenInterest, 10), nil
}

func (r *myMarketDataResolver) BestBidVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestBidVolume, 10), nil
}

func (r *myMarketDataResolver) BestOfferPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestOfferPrice, 10), nil
}

func (r *myMarketDataResolver) BestStaticOfferPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestStaticOfferPrice, 10), nil
}

func (r *myMarketDataResolver) BestStaticOfferVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestStaticOfferVolume, 10), nil
}

func (r *myMarketDataResolver) BestOfferVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestOfferVolume, 10), nil
}

func (r *myMarketDataResolver) MidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.MidPrice, 10), nil
}

func (r *myMarketDataResolver) StaticMidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.StaticMidPrice, 10), nil
}

func (r *myMarketDataResolver) MarkPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.MarkPrice, 10), nil
}

func (r *myMarketDataResolver) Timestamp(_ context.Context, m *types.MarketData) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

func (r *myMarketDataResolver) Commitments(ctx context.Context, m *types.MarketData) (*MarketDataCommitments, error) {
	// get all the commitments for the given market
	req := protoapi.LiquidityProvisionsRequest{
		Market: m.Market,
	}
	res, err := r.tradingDataClient.LiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	// now we split all the sells and buys
	sells := []*types.LiquidityOrderReference{}
	buys := []*types.LiquidityOrderReference{}

	for _, v := range res.LiquidityProvisions {
		sells = append(sells, v.Sells...)
		buys = append(buys, v.Buys...)
	}

	return &MarketDataCommitments{
		Sells: sells,
		Buys:  buys,
	}, nil
}

func (r *myMarketDataResolver) PriceMonitoringBounds(ctx context.Context, obj *types.MarketData) ([]*PriceMonitoringBounds, error) {
	ret := make([]*PriceMonitoringBounds, 0, len(obj.PriceMonitoringBounds))
	for _, b := range obj.PriceMonitoringBounds {
		bounds := &PriceMonitoringBounds{
			MinValidPrice: strconv.FormatUint(b.MinValidPrice, 10),
			MaxValidPrice: strconv.FormatUint(b.MaxValidPrice, 10),
			Trigger: &PriceMonitoringTrigger{
				HorizonSecs:          int(b.Trigger.Horizon),
				Probability:          b.Trigger.Probability,
				AuctionExtensionSecs: int(b.Trigger.AuctionExtension),
			},
			ReferencePrice: strconv.FormatFloat(b.ReferencePrice, 'f', -1, 64),
		}
		ret = append(ret, bounds)
	}
	return ret, nil
}

func (r *myMarketDataResolver) Market(ctx context.Context, m *types.MarketData) (*types.Market, error) {
	return r.r.getMarketByID(ctx, m.Market)
}

// Trigger...
func (r *myMarketDataResolver) Trigger(_ context.Context, m *types.MarketData) (AuctionTrigger, error) {
	return convertAuctionTriggerFromProto(m.Trigger)
}

func (r *myMarketDataResolver) MarketValueProxy(_ context.Context, m *types.MarketData) (string, error) {
	return m.MarketValueProxy, nil
}

func (r *myMarketDataResolver) LiquidityProviderFeeShare(_ context.Context, m *types.MarketData) ([]*LiquidityProviderFeeShare, error) {
	out := make([]*LiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare))
	for _, v := range m.LiquidityProviderFeeShare {
		out = append(out, &LiquidityProviderFeeShare{
			Party:                 &types.Party{Id: v.Party},
			EquityLikeShare:       v.EquityLikeShare,
			AverageEntryValuation: v.AverageEntryValuation,
		})
	}
	return out, nil
}

// END: MarketData resolver

// BEGIN: Market Depth Resolver

type myMarketDepthResolver VegaResolverRoot

func (r *myMarketDepthResolver) LastTrade(ctx context.Context, md *types.MarketDepth) (*types.Trade, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := protoapi.LastTradeRequest{MarketId: md.MarketId}
	res, err := r.tradingDataClient.LastTrade(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Trade, nil
}

func (r *myMarketDepthResolver) SequenceNumber(ctx context.Context, md *types.MarketDepth) (string, error) {
	return strconv.FormatUint(md.SequenceNumber, 10), nil
}

func (r *myMarketDepthResolver) Market(ctx context.Context, md *types.MarketDepth) (*types.Market, error) {
	return r.r.getMarketByID(ctx, md.MarketId)
}

// END: Market Depth Resolver

// BEGIN: Market Depth Update Resolver

type myMarketDepthUpdateResolver VegaResolverRoot

func (r *myMarketDepthUpdateResolver) SequenceNumber(ctx context.Context, md *types.MarketDepthUpdate) (string, error) {
	return strconv.FormatUint(md.SequenceNumber, 10), nil
}

func (r *myMarketDepthUpdateResolver) Market(ctx context.Context, md *types.MarketDepthUpdate) (*types.Market, error) {
	return r.r.getMarketByID(ctx, md.MarketId)
}

// END: Market Depth Update Resolver

// BEGIN: Order Resolver

type myOrderResolver VegaResolverRoot

func (r *myOrderResolver) RejectionReason(_ context.Context, o *types.Order) (*OrderRejectionReason, error) {
	if o.Reason == types.OrderError_ORDER_ERROR_UNSPECIFIED {
		return nil, nil
	}
	reason, err := convertOrderRejectionReasonFromProto(o.Reason)
	if err != nil {
		return nil, err
	}
	return &reason, nil
}

func (r *myOrderResolver) Price(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *myOrderResolver) TimeInForce(ctx context.Context, obj *types.Order) (OrderTimeInForce, error) {
	return convertOrderTimeInForceFromProto(obj.TimeInForce)
}

func (r *myOrderResolver) Type(ctx context.Context, obj *types.Order) (*OrderType, error) {
	t, err := convertOrderTypeFromProto(obj.Type)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *myOrderResolver) Side(ctx context.Context, obj *types.Order) (Side, error) {
	return convertSideFromProto(obj.Side)
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

func (r *myOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return convertOrderStatusFromProto(obj.Status)
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

// END: Order Resolver

// BEGIN: Trade Resolver

type myTradeResolver VegaResolverRoot

func (r *myTradeResolver) Market(ctx context.Context, obj *types.Trade) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myTradeResolver) Aggressor(ctx context.Context, obj *types.Trade) (Side, error) {
	return Side(obj.Aggressor.String()), nil
}

func (r *myTradeResolver) Price(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
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
	req := protoapi.PartyByIDRequest{PartyId: obj.Buyer}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
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
	req := protoapi.PartyByIDRequest{PartyId: obj.Seller}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

func (r *myTradeResolver) Type(ctx context.Context, obj *types.Trade) (TradeType, error) {
	return convertTradeTypeFromProto(obj.Type)
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
		fee.MakerFee = strconv.FormatUint(obj.BuyerFee.MakerFee, 10)
		fee.InfrastructureFee = strconv.FormatUint(obj.BuyerFee.InfrastructureFee, 10)
		fee.LiquidityFee = strconv.FormatUint(obj.BuyerFee.LiquidityFee, 10)
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
		fee.MakerFee = strconv.FormatUint(obj.SellerFee.MakerFee, 10)
		fee.InfrastructureFee = strconv.FormatUint(obj.SellerFee.InfrastructureFee, 10)
		fee.LiquidityFee = strconv.FormatUint(obj.SellerFee.LiquidityFee, 10)
	}
	return &fee, nil
}

// END: Trade Resolver

// BEGIN: Candle Resolver

type myCandleResolver VegaResolverRoot

func (r *myCandleResolver) High(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.High, 10), nil
}
func (r *myCandleResolver) Low(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Low, 10), nil
}
func (r *myCandleResolver) Open(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Open, 10), nil
}
func (r *myCandleResolver) Close(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Close, 10), nil
}
func (r *myCandleResolver) Volume(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}
func (r *myCandleResolver) Datetime(ctx context.Context, obj *types.Candle) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *myCandleResolver) Timestamp(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *myCandleResolver) Interval(ctx context.Context, obj *types.Candle) (Interval, error) {
	return convertIntervalFromProto(obj.Interval)
}

// END: Candle Resolver

// BEGIN: Price Level Resolver

type myPriceLevelResolver VegaResolverRoot

func (r *myPriceLevelResolver) Price(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *myPriceLevelResolver) Volume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.NumberOfOrders, 10), nil
}

// END: Price Level Resolver

// BEGIN: PeggedOrder Resolver

type myPeggedOrderResolver VegaResolverRoot

func (r *myPeggedOrderResolver) Reference(ctx context.Context, obj *types.PeggedOrder) (PeggedReference, error) {
	return convertPeggedReferenceFromProto(obj.Reference)
}

func (r *myPeggedOrderResolver) Offset(ctx context.Context, obj *types.PeggedOrder) (string, error) {
	return strconv.FormatInt(obj.Offset, 10), nil
}

// END: PeggedOrder Resolver

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
	return strconv.FormatInt(obj.RealisedPnl, 10), nil
}

func (r *myPositionResolver) UnrealisedPnl(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.UnrealisedPnl, 10), nil
}

func (r *myPositionResolver) AverageEntryPrice(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatUint(obj.AverageEntryPrice, 10), nil
}

func (r *myPositionResolver) Party(ctx context.Context, obj *types.Position) (*types.Party, error) {
	return getParty(ctx, r.log, r.tradingDataClient, obj.PartyId)
}

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

// END: Position Resolver

// BEGIN: Mutation Resolver

type myMutationResolver VegaResolverRoot

func (r *myMutationResolver) PrepareWithdrawal(
	ctx context.Context,
	partyID, amount, asset string,
	erc20Details *Erc20WithdrawalDetailsInput,
) (*PreparedWithdrawal, error) {
	var ext *types.WithdrawExt
	if erc20Details != nil {
		ext = erc20Details.IntoProtoExt()
	}

	amountU, err := safeStringUint64(amount)
	if err != nil {
		return nil, err
	}

	req := protoapi.PrepareWithdrawRequest{
		Withdraw: &types.WithdrawSubmission{
			PartyId: partyID,
			Asset:   asset,
			Amount:  amountU,
			Ext:     ext,
		},
	}

	res, err := r.tradingClient.PrepareWithdraw(ctx, &req)
	if err != nil {
		return nil, err
	}

	return &PreparedWithdrawal{
		Blob: base64.StdEncoding.EncodeToString(res.Blob),
	}, nil
}

func (r *myMutationResolver) SubmitTransaction(ctx context.Context, data string, sig SignatureInput, ty *SubmitTransactionType) (*TransactionSubmitted, error) {

	pty := protoapi.SubmitTransactionRequest_TYPE_ASYNC
	if ty != nil {
		switch *ty {
		case SubmitTransactionTypeSync:
			pty = protoapi.SubmitTransactionRequest_TYPE_SYNC
		case SubmitTransactionTypeCommit:
			pty = protoapi.SubmitTransactionRequest_TYPE_COMMIT
		}
	}

	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	decodedSig, err := base64.StdEncoding.DecodeString(sig.Sig)
	if err != nil {
		return nil, err
	}
	req := &protoapi.SubmitTransactionRequest{
		Tx: &types.SignedBundle{
			Tx: decodedData,
			Sig: &types.Signature{
				Sig:     decodedSig,
				Version: uint64(sig.Version),
				Algo:    sig.Algo,
			},
		},
		Type: pty,
	}
	res, err := r.tradingClient.SubmitTransaction(ctx, req)
	if err != nil {
		r.log.Error("Failed to submit transaction", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return &TransactionSubmitted{
		Success: res.Success,
	}, nil
}

func (r *myMutationResolver) PrepareOrderSubmit(ctx context.Context, market, party string, price *string, size string, side Side,
	timeInForce OrderTimeInForce, expiration *string, ty OrderType, reference *string, po *PeggedOrderInput) (*PreparedSubmitOrder, error) {

	order := &types.OrderSubmission{}

	var (
		p   uint64
		err error
	)

	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	if price != nil {
		p, err = safeStringUint64(*price)
		if err != nil {
			return nil, err
		}
	}
	order.Price = p
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
	if order.TimeInForce, err = convertOrderTimeInForceToProto(timeInForce); err != nil {
		return nil, err
	}
	if order.Side, err = convertSideToProto(side); err != nil {
		return nil, err
	}
	if order.Type, err = convertOrderTypeToProto(ty); err != nil {
		return nil, err
	}

	if po != nil {
		pegreference, err := convertPeggedReferenceToProto(po.Reference)
		if err != nil {
			return nil, err
		}
		offset, err := safeStringInt64(po.Offset)
		if err != nil {
			return nil, err
		}
		order.PeggedOrder = &types.PeggedOrder{Reference: pegreference,
			Offset: offset}
	}

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
	if reference != nil {
		order.Reference = *reference
	}

	req := protoapi.PrepareSubmitOrderRequest{
		Submission: order,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	resp, err := r.tradingClient.PrepareSubmitOrder(ctx, &req)
	if err != nil {
		r.log.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return &PreparedSubmitOrder{
		Blob: base64.StdEncoding.EncodeToString(resp.Blob),
	}, nil
}

func (r *myMutationResolver) PrepareOrderCancel(ctx context.Context, id *string, party string, market *string) (*PreparedCancelOrder, error) {
	order := &types.OrderCancellation{}

	if market != nil {
		order.MarketId = *market
	}
	if id != nil {
		order.OrderId = *id
	}
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}
	order.PartyId = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)

	req := protoapi.PrepareCancelOrderRequest{
		Cancellation: order,
	}
	pendingOrder, err := r.tradingClient.PrepareCancelOrder(ctx, &req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	return &PreparedCancelOrder{
		Blob: base64.StdEncoding.EncodeToString(pendingOrder.Blob),
	}, nil

}

func (r *myMutationResolver) PrepareProposal(
	ctx context.Context, partyID string, reference *string, proposalTerms ProposalTermsInput) (*PreparedProposal, error) {
	var ref string
	if reference != nil {
		ref = *reference
	}

	terms, err := proposalTerms.IntoProto()
	if err != nil {
		return nil, err
	}

	pendingProposal, err := r.tradingClient.PrepareProposal(ctx, &protoapi.PrepareProposalRequest{
		PartyId:   partyID,
		Reference: ref,
		Proposal:  terms,
	})
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	return &PreparedProposal{
		Blob: base64.StdEncoding.EncodeToString(pendingProposal.Blob),
		PendingProposal: &types.GovernanceData{
			Proposal: pendingProposal.PendingProposal,
		},
	}, nil
}

func (r *myMutationResolver) PrepareVote(ctx context.Context, value VoteValue, partyID, proposalID string) (*PreparedVote, error) {
	_, err := getParty(ctx, r.log, r.tradingDataClient, partyID)
	if err != nil {
		return nil, err
	}
	protoValue, err := convertVoteValueToProto(value)
	if err != nil {
		return nil, err
	}
	req := &protoapi.PrepareVoteRequest{
		Vote: &types.Vote{
			Value:      protoValue,
			PartyId:    partyID,
			ProposalId: proposalID,
		},
	}
	resp, err := r.tradingClient.PrepareVote(ctx, req)
	if err != nil {
		return nil, err
	}
	return &PreparedVote{
		Blob: base64.StdEncoding.EncodeToString(resp.Blob),
		Vote: &ProposalVote{
			Vote:       req.Vote,
			ProposalID: resp.Vote.ProposalId,
		},
	}, nil
}

func (r *myMutationResolver) PrepareOrderAmend(ctx context.Context, id string, party string, price, size string,
	expiration *string, tif OrderTimeInForce, peggedReference *PeggedReference, peggedOffset *string) (*PreparedAmendOrder, error) {
	order := &types.OrderAmendment{}

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(id) == 0 {
		return nil, errors.New("id missing or empty")
	}
	order.OrderId = id
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}
	order.PartyId = party

	var err error
	pricevalue, err := strconv.ParseUint(price, 10, 64)
	if err != nil {
		if r.log.GetLevel() == logging.DebugLevel {
			r.log.Debug("unable to convert price from string in order amend", logging.Error(err))
		}
		return nil, errors.New("invalid price, could not convert to unsigned int")
	}
	order.Price = &types.Price{Value: pricevalue}

	order.SizeDelta, err = strconv.ParseInt(size, 10, 64)
	if err != nil {
		if r.log.GetLevel() == logging.DebugLevel {
			r.log.Debug("unable to convert size from string in order amend", logging.Error(err))
		}
		return nil, errors.New("invalid size, could not convert to unsigned int")
	}

	order.TimeInForce, err = convertOrderTimeInForceToProto(tif)
	if err != nil {
		if r.log.GetLevel() == logging.DebugLevel {
			r.log.Debug("unable to parse time in force in order amend", logging.Error(err))
		}
		return nil, errors.New("invalid time in force, could not convert to vega time in force")
	}

	if expiration != nil {
		expiresAt, err := vegatime.Parse(*expiration)
		if err != nil {
			return nil, fmt.Errorf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration)
		}
		// move to pure timestamps or convert an RFC format shortly
		order.ExpiresAt = &types.Timestamp{Value: expiresAt.UnixNano()}
	}

	if peggedOffset != nil {
		po, err := strconv.ParseInt(*peggedOffset, 10, 64)
		if err != nil {
			if r.log.GetLevel() == logging.DebugLevel {
				r.log.Debug("unable to parse pegged offset in order amend", logging.Error(err))
			}
			return nil, errors.New("invalid pegged offset, could not convert to proto pegged offset")
		}
		order.PeggedOffset = &wrapperspb.Int64Value{Value: po}
	}

	order.PeggedReference, err = convertPeggedReferenceToProto(*peggedReference)
	if err != nil {
		if r.log.GetLevel() == logging.DebugLevel {
			r.log.Debug("unable to parse pegged reference in order amend", logging.Error(err))
		}
		return nil, errors.New("invalid pegged reference, could not convert to proto pegged reference")
	}

	req := protoapi.PrepareAmendOrderRequest{
		Amendment: order,
	}
	pendingOrder, err := r.tradingClient.PrepareAmendOrder(ctx, &req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	return &PreparedAmendOrder{
		Blob: base64.StdEncoding.EncodeToString(pendingOrder.Blob),
	}, nil
}

func (r *myMutationResolver) PrepareLiquidityProvision(ctx context.Context, marketID string, commitmentAmount int, fee string, sells []*LiquidityOrderInput, buys []*LiquidityOrderInput, maybeRef *string) (*PreparedLiquidityProvision, error) {
	if commitmentAmount < 0 {
		return nil, errors.New("commitmentAmount can't be negative")
	}

	pBuys, err := LiquidityOrderInputs(buys).IntoProto()
	if err != nil {
		return nil, err
	}

	pSells, err := LiquidityOrderInputs(sells).IntoProto()
	if err != nil {
		return nil, err
	}

	var ref string
	if maybeRef != nil {
		ref = *maybeRef
	}

	req := &protoapi.PrepareLiquidityProvisionRequest{
		Submission: &types.LiquidityProvisionSubmission{
			MarketId:         marketID,
			CommitmentAmount: uint64(commitmentAmount),
			Fee:              fee,
			Buys:             pBuys,
			Sells:            pSells,
			Reference:        ref,
		},
	}
	resp, err := r.tradingClient.PrepareLiquidityProvision(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	return &PreparedLiquidityProvision{
		Blob: base64.StdEncoding.EncodeToString(resp.Blob),
	}, nil
}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type mySubscriptionResolver VegaResolverRoot

func (r *mySubscriptionResolver) Margins(ctx context.Context, partyID string, marketID *string) (<-chan *types.MarginLevels, error) {
	var mktid string
	if marketID != nil {
		mktid = *marketID
	}
	req := &protoapi.MarginLevelsSubscribeRequest{
		MarketId: mktid,
		PartyId:  partyID,
	}
	stream, err := r.tradingDataClient.MarginLevelsSubscribe(ctx, req)
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

func (r *mySubscriptionResolver) MarketData(ctx context.Context, marketID *string) (<-chan *types.MarketData, error) {
	var mktid string
	if marketID != nil {
		mktid = *marketID
	}
	req := &protoapi.MarketsDataSubscribeRequest{
		MarketId: mktid,
	}
	stream, err := r.tradingDataClient.MarketsDataSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.MarketData)
	go func() {
		defer func() {
			stream.CloseSend()
			close(ch)
		}()
		for {
			m, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("marketdata: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("marketdata: stream closed", logging.Error(err))
				break
			}
			ch <- m.MarketData
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) Accounts(ctx context.Context, marketID *string, partyID *string, asset *string, typeArg *AccountType) (<-chan *types.Account, error) {
	var (
		mkt, pty string
		ty       types.AccountType
	)

	if marketID == nil && partyID == nil && asset == nil && typeArg == nil {
		// Updates on every balance update, on every account, for everyone and shouldn't be allowed for GraphQL.
		return nil, errors.New("at least one query filter must be applied for this subscription")
	}
	if marketID != nil {
		mkt = *marketID
	}
	if partyID != nil {
		pty = *partyID
	}
	if typeArg != nil {
		ty = typeArg.IntoProto()
	}

	req := &protoapi.AccountsSubscribeRequest{
		MarketId: mkt,
		PartyId:  pty,
		Type:     ty,
	}
	stream, err := r.tradingDataClient.AccountsSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.Account)
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
			c <- a.Account
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []*types.Order, error) {
	var (
		mkt, pty string
	)
	if market != nil {
		mkt = *market
	}
	if party != nil {
		pty = *party
	}

	req := &protoapi.OrdersSubscribeRequest{
		MarketId: mkt,
		PartyId:  pty,
	}
	stream, err := r.tradingDataClient.OrdersSubscribe(ctx, req)
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
			c <- o.Orders
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []*types.Trade, error) {
	var (
		mkt, pty string
	)
	if market != nil {
		mkt = *market
	}
	if party != nil {
		pty = *party
	}

	req := &protoapi.TradesSubscribeRequest{
		MarketId: mkt,
		PartyId:  pty,
	}
	stream, err := r.tradingDataClient.TradesSubscribe(ctx, req)
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

func (r *mySubscriptionResolver) Positions(ctx context.Context, party, market *string) (<-chan *types.Position, error) {
	req := &protoapi.PositionsSubscribeRequest{}
	if party != nil {
		req.PartyId = *party
	}
	if market != nil {
		req.MarketId = *market
	}
	stream, err := r.tradingDataClient.PositionsSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.Position)
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
			c <- t.Position
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	req := &protoapi.MarketDepthSubscribeRequest{
		MarketId: market,
	}
	stream, err := r.tradingDataClient.MarketDepthSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.MarketDepth)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			md, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("marketDepth: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("marketDepth: stream closed", logging.Error(err))
				break
			}
			c <- md.MarketDepth
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) MarketDepthUpdate(ctx context.Context, market string) (<-chan *types.MarketDepthUpdate, error) {
	req := &protoapi.MarketDepthUpdatesSubscribeRequest{
		MarketId: market,
	}
	stream, err := r.tradingDataClient.MarketDepthUpdatesSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.MarketDepthUpdate)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			md, err := stream.Recv()
			if err == io.EOF {
				if r.log.GetLevel() == logging.DebugLevel {
					r.log.Debug("marketDepthUpdates: stream closed by server", logging.Error(err))
				}
				break
			}
			if err != nil {
				if r.log.GetLevel() == logging.DebugLevel {
					r.log.Debug("marketDepthUpdates: stream closed", logging.Error(err))
				}
				break
			}

			c <- md.Update
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {

	pinterval, err := convertIntervalToProto(interval)
	if err != nil {
		r.log.Debug("invalid interval for candles subscriptions", logging.Error(err))
	}

	req := &protoapi.CandlesSubscribeRequest{
		MarketId: market,
		Interval: pinterval,
	}
	stream, err := r.tradingDataClient.CandlesSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.Candle)
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
	stream, err := r.tradingDataClient.ObserveGovernance(ctx, &protoapi.ObserveGovernanceRequest{})
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
	stream, err := r.tradingDataClient.ObservePartyProposals(ctx, &protoapi.ObservePartyProposalsRequest{
		PartyId: partyID,
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
	stream, err := r.tradingDataClient.ObserveProposalVotes(ctx, &protoapi.ObserveProposalVotesRequest{
		ProposalId: proposalID,
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
	stream, err := r.tradingDataClient.ObservePartyVotes(ctx, &protoapi.ObservePartyVotesRequest{
		PartyId: partyID,
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

	if proposalID != nil && len(*proposalID) == 0 {
		return r.subscribeProposalVotes(ctx, *proposalID)
	} else if partyID != nil && len(*partyID) == 0 {
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
	req := protoapi.ObserveEventBusRequest{
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

	// build the bidirectionnal stream connection
	stream, err := r.tradingDataClient.ObserveEventBus(ctx, msgSize)
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
			r.busEvents(ctx, stream, out)
		} else {
			r.busEventsWithBatch(ctx, int64(batchSize), stream, out)
		}
	}()

	return out, nil
}

func (r *mySubscriptionResolver) busEvents(
	ctx context.Context,
	stream protoapi.TradingDataService_ObserveEventBusClient,
	out chan []*BusEvent,
) {
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

func (r *mySubscriptionResolver) busEventsWithBatch(
	ctx context.Context,
	batchSize int64, // always non-0 here
	stream protoapi.TradingDataService_ObserveEventBusClient,
	out chan []*BusEvent,
) {
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

// START: Account Resolver

type myAccountResolver VegaResolverRoot

func (r *myAccountResolver) Balance(ctx context.Context, acc *types.Account) (string, error) {
	bal := fmt.Sprintf("%d", acc.Balance)
	return bal, nil
}

func (r *myAccountResolver) Market(ctx context.Context, acc *types.Account) (*types.Market, error) {
	if acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN || acc.Type == types.AccountType_ACCOUNT_TYPE_BOND {
		return r.r.getMarketByID(ctx, acc.MarketId)
	}
	return nil, nil
}

func (r *myAccountResolver) Type(ctx context.Context, obj *types.Account) (AccountType, error) {
	return convertAccountTypeFromProto(obj.Type)
}

func (r *myAccountResolver) Asset(ctx context.Context, obj *types.Account) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

// END: Account Resolver

type myStatisticsResolver VegaResolverRoot

func (r *myStatisticsResolver) BlockHeight(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BlockHeight), nil
}

func (r *myStatisticsResolver) BacklogLength(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BacklogLength), nil
}

func (r *myStatisticsResolver) TotalPeers(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalPeers), nil
}

func (r *myStatisticsResolver) Status(ctx context.Context, obj *types.Statistics) (string, error) {
	return obj.Status.String(), nil
}

func (r *myStatisticsResolver) TxPerBlock(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TxPerBlock), nil
}

func (r *myStatisticsResolver) AverageTxBytes(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.AverageTxBytes), nil
}

func (r *myStatisticsResolver) AverageOrdersPerBlock(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.AverageOrdersPerBlock), nil
}

func (r *myStatisticsResolver) TradesPerSecond(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TradesPerSecond), nil
}

func (r *myStatisticsResolver) OrdersPerSecond(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.OrdersPerSecond), nil
}

func (r *myStatisticsResolver) TotalMarkets(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalMarkets), nil
}

func (r *myStatisticsResolver) TotalAmendOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalAmendOrder), nil
}

func (r *myStatisticsResolver) TotalCancelOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalCancelOrder), nil
}

func (r *myStatisticsResolver) TotalCreateOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalCreateOrder), nil
}

func (r *myStatisticsResolver) TotalOrders(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalOrders), nil
}

func (r *myStatisticsResolver) TotalTrades(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalTrades), nil
}

func (r *myStatisticsResolver) BlockDuration(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BlockDuration), nil
}

func (r *myStatisticsResolver) CandleSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.CandleSubscriptions), nil
}

func (r *myStatisticsResolver) MarketDepthSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.MarketDepthSubscriptions), nil
}

func (r *myStatisticsResolver) MarketDepthUpdateSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.MarketDepthUpdatesSubscriptions), nil
}

func (r *myStatisticsResolver) OrderSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.OrderSubscriptions), nil
}

func (r *myStatisticsResolver) PositionsSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.PositionsSubscriptions), nil
}

func (r *myStatisticsResolver) TradeSubscriptions(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TradeSubscriptions), nil
}

func getParty(ctx context.Context, log *logging.Logger, client TradingDataServiceClient, id string) (*types.Party, error) {
	if len(id) == 0 {
		return nil, nil
	}
	res, err := client.PartyByID(ctx, &protoapi.PartyByIDRequest{PartyId: id})
	if err != nil {
		log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}
