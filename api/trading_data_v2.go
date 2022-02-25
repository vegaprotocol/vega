package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
)

var defaultPaginationV2 = entities.Pagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

type tradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	balanceStore       *sqlstore.Balances
	orderStore         *sqlstore.Orders
	networkLimitsStore *sqlstore.NetworkLimits
}

func (t *tradingDataServiceV2) QueryBalanceHistory(ctx context.Context, req *v2.QueryBalanceHistoryRequest) (*v2.QueryBalanceHistoryResponse, error) {
	if t.balanceStore == nil {
		return nil, fmt.Errorf("sql balance store not available")
	}

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	groupBy := []entities.AccountField{}
	for _, field := range req.GroupBy {
		field, err := entities.AccountFieldFromProto(field)
		if err != nil {
			return nil, fmt.Errorf("parsing group by list: %w", err)
		}
		groupBy = append(groupBy, field)
	}

	balances, err := t.balanceStore.Query(filter, groupBy)
	if err != nil {
		return nil, fmt.Errorf("querying balances: %w", err)
	}

	pbBalances := make([]*v2.AggregatedBalance, len(*balances))
	for i, balance := range *balances {
		pbBalance := balance.ToProto()
		pbBalances[i] = &pbBalance
	}

	return &v2.QueryBalanceHistoryResponse{Balances: pbBalances}, nil
}

func (t *tradingDataServiceV2) OrderByID(ctx context.Context, req *v2.OrderByIDRequest) (*v2.OrderByIDResponse, error) {
	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	version := int32(req.Version)
	order, err := t.orderStore.GetByOrderID(ctx, req.OrderId, &version)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &v2.OrderByIDResponse{Order: order.ToProto()}
	return resp, nil
}

func (t *tradingDataServiceV2) OrderByMarketAndID(ctx context.Context, req *v2.OrderByMarketAndIDRequest) (*v2.OrderByMarketAndIDResponse, error) {
	// This function is no longer needed; IDs are globally unique now, but keep it for compatibility for now
	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	order, err := t.orderStore.GetByOrderID(ctx, req.OrderId, nil)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &v2.OrderByMarketAndIDResponse{Order: order.ToProto()}
	return resp, nil
}

func (t *tradingDataServiceV2) OrderByReference(ctx context.Context, req *v2.OrderByReferenceRequest) (*v2.OrderByReferenceResponse, error) {
	orders, err := t.orderStore.GetByReference(ctx, req.Reference, entities.Pagination{})
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByReference, err)
	}

	if len(orders) == 0 {
		return nil, ErrOrderNotFound
	}
	return &v2.OrderByReferenceResponse{
		Order: orders[0].ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) OrdersByParty(ctx context.Context, req *v2.OrdersByPartyRequest) (*v2.OrdersByPartyResponse, error) {
	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.PaginationFromProto(req.Pagination)
	}

	orders, err := t.orderStore.GetByParty(ctx, req.PartyId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &v2.OrdersByPartyResponse{
		Orders: pbOrders,
	}, nil

}

func (t *tradingDataServiceV2) OrdersByMarket(ctx context.Context, req *v2.OrdersByMarketRequest) (*v2.OrdersByMarketResponse, error) {
	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.PaginationFromProto(req.Pagination)
	}

	orders, err := t.orderStore.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &v2.OrdersByMarketResponse{
		Orders: pbOrders,
	}, nil
}

func (t *tradingDataServiceV2) GetNetworkLimits(ctx context.Context, req *v2.GetNetworkLimitsRequest) (*v2.GetNetworkLimitsResponse, error) {
	limits, err := t.networkLimitsStore.GetLatest(ctx)
	if err != nil {
		return nil, apiError(codes.Unknown, ErrGetNetworkLimits, err)
	}

	return &v2.GetNetworkLimitsResponse{Limits: limits.ToProto()}, nil
}
