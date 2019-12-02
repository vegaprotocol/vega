package main

import (
	"context"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/hashicorp/go-multierror"
)

//TODO (WG 05/11/2019): instantiating dependencies internally while WIP, the final dependencies will get incjeted from outside the package.
func getDependencies(log *logging.Logger, config storage.Config) (*dependencies, error) {

	ctx, cancel := context.WithCancel(context.Background())
	var errs *multierror.Error

	orderStore, err := storage.NewOrders(log, config, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	tradeStore, err := storage.NewTrades(log, config, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	candleStore, err := storage.NewCandles(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	marketStore, err := storage.NewMarkets(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	partyStore, err := storage.NewParties(config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	accountStore, err := storage.NewAccounts(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	transferResponseStore, err := storage.NewTransferResponses(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	riskStore, err := storage.NewRisks(config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	tradeService, err := trades.NewService(log, trades.NewDefaultConfig(), tradeStore, riskStore)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = errs.ErrorOrNil()
	if err != nil {
		return nil, err
	}

	orderBuffer := buffer.NewOrder(orderStore)
	tradeBuffer := buffer.NewTrade(tradeStore)
	candleBuffer := buffer.NewCandle(candleStore)
	marketBuffer := buffer.NewMarket(marketStore)
	partyBuffer := buffer.NewParty(partyStore)
	accountBuffer := buffer.NewAccount(accountStore)
	transferResponseBuffer := buffer.NewTransferResponse(transferResponseStore)
	settleBuf := buffer.NewSettlement()

	executionConfig := execution.NewDefaultConfig("")
	timeService := vegatime.New(vegatime.NewDefaultConfig())
	engine := execution.NewEngine(
		log,
		executionConfig,
		timeService,
		orderBuffer,
		tradeBuffer,
		candleBuffer,
		marketBuffer,
		partyBuffer,
		accountBuffer,
		transferResponseBuffer,
		settleBuf,
		[]types.Market{}, // WG (21/11/2019): Please note these get added from config in scenariorunner/engine.go/NewEngine just now, but can definitely be moved here.
	)

	return &dependencies{
		ctx:          ctx,
		vegaTime:     timeService,
		execution:    engine,
		orderBuf:     orderBuffer,
		tradeBuf:     tradeBuffer,
		partyBuf:     partyBuffer,
		marketBuf:    marketBuffer,
		accountBuf:   accountBuffer,
		candleBuf:    candleBuffer,
		partyStore:   partyStore,
		orderStore:   orderStore,
		tradeStore:   tradeStore,
		marketStore:  marketStore,
		accountStore: accountStore,
		candleStore:  candleStore,
		tradeService: tradeService,
	}, nil
}

type dependencies struct {
	ctx       context.Context
	vegaTime  *vegatime.Svc
	execution *execution.Engine

	orderBuf   *buffer.Order
	tradeBuf   *buffer.Trade
	partyBuf   *buffer.Party
	marketBuf  *buffer.Market
	accountBuf *buffer.Account
	candleBuf  *buffer.Candle

	partyStore   *storage.Party
	orderStore   *storage.Order
	tradeStore   *storage.Trade
	marketStore  *storage.Market
	accountStore *storage.Account
	candleStore  *storage.Candle
	tradeService *trades.Svc
}
