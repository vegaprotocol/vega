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
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type LogNormalModelParams struct {
	Mu    num.Decimal
	R     num.Decimal
	Sigma num.Decimal
}

type LogNormalRiskModel struct {
	RiskAversionParameter num.Decimal
	Tau                   num.Decimal
	Params                *LogNormalModelParams
}

func (l LogNormalModelParams) IntoProto() *proto.LogNormalModelParams {
	mu, _ := l.Mu.Float64()
	r, _ := l.R.Float64()
	sigma, _ := l.Sigma.Float64()
	return &proto.LogNormalModelParams{
		Mu:    mu,
		R:     r,
		Sigma: sigma,
	}
}

func (l LogNormalModelParams) String() string {
	return fmt.Sprintf(
		"mu(%s) r(%s) sigma(%s)",
		l.Mu.String(),
		l.R.String(),
		l.Sigma.String(),
	)
}

func (l LogNormalModelParams) DeepClone() *LogNormalModelParams {
	return &LogNormalModelParams{
		Mu:    l.Mu,
		R:     l.R,
		Sigma: l.Sigma,
	}
}

func (l LogNormalRiskModel) IntoProto() *proto.LogNormalRiskModel {
	ra, _ := l.RiskAversionParameter.Float64()
	t, _ := l.Tau.Float64()
	var params *proto.LogNormalModelParams
	if l.Params != nil {
		params = l.Params.IntoProto()
	}
	return &proto.LogNormalRiskModel{
		RiskAversionParameter: ra,
		Tau:                   t,
		Params:                params,
	}
}

func (l LogNormalRiskModel) DeepClone() *LogNormalRiskModel {
	cpy := LogNormalRiskModel{
		RiskAversionParameter: l.RiskAversionParameter,
		Tau:                   l.Tau,
	}
	if l.Params != nil {
		cpy.Params = l.Params.DeepClone()
	}
	return &cpy
}

func (l LogNormalRiskModel) String() string {
	return fmt.Sprintf(
		"tau(%s) riskAversionParameter(%s) params(%s)",
		l.Tau.String(),
		l.RiskAversionParameter.String(),
		stringer.PtrToString(l.Params),
	)
}

type TradableInstrumentLogNormalRiskModel struct {
	LogNormalRiskModel *LogNormalRiskModel
}

func (t TradableInstrumentLogNormalRiskModel) String() string {
	return fmt.Sprintf(
		"logNormalRiskModel(%s)",
		stringer.PtrToString(t.LogNormalRiskModel),
	)
}

func (t TradableInstrumentLogNormalRiskModel) IntoProto() *proto.TradableInstrument_LogNormalRiskModel {
	return &proto.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: t.LogNormalRiskModel.IntoProto(),
	}
}

func (TradableInstrumentLogNormalRiskModel) isTRM() {}

func (t TradableInstrumentLogNormalRiskModel) trmIntoProto() interface{} {
	return t.IntoProto()
}

func (TradableInstrumentLogNormalRiskModel) rmType() rmType {
	return LogNormalRiskModelType
}

func (t TradableInstrumentLogNormalRiskModel) Equal(trm isTRM) bool {
	var ct *TradableInstrumentLogNormalRiskModel
	switch et := trm.(type) {
	case *TradableInstrumentLogNormalRiskModel:
		ct = et
	case TradableInstrumentLogNormalRiskModel:
		ct = &et
	}
	if ct == nil {
		return false
	}
	if !t.LogNormalRiskModel.Tau.Equal(ct.LogNormalRiskModel.Tau) || !t.LogNormalRiskModel.RiskAversionParameter.Equal(ct.LogNormalRiskModel.RiskAversionParameter) {
		return false
	}
	// check params
	p, cp := t.LogNormalRiskModel.Params, ct.LogNormalRiskModel.Params
	// check if all params match
	return p.Mu.Equal(cp.Mu) && p.R.Equal(cp.R) && p.Sigma.Equal(cp.Sigma)
}

func MarginCalculatorFromProto(p *proto.MarginCalculator) *MarginCalculator {
	if p == nil {
		return nil
	}
	return &MarginCalculator{
		ScalingFactors: ScalingFactorsFromProto(p.ScalingFactors),
	}
}

func ScalingFactorsFromProto(p *proto.ScalingFactors) *ScalingFactors {
	return &ScalingFactors{
		SearchLevel:       num.DecimalFromFloat(p.SearchLevel),
		InitialMargin:     num.DecimalFromFloat(p.InitialMargin),
		CollateralRelease: num.DecimalFromFloat(p.CollateralRelease),
	}
}

type MarginCalculator struct {
	ScalingFactors *ScalingFactors
}

func (m MarginCalculator) DeepClone() *MarginCalculator {
	return &MarginCalculator{
		ScalingFactors: m.ScalingFactors.DeepClone(),
	}
}

type ScalingFactors struct {
	SearchLevel       num.Decimal
	InitialMargin     num.Decimal
	CollateralRelease num.Decimal
}

