package storage

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

type epoch struct {
	seq       string
	startTime int64
	endTime   int64
	nodeIDs   []string

	delegationsPerNodePerParty map[string]map[string]pb.Delegation
}

type Epoch struct {
	Config

	mut          sync.RWMutex
	epochs       map[string]epoch
	currentEpoch string

	nodeStore Node

	log *logging.Logger
}

func NewEpoch(log *logging.Logger, c Config) *Epoch {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Epoch{
		epochs: make(map[string]epoch),
		log:    log,
		Config: c,
	}
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

func (e *Epoch) AddEpoch(seq uint64, startTime int64, endTime int64) {
	e.mut.Lock()
	defer e.mut.Unlock()

	epochSeq := strconv.FormatUint(seq, 10)

	e.epochs[epochSeq] = epoch{
		seq:       epochSeq,
		startTime: startTime,
		endTime:   startTime,
		// @TODO this is hack.. Epoch store should consume
		// some event about node participation in epoch in future
		nodeIDs: e.nodeStore.GetAllIDs(),
	}

	e.currentEpoch = epochSeq
}

func (e *Epoch) AddDelegation(de pb.Delegation) {
	e.mut.Lock()
	defer e.mut.Unlock()

	epoch, ok := e.epochs[de.EpochSeq]
	if !ok {
		e.log.Error("Failed to update event for non existing epoch", logging.String("epoch", de.EpochSeq))
	}

	delegationsPerNodes, ok := epoch.delegationsPerNodePerParty[de.NodeId]
	if !ok {
		delegationsPerNodes = map[string]pb.Delegation{}
	}

	delegationsPerNodes[de.GetParty()] = de
}

func (e *Epoch) GetTotalNodesUptime() time.Duration {
	e.mut.RLock()
	defer e.mut.RUnlock()

	var uptime time.Duration
	for _, e := range e.epochs {
		uptime += time.Unix(0, e.endTime).Sub(time.Unix(0, e.startTime))
	}

	return uptime
}

func (e *Epoch) GetEpoch() (*pb.Epoch, error) {
	e.mut.RLock()
	defer e.mut.RUnlock()

	epoch := e.epochs[e.currentEpoch]

	pe, err := e.epochProtoFromInternal(epoch)
	if err != nil {
		return nil, err
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

func (e *Epoch) epochProtoFromInternal(ie epoch) (*pb.Epoch, error) {
	seq, err := strconv.ParseUint(ie.seq, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uint from %s: %w", ie.seq, err)
	}

	validators := make([]*pb.Node, 0, len(ie.nodeIDs))
	for _, id := range ie.nodeIDs {
		node, err := e.nodeStore.GetByID(id)
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
			StartTime: ie.startTime,
			EndTime:   ie.endTime,
			// @TODO - add those later
			// FirstBlock: uint64,
			// LastBlock: uint64,
		},
		Validators:  validators,
		Delegations: delegations,
	}, nil
}
