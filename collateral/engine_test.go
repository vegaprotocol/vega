package collateral_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
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
	*collateral.Engine
	ctrl               *gomock.Controller
	buf                *mocks.MockAccountBuffer
	broker             *mocks.MockBroker
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
	t.Run("Mark to Market wins and losses do not match up, settlement not drained", testSettleBalanceNotZero)
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

func TestTokenAccounts(t *testing.T) {
	t.Run("Total tokens is zero at the start, even if we add some traders", testInitialTokens)
}

func testInitialTokens(t *testing.T) {
	eng := getTestEngine(t, "test-market", 0)
	defer eng.Finish()
	trader := "trader"
	assert.Zero(t, eng.GetTotalTokens())
	// trader doesn't exist yet:
	acc, err := eng.GetPartyTokenAccount(trader)
	assert.Error(t, err)
	assert.Nil(t, acc)
	assert.Equal(t, err, collateral.ErrPartyHasNoTokenAccount)
	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_ = eng.CreatePartyGeneralAccount(context.Background(), trader, "ETH")
	acc, err = eng.GetPartyTokenAccount(trader)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.NoError(t, eng.IncrementBalance(acc.Id, 10000))
	acc, err = eng.GetPartyTokenAccount(trader)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, uint64(acc.Balance), eng.GetTotalTokens())
	assert.NoError(t, eng.UpdateBalance(acc.Id, acc.Balance/2)) // half the amount
	acc.Balance /= 2
	assert.Equal(t, uint64(acc.Balance), eng.GetTotalTokens())
	// test subtracting something from the balance
	assert.NoError(t, eng.DecrementBalance(acc.Id, 100))
	acc.Balance -= 100
	assert.Equal(t, uint64(acc.Balance), eng.GetTotalTokens())
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", 0)
	eng.Finish()
}

func testAddMarginAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// test balance is 0 when created
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), acc.Balance)
}

func testAddMarginAccountFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Error(t, err, collateral.ErrNoGeneralAccountWhenCreateMarginAccount)

}

func testAddTrader(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(general, 100000)
	assert.Nil(t, err)

	expectedGeneralBalance := uint64(100000)

	// check the amount on each account now
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), acc.Balance)

	acc, err = eng.Engine.GetAccountByID(general)
	assert.Nil(t, err)
	assert.Equal(t, expectedGeneralBalance, acc.Balance)

}

func testTransferLoss(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts, set balance for money trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(7)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	_ = eng.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.UpdateBalance(marginMoneyTrader, 100000)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, 2*price, resp.Balances[0].Balance+responses[1].Balances[0].Balance)
	// there should be 2 ledger moves
	assert.Equal(t, 1, len(resp.Transfers))
}

func testTransferComplexLoss(t *testing.T) {
	trader := "test-trader"
	half := uint64(500)
	price := half * 2

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.Engine.IncrementBalance(marginTrader, half)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: int64(-price),
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
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
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: -price,
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}
	resp, err := eng.FinalSettlement(testMarketID, pos)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
}

func testDistributeWin(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price)
	defer eng.Finish()

	// set settlement account
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.Engine.IncrementBalance(eng.marketSettlementID, price*2)
	assert.Nil(t, err)

	// create trader accounts, add balance for money trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(6)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	// err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	// assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	expMoneyBalance := price
	eng.buf.EXPECT().Add(gomock.Any()).Times(4).Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, price, acc.Balance)
		}
		// this accounts for 2 calls
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, expMoneyBalance, acc.Balance)
			expMoneyBalance += price
		}
	})
	// eng.buf.EXPECT().Add(gomock.Any()).MinTimes(4).Do(func(acc types.Account) {
	// 	if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
	// 		assert.Equal(t, price, acc.Balance)
	// 	}
	// 	if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
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
		if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(resp.Transfers))
}

func testProcessBoth(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price*3)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	// next up, updating the balance of the traders' general accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(8).Do(func(acc types.Account) {
		// if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
		// 	assert.Equal(t, int64(3000), acc.Balance)
		// }
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, int64(2000), acc.Balance)
		}
	})
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(responses[1].Transfers))
}

