package collateral_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/collateral/mocks"
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
}

func TestRemoveDistressed(t *testing.T) {
	t.Run("Successfully remove distressed trader and transfer balance", testRemoveDistressedBalance)
	t.Run("Successfully remove distressed trader, no balance transfer", testRemoveDistressedNoBalance)
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", 0)
	eng.Finish()
}

func testAddTrader(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.Engine.UpdateBalance(general, 100000)
	assert.Nil(t, err)

	// add to the market
	err = eng.AddTraderToMarket(testMarketID, trader, testMarketAsset)
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
	_, _ = eng.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	marginMoneyTrader, _ := eng.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.UpdateBalance(marginMoneyTrader, 100000)
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
	marginTrader, _ := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginTrader, half)
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
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
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
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
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
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	_, _ = eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
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

	expMoneyBalance := price
	eng.buf.EXPECT().Add(gomock.Any()).Times(4).Do(func(acc types.Account) {
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
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
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
	eng.buf.EXPECT().Add(gomock.Any()).Times(7).Do(func(acc types.Account) {
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_MARGIN {
			// assert.Equal(t, int64(3000), acc.Balance)
		}
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
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
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

	eng.buf.EXPECT().Add(gomock.Any()).Times(6)
	responses, err := eng.FinalSettlement(testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
			// assert.Equal(t, int64(1), bal.Account.Balance)
		}
	}
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
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginMoneyTrader, price*5)
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
	responses, raw, err := eng.MarkToMarket(testMarketID, transfers)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err, "was error")
	assert.NotEmpty(t, raw)
	resp := raw[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
			// assert.Equal(t, int64(1), bal.Account.Balance)
		}
	}
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
	marginID, _ := eng.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	// add balance to margin account for trader
	err := eng.Engine.IncrementBalance(marginID, 100)
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
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
}

func testRemoveDistressedNoBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := int64(1000)
	eng := getTestEngine(t, testMarketID, insBalance)
	defer eng.Finish()

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	marginID, _ := eng.CreateTraderAccount(trader, testMarketID, testMarketAsset)

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
	assert.Equal(t, collateral.ErrAccountDoesNotExist, err)
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
	mID, gID := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.UpdateBalance(marginMoneyTrader, 5*price)
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
	evts, raw, err := eng.MarkToMarket(testMarketID, transfers)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func (e *testEngine) getTestMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	tt := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {

		mt := mtmFake{
			t: t,
		}
		tt = append(tt, mt)
	}
	return tt
}

func getTestEngine(t *testing.T, market string, insuranceBalance int64) *testEngine {
	ctrl := gomock.NewController(t)
	buf := mocks.NewMockAccountBuffer(ctrl)
	conf := collateral.NewDefaultConfig()
	buf.EXPECT().Add(gomock.Any()).Times(2)

	eng, err := collateral.New(logging.NewTestLogger(), conf, buf, time.Now())
	assert.Nil(t, err)

	// create market and traders used for tests
	insID, setID := eng.CreateMarketAccounts(testMarketID, testMarketAsset, insuranceBalance)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		buf:                buf,
		marketInsuranceID:  insID,
		marketSettlementID: setID,
		// systemAccs: accounts,
	}
}

func (te *testEngine) Finish() {
	te.systemAccs = nil
	te.ctrl.Finish()
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
	t *types.Transfer
}

func (m mtmFake) Party() string             { return "" }
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
			t: t,
		})
	}
	return r
}
