package collateral

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

const (
	// The vega asset ID for Tether USD.
	TetherUSD = "bf1e88d19db4b3ca0d1d5bdb73718a01686b18cf731ca26adedf3c8b83802bba"
)

// NOTE: All prices here are expressed in the USDT assets requiring a 6 decimals precision.

var (

	// First withdrawal, submitted by key 1d2f37299f436f3b720b8efbbb6beb4aec9145a2c4f398ccb06280f6fa2503e8
	// for a total amount of 92,932.137159 USDT
	// https://explorer.vega.xyz/txs/0xAB6D8ECF7333618523E343E3EEB39297DF2D44930A792562F1DA68AC96B2FD96
	withdrawal1USDTAmount = num.MustUintFromString("92932137159", 10)

	// Second withdrawal, submitted by key f6074d2f8924f8c1d73f51bce3faa2e615ef5930ef27a7027576cc8ebaa98d9d
	// for a total amount of 13,707.007738 USDT
	// https://explorer.vega.xyz/txs/0xDF029EE77141711FE781BB011C1761375364AD4FAB368E6203CD217DA916487D
	withdrawal2USDTAmount = num.MustUintFromString("13707007738", 10)

	// Third withdrawal, submitted by key f6074d2f8924f8c1d73f51bce3faa2e615ef5930ef27a7027576cc8ebaa98d9d
	// for a total amount of 8,100.000000 USDT
	// https://explorer.vega.xyz/txs/0xA1337F3AAB6CAE8403CABC7E5CCDF95E41C3833B6E952905A7EA734F58B34A94
	withdrawal3USDTAmount = num.MustUintFromString("8100000000", 10)

	// The total amount of USDT withdrawan by.
	totalWithdrawnUSDTAmount = num.Sum(withdrawal1USDTAmount, withdrawal2USDTAmount, withdrawal3USDTAmount)

	// This slice will contains all recommended amounts to be returns to the parties which endured a lost
	// as suggested by the `Flagged withdrawals vs. losses` from the report.
	amountsReturned = []struct {
		Pubkey string
		Amount *num.Uint
	}{
		{
			Pubkey: "89c98f0e1039935b5d7f5b8d6d0660790a8e507d0c4234b6cafb7dbf88ad25ca",
			Amount: num.MustUintFromString("110027790000", 10), // 110,027.79 USDT
		},
		{
			Pubkey: "c8f5d32a8554dbddfa80946fe9ac42d156356f869256aa0a632e5152d45b1316",
			Amount: num.MustUintFromString("2872880000", 10), // 2,872.88 USDT
		},
		{
			Pubkey: "426f40b09ea2388c22e7c409b6e979747597316939ed6b422c5b935069ad4814",
			Amount: num.MustUintFromString("128530000", 10), // 128.53 USDT
		},
		{
			Pubkey: "519d2af4058af1bed4e05859afa6a15cb1791166df8f0fe3f70a783a13232440",
			Amount: num.MustUintFromString("114110000", 10), // 114.11 USDT
		},
		{
			Pubkey: "1d150c717d349e901cc26e511f776c323c1b8a8dbb0e7717183f2a1e9f3482d7",
			Amount: num.MustUintFromString("80840000", 10), // 80.84 USDT
		},
		{
			Pubkey: "36e73d371b25f0d97ce7813d688c42e61792bda80c00c9cf6d8bf9424a539bf5",
			Amount: num.MustUintFromString("71080000", 10), // 71.08 USDT
		},
		{
			Pubkey: "0a9b24a83cb661e68a2069a413cc2603f0f4804b165621806fa8a014fb0ed4b5",
			Amount: num.MustUintFromString("50640000", 10), // 50.64 USDT
		},
		// this last entry is the key from the perpetrator, and the funds to be returned to them
		{
			Pubkey: "1d2f37299f436f3b720b8efbbb6beb4aec9145a2c4f398ccb06280f6fa2503e8",
			Amount: num.MustUintFromString("1393274897", 10), // 1,393.274897 USDT
		},
	}
)

// OnStateLoaded is a hook call by the snapshot engine,
// it will be called once the whole state of the core have been
// restored from the snaphshot, then execute the migration to
// credit back the different accounts.
func (e *Engine) OnStateLoaded(ctx context.Context) error {
	if vgcontext.InProgressUpgradeFrom(ctx, "v0.74.3") {
		ExecuteMigration744(ctx, e.broker, e.log, e)
	}
	return nil
}

// ExecuteMigration744 This function will iterate other the map and execute Deposits for all
// the keys back to their general account.
func ExecuteMigration744(
	ctx context.Context,
	broker Broker,
	log *logging.Logger,
	c *Engine,
) {
	log.Info("starting migration 74.4")

	// keep track of the ledger movement so they can be sent to the datanode
	ledgerMovements := []*types.LedgerMovement{}

	// we take a copy of the total amount which was attempted to be withdrawn
	// and will decrement it with all the deposits executed to ensure that nothing
	// is left out but also not too much funds are being credited
	totalAmount := totalWithdrawnUSDTAmount.Clone()
	for _, entry := range amountsReturned {
		// to start with, ensure that the amount left to distributed is > to the amount that needs to be credited to that pubkey
		if totalAmount.LT(entry.Amount) {
			log.Panic("total amount left to distribute is too low", logging.BigUint("amount", totalAmount), logging.BigUint("toCredit", entry.Amount))
		}

		// before the deposit we get the amount from the party general account balance
		// these should never fail has the party should already have a general account
		accountBefore, err := c.GetPartyGeneralAccount(entry.Pubkey, TetherUSD)
		if err != nil {
			log.Panic("unexpected error loading account before deposit", logging.Error(err))
		}

		log.Info("account state before deposit", logging.String("pubkey", entry.Pubkey), logging.BigUint("balance", accountBefore.Balance))

		// execute the deposit
		ledgerMovement, err := c.Deposit(ctx, entry.Pubkey, TetherUSD, entry.Amount)
		if err != nil {
			log.Panic("unexpected error during deposit", logging.Error(err))
		}

		// loading balance after the deposit to ensure the right amount was deposited
		accountAfter, err := c.GetPartyGeneralAccount(entry.Pubkey, TetherUSD)
		if err != nil {
			log.Panic("unexpected error loading account afer deposit", logging.Error(err))
		}

		log.Info("account state after deposit", logging.String("pubkey", entry.Pubkey), logging.BigUint("balance", accountAfter.Balance))

		// here we just assert the right mount has been deposited
		expectedBalance := num.Sum(accountBefore.Balance, entry.Amount)
		if accountAfter.Balance.NEQ(expectedBalance) {
			log.Panic("invalid balance after deposit", logging.BigUint("expected", expectedBalance), logging.BigUint("got", accountAfter.Balance))
		}

		// decrement the totalAmount left to distribute
		totalAmount.Sub(totalAmount, entry.Amount)

		// add  the ledgerMovement to the slice
		ledgerMovements = append(ledgerMovements, ledgerMovement)
	}

	// ensure the total amount left is zero and all balances have been updated properly
	if !totalAmount.IsZero() {
		log.Panic("funds have not been fully distributed", logging.BigUint("remaining", totalAmount))
	}

	// send the events to the datanode
	broker.Stage(events.NewLedgerMovements(ctx, ledgerMovements))
}
