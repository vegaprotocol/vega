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
	"code.vegaprotocol.io/vega/core/types/num"
)

type ValidatorScore struct {
	*Base
	NodeID               string
	EpochSeq             string
	ValidatorScore       string
	NormalisedScore      string
	RawValidatorScore    string
	ValidatorPerformance string
	MultisigScore        string
	ValidatorStatus      string
}

func NewValidatorScore(ctx context.Context, nodeID, epochSeq string, score, normalisedScore, rawValidatorScore,
	validatorPerformance num.Decimal, multisigScore num.Decimal, validatorStatus string,
) *ValidatorScore {
	return &ValidatorScore{
		Base:                 newBase(ctx, ValidatorScoreEvent),
		NodeID:               nodeID,
		EpochSeq:             epochSeq,
		ValidatorScore:       score.String(),
		NormalisedScore:      normalisedScore.String(),
		RawValidatorScore:    rawValidatorScore.String(),
		ValidatorPerformance: validatorPerformance.String(),
		MultisigScore:        multisigScore.String(),
		ValidatorStatus:      validatorStatus,
	}
}

func (vd ValidatorScore) Proto() eventspb.ValidatorScoreEvent {
	return eventspb.ValidatorScoreEvent{
		NodeId:               vd.NodeID,
		EpochSeq:             vd.EpochSeq,
		ValidatorScore:       vd.ValidatorScore,
		NormalisedScore:      vd.NormalisedScore,
		ValidatorPerformance: vd.ValidatorPerformance,
		RawValidatorScore:    vd.RawValidatorScore,
		MultisigScore:        vd.MultisigScore,
		ValidatorStatus:      vd.ValidatorStatus,
	}
}

func (vd ValidatorScore) ValidatorScoreEvent() eventspb.ValidatorScoreEvent {
	return vd.Proto()
}

func (vd ValidatorScore) StreamMessage() *eventspb.BusEvent {
	p := vd.Proto()
	busEvent := newBusEventFromBase(vd.Base)
	busEvent.Event = &eventspb.BusEvent_ValidatorScore{
		ValidatorScore: &p,
	}

	return busEvent
}

func ValidatorScoreEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorScore {
	event := be.GetValidatorScore()
	if event == nil {
		return nil
	}

	return &ValidatorScore{
		Base:                 newBaseFromBusEvent(ctx, ValidatorScoreEvent, be),
		NodeID:               event.GetNodeId(),
		EpochSeq:             event.GetEpochSeq(),
		ValidatorScore:       event.ValidatorScore,
		NormalisedScore:      event.NormalisedScore,
		RawValidatorScore:    event.RawValidatorScore,
		ValidatorPerformance: event.ValidatorPerformance,
		MultisigScore:        event.MultisigScore,
		ValidatorStatus:      event.ValidatorStatus,
	}
}
