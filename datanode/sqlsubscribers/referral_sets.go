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
)

type (
	ReferralSetCreatedEvent interface {
		events.Event
		GetProtoEvent() *eventspb.ReferralSetCreated
	}

	RefereeJoinedReferralSetEvent interface {
		events.Event
		GetProtoEvent() *eventspb.RefereeJoinedReferralSet
	}

	ReferralSetStatsUpdatedEvent interface {
		events.Event
		GetProtoEvent() *eventspb.ReferralSetStatsUpdated
	}

	ReferralSetsStore interface {
		AddReferralSet(ctx context.Context, referralSet *entities.ReferralSet) error
		RefereeJoinedReferralSet(ctx context.Context, referee *entities.ReferralSetReferee) error
		AddReferralSetStats(ctx context.Context, stats *entities.ReferralSetStats) error
	}

	ReferralSets struct {
		subscriber
		store ReferralSetsStore
	}
)

func NewReferralSets(store ReferralSetsStore) *ReferralSets {
	return &ReferralSets{
		store: store,
	}
}

func (rs *ReferralSets) Types() []events.Type {
	return []events.Type{
		events.ReferralSetCreatedEvent,
		events.RefereeJoinedReferralSetEvent,
		events.ReferralSetStatsUpdatedEvent,
	}
}

func (rs *ReferralSets) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ReferralSetCreatedEvent:
		return rs.consumeReferralSetCreatedEvent(ctx, e)
	case RefereeJoinedReferralSetEvent:
		return rs.consumeRefereeJoinedReferralSetEvent(ctx, e)
	case ReferralSetStatsUpdatedEvent:
		return rs.consumeReferralSetStatsUpdated(ctx, e)
	default:
		return nil
	}
}

func (rs *ReferralSets) consumeReferralSetCreatedEvent(ctx context.Context, e ReferralSetCreatedEvent) error {
	referralSet := entities.ReferralSetFromProto(e.GetProtoEvent(), rs.vegaTime)
	return rs.store.AddReferralSet(ctx, referralSet)
}

func (rs *ReferralSets) consumeRefereeJoinedReferralSetEvent(ctx context.Context, e RefereeJoinedReferralSetEvent) error {
	referralSetReferee := entities.ReferralSetRefereeFromProto(e.GetProtoEvent(), rs.vegaTime)
	return rs.store.RefereeJoinedReferralSet(ctx, referralSetReferee)
}

func (rs *ReferralSets) consumeReferralSetStatsUpdated(ctx context.Context, e ReferralSetStatsUpdatedEvent) error {
	stats, err := entities.ReferralSetStatsFromProto(e.GetProtoEvent(), rs.vegaTime)
	if err != nil {
		return err
	}
	return rs.store.AddReferralSetStats(ctx, stats)
}

func (rs *ReferralSets) Name() string {
	return "ReferralSets"
}