func (s ScalingFactors) DeepClone() *ScalingFactors {
	return &ScalingFactors{
		SearchLevel:       s.SearchLevel,
		InitialMargin:     s.InitialMargin,
		CollateralRelease: s.CollateralRelease,
	}
}

type MarginLevels struct {
	MaintenanceMargin      *num.Uint
	SearchLevel            *num.Uint
	InitialMargin          *num.Uint
	CollateralReleaseLevel *num.Uint
	OrderMargin            *num.Uint
	Party                  string
	MarketID               string
	Asset                  string
	Timestamp              int64
	MarginMode             MarginMode
	MarginFactor           num.Decimal
}

type RiskFactor struct {
	Market string
	Short  num.Decimal
	Long   num.Decimal
}

func (m MarginLevels) IntoProto() *proto.MarginLevels {
	return &proto.MarginLevels{
		MaintenanceMargin:      num.UintToString(m.MaintenanceMargin),
		SearchLevel:            num.UintToString(m.SearchLevel),
		InitialMargin:          num.UintToString(m.InitialMargin),
		CollateralReleaseLevel: num.UintToString(m.CollateralReleaseLevel),
		OrderMargin:            num.UintToString(m.OrderMargin),
		PartyId:                m.Party,
		MarketId:               m.MarketID,
		Asset:                  m.Asset,
		Timestamp:              m.Timestamp,
		MarginMode:             m.MarginMode,
		MarginFactor:           m.MarginFactor.String(),
	}
}

func (m MarginLevels) String() string {
	return fmt.Sprintf(
		"marketID(%s) asset(%s) party(%s) intialMargin(%s) maintenanceMargin(%s) collateralReleaseLevel(%s) searchLevel(%s) orderMargin(%s) timestamp(%v) marginMode(%d) marginFactor(%s)",
		m.MarketID,
		m.Asset,
		m.Party,
		stringer.PtrToString(m.InitialMargin),
		stringer.PtrToString(m.MaintenanceMargin),
		stringer.PtrToString(m.CollateralReleaseLevel),
		stringer.PtrToString(m.SearchLevel),
		stringer.PtrToString(m.OrderMargin),
		m.Timestamp,
		m.MarginMode,
		m.MarginFactor.String(),
	)
}

func (r RiskFactor) IntoProto() *proto.RiskFactor {
	return &proto.RiskFactor{
		Market: r.Market,
		Short:  r.Short.String(),
		Long:   r.Long.String(),
	}
}

func (r RiskFactor) String() string {
	return fmt.Sprintf(
		"marketID(%s) short(%s) long(%s)",
		r.Market,
		r.Short.String(),
		r.Long.String(),
	)
}

func (m MarginCalculator) IntoProto() *proto.MarginCalculator {
	return &proto.MarginCalculator{
		ScalingFactors: m.ScalingFactors.IntoProto(),
	}
}

func (m MarginCalculator) String() string {
	return fmt.Sprintf(
		"scalingFactors(%s)",
		stringer.PtrToString(m.ScalingFactors),
	)
}

func (s ScalingFactors) IntoProto() *proto.ScalingFactors {
	sl, _ := s.SearchLevel.Float64()
	im, _ := s.InitialMargin.Float64()
	cr, _ := s.CollateralRelease.Float64()
	return &proto.ScalingFactors{
		SearchLevel:       sl,
		InitialMargin:     im,
		CollateralRelease: cr,
	}
}

func (s ScalingFactors) String() string {
	return fmt.Sprintf(
		"searchLevel(%s) initialMargin(%s) collateralRelease(%s)",
		s.SearchLevel.String(),
		s.InitialMargin.String(),
		s.CollateralRelease.String(),
	)
}

func (s *ScalingFactors) Reset() {
	*s = ScalingFactors{}
}

type SimpleModelParams struct {
	FactorLong           num.Decimal
	FactorShort          num.Decimal
	MaxMoveUp            num.Decimal
	MinMoveDown          num.Decimal
	ProbabilityOfTrading num.Decimal
}

func isTRMFromProto(p interface{}) isTRM {
	switch tirm := p.(type) {
	case *proto.TradableInstrument_SimpleRiskModel:
		return TradableInstrumentSimpleFromProto(tirm)
	case *proto.TradableInstrument_LogNormalRiskModel:
		return TradableInstrumentLogNormalFromProto(tirm)
	}
	// default to nil simple params
	return TradableInstrumentSimpleFromProto(nil)
}

func LogNormalParamsFromProto(p *proto.LogNormalModelParams) *LogNormalModelParams {
	if p == nil {
		return nil
	}
	return &LogNormalModelParams{
		Mu:    num.DecimalFromFloat(p.Mu),
		R:     num.DecimalFromFloat(p.R),
		Sigma: num.DecimalFromFloat(p.Sigma),
	}
}

func TradableInstrumentLogNormalFromProto(p *proto.TradableInstrument_LogNormalRiskModel) *TradableInstrumentLogNormalRiskModel {
	if p == nil {
		return nil
	}
	return &TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormalRiskModel.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormalRiskModel.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormalRiskModel.Params),
		},
	}
}

