package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	timefoward "code.vegaprotocol.io/vega/cmd/examples/nullchain/timeforward"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"

	datanode "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/protos/vega"
	walletpb "code.vegaprotocol.io/protos/vega/wallet/v1"
)

func randomOrder(marketID string, side vega.Side, now time.Time) *walletpb.SubmitTransactionRequest {
	price := 1000
	percent := (price / 100)
	// perturb the price a little
	perturbation := rand.Intn(2*percent) - percent

	return OrderTxn(marketID, uint64(price+perturbation), 10, side, vega.Order_TYPE_LIMIT, now.Add(356*24*time.Hour))
}

func doSomeTrading(w *Wallets, conn *Connection) {
	// Get a market to trade in
	market, _ := conn.GetMarket()
	now, _ := conn.VegaTime()
	fmt.Printf("Starting Trading Vegatime: %s\n", now)

	parties := w.GetParties()

	for i := 0; i < 100; i++ {
		buy := randomOrder(market.Id, vega.Side_SIDE_BUY, now)
		sell := randomOrder(market.Id, vega.Side_SIDE_SELL, now)

		// Submit a buy and a sell order
		w.SubmitTransaction(conn, parties[1], buy)
		w.SubmitTransaction(conn, parties[2], sell)
		timefoward.MoveByDuration(config.BlockDuration)

	}

	totalTrades, err := conn.datanode.TradesByMarket(context.Background(), &datanode.TradesByMarketRequest{MarketId: market.Id})
	if err != nil {
		return
	}
	fmt.Println("Total trades: ", len(totalTrades.Trades))
}
