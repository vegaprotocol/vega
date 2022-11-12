package gql

import (
	"context"
	"errors"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
	vega "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/grpc"
)

// MarketDepth returns the market depth resolver.
func (r *VegaResolverRoot) MarketDepth() MarketDepthResolver {
	return (*myMarketDepthResolver)(r)
}

func (r *VegaResolverRoot) ObservableMarketDepth() ObservableMarketDepthResolver {
	return (*myObservableMarketDepthResolver)(r)
}

// MarketDepthUpdate returns the market depth update resolver.
func (r *VegaResolverRoot) MarketDepthUpdate() MarketDepthUpdateResolver {
	return (*myMarketDepthUpdateResolver)(r)
}

func (r *VegaResolverRoot) ObservableMarketDepthUpdate() ObservableMarketDepthUpdateResolver {
	return (*myObservableMarketDepthUpdateResolver)(r)
}

// MarketData returns the market data resolver.
func (r *VegaResolverRoot) MarketData() MarketDataResolver {
	return (*myMarketDataResolver)(r)
}

func (r *VegaResolverRoot) ObservableMarketData() ObservableMarketDataResolver {
	return (*myObservableMarketDataResolver)(r)
}

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

func (r *myMarketDataResolver) IndicativePrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.IndicativePrice, nil
}

func (r *myMarketDataResolver) IndicativeVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.IndicativeVolume, 10), nil
}

func (r *myMarketDataResolver) BestBidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.BestBidPrice, nil
}

func (r *myMarketDataResolver) BestStaticBidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.BestStaticBidPrice, nil
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
	return m.BestOfferPrice, nil
}

func (r *myMarketDataResolver) BestStaticOfferPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.BestStaticOfferPrice, nil
}

func (r *myMarketDataResolver) BestStaticOfferVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestStaticOfferVolume, 10), nil
}

func (r *myMarketDataResolver) BestOfferVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestOfferVolume, 10), nil
}

func (r *myMarketDataResolver) MidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.MidPrice, nil
}

func (r *myMarketDataResolver) StaticMidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.StaticMidPrice, nil
}

func (r *myMarketDataResolver) MarkPrice(_ context.Context, m *types.MarketData) (string, error) {
	return m.MarkPrice, nil
}

func (r *myMarketDataResolver) Timestamp(_ context.Context, m *types.MarketData) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

func (r *myMarketDataResolver) Commitments(ctx context.Context, m *types.MarketData) (*MarketDataCommitments, error) {
	// get all the commitments for the given market
	req := v2.ListLiquidityProvisionsRequest{
		MarketId: &m.Market,
	}
	res, err := r.tradingDataClientV2.ListLiquidityProvisions(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	// now we split all the sells and buys
	sells := []*types.LiquidityOrderReference{}
	buys := []*types.LiquidityOrderReference{}

	for _, v := range res.LiquidityProvisions.Edges {
		sells = append(sells, v.Node.Sells...)
		buys = append(buys, v.Node.Buys...)
	}

	return &MarketDataCommitments{
		Sells: sells,
		Buys:  buys,
	}, nil
}

func (r *myMarketDataResolver) PriceMonitoringBounds(ctx context.Context, obj *types.MarketData) ([]*PriceMonitoringBounds, error) {
	ret := make([]*PriceMonitoringBounds, 0, len(obj.PriceMonitoringBounds))
	for _, b := range obj.PriceMonitoringBounds {
		probability, err := strconv.ParseFloat(b.Trigger.Probability, 64)
		if err != nil {
			return nil, err
		}

		bounds := &PriceMonitoringBounds{
			MinValidPrice: b.MinValidPrice,
			MaxValidPrice: b.MaxValidPrice,
			Trigger: &PriceMonitoringTrigger{
				HorizonSecs:          int(b.Trigger.Horizon),
				Probability:          probability,
				AuctionExtensionSecs: int(b.Trigger.AuctionExtension),
			},
			ReferencePrice: b.ReferencePrice,
		}
		ret = append(ret, bounds)
	}
	return ret, nil
}

func (r *myMarketDataResolver) Market(ctx context.Context, m *types.MarketData) (*types.Market, error) {
	return r.r.getMarketByID(ctx, m.Market)
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

func (r *myMarketDataResolver) NextMarkToMarket(_ context.Context, m *types.MarketData) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.NextMarkToMarket)), nil
}

