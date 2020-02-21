package collateral

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral/mocks"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testMarketID    = "7CPSHJB35AIQBTNMIE6NLFPZGHOYRQ3D"
	testMarketAsset = "BTC"
)

type testEngine struct {
	*Engine
	ctrl               *gomock.Controller
	buf                *mocks.MockAccountBuffer
	lossBuf            *mocks.MockLossSocializationBuf
	systemAccs         []*types.Account
	marketInsuranceID  string
	marketSettlementID string
}

func TestCollateralTransfer(t *testing.T) {
	t.Run("test creating new - should create market accounts", testNew)
	t.Run("test collecting buys - both insurance and sufficient in trader accounts", testTransferLoss)
	t.Run("test collecting buys - trader account not empty, but insufficient", testTransferComplexLoss)
	t.Run("test collecting buys - trader missing some accounts", testTransferLossMissingTraderAccounts)
	t.Run("test collecting sells - cases where settle account is full + where insurance pool is tapped", testDistributeWin)
	t.Run("test collecting both buys and sells - Successfully collect buy and sell in a single call", testProcessBoth)
	t.Run("test distribution insufficient funds - Transfer losses (partial), distribute wins pro-rate", testProcessBothProRated)
}

func TestCollateralMarkToMarket(t *testing.T) {
	t.Run("Mark to Market distribution, insufficient funcs - complex scenario", testProcessBothProRatedMTM)
	t.Run("Mark to Market successful", testMTMSuccess)
}

func TestAddTraderToMarket(t *testing.T) {
	t.Run("Successful calls adding new traders (one duplicate, one actual new)", testAddTrader)
	t.Run("Can add a trader margin account if general account for asset exists", testAddMarginAccount)
	t.Run("Fail add trader margin account if no general account for asset exisrts", testAddMarginAccountFail)
}

func TestRemoveDistressed(t *testing.T) {
	t.Run("Successfully remove distressed trader and transfer balance", testRemoveDistressedBalance)
	t.Run("Successfully remove distressed trader, no balance transfer", testRemoveDistressedNoBalance)
}

func TestMarginUpdateOnOrder(t *testing.T) {
	t.Run("Successfully update margin on new order if general account balance is OK", testMarginUpdateOnOrderOK)
	t.Run("Faile update margin on new order if general account balance is OK", testMarginUpdateOnOrderFail)
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", 0)
	eng.Finish()
}

func testAddMarginAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// test balance is 0 when created
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), acc.Balance)
}

func testAddMarginAccountFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Error(t, err, ErrNoGeneralAccountWhenCreateMarginAccount)

}

func testAddTrader(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	general := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(general, 100000)
	assert.Nil(t, err)

	expectedGeneralBalance := int64(100000)

	// check the amount on each account now
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), acc.Balance)

	acc, err = eng.Engine.GetAccountByID(general)
	assert.Nil(t, err)
	assert.Equal(t, expectedGeneralBalance, acc.Balance)

}

func testTransferLoss(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts, set balance for money trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	_ = eng.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	_ = eng.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.UpdateBalance(marginMoneyTrader, 100000)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, 3*price, resp.Balances[0].Balance+responses[1].Balances[0].Balance)
	// there should be 2 ledger moves
	assert.Equal(t, 1, len(resp.Transfers))
}

func testTransferComplexLoss(t *testing.T) {
	trader := "test-trader"
	half := int64(500)
	price := half * 2

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	marginTrader, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.Engine.IncrementBalance(marginTrader, half)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: -price,
			},
			Type: types.TransferType_LOSS,
		},
	}
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from trader account, one from insurance acc
	assert.Equal(t, 2, len(resp.Transfers))
}

func testTransferLossMissingTraderAccounts(t *testing.T) {
	trader := "test-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: -price,
			},
			Type: types.TransferType_LOSS,
		},
	}
	resp, err := eng.FinalSettlement(testMarketID, pos)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, ErrAccountDoesNotExist, err)
}

func testDistributeWin(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price)
	defer eng.Finish()

	// set settlement account
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.Engine.IncrementBalance(eng.marketSettlementID, price*2)
	assert.Nil(t, err)

	// create trader accounts, add balance for money trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_ = eng.Engine.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	// err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	// assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}

	expMoneyBalance := price * 2 // size = 2
	eng.buf.EXPECT().Add(gomock.Any()).Times(5).Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, price, acc.Balance)
		}
		// this accounts for 2 calls
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, expMoneyBalance, acc.Balance)
			expMoneyBalance += price
		}
	})
	// eng.buf.EXPECT().Add(gomock.Any()).MinTimes(4).Do(func(acc types.Account) {
	// 	if acc.Owner == trader && acc.Type == types.AccountType_MARGIN {
	// 		assert.Equal(t, price, acc.Balance)
	// 	}
	// 	if acc.Owner == moneyTrader && acc.Type == types.AccountType_MARGIN {
	// 		// assert.Equal(t, 5*price+factor, acc.Balance)
	// 		assert.Equal(t, 7*price, acc.Balance)
	// 	}
	// })
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(resp.Transfers))
}

