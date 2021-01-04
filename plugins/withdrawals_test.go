package plugins_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type witPluginTest struct {
	*plugins.Withdrawal
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
}

func getWitPlugin(t *testing.T) *witPluginTest {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	p := plugins.NewWithdrawal(ctx)
	return &witPluginTest{
		Withdrawal: p,
		ctrl:       ctrl,
		ctx:        ctx,
		cfunc:      cfunc,
	}
}

func (w *witPluginTest) Finish() {
	w.cfunc() // cancel context
	defer w.ctrl.Finish()
}

func TestWithdrawalPlugin(t *testing.T) {
	t.Run("Get withdrawal by id", testGetWithdrawalByID)
	t.Run("Get withdrawal by party", testGetWithdrawalByParty)
}

func testGetWithdrawalByID(t *testing.T) {
	wit := getWitPlugin(t)
	defer wit.Finish()

	w1 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party1",
			Id:      "wid1",
			Amount:  200,
		},
	)
	w2 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party2",
			Id:      "wid2",
			Amount:  300,
		},
	)
	w3 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party1",
			Id:      "wid3",
			Amount:  500,
		},
	)

	wit.Push(w1, w2, w3)
	var (
		hasError bool = true
		retries       = 50
	)
	for hasError && retries > 0 {
		retries -= 1
		_, err1 := wit.GetByID("wid1")
		_, err2 := wit.GetByID("wid1")
		_, err3 := wit.GetByID("wid1")
		hasError = err1 == nil && err2 == nil && err3 == nil
		time.Sleep(50 * time.Millisecond)
	}
	// then test actual values
	w, err := wit.GetByID("wid1")
	assert.NoError(t, err)
	assert.Equal(t, "party1", w.PartyID)
	assert.Equal(t, 200, int(w.Amount))
	w, err = wit.GetByID("wid2")
	assert.NoError(t, err)
	assert.Equal(t, "party2", w.PartyID)
	assert.Equal(t, 300, int(w.Amount))
	w, err = wit.GetByID("wid3")
	assert.NoError(t, err)
	assert.Equal(t, "party1", w.PartyID)
	assert.Equal(t, 500, int(w.Amount))
}

func testGetWithdrawalByParty(t *testing.T) {
	wit := getWitPlugin(t)
	defer wit.ctrl.Finish()

	w1 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party1",
			Id:      "wid1",
			Amount:  200,
		},
	)
	w2 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party2",
			Id:      "wid2",
			Amount:  300,
		},
	)
	w3 := events.NewWithdrawalEvent(
		wit.ctx,
		proto.Withdrawal{
			PartyID: "party1",
			Id:      "wid3",
			Amount:  500,
		},
	)

	wit.Push(w1, w2, w3)
	var (
		hasError bool = true
		retries       = 50
	)
	for hasError && retries > 0 {
		retries -= 1
		_, err1 := wit.GetByID("wid3")
		hasError = err1 == nil
		time.Sleep(50 * time.Millisecond)
	}

	wits := wit.GetByParty("party1", false)
	assert.Len(t, wits, 2)
}
