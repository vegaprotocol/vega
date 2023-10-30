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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralProgramStartedEvent interface {
		events.Event
		GetReferralProgramStarted() *eventspb.ReferralProgramStarted
	}

	ReferralProgramUpdatedEvent interface {
		events.Event
		GetReferralProgramUpdated() *eventspb.ReferralProgramUpdated
	}

	ReferralProgramEndedEvent interface {
		events.Event
		GetReferralProgramEnded() *eventspb.ReferralProgramEnded
	}

	ReferralStore interface {
		AddReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error
		UpdateReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error
		EndReferralProgram(ctx context.Context, version uint64, endedAt time.Time, vegaTime time.Time, seqNum uint64) error
	}

	ReferralProgram struct {
		subscriber
		store ReferralStore
	}
)

func NewReferralProgram(store ReferralStore) *ReferralProgram {
	return &ReferralProgram{
		store: store,
	}
}

func (rp *ReferralProgram) Types() []events.Type {
	return []events.Type{
		events.ReferralProgramStartedEvent,
		events.ReferralProgramUpdatedEvent,
		events.ReferralProgramEndedEvent,
	}
}

func (rp *ReferralProgram) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ReferralProgramStartedEvent:
		return rp.consumeReferralProgramStartedEvent(ctx, e)
	case ReferralProgramUpdatedEvent:
		return rp.consumeReferralProgramUpdatedEvent(ctx, e)
	case ReferralProgramEndedEvent:
		return rp.consumeReferralProgramEndedEvent(ctx, e)
	default:
		return nil
	}
}

func (rp *ReferralProgram) consumeReferralProgramStartedEvent(ctx context.Context, e ReferralProgramStartedEvent) error {
	program := entities.ReferralProgramFromProto(e.GetReferralProgramStarted().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.AddReferralProgram(ctx, program)
}

func (rp *ReferralProgram) consumeReferralProgramUpdatedEvent(ctx context.Context, e ReferralProgramUpdatedEvent) error {
	program := entities.ReferralProgramFromProto(e.GetReferralProgramUpdated().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.UpdateReferralProgram(ctx, program)
}

func (rp *ReferralProgram) consumeReferralProgramEndedEvent(ctx context.Context, e ReferralProgramEndedEvent) error {
	ev := e.GetReferralProgramEnded()
	return rp.store.EndReferralProgram(ctx, ev.GetVersion(), time.Unix(0, ev.EndedAt), rp.vegaTime, e.Sequence())
}