func testProcessBoth(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price*3)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}

	// next up, updating the balance of the traders' general accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(8).Do(func(acc types.Account) {
		// if acc.Owner == moneyTrader && acc.Type == types.AccountType_MARGIN {
		// 	assert.Equal(t, int64(3000), acc.Balance)
		// }
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, int64(2000), acc.Balance)
		}
	})
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(responses[1].Transfers))
}

func testProcessBothProRated(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_ = eng.Engine.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(8)
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	// resp := responses[0]
	// // total balance of settlement account should be 3 times price
	// for _, bal := range resp.Balances {
	// 	if bal.Account.Type == types.AccountType_SETTLEMENT {
	// 		// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
	// 		assert.Equal(t, int64(1), bal.Account.Balance)
	// 	}
	// }

	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(responses[1].Transfers))
}

func testProcessBothProRatedMTM(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(8)
	// quickly get the interface mocked for this test
	transfers := getMTMTransfer(pos)
	responses, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err, "was error")
	assert.NotEmpty(t, raw)
	// resp := raw[0]
	// // total balance of settlement account should be 3 times price
	// for _, bal := range resp.Balances {
	// 	if bal.Account.Type == types.AccountType_SETTLEMENT {
	// 		// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
	// 		assert.Equal(t, int64(1), bal.Account.Balance)
	// 	}
	// }

	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(raw[1].Transfers))
}

func testRemoveDistressedBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := int64(1000)
	eng := getTestEngine(t, testMarketID, insBalance)
	defer eng.Finish()

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.CreatePartyGeneralAccount(trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add balance to margin account for trader
	err = eng.Engine.IncrementBalance(marginID, 100)
	assert.Nil(t, err)

	// set up calls expected to buffer: add the update of the balance, of system account (insurance) and one with the margin account set to 0
	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(acc types.Account) {
		// this is the trader account, we expect the balance to be 0
		if acc.Id == marginID {
			assert.Zero(t, acc.Balance)
		} else {
			// we expect the insurance balance to get the 100 balance from the margin account added to it
			assert.Equal(t, insBalance+100, acc.Balance)
		}
	})
	// events:
	data := []events.MarketPosition{
		marketPositionFake{
			party: trader,
		},
	}
	resp, err := eng.RemoveDistressed(data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	assert.Error(t, err)
	assert.Equal(t, ErrAccountDoesNotExist, err)
}

func testRemoveDistressedNoBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := int64(1000)
	eng := getTestEngine(t, testMarketID, insBalance)
	defer eng.Finish()

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.CreatePartyGeneralAccount(trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// no balance on margin account, so we don't expect there to be any balance updates in the buffer either
	// set up calls expected to buffer: add the update of the balance, of system account (insurance) and one with the margin account set to 0
	data := []events.MarketPosition{
		marketPositionFake{
			party: trader,
		},
	}
	resp, err := eng.RemoveDistressed(data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	assert.Error(t, err)
	assert.Equal(t, ErrAccountDoesNotExist, err)
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testMTMSuccess(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	gID := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.UpdateBalance(marginMoneyTrader, 5*price)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes().Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func TestInvalidMarketID(t *testing.T) {
	trader := "test-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	invalidMarketID := testMarketID + "invalid"
	evts, raw, err := eng.MarkToMarket(invalidMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestEmptyTransfer(t *testing.T) {
	trader := "test-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  0,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoMarginAccount(t *testing.T) {
	trader := "test-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoGeneralAccount(t *testing.T) {
	trader := "test-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestMTMNoTransfers(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{}
	transfers := eng.getTestMTMTransfer(pos)

	// Empty list of transfers
	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)

	// List with a single nil value
	mt := mtmFake{
		t:     nil,
		party: "test-trader",
	}
	transfers = append(transfers, mt)
	evts, raw, err = eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Equal(t, len(evts), 1)
}

func TestFinalSettlementNoTransfers(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{}

	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNoSystemAccounts(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
	}

	responses, err := eng.FinalSettlement("invalidMarketID", pos)
	assert.Error(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNoSettlementAccount(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// Find the ID for the settlement account
	settlementID := eng.accountID(testMarketID, "", "BTC", types.AccountType_SETTLEMENT)

	assert.NotNil(t, settlementID)

	err := eng.removeAccount(settlementID)
	assert.NoError(t, err)

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
	}

	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Error(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNotEnoughMargin(t *testing.T) {
	amount := int64(1000)

	eng := getTestEngine(t, testMarketID, amount/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	_ = eng.Engine.CreatePartyGeneralAccount("testTrader", testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount("testTrader", testMarketID, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Size:  100,
			Amount: &types.FinancialAmount{
				Amount: -amount,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
	}

	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(responses))
}

func TestGetPartyMarginNoAccounts(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginNoMarginAccounts(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	_ = eng.Engine.CreatePartyGeneralAccount("test-trader", testMarketAsset)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginEmpty(t *testing.T) {
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	_ = eng.Engine.CreatePartyGeneralAccount("test-trader", testMarketAsset)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	_, err := eng.Engine.CreatePartyMarginAccount("test-trader", testMarketID, testMarketAsset)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.NotNil(t, margin)
	assert.Equal(t, margin.MarginBalance(), uint64(0))
	assert.Equal(t, margin.GeneralBalance(), uint64(0))
	assert.NoError(t, err)
}

func TestMTMLossSocialization(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	lossTrader1 := "losstrader1"
	lossTrader2 := "losstrader2"
	winTrader1 := "wintrader1"
	winTrader2 := "wintrader2"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(10)
	_ = eng.Engine.CreatePartyGeneralAccount(lossTrader1, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(lossTrader1, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(margin, 500)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(lossTrader2, testMarketAsset)
	margin, err = eng.Engine.CreatePartyMarginAccount(lossTrader2, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(margin, 1100)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(winTrader1, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(winTrader1, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(margin, 0)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(winTrader2, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(winTrader2, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(margin, 700)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: lossTrader1,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -700,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: lossTrader2,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -1400,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: winTrader1,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 1400,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: winTrader2,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 700,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes().Do(func(acc types.Account) {
		if acc.Owner == winTrader1 && acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, acc.Balance, int64(1066))
		}
		if acc.Owner == winTrader2 && acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, acc.Balance, int64(534))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func testMarginUpdateOnOrderOK(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: 100,
		transfer: &types.Transfer{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				MinAmount: 100,
				Amount:    100,
				Asset:     testMarketAsset,
			},
			Type: types.TransferType_MARGIN_LOW,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_MARGIN {
			assert.Equal(t, acc.Balance, int64(100))
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)
}

func testMarginUpdateOnOrderFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: 100000,
		transfer: &types.Transfer{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				MinAmount: 100000,
				Amount:    100000,
				Asset:     testMarketAsset,
			},
			Type: types.TransferType_MARGIN_LOW,
		},
	}

	resp, closed, err := eng.Engine.MarginUpdateOnOrder(testMarketID, evt)
	assert.NotNil(t, err)
	assert.Error(t, err, ErrMinAmountNotReached.Error())
	assert.NotNil(t, closed)
	assert.Nil(t, resp)
}

func TestMarginUpdates(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	list := make([]events.Risk, 1)

	list[0] = riskFake{
		asset:  testMarketAsset,
		amount: 100,
		transfer: &types.Transfer{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				MinAmount: 100,
				Amount:    100,
				Asset:     testMarketAsset,
			},
			Type: types.TransferType_MARGIN_LOW,
		},
	}

	resp, margin, err := eng.Engine.MarginUpdate(testMarketID, list)
	assert.Nil(t, err)
	assert.Equal(t, len(margin), 0)
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Transfers[0].Amount, int64(100))
}

func TestClearMarket(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	parties := []string{trader}

	responses, err := eng.Engine.ClearMarket(testMarketID, testMarketAsset, parties)

	assert.Nil(t, err)
	assert.Equal(t, len(responses), 1)
}

func TestClearMarketNoMargin(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)

	parties := []string{trader}

	responses, err := eng.Engine.ClearMarket(testMarketID, testMarketAsset, parties)

	assert.NoError(t, err)
	assert.Equal(t, len(responses), 0)
}

func TestWithdrawalOK(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw(trader, testMarketAsset, 100)
	assert.Nil(t, err)
}

func TestWithdrawalExact(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw(trader, testMarketAsset, 500)
	assert.Nil(t, err)
}

func TestWithdrawalNotEnough(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw(trader, testMarketAsset, 600)
	assert.Error(t, err)
}

func TestWithdrawalInvalidAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw("invalid", testMarketAsset, 600)
	assert.Error(t, err)
}

func TestChangeBalance(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	acc := eng.Engine.CreatePartyGeneralAccount(trader, testMarketAsset)

	eng.Engine.IncrementBalance(acc, 500)
	account, err := eng.Engine.GetAccountByID(acc)
	assert.NoError(t, err)
	assert.Equal(t, account.Balance, int64(500))

	eng.Engine.IncrementBalance(acc, 250)
	account, err = eng.Engine.GetAccountByID(acc)
	assert.Equal(t, account.Balance, int64(750))

	eng.Engine.UpdateBalance(acc, 666)
	account, err = eng.Engine.GetAccountByID(acc)
	assert.Equal(t, account.Balance, int64(666))

	err = eng.Engine.IncrementBalance("invalid", 200)
	assert.Error(t, err, ErrAccountDoesNotExist)

	err = eng.Engine.UpdateBalance("invalid", 300)
	assert.Error(t, err, ErrAccountDoesNotExist)
}

func TestOnChainTimeUpdate(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()

	// Hard to test this so for now I am just setting the value
	// and if it does not crash I am happy
	now := time.Now()
	eng.Engine.OnChainTimeUpdate(now)
}

func TestReloadConfig(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()

	// Check that the log level is currently `debug`
	assert.Equal(t, eng.Engine.Level.Level, logging.DebugLevel)

	// Create a new config and make some changes to it
	newConfig := NewDefaultConfig()
	newConfig.Level = encoding.LogLevel{
		Level: logging.InfoLevel,
	}
	eng.Engine.ReloadConf(newConfig)

	// Verify that the log level has been changed
	assert.Equal(t, eng.Engine.Level.Level, logging.InfoLevel)
}

func (e *testEngine) getTestMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	tt := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {

		// Apply some limited validation here so we can filter out bad transfers
		if t.GetSize() != 0 {
			mt := mtmFake{
				t:     t,
				party: t.Owner,
			}
			tt = append(tt, mt)
		}
	}
	return tt
}

func getTestEngine(t *testing.T, market string, insuranceBalance int64) *testEngine {
	ctrl := gomock.NewController(t)
	buf := mocks.NewMockAccountBuffer(ctrl)
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	conf := NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	buf.EXPECT().Add(gomock.Any()).Times(2)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	eng, err := New(logging.NewTestLogger(), conf, buf, lossBuf, time.Now())
	assert.Nil(t, err)

	// create market and traders used for tests
	insID, setID := eng.CreateMarketAccounts(testMarketID, testMarketAsset, insuranceBalance)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		buf:                buf,
		lossBuf:            lossBuf,
		marketInsuranceID:  insID,
		marketSettlementID: setID,
		// systemAccs: accounts,
	}
}

func (e *testEngine) Finish() {
	e.systemAccs = nil
	e.ctrl.Finish()
}

type marketPositionFake struct {
	party           string
	size, buy, sell int64
	price           uint64
}

func (m marketPositionFake) Party() string    { return m.party }
func (m marketPositionFake) Size() int64      { return m.size }
func (m marketPositionFake) Buy() int64       { return m.buy }
func (m marketPositionFake) Sell() int64      { return m.sell }
func (m marketPositionFake) Price() uint64    { return m.price }
func (m marketPositionFake) ClearPotentials() {}

type mtmFake struct {
	t     *types.Transfer
	party string
}

func (m mtmFake) Party() string             { return m.party }
func (m mtmFake) Size() int64               { return 0 }
func (m mtmFake) Price() uint64             { return 0 }
func (m mtmFake) Buy() int64                { return 0 }
func (m mtmFake) Sell() int64               { return 0 }
func (m mtmFake) ClearPotentials()          {}
func (m mtmFake) Transfer() *types.Transfer { return m.t }

func getMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	r := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {
		r = append(r, &mtmFake{
			t:     t,
			party: t.Owner,
		})
	}
	return r
}

type riskFake struct {
	party           string
	size, buy, sell int64
	price           uint64
	margins         *types.MarginLevels
	amount          int64
	transfer        *types.Transfer
	asset           string
}

func (m riskFake) Party() string                     { return m.party }
func (m riskFake) Size() int64                       { return m.size }
func (m riskFake) Buy() int64                        { return m.buy }
func (m riskFake) Sell() int64                       { return m.sell }
func (m riskFake) Price() uint64                     { return m.price }
func (m riskFake) ClearPotentials()                  {}
func (m riskFake) Transfer() *types.Transfer         { return m.transfer }
func (m riskFake) Amount() int64                     { return m.amount }
func (m riskFake) MarginLevels() *types.MarginLevels { return m.margins }
func (m riskFake) Asset() string                     { return m.asset }
func (m riskFake) MarketID() string                  { return "" }
func (m riskFake) MarginBalance() uint64             { return 0 }
func (m riskFake) GeneralBalance() uint64            { return 0 }
