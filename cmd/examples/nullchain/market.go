package nullchain

import (
	"fmt"
	"strings"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"

	"code.vegaprotocol.io/protos/vega"
)

func CreateMarketAny(w *Wallet, conn *Connection, proposer *Party, voters ...*Party) (*vega.Market, error) {
	block, _ := conn.LastBlockHeight()
	fmt.Println("Starting blockHeight: ", block)

	now, _ := conn.VegaTime()
	fmt.Printf("Proposing Market Vegatime: %s\n", now)
	txn, reference := MarketProposalTxn(now, proposer.pubkey)
	err := w.SubmitTransaction(conn, proposer, txn)
	if err != nil {
		return nil, err
	}

	// Step foward until proposal is validated
	if err := MoveByDuration(4 * config.BlockDuration); err != nil {
		return nil, err
	}

	// Vote for the proposal
	proposal, err := conn.GetProposalByReference(reference)
	if err != nil {
		return nil, err
	}

	txn = VoteTxn(proposal.Id, vega.Vote_VALUE_YES)
	for _, voter := range voters {
		w.SubmitTransaction(conn, voter, txn)
	}

	// Move forward until enacted
	if err := MoveByDuration(20 * config.BlockDuration); err != nil {
		return nil, err
	}

	// Get the market
	markets, err := conn.GetMarkets()
	if err != nil {
		return nil, err
	}

	return markets[len(markets)-1], nil
}

func SettleMarket(w *Wallet, conn *Connection, oracle *Party) error {
	terminationTxn := OracleTxn("trading.termination", "true")
	err := w.SubmitTransaction(conn, oracle, terminationTxn)
	if err != nil {
		return err
	}

	settlementTxn := OracleTxn(strings.Join([]string{"prices", config.NormalAsset, "value"}, "."), "1000")
	err = w.SubmitTransaction(conn, oracle, settlementTxn)
	if err != nil {
		return err
	}
	// Move time so that it is processed
	return MoveByDuration(5 * config.BlockDuration)
}