func SimpleModelParamsFromProto(p *proto.SimpleModelParams) *SimpleModelParams {
	return &SimpleModelParams{
		FactorLong:           num.DecimalFromFloat(p.FactorLong),
		FactorShort:          num.DecimalFromFloat(p.FactorShort),
		MaxMoveUp:            num.DecimalFromFloat(p.MaxMoveUp),
		MinMoveDown:          num.DecimalFromFloat(p.MinMoveDown),
		ProbabilityOfTrading: num.DecimalFromFloat(p.ProbabilityOfTrading),
	}
}

type TradableInstrumentSimpleRiskModel struct {
	SimpleRiskModel *SimpleRiskModel
}

func (t TradableInstrumentSimpleRiskModel) String() string {
	return fmt.Sprintf(
		"simpleRiskModel(%s)",
		stringer.PtrToString(t.SimpleRiskModel),
	)
}

func (TradableInstrumentSimpleRiskModel) isTRM() {}

func (t TradableInstrumentSimpleRiskModel) IntoProto() *proto.TradableInstrument_SimpleRiskModel {
	return &proto.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: t.SimpleRiskModel.IntoProto(),
	}
}

func (t TradableInstrumentSimpleRiskModel) trmIntoProto() interface{} {
	return t.IntoProto()
}

func (TradableInstrumentSimpleRiskModel) rmType() rmType {
	return SimpleRiskModelType
}

// Equal returns true if the risk models match.
func (t TradableInstrumentSimpleRiskModel) Equal(trm isTRM) bool {
	var ct *TradableInstrumentSimpleRiskModel
	switch et := trm.(type) {
	case *TradableInstrumentSimpleRiskModel:
		ct = et
	case TradableInstrumentSimpleRiskModel:
		ct = &et
	}
	if ct == nil {
		return false
	}
	if !t.SimpleRiskModel.Params.FactorLong.Equal(ct.SimpleRiskModel.Params.FactorLong) {
		return false
	}
	if !t.SimpleRiskModel.Params.FactorShort.Equal(ct.SimpleRiskModel.Params.FactorShort) {
		return false
	}
	if !t.SimpleRiskModel.Params.MinMoveDown.Equal(ct.SimpleRiskModel.Params.MinMoveDown) {
		return false
	}
	if !t.SimpleRiskModel.Params.MaxMoveUp.Equal(ct.SimpleRiskModel.Params.MaxMoveUp) {
		return false
	}
	return t.SimpleRiskModel.Params.ProbabilityOfTrading.Equal(ct.SimpleRiskModel.Params.ProbabilityOfTrading)
}

func TradableInstrumentSimpleFromProto(p *proto.TradableInstrument_SimpleRiskModel) *TradableInstrumentSimpleRiskModel {
	if p == nil {
		return nil
	}
	return &TradableInstrumentSimpleRiskModel{
		SimpleRiskModel: &SimpleRiskModel{
			Params: SimpleModelParamsFromProto(p.SimpleRiskModel.Params),
		},
	}
}

type SimpleRiskModel struct {
	Params *SimpleModelParams
}

func (s SimpleRiskModel) IntoProto() *proto.SimpleRiskModel {
	return &proto.SimpleRiskModel{
		Params: s.Params.IntoProto(),
	}
}

func (s SimpleRiskModel) String() string {
	return fmt.Sprintf(
		"params(%s)",
		stringer.PtrToString(s.Params),
	)
}

func (s SimpleModelParams) IntoProto() *proto.SimpleModelParams {
	lng, _ := s.FactorLong.Float64()
	sht, _ := s.FactorShort.Float64()
	up, _ := s.MaxMoveUp.Float64()
	down, _ := s.MinMoveDown.Float64()
	prob, _ := s.ProbabilityOfTrading.Float64()
	return &proto.SimpleModelParams{
		FactorLong:           lng,
		FactorShort:          sht,
		MaxMoveUp:            up,
		MinMoveDown:          down,
		ProbabilityOfTrading: prob,
	}
}

func (s SimpleModelParams) String() string {
	return fmt.Sprintf(
		"probabilityOfTrading(%s) factor(short(%s) long(%s)) minMoveDown(%s) maxMoveUp(%s)",
		s.ProbabilityOfTrading.String(),
		s.FactorShort.String(),
		s.FactorLong.String(),
		s.MinMoveDown.String(),
		s.MaxMoveUp.String(),
	)
}

func (s SimpleModelParams) DeepClone() *SimpleModelParams {
	return &SimpleModelParams{
		FactorLong:           s.FactorLong,
		FactorShort:          s.FactorShort,
		MaxMoveUp:            s.MaxMoveUp,
		MinMoveDown:          s.MinMoveDown,
		ProbabilityOfTrading: s.ProbabilityOfTrading,
	}
}

type MarginMode = proto.MarginMode

const (
	MarginModeUnspecified    MarginMode = proto.MarginMode_MARGIN_MODE_UNSPECIFIED
	MarginModeCrossMargin    MarginMode = proto.MarginMode_MARGIN_MODE_CROSS_MARGIN
	MarginModeIsolatedMargin MarginMode = proto.MarginMode_MARGIN_MODE_ISOLATED_MARGIN
)
