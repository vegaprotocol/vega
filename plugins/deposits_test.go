package plugins_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type depPluginTest struct {
	*plugins.Deposit
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
}

func getDepPlugin(t *testing.T) *depPluginTest {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	p := plugins.NewDeposit(ctx)
	return &depPluginTest{
		Deposit: p,
		ctrl:    ctrl,
		ctx:     ctx,
		cfunc:   cfunc,
	}
}

func (w *depPluginTest) Finish() {
	w.cfunc() // cancel context
	defer w.ctrl.Finish()
}

func TestDepositPlugin(t *testing.T) {
	t.Run("Get deposit by id", testGetDepositByID)
	t.Run("Get deposit by party", testGetDepositByParty)
}

func testGetDepositByID(t *testing.T) {
	dep := getDepPlugin(t)
	defer dep.Finish()

	w1 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party1",
			Id:      "wid1",
			Amount:  "200",
		},
	)
	w2 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party2",
			Id:      "wid2",
			Amount:  "300",
		},
	)
	w3 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party1",
			Id:      "wid3",
			Amount:  "500",
		},
	)

	dep.Push(w1, w2, w3)
	var (
		hasError = true
		retries  = 50
	)
	for hasError && retries > 0 {
		retries -= 1
		_, err1 := dep.GetByID("wid1")
		_, err2 := dep.GetByID("wid1")
		_, err3 := dep.GetByID("wid1")
		hasError = err1 == nil && err2 == nil && err3 == nil
		time.Sleep(50 * time.Millisecond)
	}
	// then test actual values
	w, err := dep.GetByID("wid1")
	assert.NoError(t, err)
	assert.Equal(t, "party1", w.PartyId)
	assert.Equal(t, "200", w.Amount)
	w, err = dep.GetByID("wid2")
	assert.NoError(t, err)
	assert.Equal(t, "party2", w.PartyId)
	assert.Equal(t, "300", w.Amount)
	w, err = dep.GetByID("wid3")
	assert.NoError(t, err)
	assert.Equal(t, "party1", w.PartyId)
	assert.Equal(t, "500", w.Amount)
}

func testGetDepositByParty(t *testing.T) {
	dep := getDepPlugin(t)
	defer dep.ctrl.Finish()

	w1 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party1",
			Id:      "wid1",
			Amount:  "200",
		},
	)
	w2 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party2",
			Id:      "wid2",
			Amount:  "300",
		},
	)
	w3 := events.NewDepositEvent(
		dep.ctx,
		proto.Deposit{
			PartyId: "party1",
			Id:      "wid3",
			Amount:  "500",
		},
	)

	dep.Push(w1, w2, w3)
	var (
		hasError = true
		retries  = 50
	)
	for hasError && retries > 0 {
		retries -= 1
		_, err1 := dep.GetByID("wid3")
		hasError = err1 == nil
		time.Sleep(50 * time.Millisecond)
	}

	deps := dep.GetByParty("party1", false)
	assert.Len(t, deps, 2)
}
