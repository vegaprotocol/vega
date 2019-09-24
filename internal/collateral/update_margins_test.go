package collateral_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/events"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarginUpdates(t *testing.T) {
	t.Run("Trader is in search zone - topup to stay above", testSearchZoneTopUpToBeAbove)
	t.Run("Trader is in search zone - topup to search zone", testSearchZoneTopUpToSearchZone)
	t.Run("Trader is in close out zone - topup to be in search zone", testCloseOutZoneTopUpToSearchZone)
	t.Run("Trader is in close out zone - topup is insufficient", testCloseOutZoneTopUpInsufficient)
	t.Run("Trader is in release level - release surplus", testMoveToReleaseLevelMarginIsReleased)
}

// The mark price changes causing the trader’s margin to move into the search zone.
// A collateral search is initiated and the margin is topped back up above the search zone.
func testSearchZoneTopUpToBeAbove(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	traderID := "loltrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(traderID, testMarketID, testMarketAsset)
	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the GeneralAccount
	err := eng.Engine.UpdateBalance(general, 100)
	assert.Nil(t, err)
	// add to the market
	err = eng.Engine.AddTraderToMarket(testMarketID, traderID, testMarketAsset)
	assert.Nil(t, err)

	// have 100 in general, need to topup 50
	pos := []events.Risk{
		&fakeMarginChange{
			transfer: &types.Transfer{
				Owner: traderID,
				Size:  1,
				Amount: &types.FinancialAmount{
					Amount:    50,
					MinAmount: 0,
					Asset:     testMarketAsset,
				},
				Type: types.TransferType_MARGIN_LOW,
			},
			amount:        50,
			marginBalance: 0,
			Margin:        nil,
		},
	}

	// ensure that the maximal amount was moved as it was possible
	eng.buf.EXPECT().Add(gomock.Any()).Times(3).DoAndReturn(func(acc types.Account) {
		if acc.Type == types.AccountType_GENERAL && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(50), acc.Balance)
		}
		if acc.Type == types.AccountType_MARGIN && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(50), acc.Balance)
		}

	})

	res, closed, err := eng.MarginUpdate(testMarketID, pos)
	// get transfer requests
	assert.Equal(t, 1, len(res))
	// trader not closed
	assert.Equal(t, 0, len(closed))

	_ = margin
}

// The mark price changes causing the trader’s margin to move into the search zone.
// A collateral search is initiated and the margin is topped back up to a level which
// results in the trader still being in the search zone. No further actions are taken.
func testSearchZoneTopUpToSearchZone(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	traderID := "loltrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(traderID, testMarketID, testMarketAsset)
	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the GeneralAccount
	err := eng.Engine.UpdateBalance(general, 100)
	assert.Nil(t, err)
	// add to the market
	err = eng.Engine.AddTraderToMarket(testMarketID, traderID, testMarketAsset)
	assert.Nil(t, err)

	// have 100 in general, need to topup 110
	pos := []events.Risk{
		&fakeMarginChange{
			transfer: &types.Transfer{
				Owner: traderID,
				Size:  1,
				Amount: &types.FinancialAmount{
					Amount:    110,
					MinAmount: 0,
					Asset:     testMarketAsset,
				},
				Type: types.TransferType_MARGIN_LOW,
			},
			amount:        110,
			marginBalance: 0,
			Margin:        nil,
		},
	}

	// the amount needed was not available
	// however there was no minimal specified which means the trader was not in the closeout zone
	// so everything should be still fine at this stage
	eng.buf.EXPECT().Add(gomock.Any()).Times(3).DoAndReturn(func(acc types.Account) {
		if acc.Type == types.AccountType_GENERAL && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(0), acc.Balance)
		}
		if acc.Type == types.AccountType_MARGIN && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(100), acc.Balance)
		}

	})

	res, closed, err := eng.MarginUpdate(testMarketID, pos)
	// get transfer requests
	assert.Equal(t, 1, len(res))
	// trader not closed
	assert.Equal(t, 0, len(closed))

	_ = margin
}

// The mark price changes causing the trader’s margin to move into the close-out zone.
// A collateral search is initiated and the margin is topped back up to the search zone.
// No further actions are taken.
func testCloseOutZoneTopUpToSearchZone(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	traderID := "loltrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(traderID, testMarketID, testMarketAsset)
	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the GeneralAccount
	err := eng.Engine.UpdateBalance(general, 100)
	assert.Nil(t, err)
	// add to the market
	err = eng.Engine.AddTraderToMarket(testMarketID, traderID, testMarketAsset)
	assert.Nil(t, err)

	// have 100 in general, need to topup minimal 90 and max 150
	pos := []events.Risk{
		&fakeMarginChange{
			transfer: &types.Transfer{
				Owner: traderID,
				Size:  1,
				Amount: &types.FinancialAmount{
					Amount:    150,
					MinAmount: 90,
					Asset:     testMarketAsset,
				},
				Type: types.TransferType_MARGIN_LOW,
			},
			amount:        150,
			marginBalance: 0,
			Margin:        nil,
		},
	}

	// the amount needed was not available and a minimal amount was specified
	// so we should move everything required up to the point we fullfil the minimal amount or more
	// in this case everything in the general account will be used
	eng.buf.EXPECT().Add(gomock.Any()).Times(3).DoAndReturn(func(acc types.Account) {
		if acc.Type == types.AccountType_GENERAL && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(0), acc.Balance)
		}
		if acc.Type == types.AccountType_MARGIN && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(100), acc.Balance)
		}

	})

	res, closed, err := eng.MarginUpdate(testMarketID, pos)
	// get transfer requests
	assert.Equal(t, 1, len(res))
	// trader not closed
	assert.Equal(t, 0, len(closed))

	_ = margin
}

