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

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
)

var (
	// ErrEvtAlreadyExist we have already handled this event
	ErrEvtAlreadyExist = errors.New("event already exist")
	// ErrMissingVegaWallet we cannot find the vega wallet
	ErrMissingVegaWallet = errors.New("missing vega wallet")
	// ErrPubKeyNotAllowlisted this pubkey is not part of the allowlist
	ErrPubKeyNotAllowlisted = errors.New("pubkey not allowlisted")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/evtforward TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(context.Context, time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/evtforward Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message) error
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
	log  *logging.Logger
	cfg  Config
	cmd  Commander
	self string

	evtsmu    sync.Mutex
	ackedEvts map[string]*types.ChainEvent
	evts      map[string]tsEvt

	mu               sync.RWMutex
	bcQueueAllowlist atomic.Value // this is actually an map[string]struct{}
	currentTime      time.Time
	nodes            []nodeHash

	top ValidatorTopology
}

type tsEvt struct {
	ts  time.Time // timestamp of the block when the event has been added
	evt *types.ChainEvent
}

type nodeHash struct {
	node string
	hash uint64
}

// New creates a new instance of the event forwarder
func New(log *logging.Logger, cfg Config, cmd Commander, time TimeService, top ValidatorTopology) (*EvtForwarder, error) {
	now, err := time.GetTimeNow()
	if err != nil {
		return nil, err
	}

	var allowlist atomic.Value
	allowlist.Store(buildAllowlist(cfg))
	evtf := &EvtForwarder{
		cfg:              cfg,
		log:              log,
		cmd:              cmd,
		nodes:            []nodeHash{},
		self:             string(top.SelfVegaPubKey()),
		currentTime:      now,
		ackedEvts:        map[string]*types.ChainEvent{},
		evts:             map[string]tsEvt{},
		top:              top,
		bcQueueAllowlist: allowlist,
	}
	evtf.updateValidatorsList()
	time.NotifyOnTick(evtf.onTick)
	return evtf, nil
}

func buildAllowlist(cfg Config) map[string]struct{} {
	allowlist := make(map[string]struct{}, len(cfg.BlockchainQueueAllowlist))
	for _, v := range cfg.BlockchainQueueAllowlist {
		allowlist[v] = struct{}{}
	}
	return allowlist
}

// ReloadConf updates the internal configuration of the collateral engine
func (e *EvtForwarder) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfg = cfg
	// update the allowlist
	e.log.Info("evtforward allowlist updated",
		logging.Reflect("list", cfg.BlockchainQueueAllowlist))
	e.bcQueueAllowlist.Store(buildAllowlist(cfg))
}

// Ack will return true if the event is newly acknowledge
// if the event already exist and was already acknowledge this will return false
func (e *EvtForwarder) Ack(evt *types.ChainEvent) bool {
	var res string = "ok"
	defer func() {
		metrics.EvtForwardInc("ack", res)
	}()

	e.evtsmu.Lock()
	defer e.evtsmu.Unlock()
	key := string(crypto.Hash([]byte(evt.String())))
	_, ok, acked := e.getEvt(key)
	if ok && acked {
		res = "alreadyacked"
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

func (e *EvtForwarder) isAllowlisted(pubkey string) bool {
	allowlist := e.bcQueueAllowlist.Load().(map[string]struct{})
	_, ok := allowlist[pubkey]
	return ok
}

// Forward will forward an ChainEvent to the tendermint network
// we expect the pubkey to be an ed25519 pubkey hex encoded
func (e *EvtForwarder) Forward(ctx context.Context, evt *types.ChainEvent, pubkey string) error {
	var res string = "ok"
	defer func() {
		metrics.EvtForwardInc("forward", res)
	}()
	// check if the sender of the event is whitelisted
	if !e.isAllowlisted(pubkey) {
		res = "pubkeynotallowed"
		return ErrPubKeyNotAllowlisted
	}

	e.evtsmu.Lock()
	defer e.evtsmu.Unlock()

	key := string(crypto.Hash([]byte(evt.String())))
	_, ok, _ := e.getEvt(key)
	if ok {
		res = "dupevt"
		return ErrEvtAlreadyExist
	}

	e.evts[key] = tsEvt{ts: e.currentTime, evt: evt}
	if e.isSender(evt) {
		// we are selected to send the event, let's do it.
		return e.send(ctx, evt)
	}
	return nil
}

func (e *EvtForwarder) updateValidatorsList() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.self = string(e.top.SelfVegaPubKey())
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

func (e *EvtForwarder) send(ctx context.Context, evt *types.ChainEvent) error {
	return e.cmd.Command(ctx, txn.ChainEventCommand, evt)
}

func (e *EvtForwarder) isSender(evt *types.ChainEvent) bool {
	s := fmt.Sprintf("%v%v", evt.String(), e.currentTime.Unix())
	h := e.hash([]byte(s))
	e.mu.RLock()
	if len(e.nodes) <= 0 {
		e.mu.RUnlock()
		return false
	}
	node := e.nodes[h%uint64(len(e.nodes))]
	e.mu.RUnlock()
	return node.node == e.self
}

func (e *EvtForwarder) onTick(ctx context.Context, t time.Time) {
	e.currentTime = t

	// get an updated list of validators from the topology
	e.updateValidatorsList()

	e.mu.RLock()
	retryRate := e.cfg.RetryRate.Duration
	e.mu.RUnlock()

	e.evtsmu.Lock()
	defer e.evtsmu.Unlock()

	// try to send all event that are not acknowledged at the moment
	for k, evt := range e.evts {
		// do we need to try to forward the event again?
		if evt.ts.Add(retryRate).Before(t) {
			// set next retry
			e.evts[k] = tsEvt{ts: t, evt: evt.evt}
			if e.isSender(evt.evt) {
				// we are selected to send the event, let's do it.
				if err := e.send(ctx, evt.evt); err != nil {
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
