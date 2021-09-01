package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type ValidatorScore struct {
	*Base
	NodeID          string
	EpochSeq        string
	ValidatorScore  string
	NormalisedScore string
}

func NewValidatorScore(ctx context.Context, nodeID, epochSeq string, score, normalisedScore num.Decimal) *ValidatorScore {
	return &ValidatorScore{
		Base:            newBase(ctx, ValidatorScoreEvent),
		NodeID:          nodeID,
		EpochSeq:        epochSeq,
		ValidatorScore:  score.String(),
		NormalisedScore: normalisedScore.String(),
	}
}

func (vd ValidatorScore) Proto() eventspb.ValidatorScoreEvent {
	return eventspb.ValidatorScoreEvent{
		NodeId:          vd.NodeID,
		EpochSeq:        vd.EpochSeq,
		ValidatorScore:  vd.ValidatorScore,
		NormalisedScore: vd.NormalisedScore,
	}
}

func (vd ValidatorScore) ValidatorScoreEvent() eventspb.ValidatorScoreEvent {
	return vd.Proto()
}

func (vd ValidatorScore) StreamMessage() *eventspb.BusEvent {
	p := vd.Proto()
	return &eventspb.BusEvent{
		Id:    vd.eventID(),
		Block: vd.TraceID(),
		Type:  vd.et.ToProto(),
		Event: &eventspb.BusEvent_ValidatorScore{
			ValidatorScore: &p,
		},
	}
}

func ValidatorScoreEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorScore {
	event := be.GetValidatorScore()
	if event == nil {
		return nil
	}

	return &ValidatorScore{
		Base:            newBaseFromStream(ctx, ValidatorScoreEvent, be),
		NodeID:          event.GetNodeId(),
		EpochSeq:        event.GetEpochSeq(),
		ValidatorScore:  event.ValidatorScore,
		NormalisedScore: event.NormalisedScore,
	}
}
