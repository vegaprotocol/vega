package storage

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/vegatime"
	pb "code.vegaprotocol.io/protos/vega"
)

type epoch struct {
	seq                        string
	startTime                  int64
	expiryTime                 int64
	endTime                    int64
	nodeIDs                    []string
	delegationsPerNodePerParty map[string]map[string]pb.Delegation
}

type Epoch struct {
	Config

	mut          sync.RWMutex
	epochs       map[string]*epoch
	currentEpoch string

	nodeStore *Node
	minEpoch  *uint64
	log       *logging.Logger
}

func NewEpoch(log *logging.Logger, nodeStore *Node, c Config) *Epoch {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	e := &Epoch{
		nodeStore: nodeStore,
		epochs:    map[string]*epoch{},
		log:       log,
		Config:    c,
		minEpoch:  new(uint64),
	}

	*e.minEpoch = math.MaxUint64
	return e
}

// ReloadConf update the internal conf of the market
func (e *Epoch) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.Config = cfg
}

// AddEpoch adds new epoch and updates if epoch already exists
func (e *Epoch) AddEpoch(seq uint64, startTime, expiryTime, endTime int64) {
	epochSeq := strconv.FormatUint(seq, 10)

	e.mut.Lock()
	if epoch, ok := e.epochs[epochSeq]; ok {
		epoch.startTime = startTime
		epoch.expiryTime = expiryTime
		epoch.endTime = endTime
		e.epochs[epochSeq] = epoch

		e.currentEpoch = epochSeq

		e.mut.Unlock()
		return
	}
	e.mut.Unlock()

	e.addEpoch(epochSeq, startTime, expiryTime, endTime)

	e.mut.Lock()
	e.currentEpoch = epochSeq
	e.mut.Unlock()
}

func (e *Epoch) addEpoch(seq string, startTime, expiryTime, endTime int64) {
	e.mut.Lock()
	defer e.mut.Unlock()

	e.epochs[seq] = &epoch{
		seq:        seq,
		startTime:  startTime,
		expiryTime: expiryTime,
		endTime:    endTime,
		// @TODO this is hack.. Epoch store should consume
		// some event about node participation in epoch in future
		nodeIDs:                    e.nodeStore.GetAllIDs(),
		delegationsPerNodePerParty: map[string]map[string]pb.Delegation{},
	}

	clearOldEpochsDelegations(seq, e.minEpoch, func(epochSeq string) {
		if epoch, ok := e.epochs[epochSeq]; ok {
			epoch.delegationsPerNodePerParty = map[string]map[string]pb.Delegation{}
		}
	})

	// tell the node store we're in a new epoch
	e.nodeStore.AddEpoch(seq)
}

func (e *Epoch) AddDelegation(de pb.Delegation) {
	e.mut.RLock()
	_, ok := e.epochs[de.EpochSeq]
	e.mut.RUnlock()
	if !ok {
		e.addEpoch(de.EpochSeq, 0, 0, 0)
	}

	e.mut.Lock()
	epoch := e.epochs[de.EpochSeq]

	if _, ok := epoch.delegationsPerNodePerParty[de.NodeId]; !ok {
		epoch.delegationsPerNodePerParty[de.NodeId] = map[string]pb.Delegation{}
	}

	epoch.delegationsPerNodePerParty[de.NodeId][de.GetParty()] = de

	e.mut.Unlock()
}

func (e *Epoch) GetTotalNodesUptime() time.Duration {
	e.mut.RLock()
	defer e.mut.RUnlock()

	var uptime time.Duration
	for _, e := range e.epochs {
		// Filter out epochs that have not ended yet
		if e.endTime > 0 {
			uptime += vegatime.UnixNano(e.endTime).Sub(vegatime.UnixNano(e.startTime))
		}
	}
	return uptime
}

// GetEpochSeq returns current epoch sequence
func (e *Epoch) GetEpochSeq() string {
	e.mut.RLock()
	defer e.mut.RUnlock()

	return e.currentEpoch
}

// GetEpoch returns current epoch
func (e *Epoch) GetEpoch() (*pb.Epoch, error) {
	e.mut.RLock()
	defer e.mut.RUnlock()

	epoch, ok := e.epochs[e.currentEpoch]
	if !ok {
		return nil, fmt.Errorf("no epoch present")
	}

	pe, err := e.epochProtoFromInternal(epoch)
	if err != nil {
		return nil, fmt.Errorf("failed to convert epoch to proto: %w", err)
	}

	return pe, nil
}

func (e *Epoch) GetEpochByID(id string) (*pb.Epoch, error) {
	e.mut.RLock()
	defer e.mut.RUnlock()

	epoch, ok := e.epochs[id]
	if !ok {
		return nil, fmt.Errorf("epoch %s not found", id)
	}

	pe, err := e.epochProtoFromInternal(epoch)
	if err != nil {
		return nil, err
	}

	return pe, nil
}

func (e *Epoch) epochProtoFromInternal(ie *epoch) (*pb.Epoch, error) {
	seq, err := strconv.ParseUint(ie.seq, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uint from %s: %w", ie.seq, err)
	}

	validators := make([]*pb.Node, 0, len(ie.nodeIDs))
	for _, id := range ie.nodeIDs {
		node, err := e.nodeStore.GetByID(id, ie.seq)

		if errors.Is(err, ErrNodeDoesNotExistInThisEpoch) {
			continue // the node used to exist, was removed so we don't report it for this epoch
		}
		if err != nil {
			e.log.Error("Failed to get node by id", logging.Error(err))
			continue
		}

		validators = append(validators, node)
	}

	delegations := make([]*pb.Delegation, 0, len(ie.delegationsPerNodePerParty)*2)
	for _, delegationPerParty := range ie.delegationsPerNodePerParty {
		for _, d := range delegationPerParty {
			delegation := d
			delegations = append(delegations, &delegation)
		}
	}

	return &pb.Epoch{
		Seq: seq,
		Timestamps: &pb.EpochTimestamps{
			StartTime:  ie.startTime,
			ExpiryTime: ie.expiryTime,
			EndTime:    ie.endTime,
			// @TODO - add those later
			// FirstBlock: uint64,
			// LastBlock: uint64,
		},
		Validators:  validators,
		Delegations: delegations,
	}, nil
}
