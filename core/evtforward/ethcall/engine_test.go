package ethcall_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"code.vegaprotocol.io/vega/core/evtforward/ethcall/mocks"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TEST_CONFIG = ethcall.Config{
	Level:     encoding.LogLevel{Level: logging.DebugLevel},
	PollEvery: encoding.Duration{Duration: 100 * time.Second},
}

func TestEngine(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	forwarder := mocks.NewMockForwarder(ctrl)

	log := logging.NewTestLogger()
	e := ethcall.NewEngine(log, TEST_CONFIG, tc.client, forwarder)

	currentEthTime := tc.client.Blockchain().CurrentBlock().Time

	argsAsJson, err := ethcall.AnyArgsToJson([]any{big.NewInt(66)})
	require.NoError(t, err)

	ethCallSpec := &types.EthCallSpec{
		Address:  tc.contractAddr.Hex(),
		AbiJson:  tc.abiBytes,
		Method:   "get_uint256",
		ArgsJson: argsAsJson,
		Trigger: types.EthTimeTrigger{
			Initial: currentEthTime,
			Every:   20,
			Until:   0,
		},

		RequiredConfirmations: 0,
		Filters:               types.DataSourceSpecFilters{},
	}

	def := types.NewDataSourceDefinitionWith(ethCallSpec)
	oracleSpec := types.OracleSpec{
		ExternalDataSourceSpec: &types.ExternalDataSourceSpec{
			Spec: &types.DataSourceSpec{
				ID:   "testid",
				Data: def,
			},
		},
	}

	err = e.OnSpecActivated(context.Background(), oracleSpec)

	require.NoError(t, err)

	// Make sure engine has a previous block to compare to
	e.OnTick(ctx, time.Now())

	// Every commit advances chain time 10 seconds.
	// This one shouldn't trigger our call because we're set to fire every 20 seconds
	tc.client.Commit()
	e.OnTick(ctx, time.Now())

	// But this one should
	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)

		assert.Equal(t, cc.BlockHeight, uint64(3))
		assert.Equal(t, cc.BlockTime, uint64(30))
		assert.Equal(t, cc.SpecId, "testid")
	})
	tc.client.Commit()
	e.OnTick(ctx, time.Now())
}
