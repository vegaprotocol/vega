// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type RewardPayoutEvent interface {
	events.Event
	RewardPayoutEvent() eventspb.RewardPayoutEvent
}

type RewardStore interface {
	Add(context.Context, entities.Reward) error
}

type Reward struct {
	subscriber
	store RewardStore
}

func NewReward(store RewardStore) *Reward {
	t := &Reward{
		store: store,
	}
	return t
}

func (rs *Reward) Types() []events.Type {
	return []events.Type{events.RewardPayoutEvent}
}

func (rs *Reward) Push(ctx context.Context, evt events.Event) error {
	return rs.consume(ctx, evt.(RewardPayoutEvent))
}

func (rs *Reward) consume(ctx context.Context, event RewardPayoutEvent) error {
	protoRewardPayoutEvent := event.RewardPayoutEvent()
	reward, err := entities.RewardFromProto(protoRewardPayoutEvent, entities.TxHash(event.TxHash()), rs.vegaTime, event.Sequence())
	if err != nil {
		return errors.Wrap(err, "unable to parse reward")
	}

	if reward.VegaTime != rs.vegaTime {
		return errors.Errorf("reward timestamp does not match current VegaTime. Reward:%v",
			protoRewardPayoutEvent)
	}

	return errors.Wrap(rs.store.Add(ctx, reward), "error adding reward payout")
}