type myObservableMarketDataResolver myMarketDataResolver

func (r *myObservableMarketDataResolver) MarketID(ctx context.Context, m *types.MarketData) (string, error) {
	return m.Market, nil
}

func (r *myObservableMarketDataResolver) AuctionStart(ctx context.Context, m *types.MarketData) (*string, error) {
	return (*myMarketDataResolver)(r).AuctionStart(ctx, m)
}

func (r *myObservableMarketDataResolver) AuctionEnd(ctx context.Context, m *types.MarketData) (*string, error) {
	return (*myMarketDataResolver)(r).AuctionEnd(ctx, m)
}

func (r *myObservableMarketDataResolver) IndicativePrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).IndicativePrice(ctx, m)
}

func (r *myObservableMarketDataResolver) IndicativeVolume(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).IndicativeVolume(ctx, m)
}

func (r *myObservableMarketDataResolver) BestBidPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestBidPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) BestStaticBidPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestStaticBidPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) BestStaticBidVolume(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestStaticBidVolume(ctx, m)
}

func (r *myObservableMarketDataResolver) OpenInterest(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.OpenInterest, 10), nil
}

func (r *myObservableMarketDataResolver) BestBidVolume(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestBidVolume(ctx, m)
}

func (r *myObservableMarketDataResolver) BestOfferPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestOfferPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) BestStaticOfferPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestStaticOfferPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) BestStaticOfferVolume(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestStaticOfferVolume(ctx, m)
}

func (r *myObservableMarketDataResolver) BestOfferVolume(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).BestOfferVolume(ctx, m)
}

func (r *myObservableMarketDataResolver) MidPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).MidPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) StaticMidPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).StaticMidPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) MarkPrice(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).MarkPrice(ctx, m)
}

func (r *myObservableMarketDataResolver) Timestamp(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).Timestamp(ctx, m)
}

func (r *myObservableMarketDataResolver) PriceMonitoringBounds(ctx context.Context, obj *types.MarketData) ([]*PriceMonitoringBounds, error) {
	return (*myMarketDataResolver)(r).PriceMonitoringBounds(ctx, obj)
}

// ExtensionTrigger same as Trigger.
func (r *myObservableMarketDataResolver) ExtensionTrigger(ctx context.Context, m *types.MarketData) (vega.AuctionTrigger, error) {
	return m.ExtensionTrigger, nil
}

func (r *myObservableMarketDataResolver) MarketValueProxy(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).MarketValueProxy(ctx, m)
}

func (r *myObservableMarketDataResolver) LiquidityProviderFeeShare(ctx context.Context, m *types.MarketData) ([]*ObservableLiquidityProviderFeeShare, error) {
	out := make([]*ObservableLiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare))
	for _, v := range m.LiquidityProviderFeeShare {
		out = append(out, &ObservableLiquidityProviderFeeShare{
			PartyID:               v.Party,
			EquityLikeShare:       v.EquityLikeShare,
			AverageEntryValuation: v.AverageEntryValuation,
		})
	}
	return out, nil
}

func (r *myObservableMarketDataResolver) NextMarkToMarket(ctx context.Context, m *types.MarketData) (string, error) {
	return (*myMarketDataResolver)(r).NextMarkToMarket(ctx, m)
}

// END: MarketData resolver

// BEGIN: Market Depth Resolver

type myMarketDepthResolver VegaResolverRoot

