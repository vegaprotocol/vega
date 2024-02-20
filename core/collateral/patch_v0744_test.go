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

package collateral_test

import (
	"context"
	"testing"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	pubkeys = []string{
		"89c98f0e1039935b5d7f5b8d6d0660790a8e507d0c4234b6cafb7dbf88ad25ca",
		"c8f5d32a8554dbddfa80946fe9ac42d156356f869256aa0a632e5152d45b1316",
		"426f40b09ea2388c22e7c409b6e979747597316939ed6b422c5b935069ad4814",
		"519d2af4058af1bed4e05859afa6a15cb1791166df8f0fe3f70a783a13232440",
		"1d150c717d349e901cc26e511f776c323c1b8a8dbb0e7717183f2a1e9f3482d7",
		"36e73d371b25f0d97ce7813d688c42e61792bda80c00c9cf6d8bf9424a539bf5",
		"0a9b24a83cb661e68a2069a413cc2603f0f4804b165621806fa8a014fb0ed4b5",
		// perpetrator key:
		"1d2f37299f436f3b720b8efbbb6beb4aec9145a2c4f398ccb06280f6fa2503e8",
	}
	amountsReturned = map[string]*num.Uint{
		"89c98f0e1039935b5d7f5b8d6d0660790a8e507d0c4234b6cafb7dbf88ad25ca": num.MustUintFromString("110027790000", 10), // 110,027.79 USDT
		"c8f5d32a8554dbddfa80946fe9ac42d156356f869256aa0a632e5152d45b1316": num.MustUintFromString("2872880000", 10),   // 2,872.88 USDT
		"426f40b09ea2388c22e7c409b6e979747597316939ed6b422c5b935069ad4814": num.MustUintFromString("128530000", 10),    // 128.53 USDT
		"519d2af4058af1bed4e05859afa6a15cb1791166df8f0fe3f70a783a13232440": num.MustUintFromString("114110000", 10),    // 114.11 USDT
		"1d150c717d349e901cc26e511f776c323c1b8a8dbb0e7717183f2a1e9f3482d7": num.MustUintFromString("80840000", 10),     // 80.84 USDT
		"36e73d371b25f0d97ce7813d688c42e61792bda80c00c9cf6d8bf9424a539bf5": num.MustUintFromString("71080000", 10),     // 71.08 USDT
		"0a9b24a83cb661e68a2069a413cc2603f0f4804b165621806fa8a014fb0ed4b5": num.MustUintFromString("50640000", 10),     // 50.64 USDT
		// this last entry is the key from the perpetrator, and the funds to be returned to them
		"1d2f37299f436f3b720b8efbbb6beb4aec9145a2c4f398ccb06280f6fa2503e8": num.MustUintFromString("1393274897", 10), // 1,393.274897 USDT
	}
)

func TestPatchV0744(t *testing.T) {
	c := setupEngine(t)

	// parameters for the migration function
	ctx := context.Background()
	log := logging.NewTestLogger()

	// create the account for the parties first
	// this is only for the test context, on mainnet
	// those parties will exist
	c.broker.EXPECT().Send(gomock.Any()).Times(16)
	for _, pubkey := range pubkeys {
		_, err := c.CreatePartyGeneralAccount(ctx, pubkey, collateral.TetherUSD)
		assert.NoError(t, err)
	}

	c.broker.EXPECT().Send(gomock.Any()).Times(8)
	c.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		lme, ok := e.(*events.LedgerMovements)
		assert.True(t, ok)
		movements := lme.LedgerMovements()
		assert.Equal(t, len(pubkeys), len(movements))
		for _, lm := range movements {
			entries := lm.Entries
			assert.Equal(t, 1, len(entries))
			entry := entries[0]
			// get the pubkey
			key := entry.ToAccount.GetOwner()
			expAmt, ok := amountsReturned[key]
			assert.True(t, ok)
			assert.Equal(t, expAmt.String(), entry.Amount)
			// ensure the balance - in this test, because we have just created the accounts, matches the uint amount
			assert.Equal(t, expAmt.String(), entry.ToAccountBalance)
			// just for sanity, make sure we're still using the correct asset
			assert.Equal(t, collateral.TetherUSD, entry.ToAccount.AssetId)
		}
	})
	// anything failing in the Migration function would trigger a Panic
	// this test just assert that no panics happe
	assert.NotPanics(t,
		func() {
			collateral.ExecuteMigration744(ctx, c.broker, log, c.Engine)
		},
		"migration panic'd at runtime!",
	)
}

func setupEngine(t *testing.T) *testEngine {
	t.Helper()

	// intantiate our collateral engine
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	broker.EXPECT().Send(gomock.Any()).Times(7)

	eng := collateral.New(logging.NewTestLogger(), conf, timeSvc, broker)

	// enable the assert for the tests
	usdtAsset := types.Asset{
		ID: collateral.TetherUSD,
		Details: &types.AssetDetails{
			Symbol:   "USDT",
			Name:     "Tether USD",
			Decimals: 6,
			Quantum:  num.DecimalOne(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	}

	err := eng.EnableAsset(context.Background(), usdtAsset)
	assert.NoError(t, err)

	return &testEngine{
		Engine:  eng,
		ctrl:    ctrl,
		timeSvc: timeSvc,
		broker:  broker,
	}
}
