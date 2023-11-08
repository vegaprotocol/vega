package banking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// | Epoch                    | 1  | 2  |  3 |  4 |
// | ------------------------ |---------|----|----|
// | taker fee paid           | 10 | 20 | 5  | 8  |
// | accumulated discount     | 0  | 10 | 20 | 22 |
// | transfer fee theoretical | 5  | 15 | 3  | 4  |
// | transfer fee paid        | 5  | 5  | 0  | 0  |
func TestBankingApplyFeeDiscount(t *testing.T) {
	eng := getTestEngine(t)

	ctx := context.Background()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	asset := "vega"
	party := "party-1"

	// set 2 epochs discount window
	eng.OnFeeDiscountNumOfEpochUpdate(context.Background(), 2)

	// expect the whole fee to be paid
	discountedFee, discount := eng.ApplyFeeDiscount(asset, party, num.NewUint(5))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "0", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(10)})

	// expect discount of 10 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(asset, party, num.NewUint(15))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "10", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(20)})

	// expect discount of 3 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(asset, party, num.NewUint(3))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "3", discount.String())

	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(5)})
	// expect discount of 4 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(asset, party, num.NewUint(4))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "4", discount.String())

}
