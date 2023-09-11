// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package epochtime

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/libs/proto"
)

type Broker interface {
	Send(e events.Event)
}

// Svc represents the Service managing epoch inside Vega.
type Svc struct {
	config Config

	length time.Duration
	epoch  types.Epoch

	listeners        []func(context.Context, types.Epoch) // for when the epoch state changes
	restoreListeners []func(context.Context, types.Epoch) // for when the epoch has been restored from a snapshot

	log *logging.Logger

	broker Broker

	readyToStartNewEpoch bool
	readyToEndEpoch      bool

	// Snapshot state
	state            *types.EpochState
	pl               types.Payload
	data             []byte
	currentTime      time.Time
	needsFastForward bool
}

// NewService instantiates a new epochtime service.
func NewService(l *logging.Logger, conf Config, broker Broker) *Svc {
	s := &Svc{
		config:               conf,
		log:                  l,
		broker:               broker,
		readyToStartNewEpoch: false,
		readyToEndEpoch:      false,
	}

	s.state = &types.EpochState{}
	s.pl = types.Payload{
		Data: &types.PayloadEpoch{
			EpochState: s.state,
		},
	}

	return s
}

// ReloadConf reload the configuration for the epochtime service.
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

// OnBlockEnd handles a callback from the abci when the block ends.
func (s *Svc) OnBlockEnd(ctx context.Context) {
	if s.readyToEndEpoch {
		s.readyToStartNewEpoch = true
		s.readyToEndEpoch = false
	}
}

// NB: An epoch is ended when the first block that exceeds the expiry of the current epoch ends. As onTick is called from onBlockStart - to make epoch continuous
// and avoid no man's epoch - once we get the first block past expiry we mark get ready to end the epoch. Once we get the on block end callback we're setting
// the flag to be ready to start a new block on the next onTick (i.e. preceding the beginning of the next block). Once we get the next block's on tick we close
// the epoch and notify on its end and start a new epoch (with incremented sequence) and notify about it.
func (s *Svc) OnTick(ctx context.Context, t time.Time) {
	if t.IsZero() {
		// We haven't got a block time yet, ignore
		return
	}

	if s.needsFastForward && t.Equal(s.currentTime) {
		s.log.Debug("onTick called with the same time again", logging.Time("tick-time", t))
		return
	}

	s.currentTime = t

	if s.needsFastForward {
		s.log.Info("fast forwarding epoch starts", logging.Uint64("from-epoch", s.epoch.Seq), logging.Time("at", t))
		s.needsFastForward = false
		s.fastForward(ctx)
		s.currentTime = t
		s.log.Info("fast forwarding epochs ended", logging.Uint64("current-epoch", s.epoch.Seq))
	}

	if s.epoch.StartTime.IsZero() {
		// First block so let's create our first epoch
		s.epoch.Seq = 0
		s.epoch.StartTime = t
		s.epoch.ExpireTime = t.Add(s.length) // current time + epoch length
		s.epoch.Action = vega.EpochAction_EPOCH_ACTION_START

		// Send out new epoch event
		s.notify(ctx, s.epoch)
		return
	}

	if s.readyToStartNewEpoch {
		// close previous epoch and send an event
		s.epoch.EndTime = t
		s.epoch.Action = vega.EpochAction_EPOCH_ACTION_END
		s.notify(ctx, s.epoch)

		// Move the epoch details forward
		s.epoch.Seq++
		s.readyToStartNewEpoch = false

		// Create a new epoch
		s.epoch.StartTime = t
		s.epoch.ExpireTime = t.Add(s.length) // now + epoch length
		s.epoch.EndTime = time.Time{}
		s.epoch.Action = vega.EpochAction_EPOCH_ACTION_START
		s.notify(ctx, s.epoch)
		return
	}

	// if the block time is past the expiry - this is the last block to go into the epoch - when the block ends we end the epoch and start a new one
	if s.epoch.ExpireTime.Before(t) {
		// Set the flag to tell us to end the epoch when the block ends
		s.readyToEndEpoch = true
		return
	}
}

func (*Svc) Name() types.CheckpointName {
	return types.EpochCheckpoint
}

func (s *Svc) Checkpoint() ([]byte, error) {
	return proto.Marshal(s.epoch.IntoProto())
}

func (s *Svc) Load(ctx context.Context, data []byte) error {
	pb := &eventspb.EpochEvent{}
	if err := proto.Unmarshal(data, pb); err != nil {
		return err
	}
	e := types.NewEpochFromProto(pb)
	s.epoch = *e

	// let the time end the epoch organically
	s.readyToStartNewEpoch = false
	s.readyToEndEpoch = false
	s.notify(ctx, s.epoch)
	s.needsFastForward = true
	return nil
}

// fastForward advances time and expires/starts any epoch that would have expired/started during the time period. It would trigger the epoch events naturally
// so will have a side effect of delegations getting promoted and rewards getting calculated and potentially paid.
func (s *Svc) fastForward(ctx context.Context) {
	tt := s.currentTime
	for s.epoch.ExpireTime.Before(tt) {
		s.OnBlockEnd(ctx)
		s.OnTick(ctx, s.epoch.ExpireTime.Add(1*time.Second))
	}
	s.OnTick(ctx, tt)
}

// NotifyOnEpoch allows other services to register 2 callback functions.
// The first will be called once we enter or leave a new epoch, and the second
// will be called when the epoch service has been restored from a snapshot.
func (s *Svc) NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch)) {
	s.listeners = append(s.listeners, f)
	s.restoreListeners = append(s.restoreListeners, r)
}

func (s *Svc) notify(ctx context.Context, e types.Epoch) {
	// Push this updated epoch message onto the event bus
	s.broker.Send(events.NewEpochEvent(ctx, &e))
	for _, f := range s.listeners {
		f(ctx, e)
	}
}

func (s *Svc) OnEpochLengthUpdate(ctx context.Context, l time.Duration) error {
	s.length = l
	// @TODO down the line, we ought to send an event signaling a change in epoch length
	return nil
}
