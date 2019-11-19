package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
)

var maxPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

var defaultPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

type SummaryGenerator struct {
	context      context.Context
	tradeStore   *storage.Trade
	orderStore   *storage.Order
	partyStore   *storage.Party
	marketStore  *storage.Market
	accountStore *storage.Account
	tradeService *trades.Svc
}

func NewSummaryGenerator(
	context context.Context,
	tradeStore *storage.Trade,
	orderStore *storage.Order,
	partyStore *storage.Party,
	marketStore *storage.Market,
	accountStore *storage.Account,
	tradeService *trades.Svc) *SummaryGenerator {
	return &SummaryGenerator{
		context,
		tradeStore,
		orderStore,
		partyStore,
		marketStore,
		accountStore,
		tradeService}
}

func (s *SummaryGenerator) Summary(pagination *protoapi.Pagination) (*SummaryResponse, error) {
	p := getMaxPagination(pagination)
	s.commitAllStores()
	parties, err := s.partyStore.GetAll()
	if err != nil {
		return nil, err
	}

	partySummaries := make([]*PartySummary, len(parties))
	for i, party := range parties {
		positions, err := s.tradeService.GetPositionsByParty(s.context, party.Id)
		if err != nil {
			return nil, err
		}
		accounts, err := s.accountStore.GetByParty(party.Id)
		if err != nil {
			return nil, err
		}
		party.Positions = positions
		partySummaries[i] = &PartySummary{
			Party:    party,
			Accounts: accounts,
		}
	}

	mkts, err := s.marketStore.GetAll()
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

	return &SummaryResponse{
		Summary: &Summary{
			Markets: marketSummaries,
			Parties: partySummaries,
		},
	}, nil
}

func (s *SummaryGenerator) MarketSummary(marketId string, pagination *protoapi.Pagination) (*MarketSummaryResponse, error) {
	summary, err := s.marketSummary(marketId, getMaxPagination(pagination))
	if err != nil {
		return nil, err
	}
	return &MarketSummaryResponse{
		Summary: summary,
	}, nil

}

func (s *SummaryGenerator) marketSummary(marketId string, pagination protoapi.Pagination) (*MarketSummary, error) {
	s.commitAllStores()
	market, err := s.marketStore.GetByID(marketId)
	if err != nil {
		return nil, err
	}

	depth, err := s.orderStore.GetMarketDepth(s.context, marketId)
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

func getMaxPagination(pagination *protoapi.Pagination) protoapi.Pagination {
	return getPagination(pagination, maxPagination)
}

func GetDefaultPagination(pagination *protoapi.Pagination) protoapi.Pagination {
	return getPagination(pagination, defaultPagination)
}

func getPagination(pagination *protoapi.Pagination, fallback protoapi.Pagination) protoapi.Pagination {
	p := fallback
	if pagination != nil {
		p = *pagination
	}
	return p
}
