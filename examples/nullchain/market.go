package nullchain

import (
	"errors"
	"strings"

	config "code.vegaprotocol.io/vega/examples/nullchain/config"

	"code.vegaprotocol.io/protos/vega"
)

var ErrMarketCreationFailed = errors.New("market creation failed")

func CreateMarketAny(w *Wallet, conn *Connection, proposer *Party, voters ...*Party) (*vega.Market, error) {
	now, _ := conn.VegaTime()
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

	if len(markets) == 0 {
		return nil, ErrMarketCreationFailed
	}

	// Return the last market as that *should* be the newest one
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
	return MoveByDuration(10 * config.BlockDuration)
}
