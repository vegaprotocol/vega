package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
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

func PriceMonitoringSettingsFromProto(p *proto.PriceMonitoringSettings) *PriceMonitoringSettings {
	pms := &PriceMonitoringSettings{}
	pms.FromProto(p)
	return pms
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

func (p *PriceMonitoringSettings) FromProto(pr *proto.PriceMonitoringSettings) {
	p.UpdateFrequency = pr.UpdateFrequency
	if pr.Parameters != nil {
		p.Parameters = &PriceMonitoringParameters{}
		p.Parameters.FromProto(pr.Parameters)
	}
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

func (p *PriceMonitoringParameters) FromProto(pr *proto.PriceMonitoringParameters) {
	triggers := make([]*PriceMonitoringTrigger, 0, len(pr.Triggers))
	for _, pt := range pr.Triggers {
		t := &PriceMonitoringTrigger{}
		t.FromProto(pt)
		triggers = append(triggers, t)
	}
	p.Triggers = triggers
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
		MinValidPrice:  p.MinValidPrice.Uint64(),
		MaxValidPrice:  p.MaxValidPrice.Uint64(),
		Trigger:        trigger,
		ReferencePrice: ref,
	}
}

func (p *PriceMonitoringBounds) FromProto(pr *proto.PriceMonitoringBounds) {
	p.MinValidPrice = num.NewUint(pr.MinValidPrice)
	p.MaxValidPrice = num.NewUint(pr.MaxValidPrice)
	p.Trigger.FromProto(pr.Trigger)
	p.ReferencePrice = num.DecimalFromFloat(pr.ReferencePrice)
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
		HDec:             num.DecimalFromFloat(float64(p.Horizon)),
		Probability:      num.DecimalFromFloat(p.Probability),
		AuctionExtension: p.AuctionExtension,
	}
}

// IntoProto return proto version of the PriceMonitoringTrigger
func (p PriceMonitoringTrigger) IntoProto() *proto.PriceMonitoringTrigger {
	horizon := p.Horizon
	prob, _ := p.Probability.Float64()
	return &proto.PriceMonitoringTrigger{
		Horizon:          horizon,
		Probability:      prob,
		AuctionExtension: p.AuctionExtension,
	}
}

func (p *PriceMonitoringTrigger) FromProto(pr *proto.PriceMonitoringTrigger) {
	p.Horizon = pr.Horizon
	p.HDec = num.DecimalFromFloat(float64(pr.Horizon))
	p.Probability = num.DecimalFromFloat(pr.Probability)
	p.AuctionExtension = pr.AuctionExtension
}
