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

package products_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/datasource"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScalingOfSettlementData(t *testing.T) {
	t.Run("No scaling needed for settlement data for asset decimals", testNoScalingNeeded)
	t.Run("Need to scale up the settlement data for asset decimals", testScalingUpNeeded)
	t.Run("Need to scale down the settlement data for asset decimals no loss of precision", testScalingDownNeeded)
	t.Run("Need to scale down the settlement data for asset decimals with loss of precision", testScalingDownNeededWithPrecisionLoss)
	t.Run("a future product can be updated", testUpdateFuture)
}

func testNoScalingNeeded(t *testing.T) {
	// Create test future with settlement data type integer with decimals (that represents a decimal)
	ft := testFuture(t, datapb.PropertyKey_TYPE_INTEGER)

	n := &num.Numeric{}
	n.SetUint(num.NewUint(100000))
	// settlement data is in 5 decimal places, asset in 5 decimal places => no scaling
	scaled, err := ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 5,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100000), scaled)

	// Create test future with settlement data type decimal with decimals (that represents a decimal)
	ft = testFuture(t, datapb.PropertyKey_TYPE_DECIMAL)

	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	dec := num.DecimalFromFloat(10000.01101)
	n.SetDecimal(&dec)
	scaled, err = ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 5,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1000001101), scaled)
}

func testScalingUpNeeded(t *testing.T) {
	// Create test future with settlement data type integer with decimals (that represents a decimal)
	ft := testFuture(t, datapb.PropertyKey_TYPE_INTEGER)

	n := &num.Numeric{}
	n.SetUint(num.NewUint(100000))
	// settlement data is in 5 decimal places, asset in 10 decimal places => x10^5
	scaled, err := ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 10,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(10000000000), scaled)

	// Create test future with settlement data type decimal with decimals (that represents a decimal)
	ft = testFuture(t, datapb.PropertyKey_TYPE_DECIMAL)

	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	dec := num.DecimalFromFloat(10000.00001)
	n.SetDecimal(&dec)
	scaled, err = ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 10,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100000000100000), scaled)
}

func testScalingDownNeeded(t *testing.T) {
	// Create test future with settlement data type integer with decimals (that represents a decimal)
	ft := testFuture(t, datapb.PropertyKey_TYPE_INTEGER)

	n := &num.Numeric{}
	n.SetUint(num.NewUint(100000))
	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 3,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1000), scaled)

	// Create test future with settlement data type decimal with decimals (that represents a decimal)
	ft = testFuture(t, datapb.PropertyKey_TYPE_DECIMAL)

	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	dec := num.DecimalFromFloat(10000.00001)
	n.SetDecimal(&dec)
	_, err = ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 3,
	)
	require.ErrorIs(t, products.ErrSettlementDataDecimalsNotSupportedByAsset, err)
}

func testScalingDownNeededWithPrecisionLoss(t *testing.T) {
	// Create test future with settlement data type integer with decimals (that represents a decimal)
	ft := testFuture(t, datapb.PropertyKey_TYPE_INTEGER)

	n := &num.Numeric{}
	n.SetUint(num.NewUint(123456))
	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 3,
	)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1234), scaled)

	// settlement data is in 5 decimal places, asset in 3 decimal places => x10^-2
	dec := num.DecimalFromFloat(12345.678912)
	n.SetDecimal(&dec)
	_, err = ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 3,
	)
	require.ErrorIs(t, products.ErrSettlementDataDecimalsNotSupportedByAsset, err)

	dec = num.DecimalFromFloat(12345.000)
	n.SetDecimal(&dec)
	scaled, err = ft.future.ScaleSettlementDataToDecimalPlaces(
		n, 4,
	)

	require.NoError(t, err)
	require.Equal(t, num.NewUint(123450000), scaled)
}

func testUpdateFuture(t *testing.T) {
	// Create test future with settlement data type integer with decimals (that represents a decimal)
	ft := testFuture(t, datapb.PropertyKey_TYPE_INTEGER)

	fp := getTestFutureProd(t, datapb.PropertyKey_TYPE_INTEGER, 10)

	// two new subscription
	ft.oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(3), func(ctx context.Context, sid spec.SubscriptionID) {}, nil)
	ft.oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(4), func(ctx context.Context, sid spec.SubscriptionID) {}, nil)

	ft.future.Update(context.Background(), &types.InstrumentFuture{Future: fp}, ft.oe)

	assert.Equal(t, 2, ft.unsub)
}

type tstFuture struct {
	oe     *mocks.MockOracleEngine
	future *products.Future
	unsub  int
}

func (tf *tstFuture) unsubscribe(_ context.Context, _ spec.SubscriptionID) {
	tf.unsub++
}

func getTestFutureProd(t *testing.T, propertyTpe datapb.PropertyKey_Type, dp uint64) *types.Future {
	t.Helper()
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	f := &types.Future{
		SettlementAsset: "ETH",
		QuoteName:       "ETH",
		DataSourceSpecForTradingTermination: &datasource.Spec{
			Data: datasource.NewDefinition(
				datasource.ContentTypeOracle,
			).SetOracleConfig(
				&signedoracle.SpecConfiguration{
					Signers: pubKeys,
					Filters: []*dstypes.SpecFilter{
						{
							Key: &dstypes.SpecPropertyKey{
								Name: "trading.termination",
								Type: datapb.PropertyKey_TYPE_BOOLEAN,
							},
							Conditions: nil,
						},
					},
				},
			),
		},
		DataSourceSpecBinding: &datasource.SpecBindingForFuture{
			SettlementDataProperty:     "price.ETH.value",
			TradingTerminationProperty: "trading.termination",
		},
	}

	f.DataSourceSpecForSettlementData = &datasource.Spec{
		Data: datasource.NewDefinition(
			datasource.ContentTypeOracle,
		).SetOracleConfig(
			&signedoracle.SpecConfiguration{
				Signers: pubKeys,
				Filters: []*dstypes.SpecFilter{
					{
						Key: &dstypes.SpecPropertyKey{
							Name:                "price.ETH.value",
							Type:                propertyTpe,
							NumberDecimalPlaces: &dp,
						},
						Conditions: nil,
					},
				},
			},
		),
	}

	return f
}

func testFuture(t *testing.T, propertyTpe datapb.PropertyKey_Type) *tstFuture {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	var dp uint64 = 5
	f := getTestFutureProd(t, propertyTpe, dp)

	testFuture := &tstFuture{
		oe: oe,
	}

	ctx := context.Background()
	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(1), testFuture.unsubscribe, nil)

	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(2), testFuture.unsubscribe, nil)

	future, err := products.NewFuture(ctx, log, f, oe, uint32(dp))
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	testFuture.future = future
	return testFuture
}

func subscriptionID(i uint64) spec.SubscriptionID {
	return spec.SubscriptionID(i)
}
