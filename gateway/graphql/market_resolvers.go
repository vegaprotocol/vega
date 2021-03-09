package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/vegatime"
)

type myMarketResolver VegaResolverRoot

func (r *myMarketResolver) LiquidityProvisions(
	ctx context.Context,
	market *types.Market,
	party *string,
) ([]*types.LiquidityProvision, error) {
	var pid string
	if party != nil {
		pid = *party
	}

	req := protoapi.LiquidityProvisionsRequest{
		Party:  pid,
		Market: market.Id,
	}
	res, err := r.tradingDataClient.LiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return res.LiquidityProvisions, nil
}

func (r *myMarketResolver) Data(ctx context.Context, market *types.Market) (*types.MarketData, error) {
	req := protoapi.MarketDataByIDRequest{
		MarketId: market.Id,
	}
	res, err := r.tradingDataClient.MarketDataByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.MarketData, nil
}

func (r *myMarketResolver) Orders(ctx context.Context, market *types.Market,
	skip, first, last *int) ([]*types.Order, error) {
	p := makePagination(skip, first, last)
	req := protoapi.OrdersByMarketRequest{
		MarketId:   market.Id,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrdersByMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Orders, nil
}

func (r *myMarketResolver) Trades(ctx context.Context, market *types.Market,
	skip, first, last *int) ([]*types.Trade, error) {
	p := makePagination(skip, first, last)
	req := protoapi.TradesByMarketRequest{
		MarketId:   market.Id,
		Pagination: p,
	}
	res, err := r.tradingDataClient.TradesByMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return res.Trades, nil
}

func (r *myMarketResolver) Depth(ctx context.Context, market *types.Market, maxDepth *int) (*types.MarketDepth, error) {

	if market == nil {
		return nil, errors.New("market missing or empty")
	}

	req := protoapi.MarketDepthRequest{MarketId: market.Id}
	if maxDepth != nil {
		if *maxDepth <= 0 {
			return nil, errors.New("invalid maxDepth, must be a positive number")
		}
		req.MaxDepth = uint64(*maxDepth)
	}

	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	res, err := r.tradingDataClient.MarketDepth(ctx, &req)
	if err != nil {
		r.log.Error("trading data client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return &types.MarketDepth{
		MarketId:       res.MarketId,
		Buy:            res.Buy,
		Sell:           res.Sell,
		SequenceNumber: res.SequenceNumber,
	}, nil
}

func (r *myMarketResolver) Candles(ctx context.Context, market *types.Market,
	sinceRaw string, interval Interval) ([]*types.Candle, error) {
	pinterval, err := convertIntervalToProto(interval)
	if err != nil {
		r.log.Debug("interval convert error", logging.Error(err))
	}

	since, err := vegatime.Parse(sinceRaw)
	if err != nil {
		return nil, err
	}

	var mkt string
	if market != nil {
		mkt = market.Id
	}

	req := protoapi.CandlesRequest{
		MarketId:       mkt,
		SinceTimestamp: since.UnixNano(),
		Interval:       pinterval,
	}
	res, err := r.tradingDataClient.Candles(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Candles, nil
}

// Accounts ...
// if partyID specified get margin account for the given market
// if nil return the insurance pool for the market
func (r *myMarketResolver) Accounts(ctx context.Context, market *types.Market, partyID *string) ([]*types.Account, error) {
	// get margin account for a party
	if partyID != nil {
		req := protoapi.PartyAccountsRequest{
			PartyId:  *partyID,
			MarketId: market.Id,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    "",
		}
		res, err := r.tradingDataClient.PartyAccounts(ctx, &req)
		if err != nil {
			r.log.Error("unable to get PartyAccounts",
				logging.Error(err),
				logging.String("market-id", market.Id),
				logging.String("party-id", *partyID))
			return []*types.Account{}, customErrorFromStatus(err)
		}
		return res.Accounts, nil
	}
	// get accounts for the market
	req := protoapi.MarketAccountsRequest{
		MarketId: market.Id,
		Asset:    "", // all assets
	}
	res, err := r.tradingDataClient.MarketAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get MarketAccounts",
			logging.Error(err),
			logging.String("market-id", market.Id))
		return []*types.Account{}, customErrorFromStatus(err)
	}
	return res.Accounts, nil
}

func (r *myMarketResolver) DecimalPlaces(ctx context.Context, obj *types.Market) (int, error) {
	return int(obj.DecimalPlaces), nil
}

func (r *myMarketResolver) Name(ctx context.Context, obj *types.Market) (string, error) {
	return obj.TradableInstrument.Instrument.Name, nil
}

func (r *myMarketResolver) OpeningAuction(ctx context.Context, obj *types.Market) (*AuctionDuration, error) {
	return &AuctionDuration{
		DurationSecs: int(obj.OpeningAuction.Duration),
		Volume:       int(obj.OpeningAuction.Volume),
	}, nil
}

func (r *myMarketResolver) PriceMonitoringSettings(ctx context.Context, obj *types.Market) (*PriceMonitoringSettings, error) {
	return PriceMonitoringSettingsFromProto(obj.PriceMonitoringSettings)
}

func (r *myMarketResolver) TradingModeConfig(ctx context.Context, obj *types.Market) (TradingMode, error) {
	return TradingModeConfigFromProto(obj.TradingModeConfig)
}

func (r *myMarketResolver) TargetStakeParameters(ctx context.Context, obj *types.Market) (*TargetStakeParameters, error) {
	return &TargetStakeParameters{
		TimeWindow:    int(obj.TargetStakeParameters.TimeWindow),
		ScalingFactor: obj.TargetStakeParameters.ScalingFactor,
	}, nil

}

func (r *myMarketResolver) TradingMode(ctx context.Context, obj *types.Market) (MarketTradingMode, error) {
	return convertMarketTradingModeFromProto(obj.TradingMode)
}

func (r *myMarketResolver) State(ctx context.Context, obj *types.Market) (MarketState, error) {
	return convertMarketStateFromProto(obj.State)
}

func (r *myMarketResolver) Proposal(ctx context.Context, obj *types.Market) (*types.GovernanceData, error) {
	resp, err := r.tradingDataClient.GetProposalByID(ctx, &protoapi.GetProposalByIDRequest{
		ProposalId: obj.Id,
	})
	// it's possible to not find a proposal as of now.
	// some market are loaded at startup, without
	// going through the proposal phase
	if err != nil {
		return nil, nil
	}
	return resp.Data, nil
}

/*func (r *myMarketResolver) MarketTimestamps(ctx context.Context, obj *types.Market) (*MarketTimestamps, error) {
	mts := &MarketTimestamps{
		Pending: vegatime.Format(vegatime.Unix(obj.MarketTimestamps.Pending, 0)),
		Open:    vegatime.Format(vegatime.Unix(obj.MarketTimestamps.Open, 0)),
		Close:   vegatime.Format(vegatime.Unix(obj.MarketTimestamps.Close, 0)),
	}
	return mts, nil
}*/
