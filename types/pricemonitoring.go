package types

import (
	"errors"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
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
	HorizonDec       num.Decimal
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

func (p PriceMonitoringSettings) DeepClone() *PriceMonitoringSettings {
	return &PriceMonitoringSettings{
		Parameters:      p.Parameters.DeepClone(),
		UpdateFrequency: p.UpdateFrequency,
	}
}

func PriceMonitoringSettingsFromProto(pr *proto.PriceMonitoringSettings) *PriceMonitoringSettings {
	if pr == nil {
		return nil
	}
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
	var trigger *proto.PriceMonitoringTrigger
	if p.Trigger != nil {
		trigger = p.Trigger.IntoProto()
	}
	return &proto.PriceMonitoringBounds{
		MinValidPrice:  num.UintToString(p.MinValidPrice),
		MaxValidPrice:  num.UintToString(p.MaxValidPrice),
		Trigger:        trigger,
		ReferencePrice: p.ReferencePrice.String(),
	}
}

func PriceMonitoringBoundsFromProto(pr *proto.PriceMonitoringBounds) (*PriceMonitoringBounds, error) {
	minValid, overflowed := num.UintFromString(pr.MinValidPrice, 10)
	if overflowed {
		return nil, errors.New("invalid min valid price")
	}
	maxValid, overflowed := num.UintFromString(pr.MaxValidPrice, 10)
	if overflowed {
		return nil, errors.New("invalid max valid price")
	}
	refPrice, err := num.DecimalFromString(pr.ReferencePrice)
	if err != nil {
		return nil, fmt.Errorf("invalid reference price: %w", err)
	}
	p := PriceMonitoringBounds{
		MinValidPrice:  minValid,
		MaxValidPrice:  maxValid,
		ReferencePrice: refPrice,
	}
	if pr.Trigger != nil {
		p.Trigger = PriceMonitoringTriggerFromProto(pr.Trigger)
	}
	return &p, nil
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
	probability, err := num.DecimalFromString(p.Probability)
	if err != nil {
		probability = num.DecimalZero()
	}
	return &PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		HorizonDec:       num.DecimalFromInt64(p.Horizon),
		Probability:      probability,
		AuctionExtension: p.AuctionExtension,
	}
}

// IntoProto return proto version of the PriceMonitoringTrigger.
func (p PriceMonitoringTrigger) IntoProto() *proto.PriceMonitoringTrigger {
	return &proto.PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		Probability:      p.Probability.String(),
		AuctionExtension: p.AuctionExtension,
	}
}

func (p PriceMonitoringTrigger) DeepClone() *PriceMonitoringTrigger {
	return &PriceMonitoringTrigger{
		Horizon:          p.Horizon,
		HorizonDec:       p.HorizonDec,
		Probability:      p.Probability,
		AuctionExtension: p.AuctionExtension,
	}
}
