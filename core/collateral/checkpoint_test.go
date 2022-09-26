package collateral_test

import (
	"context"
	"testing"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkpointTestEngine struct {
	*collateral.Engine
	ctrl   *gomock.Controller
	broker *bmocks.MockBroker
}

func newCheckpointTestEngine(t *testing.T) *checkpointTestEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}

	broker.EXPECT().Send(gomock.Any()).Times(3)

	e := collateral.New(logging.NewTestLogger(), conf, timeSvc, broker)
	e.EnableAsset(context.Background(), types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:     "VEGA",
			Symbol:   "VEGA",
			Decimals: 5,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	})

	return &checkpointTestEngine{
		Engine: e,
		ctrl:   ctrl,
		broker: broker,
	}
}

func TestCheckPointLoadingWithAlias(t *testing.T) {
	e := newCheckpointTestEngine(t)
	defer e.ctrl.Finish()

	e.broker.EXPECT().Send(gomock.Any()).Times(3).Do(func(e events.Event) {
		ledgerMovmenentsE, ok := e.(*events.LedgerMovements)
		if !ok {
			return
		}

		mvts := ledgerMovmenentsE.LedgerMovements()
		assert.Len(t, mvts, 2)
		assert.Len(t, mvts[0].Entries, 1)
		// no owner + from externa
		assert.Nil(t, mvts[0].Entries[0].FromAccount.Owner)
		assert.Equal(t, mvts[0].Entries[0].FromAccount.Type, types.AccountTypeExternal)
		assert.Equal(t, mvts[0].Entries[0].Amount, "1000")
		// to no owner + to reward
		assert.Nil(t, mvts[0].Entries[0].ToAccount.Owner)
		assert.Equal(t, mvts[0].Entries[0].ToAccount.Type, types.AccountTypeGlobalReward)

		// second transfer
		assert.Len(t, mvts[1].Entries, 1)
		// no owner + from externa
		assert.Nil(t, mvts[1].Entries[0].FromAccount.Owner)
		assert.Equal(t, mvts[1].Entries[0].FromAccount.Type, types.AccountTypeExternal)
		assert.Equal(t, mvts[1].Entries[0].Amount, "2000")
		// to no owner + to reward
		assert.Nil(t, mvts[1].Entries[0].ToAccount.Owner)
		assert.Equal(t, mvts[1].Entries[0].ToAccount.Type, types.AccountTypeGlobalReward)
	})

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
