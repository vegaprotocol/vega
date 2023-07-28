// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	"github.com/stretchr/testify/require"
)

func TestScalingOfSettlementData(t *testing.T) {
	t.Run("No scaling needed for settlement data for asset decimals", testNoScalingNeeded)
	t.Run("Need to scale up the settlement data for asset decimals", testScalingUpNeeded)
	t.Run("Need to scale down the settlement data for asset decimals no loss of precision", testScalingDownNeeded)
	t.Run("Need to scale down the settlement data for asset decimals with loss of precision", testScalingDownNeededWithPrecisionLoss)
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

type tstFuture struct {
	oe     *mocks.MockOracleEngine
	future *products.Future
}

func testFuture(t *testing.T, propertyTpe datapb.PropertyKey_Type) *tstFuture {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	var dp uint64 = 5
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

	ctx := context.Background()
	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(1), func(ctx context.Context, sid spec.SubscriptionID) {}, nil)

	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(2), func(ctx context.Context, sid spec.SubscriptionID) {}, nil)

	future, err := products.NewFuture(ctx, log, f, oe, uint32(dp))
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	return &tstFuture{
		future: future,
		oe:     oe,
	}
}

func subscriptionID(i uint64) spec.SubscriptionID {
	return spec.SubscriptionID(i)
}
