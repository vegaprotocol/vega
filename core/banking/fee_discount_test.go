// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	asset := "vega"
	party := "party-1"

	// set 2 epochs discount window
	eng.OnFeeDiscountNumOfEpochUpdate(context.Background(), num.NewUint(2))

	// expect the whole fee to be paid
	discountedFee, discount := eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(5))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "0", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(10)})

	// expect discount of 10 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(15))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "10", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(20)})

	// expect discount of 3 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(3))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "3", discount.String())

	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(5)})
	// expect discount of 4 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(4))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "4", discount.String())

}
