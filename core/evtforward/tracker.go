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

package evtforward

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethverifier Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

type EVMEngine interface {
	VerifyHeartbeat(context context.Context, height uint64, chainID string, contract string, blockTime uint64) error
	UpdateStartingBlock(string, uint64)
}

type bridge struct {
	// map from contract address -> block height of the last seen address
	contractAddresses map[string]uint64
	engine            EVMEngine
}

type Tracker struct {
	log         *logging.Logger
	witness     Witness
	timeService TimeService

	// map from chain-id -> bridge contract information
	bridges map[string]*bridge

	pendingHeartbeats   []*pendingHeartbeat
	finalizedHeartbeats []*pendingHeartbeat
}

type pendingHeartbeat struct {
	height          uint64
	blockTime       uint64
	chainID         string
	contractAddress string
	check           func(ctx context.Context) error
}

func (p pendingHeartbeat) GetID() string {
	h := strconv.FormatUint(p.height, 10)
	t := strconv.FormatUint(p.blockTime, 10)
	bytes := []byte(h + t + p.chainID + p.contractAddress)
	return hex.EncodeToString(vgcrypto.Hash(bytes))
}

func (p pendingHeartbeat) GetChainID() string {
	return p.chainID
}

func (p pendingHeartbeat) GetType() types.NodeVoteType {
	return types.NodeVoteTypeEthereumHeartbeat
}

func (p *pendingHeartbeat) Check(ctx context.Context) error { return p.check(ctx) }

func NewTracker(log *logging.Logger, witness Witness, ts TimeService) *Tracker {
	return &Tracker{
		log:         log,
		bridges:     map[string]*bridge{},
		timeService: ts,
		witness:     witness,
	}
}

func (t *Tracker) RegisterForwarder(fwd EVMEngine, chainID string, addresses ...string) {
	contracts := map[string]uint64{}
	for _, address := range addresses {
		contracts[address] = 0
	}

	t.bridges[chainID] = &bridge{
		engine:            fwd,
		contractAddresses: contracts,
	}
}

func (t *Tracker) ProcessHeartbeat(address, chainID string, height uint64, blockTime uint64) error {
	// check if the heartbeat is too old, we don't care if we've already seen something at a higher block height

	bridge, ok := t.bridges[chainID]
	if !ok {
		return fmt.Errorf("bridge does not exist for chain-id: %s", chainID)
	}

	last, ok := bridge.contractAddresses[address]
	if !ok {
		return fmt.Errorf("contract address does not correspond to a bridge contract: %s", address)
	}

	if height <= last {
		return fmt.Errorf("heartbeat is stale")
	}

	fwd := bridge.engine

	// continue with verification
	pending := &pendingHeartbeat{
		height:          height,
		blockTime:       blockTime,
		chainID:         chainID,
		contractAddress: address,
		check:           func(ctx context.Context) error { return fwd.VerifyHeartbeat(ctx, height, chainID, address, blockTime) },
	}

	t.pendingHeartbeats = append(t.pendingHeartbeats, pending)

	t.log.Info("bridge heartbeat received, starting validation",
		logging.String("chain-id", chainID),
		logging.String("contract-address", address),
		logging.Uint64("height", height),
	)

	err := t.witness.StartCheck(
		pending, t.onVerified, t.timeService.GetTimeNow().Add(30*time.Minute))
	if err != nil {
		t.log.Error("could not start witness routine", logging.String("id", pending.GetID()))
		t.removeHeartbeat(pending.GetID())
	}
	return nil
}

