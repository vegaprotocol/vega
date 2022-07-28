// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
