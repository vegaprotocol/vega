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
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/staking"
	"code.vegaprotocol.io/vega/core/staking/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type stakeVerifierTest struct {
	*staking.StakeVerifier

	ctrl    *gomock.Controller
	tsvc    *mocks.MockTimeService
	broker  *bmocks.MockBroker
	accs    *staking.Accounting
	ocv     *mocks.MockEthOnChainVerifier
	witness *mocks.MockWitness
	evtfwd  *mocks.MockEvtForwarder
	evtSrc  *mocks.MockEthereumEventSource

	onTick func(context.Context, time.Time)
}

func getStakeVerifierTest(t *testing.T) *stakeVerifierTest {
	t.Helper()
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	log := logging.NewTestLogger()
	cfg := staking.NewDefaultConfig()
	ocv := mocks.NewMockEthOnChainVerifier(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	witness := mocks.NewMockWitness(ctrl)
	evtfwd := mocks.NewMockEvtForwarder(ctrl)
	evtSrc := mocks.NewMockEthereumEventSource(ctrl)

	accs := staking.NewAccounting(log, cfg, ts, broker, nil, evtfwd, witness, true, evtSrc)

	svt := &stakeVerifierTest{
		StakeVerifier: staking.NewStakeVerifier(log, cfg, accs, witness, ts, broker, ocv, evtSrc),
		ctrl:          ctrl,
		broker:        broker,
		accs:          accs,
		ocv:           ocv,
		tsvc:          ts,
		witness:       witness,
		evtfwd:        evtfwd,
		evtSrc:        evtSrc,
	}
	svt.onTick = svt.StakeVerifier.OnTick

	return svt
}

func TestStakeVerifier(t *testing.T) {
	t.Run("can process stake event deposited OK", testProcessStakeEventDepositedOK)
	t.Run("can process stake event deposited KO", testProcessStakeEventDepositedKO)
	t.Run("can process stake event removed OK", testProcessStakeEventRemovedOK)
	t.Run("can process stake event removed KO", testProcessStakeEventRemovedKO)
	t.Run("can process multiple events OK", testProcessStakeEventMultiOK)
	t.Run("duplicates", testDuplicates)
}

func testProcessStakeEventDepositedOK(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(2)

	var f func(interface{}, bool)
	var evt interface{}
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := stakev.ProcessStakeDeposited(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, true)

	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.ocv.EXPECT().GetStakingBridgeAddresses().Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	balance, err := stakev.accs.GetAvailableBalance("somepubkey")
	assert.NoError(t, err)
	assert.Equal(t, 1000, int(balance.Uint64()))
}

func testProcessStakeEventDepositedKO(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)

	var f func(interface{}, bool)
	var evt interface{}
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := stakev.ProcessStakeDeposited(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, false)

	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	balance, err := stakev.accs.GetAvailableBalance("somepubkey")
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, 0, int(balance.Uint64()))
}

func testProcessStakeEventRemovedOK(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(2)

	var f func(interface{}, bool)
	var evt interface{}
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	event := &types.StakeRemoved{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := stakev.ProcessStakeRemoved(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, true)

	stakev.ocv.EXPECT().GetStakingBridgeAddresses().Times(1)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	// we get a 0 balance, as the only event is a removed.
	balance, err := stakev.accs.GetAvailableBalance("somepubkey")
	assert.NoError(t, err)
	assert.Equal(t, 0, int(balance.Uint64()))
}

func testProcessStakeEventRemovedKO(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)

	var f func(interface{}, bool)
	var evt interface{}
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	event := &types.StakeRemoved{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := stakev.ProcessStakeRemoved(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, false)

	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	balance, err := stakev.accs.GetAvailableBalance("somepubkey")
	assert.EqualError(t, err, staking.ErrNoBalanceForParty.Error())
	assert.Equal(t, 0, int(balance.Uint64()))
}

func testProcessStakeEventMultiOK(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(2)
	stakev.ocv.EXPECT().GetStakingBridgeAddresses().AnyTimes()

	var f func(interface{}, bool)
	var evt interface{}
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := stakev.ProcessStakeDeposited(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, true)

	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	balance, err := stakev.accs.GetAvailableBalance("somepubkey")
	assert.NoError(t, err)
	assert.Equal(t, 1000, int(balance.Uint64()))

	// no we remove some stake

	stakev.tsvc.EXPECT().GetTimeNow().Times(2)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	f = nil
	evt = nil
	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(evtR validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			f = fn
			evt = evtR
			return nil
		})

	eventR := &types.StakeRemoved{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid2",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(500),
		BlockTime:       200000,
	}

	err = stakev.ProcessStakeRemoved(context.Background(), eventR)

	assert.NoError(t, err)
	assert.NotNil(t, f)

	// now we'll use the callback to set the event OK
	// no expectation there.
	f(evt, true)

	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)
	stakev.onTick(context.Background(), time.Unix(10, 0))

	balance, err = stakev.accs.GetAvailableBalance("somepubkey")
	assert.NoError(t, err)
	assert.Equal(t, 500, int(balance.Uint64()))
}

func testDuplicates(t *testing.T) {
	stakev := getStakeVerifierTest(t)
	defer stakev.ctrl.Finish()
	assert.NotNil(t, stakev)

	stakev.tsvc.EXPECT().GetTimeNow().Times(1)
	stakev.broker.EXPECT().Send(gomock.Any()).Times(1)

	stakev.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()
	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	// no error at first
	err := stakev.ProcessStakeDeposited(context.Background(), event)
	assert.NoError(t, err)
	// same event
	err = stakev.ProcessStakeDeposited(context.Background(), event)
	assert.EqualError(t, err, staking.ErrDuplicatedStakeDepositedEvent.Error())

	event2 := &types.StakeRemoved{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}
	// stake removed now
	err = stakev.ProcessStakeRemoved(context.Background(), event2)
	assert.EqualError(t, err, staking.ErrDuplicatedStakeRemovedEvent.Error())
}
