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

package staking

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	pbevents "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Checkpoint struct {
	log            *logging.Logger
	accounting     *Accounting
	stakeVerifier  *StakeVerifier
	ethEventSource EthereumEventSource
}

func NewCheckpoint(
	log *logging.Logger,
	accounting *Accounting,
	stakeVerifier *StakeVerifier,
	ethEventSource EthereumEventSource,
) *Checkpoint {
	return &Checkpoint{
		log:            log,
		accounting:     accounting,
		stakeVerifier:  stakeVerifier,
		ethEventSource: ethEventSource,
	}
}

func (c *Checkpoint) Name() types.CheckpointName {
	return types.StakingCheckpoint
}

func (c *Checkpoint) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Staking{
		Accepted:      c.getAcceptedEvents(),
		LastBlockSeen: c.getLastBlockSeen(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Checkpoint) Load(ctx context.Context, data []byte) error {
	b := checkpoint.Staking{}
	if err := proto.Unmarshal(data, &b); err != nil {
		return err
	}

	// first we deduplicates those events, this is a fix for v0.50.4
	dedup := dedupEvents(b.Accepted)

	for _, evt := range dedup {
		stakeLinking := types.StakeLinkingFromProto(evt)

		// this will send all necessary events as well
		c.accounting.AddEvent(ctx, stakeLinking)
		// now add event to the hash mapping
		if !c.stakeVerifier.ensureNotDuplicate(stakeLinking.ID, stakeLinking.Hash()) {
			c.log.Panic("invalid checkpoint, duplicate event stored",
				logging.String("event-id", stakeLinking.ID),
			)
		}
	}

	stakeLinkingEvents := make([]events.Event, 0, len(b.Accepted))
	for _, acc := range c.accounting.hashableAccounts {
		for _, e := range acc.Events {
			stakeLinkingEvents = append(stakeLinkingEvents, events.NewStakeLinking(ctx, *e))
		}
	}

	c.accounting.broker.SendBatch(stakeLinkingEvents)

	// 0 is default value, we assume that it was then not set
	if b.LastBlockSeen != 0 {
		for _, addr := range c.stakeVerifier.ocv.GetStakingBridgeAddresses() {
			c.ethEventSource.UpdateContractBlock(addr, c.accounting.chainID, b.LastBlockSeen)
		}
	}

	return nil
}

func (c *Checkpoint) getAcceptedEvents() []*pbevents.StakeLinking {
	out := make([]*pbevents.StakeLinking, 0, len(c.accounting.hashableAccounts))

	for _, acc := range c.accounting.hashableAccounts {
		for _, evt := range acc.Events {
			out = append(out, evt.IntoProto())
		}
	}
	return out
}

// getLastBlockSeen will return the oldest pending transaction block
// from the stake verifier. By doing so we can restart listening to ethereum
// from the block of the oldest non accepted / verified stake linking event
// which should ensure that we haven't missed any.
func (c *Checkpoint) getLastBlockSeen() uint64 {
	if block := c.stakeVerifier.getLastBlockSeen(); block != 0 {
		return block
	}

	// now if block is still 0, we use the accounting stuff to find
	// the newest block verified then instead ...
	return c.accounting.getLastBlockSeen()
}

type key struct {
	txHash                string
	logIndex, blockHeight uint64
}

func dedupEvents(evts []*pbevents.StakeLinking) []*pbevents.StakeLinking {
	evtsM := map[key]*pbevents.StakeLinking{}
	for _, v := range evts {
		k := key{v.TxHash, v.LogIndex, v.BlockHeight}
		evt, ok := evtsM[k]
		if !ok {
			// we haven't seen this event, just add it and move on
			evtsM[k] = v
			continue
		}
		// we have seen this one already, let's save to earliest one only
		if evt.FinalizedAt > v.FinalizedAt {
			evtsM[k] = v
		}
	}

	// now we sort and return
	out := make([]*pbevents.StakeLinking, 0, len(evtsM))
	for _, v := range evtsM {
		out = append(out, v)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out
}
