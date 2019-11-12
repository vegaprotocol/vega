package core

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"
)

var maxPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

type SummaryGenerator struct {
	context            context.Context
	marketDataProvider api.MarketDataProvider
	tradeStore         *storage.Trade
	orderStore         *storage.Order
	partyStore         *storage.Party
	marketStore        *storage.Market
}

func NewSummaryGenerator(
	context context.Context,
	marketDataProvider api.MarketDataProvider,
	tradeStore *storage.Trade,
	orderStore *storage.Order,
	partyStore *storage.Party,
	marketStore *storage.Market) *SummaryGenerator {
	return &SummaryGenerator{
		context,
		marketDataProvider,
		tradeStore,
		orderStore,
		partyStore,
		marketStore}
}

func (s *SummaryGenerator) ProtocolSummary(pagination *protoapi.Pagination) (*ProtocolSummaryResponse, error) {
	p := getPagination(pagination)
	parties, err := s.partyStore.GetAll()
	if err != nil {
		return nil, err
	}

	mkts, err := s.marketDataProvider.GetAll(s.context)
	if err != nil {
		return nil, err
	}
	marketSummaries := make([]*MarketSummary, len(mkts))
	for i, mkt := range mkts {
		summary, err := s.marketSummary(mkt.Id, p)
		if err != nil {
			return nil, err
		}
		marketSummaries[i] = summary
	}

	return &ProtocolSummaryResponse{
		Markets: marketSummaries,
		Parties: parties,
	}, nil
}

func (s *SummaryGenerator) MarketSummary(marketId string, pagination *protoapi.Pagination) (*MarketSummaryResponse, error) {
	summary, err := s.marketSummary(marketId, getPagination(pagination))
	if err != nil {
		return nil, err
	}
	return &MarketSummaryResponse{
		Summary: summary,
	}, nil

}

func (s *SummaryGenerator) marketSummary(marketId string, pagination protoapi.Pagination) (*MarketSummary, error) {
	s.commitAllStores()
	market, err := s.marketDataProvider.GetByID(s.context, marketId)
	if err != nil {
		return nil, err
	}
	depth, err := s.marketDataProvider.GetDepth(s.context, marketId)
	if err != nil {
		return nil, err
	}
	trades, err := s.tradeStore.GetByMarket(s.context, marketId, pagination.Skip, pagination.Limit, pagination.Descending)
	if err != nil {
		return nil, err
	}
	orders, err := s.orderStore.GetByMarket(s.context, marketId, pagination.Skip, pagination.Limit, pagination.Descending, nil)
	if err != nil {
		return nil, err
	}

	return &MarketSummary{
		Market:      market,
		Trades:      trades,
		Orders:      orders,
		MarketDepth: depth,
	}, nil
}

func (s *SummaryGenerator) commitAllStores() {
	s.marketStore.Commit()
	s.orderStore.Commit()
	s.tradeStore.Commit()
}

func getPagination(pagination *protoapi.Pagination) protoapi.Pagination {
	p := maxPagination
	if pagination != nil {
		p = *pagination
	}
	return p
}
