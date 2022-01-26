package evtforward

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
)

var (
	// ErrEvtAlreadyExist we have already handled this event.
	ErrEvtAlreadyExist = errors.New("event already exist")
	// ErrPubKeyNotAllowlisted this pubkey is not part of the allowlist.
	ErrPubKeyNotAllowlisted = errors.New("pubkey not allowlisted")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/evtforward TimeService
type TimeService interface {
	GetTimeNow() time.Time
	NotifyOnTick(f func(context.Context, time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/evtforward Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error))
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/evtforward ValidatorTopology
type ValidatorTopology interface {
	SelfNodeID() string
	AllNodeIDs() []string
}

// Forwarder receive events from the blockchain queue
// and will try to send them to the vega chain.
// this will select a node in the network to forward the event.
type Forwarder struct {
	log  *logging.Logger
	cfg  Config
	cmd  Commander
	self string

	evtsmu    sync.Mutex
	ackedEvts map[string]*commandspb.ChainEvent
	evts      map[string]tsEvt

	mu               sync.RWMutex
	bcQueueAllowlist atomic.Value // this is actually an map[string]struct{}
	currentTime      time.Time
	nodes            []string

	top  ValidatorTopology
	efss *efSnapshotState
}

type tsEvt struct {
	ts  time.Time // timestamp of the block when the event has been added
	evt *commandspb.ChainEvent
}

// New creates a new instance of the event forwarder.
func New(log *logging.Logger, cfg Config, cmd Commander, time TimeService, top ValidatorTopology) *Forwarder {
	log = log.Named(forwarderLogger)
	log.SetLevel(cfg.Level.Get())
	var allowlist atomic.Value
	allowlist.Store(buildAllowlist(cfg))
	forwarder := &Forwarder{
		cfg:              cfg,
		log:              log,
		cmd:              cmd,
		nodes:            []string{},
		self:             top.SelfNodeID(),
		currentTime:      time.GetTimeNow(),
		ackedEvts:        map[string]*commandspb.ChainEvent{},
		evts:             map[string]tsEvt{},
		top:              top,
		bcQueueAllowlist: allowlist,
		efss: &efSnapshotState{
			changed:    true,
			hash:       []byte{},
			serialised: []byte{},
		},
	}
	forwarder.updateValidatorsList()
	time.NotifyOnTick(forwarder.onTick)
	return forwarder
}

func buildAllowlist(cfg Config) map[string]struct{} {
	allowlist := make(map[string]struct{}, len(cfg.BlockchainQueueAllowlist))
	for _, v := range cfg.BlockchainQueueAllowlist {
		allowlist[v] = struct{}{}
	}
	return allowlist
}

// ReloadConf updates the internal configuration of the Event Forwarder engine.
func (f *Forwarder) ReloadConf(cfg Config) {
	f.log.Info("reloading configuration")
	if f.log.GetLevel() != cfg.Level.Get() {
		f.log.Info("updating log level",
			logging.String("old", f.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		f.log.SetLevel(cfg.Level.Get())
	}

	f.cfg = cfg
	// update the allowlist
	f.log.Info("evtforward allowlist updated",
		logging.Reflect("list", cfg.BlockchainQueueAllowlist))
	f.bcQueueAllowlist.Store(buildAllowlist(cfg))
}

// Ack will return true if the event is newly acknowledged.
// If the event already exist and was already acknowledged, this will return
// false.
func (f *Forwarder) Ack(evt *commandspb.ChainEvent) bool {
	res := "ok"
	defer func() {
		metrics.EvtForwardInc("ack", res)
	}()

	f.evtsmu.Lock()
	defer f.evtsmu.Unlock()

	key, err := f.getEvtKey(evt)
	if err != nil {
		f.log.Error("could not get event key", logging.Error(err))
		return false
	}
	_, ok, acked := f.getEvt(key)
	if ok && acked {
		f.log.Error("event already acknowledged",
			logging.String("evt", evt.String()),
		)
		res = "alreadyacked"
		// this was already acknowledged, nothing to be done, return false
		return false
	}
	if ok {
		// exists but was not acknowledged
		// we just remove it from the non-acked table
		delete(f.evts, key)
	}

	// now add it to the acknowledged evts
	f.ackedEvts[key] = evt
	f.efss.changed = true
	f.log.Info("new event acknowledged", logging.String("event", evt.String()))
	return true
}

func (f *Forwarder) isAllowlisted(pubkey string) bool {
	allowlist := f.bcQueueAllowlist.Load().(map[string]struct{})
	_, ok := allowlist[pubkey]
	return ok
}

// Forward will forward a ChainEvent to the tendermint network.
// We expect the pubkey to be an ed25519, hex encoded, key.
func (f *Forwarder) Forward(ctx context.Context, evt *commandspb.ChainEvent, pubkey string) error {
	res := "ok"
	defer func() {
		metrics.EvtForwardInc("forward", res)
	}()

	if f.log.IsDebug() {
		f.log.Debug("new event received to be forwarded",
			logging.String("event", evt.String()),
		)
	}

	// check if the sender of the event is whitelisted
	if !f.isAllowlisted(pubkey) {
		res = "pubkeynotallowed"
		return ErrPubKeyNotAllowlisted
	}

	f.evtsmu.Lock()
	defer f.evtsmu.Unlock()

	key, err := f.getEvtKey(evt)
	if err != nil {
		return err
	}
	_, ok, ack := f.getEvt(key)
	if ok {
		f.log.Error("event already processed",
			logging.String("evt", evt.String()),
			logging.Bool("acknowledged", ack),
		)
		res = "dupevt"
		return ErrEvtAlreadyExist
	}

	f.evts[key] = tsEvt{ts: f.currentTime, evt: evt}
	if f.isSender(evt) {
		// we are selected to send the event, let's do it.
		f.send(ctx, evt)
	}
	return nil
}

// ForwardFromSelf will forward event seen by the node itself, not from
// an external service like the eef for example.
func (f *Forwarder) ForwardFromSelf(evt *commandspb.ChainEvent) {
	f.evtsmu.Lock()
	defer f.evtsmu.Unlock()

	key, err := f.getEvtKey(evt)
	if err != nil {
		// no way this event would be badly formatted
		// it is sent by the node, a badly formatted event
		// would mean a code bug
		f.log.Panic("invalid event to be forwarded",
			logging.String("event", evt.String()),
			logging.Error(err),
		)
	}

	_, ok, ack := f.getEvt(key)
	if ok {
		f.log.Error("event already processed",
			logging.String("evt", evt.String()),
			logging.Bool("acknowledged", ack),
		)
	}

	f.evts[key] = tsEvt{ts: f.currentTime, evt: evt}
}

func (f *Forwarder) updateValidatorsList() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.self = f.top.SelfNodeID()
	f.nodes = f.top.AllNodeIDs()
	sort.SliceStable(f.nodes, func(i, j int) bool {
		return f.nodes[i] < f.nodes[j]
	})
}

// getEvt assumes the lock is acquired before being called.
func (f *Forwarder) getEvt(key string) (evt *commandspb.ChainEvent, ok bool, acked bool) {
	if evt, ok = f.ackedEvts[key]; ok {
		return evt, true, true
	}

	if tsEvt, ok := f.evts[key]; ok {
		return tsEvt.evt, true, false
	}

	return nil, false, false
}

func (f *Forwarder) send(ctx context.Context, evt *commandspb.ChainEvent) {
	if f.log.IsDebug() {
		f.log.Debug("trying to send event",
			logging.String("event", evt.String()),
		)
	}

	// error doesn't matter here
	f.cmd.CommandSync(ctx, txn.ChainEventCommand, evt, func(err error) {
		if err != nil {
			f.log.Error("could not send command", logging.String("tx-id", evt.TxId), logging.Error(err))
		}
	})
}

func (f *Forwarder) isSender(evt *commandspb.ChainEvent) bool {
	key, err := f.makeEvtHashKey(evt)
	if err != nil {
		f.log.Error("could not marshal event", logging.Error(err))
		return false
	}
	h := f.hash(key) + uint64(f.currentTime.Unix())

	f.mu.RLock()
	if len(f.nodes) <= 0 {
		f.mu.RUnlock()
		return false
	}
	node := f.nodes[h%uint64(len(f.nodes))]
	f.mu.RUnlock()

	return node == f.self
}

func (f *Forwarder) onTick(ctx context.Context, t time.Time) {
	f.currentTime = t

	// get an updated list of validators from the topology
	f.updateValidatorsList()

	f.mu.RLock()
	retryRate := f.cfg.RetryRate.Duration
	f.mu.RUnlock()

	f.evtsmu.Lock()
	defer f.evtsmu.Unlock()

	// try to send all event that are not acknowledged at the moment
	for k, evt := range f.evts {
		// do we need to try to forward the event again?
		if evt.ts.Add(retryRate).Before(t) {
			// set next retry
			f.evts[k] = tsEvt{ts: t, evt: evt.evt}
			if f.isSender(evt.evt) {
				// we are selected to send the event, let's do it.
				f.send(ctx, evt.evt)
			}
		}
	}
}

func (f *Forwarder) getEvtKey(evt *commandspb.ChainEvent) (string, error) {
	mevt, err := f.marshalEvt(evt)
	if err != nil {
		return "", fmt.Errorf("invalid event: %w", err)
	}

	return string(crypto.Hash(mevt)), nil
}

func (f *Forwarder) marshalEvt(evt *commandspb.ChainEvent) ([]byte, error) {
	pbuf := proto.Buffer{}
	pbuf.Reset()
	pbuf.SetDeterministic(true)
	if err := pbuf.Marshal(evt); err != nil {
		return nil, err
	}
	return pbuf.Bytes(), nil
}

func (f *Forwarder) makeEvtHashKey(evt *commandspb.ChainEvent) ([]byte, error) {
	// deterministic marshal of the event
	pbuf, err := f.marshalEvt(evt)
	if err != nil {
		return nil, err
	}
	return pbuf, nil
}

func (f *Forwarder) hash(key []byte) uint64 {
	h := fnv.New64a()
	h.Write(key)

	return h.Sum64()
}
