package main

import (
	"fmt"

	timefoward "code.vegaprotocol.io/vega/cmd/examples/nullchain/timeforward"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"

	"code.vegaprotocol.io/protos/vega"
)

func proposeAndVoteMarket(w *Wallets, conn *Connection) {
	block, _ := conn.LastBlockHeight()
	fmt.Println("Starting blockHeight: ", block)

	// Get some users from the wallet
	parties := w.GetParties()

	now, _ := conn.VegaTime()
	fmt.Printf("Proposing Market Vegatime: %s\n")
	txn := MarketProposalTxn(now, parties[0].pubkey)
	err := w.SubmitTransaction(conn, parties[0], txn)
	if err != nil {
		return
	}

	// Step foward until proposal is validated
	timefoward.MoveByDuration(2 * config.BlockDuration)

	// Step forward a block so that it is validated
	timefoward.MoveByDuration(2 * config.BlockDuration)

	// Vote for the proposal
	proposalID, _ := conn.GetProposalByParty(parties[0])
	txn = VoteTxn(proposalID, vega.Vote_VALUE_YES)

	w.SubmitTransaction(conn, parties[1], txn)
	w.SubmitTransaction(conn, parties[2], txn)

	// Move forward until enacted
	timefoward.MoveByDuration(20 * config.BlockDuration)
	block, _ = conn.LastBlockHeight()
	fmt.Println("Ending blockHeight:   ", block)
	fmt.Println()

	// Get the market
	market, _ := conn.GetMarket()
	fmt.Println("MarketID:    ", market.Id)
	fmt.Println("State:       ", market.State)
	fmt.Println("TradingMode: ", market.TradingMode)
}
