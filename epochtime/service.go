package epochtime

import (
	"context"
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

type Broker interface {
	Send(e events.Event)
}

// Svc represents the Service managing epoch inside Vega.
type Svc struct {
	config Config

	length time.Duration
	epoch  types.Epoch

	listeners []func(context.Context, types.Epoch)

	log *logging.Logger

	broker Broker

	readyToStartNewEpoch bool
	readyToEndEpoch      bool

	// Snapshot state
	data []byte
	hash []byte
}

type VegaTime interface {
	NotifyOnTick(func(context.Context, time.Time))
}

// NewService instantiates a new epochtime service
func NewService(l *logging.Logger, conf Config, vt VegaTime, broker Broker) *Svc {
	s := &Svc{config: conf,
		log:                  l,
		broker:               broker,
		readyToStartNewEpoch: false,
		readyToEndEpoch:      false,
	}

	// Subscribe to the vegatime onblocktime event
	vt.NotifyOnTick(s.onTick)

	return s
}

// ReloadConf reload the configuration for the epochtime service
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

//OnBlockEnd handles a callback from the abci when the block ends
func (s *Svc) OnBlockEnd(ctx context.Context) {
	if s.readyToEndEpoch {
		s.readyToStartNewEpoch = true
		s.readyToEndEpoch = false

		// take snapshot
		s.serialise()
	}
}

//NB: An epoch is ended when the first block that exceeds the expiry of the current epoch ends. As onTick is called from onBlockStart - to make epoch continuous
//and avoid no man's epoch - once we get the first block past expiry we mark get ready to end the epoch. Once we get the on block end callback we're setting
//the flag to be ready to start a new block on the next onTick (i.e. preceding the beginning of the next block). Once we get the next block's on tick we close
//the epoch and notify on its end and start a new epoch (with incremented sequence) and notify about it.
func (s *Svc) onTick(ctx context.Context, t time.Time) {

	if t.IsZero() {
		// We haven't got a block time yet, ignore
		return
	}

	if s.epoch.StartTime.IsZero() {
		// First block so let's create our first epoch
		s.epoch.Seq = 0
		s.epoch.StartTime = t
		s.epoch.ExpireTime = t.Add(s.length) // current time + epoch length
		s.epoch.Action = vega.EpochAction_EPOCH_ACTION_START

		// Send out new epoch event
		s.notify(ctx, s.epoch)

		// take snapshot
		s.serialise()
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

		// take snapshot
		s.serialise()
		return
	}

	// if the block time is past the expiry - this is the last block to go into the epoch - when the block ends we end the epoch and start a new one
	if s.epoch.ExpireTime.Before(t) {
		// Set the flag to tell us to end the epoch when the block ends
		s.readyToEndEpoch = true

		// take snapshot
		s.serialise()
		return
	}
}

func (*Svc) Name() types.CheckpointName {
	return types.EpochCheckpoint
}

func (s *Svc) Checkpoint() ([]byte, error) {
	return proto.Marshal(s.epoch.IntoProto())
}

func (s *Svc) Load(_ context.Context, data []byte) error {
	pb := &eventspb.EpochEvent{}
	if err := proto.Unmarshal(data, pb); err != nil {
		return err
	}
	e := types.NewEpochFromProto(pb)
	s.epoch = *e
	if e.Action == vega.EpochAction_EPOCH_ACTION_START {
		s.readyToStartNewEpoch = true
	}
	return nil
}

// NotifyOnEpoch allows other services to register a callback function
// which will be called once we enter a new epoch
func (s *Svc) NotifyOnEpoch(f func(context.Context, types.Epoch)) {
	s.listeners = append(s.listeners, f)
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
