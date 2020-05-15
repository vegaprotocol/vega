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
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/protobuf/proto"
)

var (
	ErrEvtAlreadyExist   = errors.New("event already exist")
	ErrMissingVegaWallet = errors.New("missing vega wallet§")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/evtforward TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/evtforward Commander
type Commander interface {
	Command(key nodewallet.Wallet, cmd blockchain.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/evtforward NodeWallet
type NodeWallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
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
	wallet      nodewallet.Wallet
}

type tsEvt struct {
	ts  time.Time // timestamp of the block when the event has been added
	evt *types.ChainEvent
}

type nodeHash struct {
	node string
	hash uint64
}

func New(log *logging.Logger, cfg Config, cmd Commander, self []byte, time TimeService, nwallet NodeWallet) (*EvtForwarder, error) {
	now, err := time.GetTimeNow()
	if err != nil {
		return nil, err
	}

	wallet, ok := nwallet.Get(nodewallet.Vega)
	if !ok {
		return nil, ErrMissingVegaWallet
	}

	evtf := &EvtForwarder{
		cfg:         cfg,
		log:         log,
		cmd:         cmd,
		nodes:       []nodeHash{},
		self:        string(self),
		currentTime: now,
		wallet:      wallet,
		ackedEvts:   map[string]*types.ChainEvent{},
		evts:        map[string]tsEvt{},
	}
	evtf.AddNodePubKey(self)
	time.NotifyOnTick(evtf.onTick)
	return evtf, nil
}

func (e *EvtForwarder) AddNodePubKey(key []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	h := e.hash(key)
	// first search if it exists
	i := sort.Search(len(e.nodes), func(i int) bool { return e.nodes[i].hash >= h })
	if i < len(e.nodes) && e.nodes[i].hash == h {
		// h already exists, return now
		return
	}

	// add the new node key
	e.nodes = append(e.nodes, nodeHash{string(key), h})
	// then sort
	// Sort by name, preserving original order
	sort.SliceStable(e.nodes, func(i, j int) bool { return e.nodes[i].hash < e.nodes[j].hash })
}

func (e *EvtForwarder) DelNodePubKey(key []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	h := e.hash(key)
	// first search if it exists
	i := sort.Search(len(e.nodes), func(i int) bool { return e.nodes[i].hash >= h })
	if i < len(e.nodes) && e.nodes[i].hash == h {
		// we found it, less remove now
		e.nodes = e.nodes[:i+copy(e.nodes[i:], e.nodes[i+1:])]
	}
}

func (n *EvtForwarder) getEvt(key string) (evt *types.ChainEvent, ok bool, acked bool) {
	evt, ok = n.ackedEvts[key]
	if ok {
		return evt, true, true
	}

	tsEvt, ok := n.evts[key]
	if ok {
		return tsEvt.evt, true, false
	}

	return nil, false, false
}

func (n *EvtForwarder) Forward(evt *types.ChainEvent) error {
	key := string(n.hash([]byte(evt.String())))
	_, ok, _ := n.getEvt(key)
	if ok {
		return ErrEvtAlreadyExist
	}

	n.evts[key] = tsEvt{ts: n.currentTime, evt: evt}
	if n.isSender(evt) {
		// we are selected to send the event, let's do it.
		return n.send(evt)
	}
	return nil
}

func (n *EvtForwarder) send(evt *types.ChainEvent) error {
	return n.cmd.Command(n.wallet, blockchain.ChainEventCommand, evt)
}

func (n *EvtForwarder) isSender(evt *types.ChainEvent) bool {
	s := fmt.Sprintf("%v%v", evt.String(), n.currentTime)
	h := n.hash([]byte(s))
	node := n.nodes[h%uint64(len(n.nodes))]
	return node.node == n.self
}

// Ack will return true if the event is newly acknowledge
// if the event already exist and was already acknowledge this will return false
func (n *EvtForwarder) Ack(evt *types.ChainEvent) bool {
	key := string(n.hash([]byte(evt.String())))
	_, ok, acked := n.getEvt(key)
	if ok && acked {
		// this was already acknowledged, nothing to be done, return false
		return false
	}
	if ok {
		// exists but was not acknowleded
		// we just remove it from the non-acked table
		delete(n.evts, string(key))
	}

	// now add it to the acknowledged evts
	n.ackedEvts[key] = evt
	return true
}

func (n *EvtForwarder) onTick(t time.Time) {
	n.currentTime = t

	// try to send all event that are not acknowledged at the moment
	for k, evt := range n.evts {
		// do we need to try to forward the event again?
		if evt.ts.Add(n.cfg.RetryRate.Duration).Before(t) {
			// set next retry
			n.evts[k] = tsEvt{ts: t, evt: evt.evt}
			if n.isSender(evt.evt) {
				// we are selected to send the event, let's do it.
				err := n.send(evt.evt)
				n.log.Error("unable to send event", logging.Error(err))
				continue
			}
		}
	}
}

func (n *EvtForwarder) hash(key []byte) uint64 {
	h := fnv.New64a()
	h.Write(key)
	return h.Sum64()
}
