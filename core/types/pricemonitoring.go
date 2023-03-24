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

package types

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/libs/num"
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
}

func (p PriceMonitoringBounds) String() string {
	return fmt.Sprintf(
		"minValidPrice(%s) maxValidPrice(%s) trigger(%s) referencePrice(%s)",
		uintPointerToString(p.MinValidPrice),
		uintPointerToString(p.MaxValidPrice),
		reflectPointerToString(p.Trigger),
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
	return fmt.Sprintf("parameters(%s)", reflectPointerToString(p.Parameters))
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
	HorizonDec       num.Decimal
	Probability      num.Decimal
	Horizon          int64
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