func (r *myMarketDepthResolver) LastTrade(ctx context.Context, md *types.MarketDepth) (*types.Trade, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := v2.GetLastTradeRequest{MarketId: md.MarketId}
	res, err := r.tradingDataClientV2.GetLastTrade(ctx, &req)
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

type myObservableMarketDepthResolver myMarketDepthResolver

func (r *myObservableMarketDepthResolver) LastTrade(ctx context.Context, md *types.MarketDepth) (*MarketDepthTrade, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := v2.GetLastTradeRequest{MarketId: md.MarketId}
	res, err := r.tradingDataClientV2.GetLastTrade(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return &MarketDepthTrade{ID: res.Trade.Id, Price: res.Trade.Price, Size: strconv.FormatUint(res.Trade.Size, 10)}, nil
}

func (r *myObservableMarketDepthResolver) SequenceNumber(ctx context.Context, md *types.MarketDepth) (string, error) {
	return (*myMarketDepthResolver)(r).SequenceNumber(ctx, md)
}

// END: Market Depth Resolver

// BEGIN: Market Depth Update Resolver

type myMarketDepthUpdateResolver VegaResolverRoot

func (r *myMarketDepthUpdateResolver) SequenceNumber(ctx context.Context, md *types.MarketDepthUpdate) (string, error) {
	return strconv.FormatUint(md.SequenceNumber, 10), nil
}

func (r *myMarketDepthUpdateResolver) PreviousSequenceNumber(ctx context.Context, md *types.MarketDepthUpdate) (string, error) {
	return strconv.FormatUint(md.PreviousSequenceNumber, 10), nil
}

func (r *myMarketDepthUpdateResolver) Market(ctx context.Context, md *types.MarketDepthUpdate) (*types.Market, error) {
	return r.r.getMarketByID(ctx, md.MarketId)
}

type myObservableMarketDepthUpdateResolver myMarketDepthUpdateResolver

func (r *myObservableMarketDepthUpdateResolver) SequenceNumber(ctx context.Context, md *types.MarketDepthUpdate) (string, error) {
	return (*myMarketDepthUpdateResolver)(r).SequenceNumber(ctx, md)
}

func (r *myObservableMarketDepthUpdateResolver) PreviousSequenceNumber(ctx context.Context, md *types.MarketDepthUpdate) (string, error) {
	return (*myMarketDepthUpdateResolver)(r).PreviousSequenceNumber(ctx, md)
}

// END: Market Depth Update Resolver

func (r *mySubscriptionResolver) MarketsDepth(ctx context.Context, marketIds []string) (<-chan []*types.MarketDepth, error) {
	req := &v2.ObserveMarketsDepthRequest{
		MarketIds: marketIds,
	}
	stream, err := r.tradingDataClientV2.ObserveMarketsDepth(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	return grpcStreamToGraphQlChannel[*v2.ObserveMarketsDepthResponse](r.log, "marketsDepth", stream,
		func(md *v2.ObserveMarketsDepthResponse) []*types.MarketDepth {
			return md.MarketDepth
		}), nil
}

func (r *mySubscriptionResolver) MarketsDepthUpdate(ctx context.Context, marketIDs []string) (<-chan []*types.MarketDepthUpdate, error) {
	req := &v2.ObserveMarketsDepthUpdatesRequest{
		MarketIds: marketIDs,
	}
	stream, err := r.tradingDataClientV2.ObserveMarketsDepthUpdates(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	return grpcStreamToGraphQlChannel[*v2.ObserveMarketsDepthUpdatesResponse](r.log, "marketsDepthUpdate", stream,
		func(md *v2.ObserveMarketsDepthUpdatesResponse) []*types.MarketDepthUpdate {
			return md.Update
		}), nil
}

func (r *mySubscriptionResolver) MarketsData(ctx context.Context, marketIds []string) (<-chan []*types.MarketData, error) {
	req := &v2.ObserveMarketsDataRequest{
		MarketIds: marketIds,
	}
	stream, err := r.tradingDataClientV2.ObserveMarketsData(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	return grpcStreamToGraphQlChannel[*v2.ObserveMarketsDataResponse](r.log, "marketsdata", stream,
		func(md *v2.ObserveMarketsDataResponse) []*types.MarketData {
			return md.MarketData
		}), nil
}

type grpcStream[T any] interface {
	Recv() (T, error)
	grpc.ClientStream
}

func grpcStreamToGraphQlChannel[T any, Y any](log *logging.Logger, observableType string, stream grpcStream[T], grpcStreamTypeToGraphQlType func(T) Y) chan Y {
	c := make(chan Y)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			md, err := stream.Recv()
			if err == io.EOF {
				log.Error(observableType+": stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				log.Error(observableType+": stream closed", logging.Error(err))
				break
			}
			c <- grpcStreamTypeToGraphQlType(md)
		}
	}()
	return c
}
