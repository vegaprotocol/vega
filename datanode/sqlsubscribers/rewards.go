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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
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

func (rs *Reward) Name() string {
	return "Reward"
}
