package types

import (
	proto "code.vegaprotocol.io/data-node/proto/vega"
	"code.vegaprotocol.io/data-node/types/num"
)

type PriceMonitoringSettings struct {
	Parameters      *PriceMonitoringParameters
	UpdateFrequency int64
}

type PriceMonitoringParameters struct {
	Triggers []*PriceMonitoringTrigger
}

type PriceMonitoringTrigger struct {
	Horizon          int64
	HDec             num.Decimal
	Probability      num.Decimal
	AuctionExtension int64
}

type PriceMonitoringBounds struct {
	MinValidPrice  *num.Uint
	MaxValidPrice  *num.Uint
	Trigger        *PriceMonitoringTrigger
	ReferencePrice num.Decimal
}

func (p PriceMonitoringSettings) IntoProto() *proto.PriceMonitoringSettings {
	var parameters *proto.PriceMonitoringParameters
	if p.Parameters != nil {
		parameters = p.Parameters.IntoProto()
	}
	return &proto.PriceMonitoringSettings{
		Parameters:      parameters,
		UpdateFrequency: p.UpdateFrequency,
	}
}

func PriceMonitoringSettingsFromProto(pr *proto.PriceMonitoringSettings) *PriceMonitoringSettings {
	p := PriceMonitoringSettings{
		UpdateFrequency: pr.UpdateFrequency,
	}
	if pr.Parameters != nil {
		p.Parameters = PriceMonitoringParametersFromProto(pr.Parameters)
	}
	return &p
}

func PriceMonitoringParametersFromProto(p *proto.PriceMonitoringParameters) *PriceMonitoringParameters {
	triggers := make([]*PriceMonitoringTrigger, 0, len(p.Triggers))
	for _, t := range p.Triggers {
		triggers = append(triggers, PriceMonitoringTriggerFromProto(t))
	}
	return &PriceMonitoringParameters{
		Triggers: triggers,
	}
}

func (p PriceMonitoringParameters) IntoProto() *proto.PriceMonitoringParameters {
	triggers := make([]*proto.PriceMonitoringTrigger, 0, len(p.Triggers))
	for _, t := range p.Triggers {
		triggers = append(triggers, t.IntoProto())
	}
	return &proto.PriceMonitoringParameters{
		Triggers: triggers,
	}
}

func (p PriceMonitoringParameters) DeepClone() *PriceMonitoringParameters {
	cpy := PriceMonitoringParameters{
		Triggers: make([]*PriceMonitoringTrigger, 0, len(p.Triggers)),
	}
	for _, t := range p.Triggers {
		cpy.Triggers = append(cpy.Triggers, t.DeepClone())
	}
	return &cpy
}

func (p *PriceMonitoringParameters) Reset() {
	*p = PriceMonitoringParameters{}
}

func (p PriceMonitoringBounds) IntoProto() *proto.PriceMonitoringBounds {
	ref, _ := p.ReferencePrice.Float64()
	var trigger *proto.PriceMonitoringTrigger
	if p.Trigger != nil {
		trigger = p.Trigger.IntoProto()
	}
	return &proto.PriceMonitoringBounds{
		MinValidPrice:  num.UintToUint64(p.MinValidPrice),
		MaxValidPrice:  num.UintToUint64(p.MaxValidPrice),
		Trigger:        trigger,
		ReferencePrice: ref,
	}
}

func PriceMonitoringBoundsFromProto(pr *proto.PriceMonitoringBounds) *PriceMonitoringBounds {
	p := PriceMonitoringBounds{
		MinValidPrice:  num.NewUint(pr.MinValidPrice),
		MaxValidPrice:  num.NewUint(pr.MaxValidPrice),
		ReferencePrice: num.DecimalFromFloat(pr.ReferencePrice),
	}
	if pr.Trigger != nil {
		p.Trigger = PriceMonitoringTriggerFromProto(pr.Trigger)
	}
	return &p
}

func (p PriceMonitoringBounds) DeepClone() *PriceMonitoringBounds {
	cpy := p
	if p.MinValidPrice != nil {
		cpy.MinValidPrice = p.MinValidPrice.Clone()
	}
	if p.MaxValidPrice != nil {
		cpy.MaxValidPrice = p.MaxValidPrice.Clone()
	}
	return &cpy
}

func PriceMonitoringTriggerFromProto(p *proto.PriceMonitoringTrigger) *PriceMonitoringTrigger {
	return &PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		HDec:             num.DecimalFromInt64(p.Horizon),
		Probability:      num.DecimalFromFloat(p.Probability),
		AuctionExtension: p.AuctionExtension,
	}
}

// IntoProto return proto version of the PriceMonitoringTrigger
func (p PriceMonitoringTrigger) IntoProto() *proto.PriceMonitoringTrigger {
	prob, _ := p.Probability.Float64()
	return &proto.PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		Probability:      prob,
		AuctionExtension: p.AuctionExtension,
	}
}

func (p PriceMonitoringTrigger) DeepClone() *PriceMonitoringTrigger {
	return &PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		HDec:             p.HDec,
		Probability:      p.Probability,
		AuctionExtension: p.AuctionExtension,
	}
}
