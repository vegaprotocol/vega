package staking

import (
	"context"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	events "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
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

	for _, evt := range b.Accepted {
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

	// 0 is default value, we assume that it was then not set
	if b.LastBlockSeen != 0 {
		c.ethEventSource.UpdateStakingStartingBlock(b.LastBlockSeen)
	}

	return nil
}

func (c *Checkpoint) getAcceptedEvents() []*events.StakeLinking {
	out := make([]*events.StakeLinking, 0, len(c.accounting.hashableAccounts))

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
	var block uint64
	for _, p := range c.stakeVerifier.pendingSDs {
		if block == 0 {
			block = p.BlockNumber
			continue
		}

		if p.BlockNumber < block {
			block = p.BlockNumber
		}
	}

	for _, p := range c.stakeVerifier.pendingSRs {
		if block == 0 {
			block = p.BlockNumber
			continue
		}

		if p.BlockNumber < block {
			block = p.BlockNumber
		}
	}

	// now if block is still 0, we use the accounting stuff to find
	// the newest block verified then instead ...
	if block == 0 {
		for _, acc := range c.accounting.hashableAccounts {
			if len(acc.Events) == 0 {
				continue
			}
			height := acc.Events[len(acc.Events)-1].BlockHeight
			if block < height {
				block = height
			}
		}
	}

	return block
}
