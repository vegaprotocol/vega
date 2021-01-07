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
		MarketID: market.Id,
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
		MarketID:   market.Id,
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
		MarketID:   market.Id,
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

	req := protoapi.MarketDepthRequest{MarketID: market.Id}
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
		MarketID:       res.MarketID,
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
		MarketID:       mkt,
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
			PartyID:  *partyID,
			MarketID: market.Id,
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
		MarketID: market.Id,
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

func (r *myMarketResolver) Fees(ctx context.Context, obj *types.Market) (*Fees, error) {
	return &Fees{
		Factors: &FeeFactors{
			MakerFee:          obj.Fees.Factors.MakerFee,
			InfrastructureFee: obj.Fees.Factors.InfrastructureFee,
			LiquidityFee:      obj.Fees.Factors.LiquidityFee,
		},
	}, nil
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
