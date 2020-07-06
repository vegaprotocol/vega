package evtforward

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
)

var (
	ErrEvtAlreadyExist   = errors.New("event already exist")
	ErrMissingVegaWallet = errors.New("missing vega wallet")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/evtforward TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/evtforward Commander
type Commander interface {
	Command(cmd blockchain.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/evtforward ValidatorTopology
type ValidatorTopology interface {
	SelfVegaPubKey() []byte
	Exists(key []byte) bool
	AllPubKeys() [][]byte
}

// EvtForwarder receive events from the blockchain queue
// and will try to send them to the vega chain.
// this will select a node in the network to forward the event
type EvtForwarder struct {
	log         *logging.Logger
	cfg         Config
	cmd         Commander
	nodes       []nodeHash
	self        string
	mu          sync.RWMutex
	ackedEvts   map[string]*types.ChainEvent
	evts        map[string]tsEvt
	currentTime time.Time
	top         ValidatorTopology
}

type tsEvt struct {
	ts  time.Time // timestamp of the block when the event has been added
	evt *types.ChainEvent
}

type nodeHash struct {
	node string
	hash uint64
}

func New(log *logging.Logger, cfg Config, cmd Commander, time TimeService, top ValidatorTopology) (*EvtForwarder, error) {
	now, err := time.GetTimeNow()
	if err != nil {
		return nil, err
	}

	evtf := &EvtForwarder{
		cfg:         cfg,
		log:         log,
		cmd:         cmd,
		nodes:       []nodeHash{},
		self:        string(top.SelfVegaPubKey()),
		currentTime: now,
		ackedEvts:   map[string]*types.ChainEvent{},
		evts:        map[string]tsEvt{},
		top:         top,
	}
	evtf.updateValidatorsList()
	time.NotifyOnTick(evtf.onTick)
	return evtf, nil
}

// Ack will return true if the event is newly acknowledge
// if the event already exist and was already acknowledge this will return false
func (e *EvtForwarder) Ack(evt *types.ChainEvent) bool {
	key := string(e.hash([]byte(evt.String())))
	_, ok, acked := e.getEvt(key)
	if ok && acked {
		// this was already acknowledged, nothing to be done, return false
		return false
	}
	if ok {
		// exists but was not acknowleded
		// we just remove it from the non-acked table
		delete(e.evts, string(key))
	}

	// now add it to the acknowledged evts
	e.ackedEvts[key] = evt
	return true
}

func (e *EvtForwarder) Forward(evt *types.ChainEvent) error {
	key := string(e.hash([]byte(evt.String())))
	_, ok, _ := e.getEvt(key)
	if ok {
		return ErrEvtAlreadyExist
	}

	e.evts[key] = tsEvt{ts: e.currentTime, evt: evt}
	if e.isSender(evt) {
		// we are selected to send the event, let's do it.
		return e.send(evt)
	}
	return nil
}

func (e *EvtForwarder) updateValidatorsList() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// reset slice
	// preemptive alloc, we can expect to have most likely
	// as much validator
	e.nodes = make([]nodeHash, 0, len(e.nodes))
	// get all keys
	for _, key := range e.top.AllPubKeys() {
		h := e.hash(key)
		e.nodes = append(e.nodes, nodeHash{string(key), h})
	}
	sort.SliceStable(e.nodes, func(i, j int) bool { return e.nodes[i].hash < e.nodes[j].hash })
}

func (e *EvtForwarder) getEvt(key string) (evt *types.ChainEvent, ok bool, acked bool) {
	if evt, ok = e.ackedEvts[key]; ok {
		return evt, true, true
	}

	if tsEvt, ok := e.evts[key]; ok {
		return tsEvt.evt, true, false
	}

	return nil, false, false
}

func (e *EvtForwarder) send(evt *types.ChainEvent) error {
	return e.cmd.Command(blockchain.ChainEventCommand, evt)
}

func (e *EvtForwarder) isSender(evt *types.ChainEvent) bool {
	s := fmt.Sprintf("%v%v", evt.String(), e.currentTime.Unix())
	h := e.hash([]byte(s))
	e.mu.RLock()
	node := e.nodes[h%uint64(len(e.nodes))]
	e.mu.RUnlock()
	return node.node == e.self
}

func (e *EvtForwarder) onTick(t time.Time) {
	e.currentTime = t

	// get an updated list of validators from the topology
	e.updateValidatorsList()

	e.mu.RLock()
	retryRate := e.cfg.RetryRate.Duration
	e.mu.RUnlock()

	// try to send all event that are not acknowledged at the moment
	for k, evt := range e.evts {
		// do we need to try to forward the event again?
		if evt.ts.Add(retryRate).Before(t) {
			// set next retry
			e.evts[k] = tsEvt{ts: t, evt: evt.evt}
			if e.isSender(evt.evt) {
				// we are selected to send the event, let's do it.
				if err := e.send(evt.evt); err != nil {
					e.log.Error("unable to send event", logging.Error(err))
				}
			}
		}
	}
}

func (e *EvtForwarder) hash(key []byte) uint64 {
	h := fnv.New64a()
	h.Write(key)
	return h.Sum64()
}