// The mark price changes causing the trader’s margin to move into the close-out zone.
// A collateral search is initiated and the margin is topped back up to a level which
// results in the trader still being in the close-out zone. This trader becomes a werewolf.
func testCloseOutZoneTopUpInsufficient(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	traderID := "loltrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(traderID, testMarketID, testMarketAsset)
	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the GeneralAccount
	err := eng.Engine.UpdateBalance(general, 100)
	assert.Nil(t, err)
	// add to the market
	err = eng.Engine.AddTraderToMarket(testMarketID, traderID, testMarketAsset)
	assert.Nil(t, err)

	// have 100 in general, need to topup minimal 110 and max 150
	pos := []events.Risk{
		&fakeMarginChange{
			transfer: &types.Transfer{
				Owner: traderID,
				Size:  1,
				Amount: &types.FinancialAmount{
					Amount:    150,
					MinAmount: 110,
					Asset:     testMarketAsset,
				},
				Type: types.TransferType_MARGIN_LOW,
			},
			amount:        150,
			marginBalance: 0,
			Margin:        nil,
		},
	}

	// the amount needed was not available and a minimal amount was specified
	// however there is not enough money in the general account even to fullfill the minimum
	// amount.
	// the money should be move, altho the trader will be returned in the list of traders
	// distressed
	eng.buf.EXPECT().Add(gomock.Any()).Times(3).DoAndReturn(func(acc types.Account) {
		if acc.Type == types.AccountType_GENERAL && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(0), acc.Balance)
		}
		if acc.Type == types.AccountType_MARGIN && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(100), acc.Balance)
		}

	})

	res, closed, err := eng.MarginUpdate(testMarketID, pos)
	// get transfer requests
	assert.Equal(t, 0, len(res))
	// trader not closed
	assert.Equal(t, 1, len(closed))

	_ = margin
}

// The mark price changes causing the trader’s margin to move in to the release level.
// Margin should be released back to the trader.
func testMoveToReleaseLevelMarginIsReleased(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	traderID := "loltrader"

	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(traderID, testMarketID, testMarketAsset)
	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the GeneralAccount
	err := eng.Engine.UpdateBalance(general, 100)
	assert.Nil(t, err)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	// add X to the MarginAccount
	err = eng.Engine.UpdateBalance(margin, 70)
	assert.Nil(t, err)
	// add to the market
	err = eng.Engine.AddTraderToMarket(testMarketID, traderID, testMarketAsset)
	assert.Nil(t, err)

	// have 100 in general and 70 in margin, we release 30
	pos := []events.Risk{
		&fakeMarginChange{
			transfer: &types.Transfer{
				Owner: traderID,
				Size:  1,
				Amount: &types.FinancialAmount{
					Amount: 30,
					Asset:  testMarketAsset,
				},
				Type: types.TransferType_MARGIN_HIGH,
			},
			amount:        30,
			marginBalance: 70,
			Margin:        nil,
		},
	}

	// we release some margin, we endup top up the general account
	// and remove some from the margin account
	eng.buf.EXPECT().Add(gomock.Any()).Times(3).DoAndReturn(func(acc types.Account) {
		if acc.Type == types.AccountType_GENERAL && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(130), acc.Balance)
		}
		if acc.Type == types.AccountType_MARGIN && traderID == acc.Owner {
			// less monies in general account
			assert.Equal(t, int64(40), acc.Balance)
		}

	})

	res, closed, err := eng.MarginUpdate(testMarketID, pos)
	// get transfer requests
	assert.Equal(t, 1, len(res))
	// trader not closed
	assert.Equal(t, 0, len(closed))

	_ = margin

}

type fakeMarginChange struct {
	events.Margin
	amount        int64
	transfer      *types.Transfer
	marginBalance uint64
}

func (m fakeMarginChange) Amount() int64 {
	return m.amount
}

// Transfer - it's actually part of the embedded interface already, but we have to mask it, because this type contains another transfer
func (m fakeMarginChange) Transfer() *types.Transfer {
	return m.transfer
}

func (m fakeMarginChange) MarginBalance() uint64 {
	return m.marginBalance
}

func (m fakeMarginChange) MarginLevels() *types.MarginLevels {
	return nil
}
