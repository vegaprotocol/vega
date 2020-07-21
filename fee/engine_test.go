package fee_test

import (
	"testing"

	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

const (
	testAsset = "ETH"
)

var (
	testFees = types.Fees{
		Factors: &proto.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
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
}

func testUpdateFeeFactors(t *testing.T) {
	eng := getTestFee(t)
	okFees := types.Fees{
		Factors: &proto.FeeFactors{
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
		Factors: &proto.FeeFactors{
			LiquidityFee:      "asdasd",
			InfrastructureFee: "0.5",
			MakerFee:          "0.25",
		},
	}
	err := eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)

	koFees = types.Fees{
		Factors: &proto.FeeFactors{
			LiquidityFee:      "0.1",
			InfrastructureFee: "asdas",
			MakerFee:          "0.25",
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
	koFees = types.Fees{
		Factors: &proto.FeeFactors{
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
	assert.Equal(t, 449, int(party1Amount))
	_ = party1Amount

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var (
		pay, recv, infra, liquidity int
	)
	for _, v := range transfers {
		if v.Type == proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY {
			liquidity += 1
		}
		if v.Type == proto.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY {
			infra += 1
		}
		if v.Type == proto.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE {
			recv += 1
		}
		if v.Type == proto.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY {
			pay += 1
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
}
