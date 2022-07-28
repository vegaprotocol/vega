package collateral_test

import (
	"context"
	"testing"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func checkpointTestEngine(t *testing.T) *collateral.Engine {
	t.Helper()
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	e := collateral.New(logging.NewTestLogger(), conf, timeSvc, broker)
	e.EnableAsset(context.Background(), types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:        "VEGA",
			Symbol:      "VEGA",
			Decimals:    5,
			TotalSupply: num.NewUint(1000),
			Quantum:     num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.Zero(),
				},
			},
		},
	})
	return e
}

func TestCheckPointLoadingWithAlias(t *testing.T) {
	e := checkpointTestEngine(t)

	ab := []*checkpoint.AssetBalance{
		{Party: "*", Asset: "VEGA", Balance: "1000"},
		{Party: "*ACCOUNT_TYPE_GLOBAL_REWARD", Asset: "VEGA", Balance: "2000"},
	}

	msg := &checkpoint.Collateral{
		Balances: ab,
	}

	ret, err := proto.Marshal(msg)
	require.NoError(t, err)

	e.Load(context.Background(), ret)

	acc, err := e.GetGlobalRewardAccount("VEGA")
	require.NoError(t, err)
	require.Equal(t, "3000", acc.Balance.String())

	_, err = e.GetPartyGeneralAccount("*ACCOUNT_TYPE_GLOBAL_REWARD", "VEGA")
	require.Error(t, err)
}
