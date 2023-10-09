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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ValidatorRanking struct {
	*Base
	NodeID           string
	EpochSeq         string
	StakeScore       string
	PerformanceScore string
	Ranking          string
	PreviousStatus   string
	Status           string
	TMVotingPower    int
}

func NewValidatorRanking(ctx context.Context, epochSeq, nodeID, stakeScore, performanceScore, ranking, previousStatus, status string, votingPower int) *ValidatorRanking {
	return &ValidatorRanking{
		Base:             newBase(ctx, ValidatorRankingEvent),
		NodeID:           nodeID,
		EpochSeq:         epochSeq,
		StakeScore:       stakeScore,
		PerformanceScore: performanceScore,
		Ranking:          ranking,
		PreviousStatus:   previousStatus,
		Status:           status,
		TMVotingPower:    votingPower,
	}
}

func (vr ValidatorRanking) Proto() eventspb.ValidatorRankingEvent {
	return eventspb.ValidatorRankingEvent{
		NodeId:           vr.NodeID,
		EpochSeq:         vr.EpochSeq,
		StakeScore:       vr.StakeScore,
		PerformanceScore: vr.PerformanceScore,
		RankingScore:     vr.Ranking,
		PreviousStatus:   vr.PreviousStatus,
		NextStatus:       vr.Status,
		TmVotingPower:    uint32(vr.TMVotingPower),
	}
}

func (vr ValidatorRanking) ValidatorRankingEvent() eventspb.ValidatorRankingEvent {
	return vr.Proto()
}

func (vr ValidatorRanking) StreamMessage() *eventspb.BusEvent {
	p := vr.Proto()
	busEvent := newBusEventFromBase(vr.Base)
	busEvent.Event = &eventspb.BusEvent_RankingEvent{
		RankingEvent: &p,
	}

	return busEvent
}

func ValidatorRankingEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorRanking {
	event := be.GetRankingEvent()
	if event == nil {
		return nil
	}

	return &ValidatorRanking{
		Base:             newBaseFromBusEvent(ctx, ValidatorRankingEvent, be),
		NodeID:           event.GetNodeId(),
		EpochSeq:         event.GetEpochSeq(),
		StakeScore:       event.GetStakeScore(),
		PerformanceScore: event.GetPerformanceScore(),
		Ranking:          event.GetRankingScore(),
		PreviousStatus:   event.GetPreviousStatus(),
		Status:           event.GetNextStatus(),
		TMVotingPower:    int(event.GetTmVotingPower()),
	}
}