func testSettleBalanceNotZero(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	gID := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.UpdateBalance(marginMoneyTrader, 6*price)
	assert.Nil(t, err)
	pos := []*types.Transfer{
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price) * 2, // lost 2xprice, trader only won half
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	transfers := eng.getTestMTMTransfer(pos)
	_, _, err = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	// this should return an error
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrSettlementBalanceNotZero, err)
}

func testProcessBothProRated(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(7)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(8)
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	// resp := responses[0]
	// // total balance of settlement account should be 3 times price
	// for _, bal := range resp.Balances {
	// 	if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
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
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(8)
	// quickly get the interface mocked for this test
	transfers := getMTMTransfer(pos)
	responses, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err, "was error")
	assert.NotEmpty(t, raw)
	// resp := raw[0]
	// // total balance of settlement account should be 3 times price
	// for _, bal := range resp.Balances {
	// 	if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
	// 		// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
	// 		assert.Equal(t, int64(1), bal.Account.Balance)
	// 	}
	// }

	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(raw[1].Transfers))
}

func testRemoveDistressedBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := uint64(1000)
	eng := getTestEngine(t, testMarketID, insBalance)
	defer eng.Finish()

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
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
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
}

func testRemoveDistressedNoBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := uint64(1000)
	eng := getTestEngine(t, testMarketID, insBalance)
	defer eng.Finish()

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// no balance on margin account, so we don't expect there to be any balance updates in the buffer either
	// set up calls expected to buffer: add the update of the balance, of system account (insurance) and one with the margin account set to 0
	data := []events.MarketPosition{
		marketPositionFake{
			party: trader,
		},
	}
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testMTMSuccess(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	gID := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.UpdateBalance(marginMoneyTrader, 5*price)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: int64(price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes().Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func TestInvalidMarketID(t *testing.T) {
	trader := "test-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	invalidMarketID := testMarketID + "invalid"
	evts, raw, err := eng.MarkToMarket(context.Background(), invalidMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestEmptyTransfer(t *testing.T) {
	trader := "test-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(0),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoMarginAccount(t *testing.T) {
	trader := "test-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoGeneralAccount(t *testing.T) {
	trader := "test-trader"
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestMTMNoTransfers(t *testing.T) {
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{}
	transfers := eng.getTestMTMTransfer(pos)

	// Empty list of transfers
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)

	// List with a single nil value
	mt := mtmFake{
		t:     nil,
		party: "test-trader",
	}
	transfers = append(transfers, mt)
	evts, raw, err = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Equal(t, len(evts), 1)
}

func TestFinalSettlementNoTransfers(t *testing.T) {
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{}

	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNoSystemAccounts(t *testing.T) {
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Amount: &types.FinancialAmount{
				Amount: int64(-price),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	responses, err := eng.FinalSettlement("invalidMarketID", pos)
	assert.Error(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNotEnoughMargin(t *testing.T) {
	amount := uint64(1000)

	eng := getTestEngine(t, testMarketID, amount/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "testTrader", testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), "testTrader", testMarketID, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Amount: &types.FinancialAmount{
				Amount: int64(-amount * 100),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(responses))
}

func TestGetPartyMarginNoAccounts(t *testing.T) {
	price := uint64(1000)

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
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "test-trader", testMarketAsset)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginEmpty(t *testing.T) {
	price := uint64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "test-trader", testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), "test-trader", testMarketID, testMarketAsset)

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
	eng.buf.EXPECT().Add(gomock.Any()).Times(14)
	eng.broker.EXPECT().Send(gomock.Any()).Times(12)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), lossTrader1, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), lossTrader1, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(margin, 500)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), lossTrader2, testMarketAsset)
	margin, err = eng.Engine.CreatePartyMarginAccount(context.Background(), lossTrader2, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(margin, 1100)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), winTrader1, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), winTrader1, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(margin, 0)
	assert.Nil(t, err)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), winTrader2, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), winTrader2, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(margin, 700)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: lossTrader1,
			Amount: &types.FinancialAmount{
				Amount: -700,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: lossTrader2,
			Amount: &types.FinancialAmount{
				Amount: -1400,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: winTrader1,
			Amount: &types.FinancialAmount{
				Amount: 1400,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: winTrader2,
			Amount: &types.FinancialAmount{
				Amount: 700,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes().Do(func(acc types.Account) {
		if acc.Owner == winTrader1 && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(1066))
		}
		if acc.Owner == winTrader2 && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(534))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func testMarginUpdateOnOrderOK(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: 100,
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: 100,
				Asset:  testMarketAsset,
			},
			MinAmount: 100,
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(100))
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)
}

func testMarginUpdateOnOrderFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: 100000,
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: 100000,
				Asset:  testMarketAsset,
			},
			MinAmount: 100000,
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.NotNil(t, err)
	assert.Error(t, err, collateral.ErrMinAmountNotReached.Error())
	assert.NotNil(t, closed)
	assert.Nil(t, resp)
}

func TestMarginUpdates(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(6)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	list := make([]events.Risk, 1)

	list[0] = riskFake{
		asset:  testMarketAsset,
		amount: 100,
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: 100,
				Asset:  testMarketAsset,
			},
			MinAmount: 100,
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	resp, margin, err := eng.Engine.MarginUpdate(testMarketID, list)
	assert.Nil(t, err)
	assert.Equal(t, len(margin), 0)
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Transfers[0].Amount, uint64(100))
}

func TestClearMarket(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(6)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
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
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
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
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw(trader, testMarketAsset, 100)
	assert.Nil(t, err)
}

func TestWithdrawalExact(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw(trader, testMarketAsset, 500)
	assert.Nil(t, err)
}

func TestWithdrawalNotEnough(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.buf.EXPECT().Add(gomock.Any()).Times(5)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
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
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(acc, 500)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.Engine.Withdraw("invalid", testMarketAsset, 600)
	assert.Error(t, err)
}

func TestChangeBalance(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "oktrader"

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	acc := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)

	eng.Engine.IncrementBalance(acc, 500)
	account, err := eng.Engine.GetAccountByID(acc)
	assert.NoError(t, err)
	assert.Equal(t, account.Balance, uint64(500))

	eng.Engine.IncrementBalance(acc, 250)
	account, err = eng.Engine.GetAccountByID(acc)
	assert.Equal(t, account.Balance, uint64(750))

	eng.Engine.UpdateBalance(acc, 666)
	account, err = eng.Engine.GetAccountByID(acc)
	assert.Equal(t, account.Balance, uint64(666))

	err = eng.Engine.IncrementBalance("invalid", 200)
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)

	err = eng.Engine.UpdateBalance("invalid", 300)
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)
}

func TestReloadConfig(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()

	// Check that the log level is currently `debug`
	assert.Equal(t, eng.Engine.Level.Level, logging.DebugLevel)

	// Create a new config and make some changes to it
	newConfig := collateral.NewDefaultConfig()
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
		if t.Amount.Amount != 0 {
			mt := mtmFake{
				t:     t,
				party: t.Owner,
			}
			tt = append(tt, mt)
		}
	}
	return tt
}

func getTestEngine(t *testing.T, market string, insuranceBalance uint64) *testEngine {
	ctrl := gomock.NewController(t)
	buf := mocks.NewMockAccountBuffer(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	// 2 new events expected
	broker.EXPECT().Send(gomock.Any()).Times(2)
	// system accounts created
	buf.EXPECT().Add(gomock.Any()).Times(2)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	eng, err := collateral.New(logging.NewTestLogger(), conf, broker, buf, lossBuf, time.Now())
	assert.Nil(t, err)

	// create market and traders used for tests
	insID, setID := eng.CreateMarketAccounts(context.Background(), testMarketID, testMarketAsset, insuranceBalance)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		buf:                buf,
		broker:             broker,
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
