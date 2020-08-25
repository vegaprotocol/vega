package fee_test

import (
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

const (
	testAsset = "ETH"
)

var (
	testFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.1",
			InfrastructureFee: "0.05",
			MakerFee:          "0.02",
		},
	}
)

type testFee struct {
	*fee.Engine
}

func getTestFee(t *testing.T) *testFee {
	eng, err := fee.New(
		logging.NewTestLogger(),
		fee.NewDefaultConfig(),
		testFees,
		testAsset,
	)
	assert.NoError(t, err)
	return &testFee{eng}
}

func TestFeeEngine(t *testing.T) {
	t.Run("update fee factors with invalid input", testUpdateFeeFactorsError)
	t.Run("update fee factors with valid input", testUpdateFeeFactors)
	t.Run("calculate continuous trading fee empty trade", testCalcContinuousTradingErrorEmptyTrade)
	t.Run("calcualte continuous trading fee", testCalcContinuousTrading)
	t.Run("calcualte continuous trading fee + check amounts", testCalcContinuousTradingAndCheckAmounts)

	t.Run("calculate continuous trading fee empty trade", testCalcContinuousTradingErrorEmptyTrade)
	t.Run("calcualte continuous trading fee", testCalcContinuousTrading)

	t.Run("calculate auction trading fee empty trade", testCalcAuctionTradingErrorEmptyTrade)
	t.Run("calcualte auction trading fee", testCalcAuctionTrading)

	t.Run("calculate batch auction trading fee empty trade", testCalcBatchAuctionTradingErrorEmptyTrade)
	t.Run("calcualte batch auction trading fee same batch", testCalcBatchAuctionTradingSameBatch)
	t.Run("calcualte batch auction trading fee different batches", testCalcBatchAuctionTradingDifferentBatches)

}

func testUpdateFeeFactors(t *testing.T) {
	eng := getTestFee(t)
	okFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.1",
			InfrastructureFee: "0.5",
			MakerFee:          "0.25",
		},
	}
	err := eng.UpdateFeeFactors(okFees)
	assert.NoError(t, err)
}

func testUpdateFeeFactorsError(t *testing.T) {
	eng := getTestFee(t)
	koFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "asdasd",
			InfrastructureFee: "0.5",
			MakerFee:          "0.25",
		},
	}
	err := eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)

	koFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.1",
			InfrastructureFee: "asdas",
			MakerFee:          "0.25",
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
	koFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.1",
			InfrastructureFee: "0.5",
			MakerFee:          "asdas",
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
}

func testCalcContinuousTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	_, err := eng.CalculateForContinuousMode([]*types.Trade{})
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcContinuousTradingAndCheckAmounts(t *testing.T) {
	eng := getTestFee(t)
	eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          "0.000250",
			InfrastructureFee: "0.0005",
			LiquidityFee:      "0.001",
		},
	})
	trades := []*types.Trade{
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     100000,
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
			assert.Equal(t, 500, int(v.Amount.Amount))
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
			assert.Equal(t, 250, int(v.Amount.Amount))
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
			assert.Equal(t, 125, int(v.Amount.Amount))
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
			assert.Equal(t, 125, int(v.Amount.Amount))
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))

}
func testCalcContinuousTrading(t *testing.T) {
	eng := getTestFee(t)
	trades := []*types.Trade{
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      10,
			Price:     10000,
		},
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party3",
			Size:      1,
			Price:     10300,
		},
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party4",
			Size:      7,
			Price:     10300,
		},
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      2,
			Price:     10500,
		},
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party5",
			Size:      5,
			Price:     11000,
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, 43928, int(party1Amount))
	_ = party1Amount

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
}

func testCalcAuctionTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	_, err := eng.CalculateForAuctionMode([]*types.Trade{})
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcAuctionTrading(t *testing.T) {
	eng := getTestFee(t)
	trades := []*types.Trade{
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      1,
			Price:     100,
		},
	}

	ft, err := eng.CalculateForAuctionMode(trades)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05) = 15
	// 15 / 2 = 7.5
	// internally the engine Ceil all fees.
	// so here we will expect 8 for each
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, 8, int(party1Amount))
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, 8, int(party2Amount))

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, 2)
	assert.Equal(t, infra, 2)
	assert.Equal(t, recv, 0)
	assert.Equal(t, pay, 0)
}

func testCalcBatchAuctionTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	_, err := eng.CalculateForFrequentBatchesAuctionMode([]*types.Trade{})
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcBatchAuctionTradingSameBatch(t *testing.T) {
	eng := getTestFee(t)
	trades := []*types.Trade{
		{
			Aggressor:          types.Side_SIDE_SELL,
			Seller:             "party1",
			Buyer:              "party2",
			Size:               1,
			Price:              100,
			SellerAuctionBatch: 10,
			BuyerAuctionBatch:  10,
		},
	}

	ft, err := eng.CalculateForFrequentBatchesAuctionMode(trades)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05) = 15
	// 15 / 2 = 7.5
	// internally the engine Ceil all fees.
	// so here we will expect 8 for each
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, 8, int(party1Amount))
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, 8, int(party2Amount))

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, 2)
	assert.Equal(t, infra, 2)
	assert.Equal(t, recv, 0)
	assert.Equal(t, pay, 0)
}

func testCalcBatchAuctionTradingDifferentBatches(t *testing.T) {
	eng := getTestFee(t)
	trades := []*types.Trade{
		{
			Aggressor:          types.Side_SIDE_SELL,
			Seller:             "party1",
			Buyer:              "party2",
			Size:               1,
			Price:              100,
			SellerAuctionBatch: 11,
			BuyerAuctionBatch:  10,
		},
	}

	ft, err := eng.CalculateForFrequentBatchesAuctionMode(trades)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05 + 100 *0.02) = 17
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, 17, int(party1Amount))
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, 0, int(party2Amount))

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, 1)
	assert.Equal(t, pay, 1)
}

func testCalcPositionResolution(t *testing.T) {
	eng := getTestFee(t)
	trades := []*types.Trade{
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party1",
			Buyer:     "network",
			Size:      3,
			Price:     1000,
		},
		{
			Aggressor: types.Side_SIDE_SELL,
			Seller:    "party2",
			Buyer:     "network",
			Size:      2,
			Price:     1100,
		},
	}

	positions := []events.MarketPosition{
		fakeMktPos{party: "bad-party1", size: -10},
		fakeMktPos{party: "bad-party2", size: 7},
		fakeMktPos{party: "bad-party3", size: -2},
		fakeMktPos{party: "bad-party4", size: 10},
	}

	ft, err := eng.CalculateFeeForPositionResolution(trades, positions)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	party1Amount, ok := feeAmounts["bad-party1"]
	assert.True(t, ok)
	assert.Equal(t, 307, int(party1Amount))
	party2Amount, ok := feeAmounts["bad-party2"]
	assert.True(t, ok)
	assert.Equal(t, 217, int(party2Amount))
	party3Amount, ok := feeAmounts["bad-party3"]
	assert.True(t, ok)
	assert.Equal(t, 65, int(party3Amount))
	party4Amount, ok := feeAmounts["bad-party4"]
	assert.True(t, ok)
	assert.Equal(t, 307, int(party4Amount))

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == types.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, len(trades)*len(positions))
	assert.Equal(t, infra, len(trades)*len(positions))
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades)*len(positions))
}

type fakeMktPos struct {
	party         string
	size          int64
	vwBuy, vwSell uint64
}

func (f fakeMktPos) Party() string { return f.party }
func (f fakeMktPos) Size() int64   { return f.size }
func (f fakeMktPos) Buy() int64    { return 0 }
func (f fakeMktPos) Sell() int64   { return 0 }
func (f fakeMktPos) Price() uint64 { return 0 }

func (f fakeMktPos) VWBuy() uint64 {
	return f.vwBuy
}

func (f fakeMktPos) VWSell() uint64 {
	return f.vwSell
}
