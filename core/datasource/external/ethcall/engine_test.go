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

package ethcall_test

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall/mocks"
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
	e := ethcall.NewEngine(log, TEST_CONFIG, true, tc.client, forwarder)

	currentEthTime := tc.client.Blockchain().CurrentBlock().Time

	argsAsJson, err := ethcall.AnyArgsToJson([]any{big.NewInt(66)})
	require.NoError(t, err)

	ethCallSpec := &ethcallcommon.Spec{
		Address:  tc.contractAddr.Hex(),
		AbiJson:  tc.abiBytes,
		Method:   "get_uint256",
		ArgsJson: argsAsJson,
		Trigger: ethcallcommon.TimeTrigger{
			Initial: currentEthTime,
			Every:   20,
			Until:   0,
		},

		RequiredConfirmations: 0,
		Filters:               common.SpecFilters{},
	}

	def := datasource.NewDefinitionWith(ethCallSpec)
	oracleSpec := datasource.Spec{
		ID:   "testid",
		Data: def,
		//},
	}

	err = e.OnSpecActivated(context.Background(), oracleSpec)

	require.NoError(t, err)

	// Make sure engine has a previous block to compare to
	e.Poll(ctx, time.Now())

	// Every commit advances chain time 10 seconds.
	// This one shouldn't trigger our call because we're set to fire every 20 seconds
	tc.client.Commit()
	e.Poll(ctx, time.Now())

	// But this one should
	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)

		assert.Equal(t, cc.BlockHeight, uint64(3))
		assert.Equal(t, cc.BlockTime, uint64(30))
		assert.Equal(t, cc.SpecId, "testid")
	})
	tc.client.Commit()
	e.Poll(ctx, time.Now())

	// Now try advancing advancing eth time 40 seconds through a two triggers and
	// check that we get called twice given a single call to OnTick()
	tc.client.Commit()
	tc.client.Commit()
	tc.client.Commit()
	tc.client.Commit()

	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)
		assert.Equal(t, cc.BlockHeight, uint64(5))
		assert.Equal(t, cc.BlockTime, uint64(50))
	})

	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)
		assert.Equal(t, cc.BlockHeight, uint64(7))
		assert.Equal(t, cc.BlockTime, uint64(70))
	})

	e.Poll(ctx, time.Now())

	// Now deactivate the spec and make sure we don't get called again
	tc.client.Commit()
	tc.client.Commit()

	e.OnSpecDeactivated(context.Background(), oracleSpec)
	e.Poll(ctx, time.Now())
}

func TestEngineWithErrorSpec(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	forwarder := mocks.NewMockForwarder(ctrl)

	log := logging.NewTestLogger()
	e := ethcall.NewEngine(log, TEST_CONFIG, true, tc.client, forwarder)

	currentEthTime := tc.client.Blockchain().CurrentBlock().Time

	argsAsJson, err := ethcall.AnyArgsToJson([]any{big.NewInt(66)})
	require.NoError(t, err)

	// To simulate a contract call error, we'll change the method name
	tc.abiBytes = []byte(strings.Replace(string(tc.abiBytes), "get_uint256", "get_uint256doesnotexist", -1))

	ethCallSpec := &ethcallcommon.Spec{
		Address:  tc.contractAddr.Hex(),
		AbiJson:  tc.abiBytes,
		Method:   "get_uint256doesnotexist",
		ArgsJson: argsAsJson,
		Trigger: ethcallcommon.TimeTrigger{
			Initial: currentEthTime,
			Every:   20,
			Until:   0,
		},

		RequiredConfirmations: 0,
		Filters:               common.SpecFilters{},
	}

	def := datasource.NewDefinitionWith(ethCallSpec)
	oracleSpec := datasource.Spec{
		//&types.DataSourceSpec{
		ID:   "testid",
		Data: def,
		//	},
	}

	err = e.OnSpecActivated(context.Background(), oracleSpec)

	require.NoError(t, err)

	// Make sure engine has a previous block to compare to
	e.Poll(ctx, time.Now())

	// Every commit advances chain time 10 seconds.
	// This one shouldn't trigger our call because we're set to fire every 20 seconds
	tc.client.Commit()
	e.Poll(ctx, time.Now())

	// But this one should
	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)

		assert.Equal(t, cc.BlockHeight, uint64(3))
		assert.Equal(t, cc.BlockTime, uint64(30))
		assert.Equal(t, cc.SpecId, "testid")
		assert.NotNil(t, cc.Error)
	})
	tc.client.Commit()
	e.Poll(ctx, time.Now())
}
