package staking_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

type accountingTest struct {
	*staking.Accounting
	log *logging.Logger
}

func getAccountingTest() *accountingTest {
	log := logging.NewTestLogger()
	return &accountingTest{
		Accounting: staking.NewAccounting(log),
		log:        log,
	}
}

func TestStakingAccounting(t *testing.T) {
	t.Run("error party don't exists", testPartyDontExists)
	t.Run("get available balance at", testAccountingGetAvailableBalanceAt)
	t.Run("get available balance in range", testAccountingGetAvailableBalanceInRange)
}

func testPartyDontExists(t *testing.T) {
	acc := getAccountingTest()

	balance, err := acc.GetAvailableBalance("nope")
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.Zero(), balance)
	balance, err = acc.GetAvailableBalanceAt("nope", time.Unix(10, 0))
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.Zero(), balance)
	balance, err = acc.GetAvailableBalanceInRange("nope", time.Unix(10, 0), time.Unix(20, 0))
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, num.Zero(), balance)
}

func testAccountingGetAvailableBalanceInRange(t *testing.T) {
	acc := getAccountingTest()
	cases := []struct {
		evt    types.StakingEvent
		expect error
	}{
		{
			evt: types.StakingEvent{
				ID:     "someid1",
				Kind:   types.StakingEventKindDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid2",
				Kind:   types.StakingEventKindRemoved,
				TS:     110,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid3",
				Kind:   types.StakingEventKindDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid4",
				Kind:   types.StakingEventKindRemoved,
				TS:     125,
				Party:  testParty,
				Amount: num.NewUint(6),
			},
			expect: nil,
		},
	}

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
	assert.Equal(t, num.NewUint(10), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 111))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(9), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 121))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(10), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		testParty, time.Unix(0, 101), time.Unix(0, 126))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(8), balance)
}

func testAccountingGetAvailableBalanceAt(t *testing.T) {
	acc := getAccountingTest()
	cases := []struct {
		evt    types.StakingEvent
		expect error
	}{
		{
			evt: types.StakingEvent{
				ID:     "someid1",
				Kind:   types.StakingEventKindDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid2",
				Kind:   types.StakingEventKindRemoved,
				TS:     110,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid3",
				Kind:   types.StakingEventKindDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
	}

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
