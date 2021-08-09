package staking_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

const (
	testParty = "bob"
)

func TestStakingAccount(t *testing.T) {
	t.Run("account with no event have a zero balance", testGetZeroBalanceWithNoEvent)
	t.Run("event error validation", testEventErrorValidation)
	t.Run("test events sorting", testEventSorting)
	t.Run("get available balance at", testGetAvailableBalanceAt)
	t.Run("get available balance in range", testGetAvailableBalanceInRange)
}

func testGetAvailableBalanceInRange(t *testing.T) {
	acc := staking.NewStakingAccount(testParty)
	cases := []struct {
		evt    types.StakingEvent
		expect error
	}{
		{
			evt: types.StakingEvent{
				ID:     "someid1",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid2",
				Type:   types.StakingEventTypeRemoved,
				TS:     110,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid3",
				Type:   types.StakingEventTypeDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid4",
				Type:   types.StakingEventTypeRemoved,
				TS:     125,
				Party:  testParty,
				Amount: num.NewUint(6),
			},
			expect: nil,
		},
	}

	for _, c := range cases {
		c := c
		err := acc.AddEvent(&c.evt)
		assert.Equal(t, c.expect, err)
	}

	balance, err := acc.GetAvailableBalanceInRange(
		time.Unix(0, 10), time.Unix(0, 20))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		time.Unix(0, 10), time.Unix(0, 110))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		time.Unix(0, 101), time.Unix(0, 109))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(10), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		time.Unix(0, 101), time.Unix(0, 111))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(9), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		time.Unix(0, 101), time.Unix(0, 121))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(10), balance)

	balance, err = acc.GetAvailableBalanceInRange(
		time.Unix(0, 101), time.Unix(0, 126))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(8), balance)
}

func testGetAvailableBalanceAt(t *testing.T) {
	acc := staking.NewStakingAccount(testParty)
	cases := []struct {
		evt    types.StakingEvent
		expect error
	}{
		{
			evt: types.StakingEvent{
				ID:     "someid1",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(10),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid2",
				Type:   types.StakingEventTypeRemoved,
				TS:     110,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid3",
				Type:   types.StakingEventTypeDeposited,
				TS:     120,
				Party:  testParty,
				Amount: num.NewUint(5),
			},
			expect: nil,
		},
	}

	for _, c := range cases {
		c := c
		err := acc.AddEvent(&c.evt)
		assert.Equal(t, c.expect, err)
	}

	balance, err := acc.GetAvailableBalanceAt(time.Unix(0, 10))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), balance)
	balance, err = acc.GetAvailableBalanceAt(time.Unix(0, 120))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(14), balance)
	balance, err = acc.GetAvailableBalanceAt(time.Unix(0, 115))
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(9), balance)
}

func testEventSorting(t *testing.T) {
	acc := staking.NewStakingAccount(testParty)

	// we have 2 events added for the same timestamp
	// although the first one is a remove
	// second one is a deposit
	// so we would expect at the end for the events to
	// be sorted and return the actual balance
	evts := []types.StakingEvent{
		{
			ID:     "someid2",
			Type:   types.StakingEventTypeRemoved,
			TS:     100,
			Party:  testParty,
			Amount: num.NewUint(1),
		},
		{
			ID:     "someid1",
			Type:   types.StakingEventTypeDeposited,
			TS:     100,
			Party:  testParty,
			Amount: num.NewUint(100),
		},
	}

	err := acc.AddEvent(&evts[0])
	assert.EqualError(t, staking.ErrNegativeBalance, err.Error())
	err = acc.AddEvent(&evts[1])
	assert.NoError(t, err)
	// now assert the final balance
	assert.Equal(t, num.NewUint(99), acc.GetAvailableBalance())
}

func testGetZeroBalanceWithNoEvent(t *testing.T) {
	acc := staking.NewStakingAccount(testParty)
	assert.Equal(t, num.Zero(), acc.GetAvailableBalance())

}

func testEventErrorValidation(t *testing.T) {
	acc := staking.NewStakingAccount(testParty)

	cases := []struct {
		evt    types.StakingEvent
		expect error
	}{
		{ // invalid id
			evt: types.StakingEvent{
				ID:     "",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: staking.ErrMissingEventID,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   10,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: staking.ErrInvalidEventKind,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   types.StakingEventTypeDeposited,
				TS:     0,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: staking.ErrMissingTimestamp,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.Zero(),
			},
			expect: staking.ErrInvalidAmount,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  "not-a-party",
				Amount: num.NewUint(10),
			},
			expect: staking.ErrInvalidParty,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: nil,
		},
		{
			evt: types.StakingEvent{
				ID:     "someid",
				Type:   types.StakingEventTypeDeposited,
				TS:     100,
				Party:  testParty,
				Amount: num.NewUint(1),
			},
			expect: staking.ErrEventAlreadyExists,
		},
	}

	for _, c := range cases {
		c := c
		err := acc.AddEvent(&c.evt)
		assert.Equal(t, c.expect, err)
	}
}
