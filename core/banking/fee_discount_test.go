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

func TestBankingApplyFeeDiscount(t *testing.T) {
	eng := getTestEngine(t)

	ctx := context.Background()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	asset := "vega"
	party := "party-1"

	eng.OnTransferFeeDiscountDecayFractionUpdate(context.Background(), num.DecimalFromFloat(0.5))

	assert.Equal(t, "0", eng.AvailableFeeDiscount(asset, party).String())

	// expect the whole fee to be paid
	discountedFee, discount := eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(5))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "0", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(10)})

	assert.Equal(t, "10", eng.AvailableFeeDiscount(asset, party).String())

	// expect discount of 10 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(15))
	assert.Equal(t, "5", discountedFee.String())
	assert.Equal(t, "10", discount.String())
	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(20)})

	assert.Equal(t, "20", eng.AvailableFeeDiscount(asset, party).String())

	// expect discount of 3 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(3))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "3", discount.String())

	assert.Equal(t, "17", eng.AvailableFeeDiscount(asset, party).String())

	// move to another epoch
	eng.RegisterTakerFees(ctx, asset, map[string]*num.Uint{party: num.NewUint(5)})

	// it's 13 because 9 was decayed and extra 5 added = 17-8+5
	assert.Equal(t, "13", eng.AvailableFeeDiscount(asset, party).String())

	// expect discount of 4 to be applied
	discountedFee, discount = eng.ApplyFeeDiscount(ctx, asset, party, num.NewUint(4))
	assert.Equal(t, "0", discountedFee.String())
	assert.Equal(t, "4", discount.String())

}
