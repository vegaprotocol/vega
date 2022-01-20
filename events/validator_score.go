package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type ValidatorScore struct {
	*Base
	NodeID               string
	EpochSeq             string
	ValidatorScore       string
	NormalisedScore      string
	RawValidatorScore    string
	ValidatorPerformance string
}

func NewValidatorScore(ctx context.Context, nodeID, epochSeq string, score, normalisedScore, rawValidatorScore,
	validatorPerformance num.Decimal) *ValidatorScore {
	return &ValidatorScore{
		Base:                 newBase(ctx, ValidatorScoreEvent),
		NodeID:               nodeID,
		EpochSeq:             epochSeq,
		ValidatorScore:       score.String(),
		NormalisedScore:      normalisedScore.String(),
		RawValidatorScore:    rawValidatorScore.String(),
		ValidatorPerformance: validatorPerformance.String(),
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
	}
}

func (vd ValidatorScore) ValidatorScoreEvent() eventspb.ValidatorScoreEvent {
	return vd.Proto()
}

func (vd ValidatorScore) StreamMessage() *eventspb.BusEvent {
	p := vd.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      vd.eventID(),
		Block:   vd.TraceID(),
		ChainId: vd.ChainID(),
		Type:    vd.et.ToProto(),
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
		Base:                 newBaseFromStream(ctx, ValidatorScoreEvent, be),
		NodeID:               event.GetNodeId(),
		EpochSeq:             event.GetEpochSeq(),
		ValidatorScore:       event.ValidatorScore,
		NormalisedScore:      event.NormalisedScore,
		RawValidatorScore:    event.RawValidatorScore,
		ValidatorPerformance: event.ValidatorPerformance,
	}
}