func (t *Tracker) removeHeartbeat(id string) error {
	for i, v := range t.pendingHeartbeats {
		if v.GetID() == id {
			t.pendingHeartbeats = t.pendingHeartbeats[:i+copy(t.pendingHeartbeats[i:], t.pendingHeartbeats[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("could not remove heartbeat: %s", id)
}

func (t *Tracker) onVerified(event interface{}, ok bool) {
	pv, isHeartbeat := event.(*pendingHeartbeat)
	if !isHeartbeat {
		t.log.Errorf("expected pending heartbeat: %T", event)
		return
	}

	if err := t.removeHeartbeat(pv.GetID()); err != nil {
		t.log.Error("could not remove pending heartbeat", logging.Error(err))
	}

	if ok {
		t.finalizedHeartbeats = append(t.finalizedHeartbeats, pv)
	}
}

// UpdateContractBlock if an external engine has processed a chain-event for the contract at this address
// then the last-seen block for the contract is updated.
func (t *Tracker) UpdateContractBlock(address, chainID string, height uint64) {
	bridge, ok := t.bridges[chainID]
	if !ok {
		return
	}

	lastSeen, ok := bridge.contractAddresses[address]
	if !ok || lastSeen < height {
		bridge.contractAddresses[address] = height
	}
}

func (t *Tracker) OnTick(ctx context.Context, tt time.Time) {
	for _, heartbeat := range t.finalizedHeartbeats {
		bridge := t.bridges[heartbeat.chainID]

		lastSeen, ok := bridge.contractAddresses[heartbeat.contractAddress]
		if !ok || lastSeen < heartbeat.height {
			bridge.contractAddresses[heartbeat.contractAddress] = heartbeat.height
		}
	}
	t.finalizedHeartbeats = nil
}

func (t *Tracker) serialise() ([]byte, error) {
	pending := make([]*snapshotpb.EVMFwdPendingHeartbeat, 0, len(t.pendingHeartbeats))
	for _, p := range t.pendingHeartbeats {
		pending = append(pending,
			&snapshotpb.EVMFwdPendingHeartbeat{
				BlockHeight:     p.height,
				BlockTime:       p.blockTime,
				ContractAddress: p.contractAddress,
				ChainId:         p.chainID,
			},
		)
	}

	lastSeen := make([]*snapshotpb.EVMFwdLastSeen, 0, len(t.bridges))

	chainIDs := maps.Keys(t.bridges)
	sort.Strings(chainIDs)

	for _, cid := range chainIDs {
		seenBlocks := t.bridges[cid].contractAddresses
		contracts := maps.Keys(t.bridges[cid].contractAddresses)
		sort.Strings(contracts)
		for _, addr := range contracts {
			t.log.Info("serialising last-seen contract block",
				logging.String("chain-id", cid),
				logging.String("contract-address", addr),
				logging.Uint64("height", seenBlocks[addr]),
			)
			lastSeen = append(lastSeen,
				&snapshotpb.EVMFwdLastSeen{
					ChainId:         cid,
					ContractAddress: addr,
					BlockHeight:     seenBlocks[addr],
				},
			)
		}
	}

	pl := types.Payload{
		Data: &types.PayloadEVMFwdHeartbeats{
			EVMFwdHeartbeats: &snapshotpb.EVMFwdHeartbeats{
				PendingHeartbeats: pending,
				LastSeen:          lastSeen,
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

func (t *Tracker) restorePendingHeartbeats(_ context.Context, heartbeats []*snapshotpb.EVMFwdPendingHeartbeat) {
	t.pendingHeartbeats = make([]*pendingHeartbeat, 0, len(heartbeats))

	for _, hb := range heartbeats {
		bridge, ok := t.bridges[hb.ChainId]
		if !ok {
			t.log.Panic("cannot restore pending heartbeat, bridge not registered", logging.String("chain-id", hb.ChainId))
		}

		pending := &pendingHeartbeat{
			height:          hb.BlockHeight,
			blockTime:       hb.BlockTime,
			chainID:         hb.ChainId,
			contractAddress: hb.ContractAddress,
			check: func(ctx context.Context) error {
				return bridge.engine.VerifyHeartbeat(ctx, hb.BlockHeight, hb.ChainId, hb.ContractAddress, hb.BlockTime)
			},
		}

		t.pendingHeartbeats = append(t.pendingHeartbeats, pending)

		if err := t.witness.RestoreResource(pending, t.onVerified); err != nil {
			t.log.Panic("unable to restore pending heartbeat resource", logging.String("ID", pending.GetID()), logging.Error(err))
		}
	}
}

func (t *Tracker) restoreLastSeen(
	_ context.Context,
	lastSeen []*snapshotpb.EVMFwdLastSeen,
) {
	for _, ls := range lastSeen {
		bridge, ok := t.bridges[ls.ChainId]
		if !ok {
			t.log.Panic("cannot restore last seen block, bridge not registered", logging.String("chain-id", ls.ChainId))
		}
		bridge.contractAddresses[ls.ContractAddress] = ls.BlockHeight
		t.log.Info("restored last seen block height",
			logging.String("chain-id", ls.ChainId),
			logging.String("contract-address", ls.ContractAddress),
			logging.Uint64("last-seen", ls.BlockHeight),
		)
	}
}

func (t *Tracker) Namespace() types.SnapshotNamespace {
	return types.EVMHeartbeatSnapshot
}

func (t *Tracker) Keys() []string {
	return []string{"all"}
}

func (t *Tracker) Stopped() bool {
	return false
}

func (t *Tracker) GetState(_ string) ([]byte, []types.StateProvider, error) {
	data, err := t.serialise()
	return data, nil, err
}

func (t *Tracker) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadEVMFwdHeartbeats:
		t.restorePendingHeartbeats(ctx, pl.EVMFwdHeartbeats.PendingHeartbeats)
		t.restoreLastSeen(ctx, pl.EVMFwdHeartbeats.LastSeen)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *Tracker) OnStateLoaded(_ context.Context) error {
	for _, bridge := range t.bridges {
		for address, lastSeen := range bridge.contractAddresses {
			t.log.Info("updating starting block after restore",
				logging.String("address", address),
				logging.Uint64("last-seen", lastSeen),
			)
			bridge.engine.UpdateStartingBlock(address, lastSeen)
		}
	}
	return nil
}
