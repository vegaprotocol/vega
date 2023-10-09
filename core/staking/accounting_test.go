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

package staking_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/staking"
	smocks "code.vegaprotocol.io/vega/core/staking/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accountingTest struct {
	*staking.Accounting
	log     *logging.Logger
	ctrl    *gomock.Controller
	tsvc    *smocks.MockTimeService
	broker  *mocks.MockBroker
	evtfwd  *smocks.MockEvtForwarder
	witness *smocks.MockWitness

	onTick func(context.Context, time.Time)
}

func getAccountingTest(t *testing.T) *accountingTest {
	t.Helper()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	ts := smocks.NewMockTimeService(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	evtfwd := smocks.NewMockEvtForwarder(ctrl)
	witness := smocks.NewMockWitness(ctrl)
	var onTick func(context.Context, time.Time)

	return &accountingTest{
		Accounting: staking.NewAccounting(
			log, staking.NewDefaultConfig(), ts, broker, nil, evtfwd, witness, true),
		log:     log,
		ctrl:    ctrl,
		tsvc:    ts,
		broker:  broker,
		evtfwd:  evtfwd,
		witness: witness,
		onTick:  onTick,
	}
}

func TestStakingAccounting(t *testing.T) {
	t.Run("error party don't exists", testPartyDontExists)
	t.Run("get available balance at", testAccountingGetAvailableBalanceAt)
	t.Run("get available balance in range", testAccountingGetAvailableBalanceInRange)
	t.Run("generate Hash", testAccountingGenerateHash)
}

func testPartyDontExists(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()

	balance, err := acc.GetAvailableBalance("nope")
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.UintZero(), balance)
	balance, err = acc.GetAvailableBalanceAt("nope", time.Unix(10, 0))
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.UintZero(), balance)
	balance, err = acc.GetAvailableBalanceInRange("nope", time.Unix(10, 0), time.Unix(20, 0))
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.UintZero(), balance)
}

func testAccountingGetAvailableBalanceInRange(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()
	cases := []struct {
		evt    types.StakeLinking
		expect error
	}{
		{
			evt: types.StakeLinking{
				ID:     "someid1",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid2",
				Type:   types.StakeLinkingTypeRemoved,
				TS:     105,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid3",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     106,
				Party:  testParty,
				Amount: num.NewUint(3),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid4",
				Type:   types.StakeLinkingTypeRemoved,
				TS:     107,
				Party:  testParty,
				Amount: num.NewUint(4),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid5",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid6",
				Type:   types.StakeLinkingTypeRemoved,
				TS:     125,
				Party:  testParty,
				Amount: num.NewUint(6),
			},
			expect: nil,
		},
	}

	acc.broker.EXPECT().Send(gomock.Any()).Times(1)

	for _, c := range cases {
		c := c
		acc.AddEvent(context.Background(), &c.evt)
	}

	balance, err := acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 10), time.Unix(0, 20))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 10), time.Unix(0, 110))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 109))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(8), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 111))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(8), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 121))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(8), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 126))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(7), balance)
}

func testAccountingGetAvailableBalanceAt(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()
	cases := []struct {
		evt    types.StakeLinking
		expect error
	}{
		{
			evt: types.StakeLinking{
				ID:     "someid1",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid2",
				Type:   types.StakeLinkingTypeRemoved,
				TS:     110,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid3",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
	}

	acc.broker.EXPECT().Send(gomock.Any()).Times(1)

	for _, c := range cases {
		c := c
		acc.AddEvent(context.Background(), &c.evt)
	}

	balance, err := acc.GetAvailableBalanceAt(testParty, time.Unix(0, 10))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)
	balance, err = acc.GetAvailableBalanceAt(testParty, time.Unix(0, 120))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(14), balance)
	balance, err = acc.GetAvailableBalanceAt(testParty, time.Unix(0, 115))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(9), balance)
}

func testAccountingGenerateHash(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()
	cases := []struct {
		evt    types.StakeLinking
		expect error
	}{
		{
			evt: types.StakeLinking{
				ID:     "someid1",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     100,
				Party:  "party1",
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid2",
				Type:   types.StakeLinkingTypeRemoved,
				TS:     110,
				Party:  "party1",
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid3",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     120,
				Party:  "party2",
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
		{
			evt: types.StakeLinking{
				ID:     "someid4",
				Type:   types.StakeLinkingTypeDeposited,
				TS:     120,
				Party:  "party3",
				Amount: num.NewUint(42),
			},
			expect: nil,
		},
	}

	acc.broker.EXPECT().Send(gomock.Any()).Times(3)

	for _, c := range cases {
		c := c
		acc.AddEvent(context.Background(), &c.evt)
	}

	require.Equal(t,
		"ab5a48b34ac9f8c33a0441b6af04c84e2759086882b93aec972f4a709f93f8e9",
		hex.EncodeToString(acc.Hash()),
		"hash is not deterministic",
	)
}
