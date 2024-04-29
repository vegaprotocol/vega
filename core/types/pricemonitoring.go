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

package types

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type PriceMonitoringSettings struct {
	Parameters *PriceMonitoringParameters
}

type PriceMonitoringParameters struct {
	Triggers []*PriceMonitoringTrigger
}

type PriceMonitoringBoundsList []*PriceMonitoringBounds

func (ls PriceMonitoringBoundsList) String() string {
	if ls == nil {
		return "[]"
	}
	strs := make([]string, 0, len(ls))
	for _, l := range ls {
		strs = append(strs, l.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type PriceMonitoringBounds struct {
	MinValidPrice  *num.Uint
	MaxValidPrice  *num.Uint
	Trigger        *PriceMonitoringTrigger
	ReferencePrice num.Decimal
	Active         bool
}

func (p PriceMonitoringBounds) String() string {
	return fmt.Sprintf(
		"minValidPrice(%s) maxValidPrice(%s) trigger(%s) referencePrice(%s)",
		stringer.PtrToString(p.MinValidPrice),
		stringer.PtrToString(p.MaxValidPrice),
		stringer.PtrToString(p.Trigger),
		p.ReferencePrice.String(),
	)
}

func (p PriceMonitoringSettings) IntoProto() *proto.PriceMonitoringSettings {
	var parameters *proto.PriceMonitoringParameters
	if p.Parameters != nil {
		parameters = p.Parameters.IntoProto()
	}
	return &proto.PriceMonitoringSettings{
		Parameters: parameters,
	}
}

func (p PriceMonitoringSettings) DeepClone() *PriceMonitoringSettings {
	return &PriceMonitoringSettings{
		Parameters: p.Parameters.DeepClone(),
	}
}

func (p PriceMonitoringSettings) String() string {
	return fmt.Sprintf("parameters(%s)", stringer.PtrToString(p.Parameters))
}

func PriceMonitoringSettingsFromProto(pr *proto.PriceMonitoringSettings) *PriceMonitoringSettings {
	if pr == nil {
		return nil
	}
	p := PriceMonitoringSettings{}
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

func (p PriceMonitoringParameters) String() string {
	return fmt.Sprintf(
		"triggers(%v)",
		PriceMonitoringTriggers(p.Triggers).String(),
	)
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
		ReferencePrice: p.ReferencePrice.BigInt().String(),
		Active:         p.Active,
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
		Active:         pr.Active,
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

type PriceMonitoringTriggers []*PriceMonitoringTrigger

func (ts PriceMonitoringTriggers) String() string {
	if ts == nil {
		return "[]"
	}
	strs := make([]string, 0, len(ts))
	for _, t := range ts {
		strs = append(strs, t.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type PriceMonitoringTrigger struct {
	Horizon          int64
	HorizonDec       num.Decimal
	Probability      num.Decimal
	AuctionExtension int64
}

func (p PriceMonitoringTrigger) String() string {
	return fmt.Sprintf(
		"horizonDec(%s) horizon(%v) probability(%s), auctionExtension(%v)",
		p.HorizonDec.String(),
		p.Horizon,
		p.Probability.String(),
		p.AuctionExtension,
	)
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
