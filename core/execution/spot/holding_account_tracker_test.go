package spot_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testHat struct {
	*spot.HoldingAccountTracker
	collat *collateral.Engine
}

func getTestHat(t *testing.T) *testHat {
	t.Helper()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	collat := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)
	asset := types.Asset{
		ID: "BTC",
		Details: &types.AssetDetails{
			Symbol:   "BTC",
			Name:     "BTC",
			Decimals: 0,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	}
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	err := collat.EnableAsset(context.Background(), asset)
	require.NoError(t, err)
	id, err := collat.CreatePartyGeneralAccount(context.Background(), "zohar", "BTC")
	require.NoError(t, err)
	require.NoError(t, collat.IncrementBalance(context.Background(), id, num.NewUint(1500)))

	return &testHat{
		HoldingAccountTracker: spot.NewHoldingAccountTracker("market1", log, collat),
		collat:                collat,
	}
}

func TestReleaseAllFromHoldingAccount(t *testing.T) {
	hat := getTestHat(t)

	generalAccount, err := hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1500), generalAccount.Balance)

	_, err = hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(1000), num.NewUint(2))
	require.NoError(t, err)

	_, err = hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(200), num.NewUint(3))
	require.Error(t, fmt.Errorf("funds for the order have already been transferred to the holding account"), err)

	holdingQty, holdingFee := hat.GetCurrentHolding("1")
	require.Equal(t, num.NewUint(1000), holdingQty)
	require.Equal(t, num.NewUint(2), holdingFee)

	generalAccount, err = hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(498), generalAccount.Balance)

	_, err = hat.ReleaseAllFromHoldingAccount(context.Background(), "1", "zohar", "BTC")
	require.NoError(t, err)

	holdingQty, holdingFee = hat.GetCurrentHolding("1")
	require.Equal(t, num.UintZero(), holdingQty)
	require.Equal(t, num.UintZero(), holdingFee)

	generalAccount, err = hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1500), generalAccount.Balance)
}

func TestReleaseQuantityHoldingAccount(t *testing.T) {
	hat := getTestHat(t)

	generalAccount, err := hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1500), generalAccount.Balance)

	_, err = hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(1000), num.NewUint(2))
	require.NoError(t, err)
	holdingQty, holdingFee := hat.GetCurrentHolding("1")
	require.Equal(t, num.NewUint(1000), holdingQty)
	require.Equal(t, num.NewUint(2), holdingFee)

	generalAccount, err = hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(498), generalAccount.Balance)

	_, err = hat.ReleaseQuantityHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(500), num.NewUint(1))
	require.NoError(t, err)
	holdingQty, holdingFee = hat.GetCurrentHolding("1")
	require.Equal(t, num.NewUint(500), holdingQty)
	require.Equal(t, num.NewUint(1), holdingFee)

	generalAccount, err = hat.collat.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(999), generalAccount.Balance)
}

func TestReleaseFeeFromHoldingAccount(t *testing.T) {
	hat := getTestHat(t)

	_, err := hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(1000), num.NewUint(2))
	require.NoError(t, err)

	holdingQty, holdingFee := hat.GetCurrentHolding("1")
	require.Equal(t, num.NewUint(1000), holdingQty)
	require.Equal(t, num.NewUint(2), holdingFee)

	le, err := hat.ReleaseFeeFromHoldingAccount(context.Background(), "1", "zohar", "BTC")
	require.NoError(t, err)

	_, holdingFee = hat.GetCurrentHolding("1")
	require.Equal(t, num.UintZero(), holdingFee)
	require.Equal(t, num.NewUint(2), le.Balances[0].Balance)
}

func TestTransferFeeToHoldingAccount(t *testing.T) {
	hat := getTestHat(t)
	_, err := hat.TransferFeeToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(2))
	require.NoError(t, err)

	holdingQty, holdingFee := hat.GetCurrentHolding("1")
	require.Equal(t, num.UintZero(), holdingQty)
	require.Equal(t, num.NewUint(2), holdingFee)
}

func TestSnapshot(t *testing.T) {
	hat := getTestHat(t)
	_, err := hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(1000), num.NewUint(2))
	require.NoError(t, err)

	state, _, err := hat.GetState("market1")
	require.NoError(t, err)
	var active snapshot.Payload
	require.NoError(t, proto.Unmarshal(state, &active))
	payload := types.PayloadFromProto(&active)

	hat2 := getTestHat(t)
	hat2.LoadState(context.Background(), payload)
	state2, _, err := hat2.GetState("market1")
	require.NoError(t, err)

	require.True(t, bytes.Equal(state, state2))
}

func TestTransferToHoldingAccount(t *testing.T) {
	hat := getTestHat(t)

	_, err := hat.TransferToHoldingAccount(context.Background(), "1", "zohar", "BTC", num.NewUint(1000), num.NewUint(2))
	require.NoError(t, err)

	holdingQty, holdingFee := hat.GetCurrentHolding("1")
	require.Equal(t, num.NewUint(1000), holdingQty)
	require.Equal(t, num.NewUint(2), holdingFee)
}
