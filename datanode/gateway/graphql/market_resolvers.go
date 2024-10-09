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
	"errors"

	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type myMarketResolver VegaResolverRoot

func (r *myMarketResolver) EnableTxReordering(ctx context.Context, obj *types.Market) (bool, error) {
	return obj.EnableTransactionReordering, nil
}

func (r *myMarketResolver) LiquidityProvisionsConnection(
	ctx context.Context,
	market *types.Market,
	party *string,
	live *bool,
	pagination *v2.Pagination,
) (*v2.LiquidityProvisionsConnection, error) {
	var pid string
	if party != nil {
		pid = *party
	}

	var marketID string
	if market != nil {
		marketID = market.Id
	}

	var l bool
	if live != nil {
		l = *live
	}

	req := v2.ListLiquidityProvisionsRequest{
		PartyId:    &pid,
		MarketId:   &marketID,
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

func (r *myMarketResolver) LiquidityProvisions(ctx context.Context, market *vega.Market, party *string, live *bool, pagination *v2.Pagination) (
	*v2.LiquidityProvisionsWithPendingConnection, error,
) {
	var pid string
	if party != nil {
		pid = *party
	}

	var marketID string
	if market != nil {
		marketID = market.Id
	}

	var l bool
	if live != nil {
		l = *live
	}

	req := v2.ListAllLiquidityProvisionsRequest{
		PartyId:    &pid,
		MarketId:   &marketID,
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

func (r *myMarketResolver) Data(ctx context.Context, market *types.Market) (*types.MarketData, error) {
	req := v2.GetLatestMarketDataRequest{
		MarketId: market.Id,
	}
	res, err := r.tradingDataClientV2.GetLatestMarketData(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.MarketData, nil
}

func (r *myMarketResolver) OrdersConnection(
	ctx context.Context,
	market *types.Market,
	pagination *v2.Pagination,
	filter *OrderByPartyIdsFilter,
) (*v2.OrderConnection, error) {
	req := v2.ListOrdersRequest{
		Pagination: pagination,
		Filter: &v2.OrderFilter{
			MarketIds: []string{market.Id},
		},
	}

	if filter != nil {
		req.Filter.PartyIds = filter.PartyIds
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

func (r *myMarketResolver) TradesConnection(ctx context.Context, market *types.Market, dateRange *v2.DateRange, pagination *v2.Pagination) (*v2.TradeConnection, error) {
	req := v2.ListTradesRequest{
		MarketIds:  []string{market.Id},
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

func (r *myMarketResolver) Depth(ctx context.Context, market *types.Market, maxDepth *int) (*types.MarketDepth, error) {
	if market == nil {
		return nil, errors.New("market missing or empty")
	}

	req := v2.GetLatestMarketDepthRequest{MarketId: market.Id}
	if maxDepth != nil {
		if *maxDepth <= 0 {
			return nil, errors.New("invalid maxDepth, must be a positive number")
		}
		reqDepth := uint64(*maxDepth)
		req.MaxDepth = &reqDepth
	}

	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	res, err := r.tradingDataClientV2.GetLatestMarketDepth(ctx, &req)
	if err != nil {
		r.log.Error("trading data client", logging.Error(err))
		return nil, err
	}

	return &types.MarketDepth{
		MarketId:       res.MarketId,
		Buy:            res.Buy,
		Sell:           res.Sell,
		SequenceNumber: res.SequenceNumber,
	}, nil
}

func (r *myMarketResolver) AccountsConnection(ctx context.Context, market *types.Market, partyID *string, pagination *v2.Pagination, includeDerivedParties *bool) (*v2.AccountsConnection, error) {
	filter := v2.AccountFilter{MarketIds: []string{market.Id}}
	ptyID := ""

	if partyID != nil {
		// get margin account for a party
		ptyID = *partyID
		filter.PartyIds = []string{ptyID}
		filter.AccountTypes = []types.AccountType{types.AccountType_ACCOUNT_TYPE_MARGIN}
	} else {
		filter.AccountTypes = []types.AccountType{
			types.AccountType_ACCOUNT_TYPE_INSURANCE,
			types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		}
	}

	req := v2.ListAccountsRequest{Filter: &filter, Pagination: pagination, IncludeDerivedParties: includeDerivedParties}

	res, err := r.tradingDataClientV2.ListAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get market accounts",
			logging.Error(err),
			logging.String("market-id", market.Id),
			logging.String("party-id", ptyID))
		return nil, err
	}
	return res.Accounts, nil
}

func (r *myMarketResolver) DecimalPlaces(_ context.Context, obj *types.Market) (int, error) {
	return int(obj.DecimalPlaces), nil
}

func (r *myMarketResolver) PositionDecimalPlaces(_ context.Context, obj *types.Market) (int, error) {
	return int(obj.PositionDecimalPlaces), nil
}

func (r *myMarketResolver) OpeningAuction(_ context.Context, obj *types.Market) (*AuctionDuration, error) {
	return &AuctionDuration{
		DurationSecs: int(obj.OpeningAuction.Duration),
		Volume:       int(obj.OpeningAuction.Volume),
	}, nil
}

func (r *myMarketResolver) PriceMonitoringSettings(_ context.Context, obj *types.Market) (*PriceMonitoringSettings, error) {
	return PriceMonitoringSettingsFromProto(obj.PriceMonitoringSettings)
}

func (r *myMarketResolver) LiquidityMonitoringParameters(_ context.Context, obj *types.Market) (*LiquidityMonitoringParameters, error) {
	return &LiquidityMonitoringParameters{
		TargetStakeParameters: &TargetStakeParameters{
			TimeWindow:    int(obj.LiquidityMonitoringParameters.TargetStakeParameters.TimeWindow),
			ScalingFactor: obj.LiquidityMonitoringParameters.TargetStakeParameters.ScalingFactor,
		},
	}, nil
}

func (r *myMarketResolver) MarketProposal(ctx context.Context, obj *types.Market) (ProposalNode, error) {
	resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
		ProposalId: &obj.Id,
	})
	// it's possible to not find a proposal as of now.
	// some market are loaded at startup, without
	// going through the proposal phase
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	resolver := (*proposalEdgeResolver)(r)
	if resp.GetData().ProposalType == vega.GovernanceData_TYPE_BATCH {
		return resolver.BatchProposal(ctx, resp.GetData())
	}

	return resp.Data, nil
}

func (r *myMarketResolver) Proposal(ctx context.Context, obj *types.Market) (*vega.GovernanceData, error) {
	resp, err := r.tradingDataClientV2.GetGovernanceData(ctx, &v2.GetGovernanceDataRequest{
		ProposalId: &obj.Id,
	})
	// it's possible to not find a proposal as of now.
	// some market are loaded at startup, without
	// going through the proposal phase
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	return resp.Data, nil
}

func (r *myMarketResolver) RiskFactors(ctx context.Context, obj *types.Market) (*types.RiskFactor, error) {
	rf, err := r.tradingDataClientV2.GetRiskFactors(ctx, &v2.GetRiskFactorsRequest{
		MarketId: obj.Id,
	})
	if err != nil {
		return nil, err
	}

	return rf.RiskFactor, nil
}

func (r *myMarketResolver) CandlesConnection(ctx context.Context, market *types.Market, sinceRaw string, toRaw *string,
	interval vega.Interval, pagination *v2.Pagination,
) (*v2.CandleDataConnection, error) {
	return handleCandleConnectionRequest(ctx, r.tradingDataClientV2, market, sinceRaw, toRaw, interval, pagination)
}

func (r *myMarketResolver) LiquiditySLAParameters(ctx context.Context, obj *types.Market) (*types.LiquiditySLAParameters, error) {
	return obj.LiquiditySlaParams, nil
}

func (r *myMarketResolver) AllowedEmptyAMMLevels(ctx context.Context, obj *types.Market) (int, error) {
	return int(obj.AllowedEmptyAmmLevels), nil
}
