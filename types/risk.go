package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type LogNormalModelParams struct {
	Mu    num.Decimal
	R     num.Decimal
	Sigma num.Decimal
}

type TradableInstrumentLogNormalRiskModel struct {
	LogNormalRiskModel *LogNormalRiskModel
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
	return l.IntoProto().String()
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
	return LOGNORMAL_RISK_MODEL
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

type ScalingFactors struct {
	SearchLevel       num.Decimal
	InitialMargin     num.Decimal
	CollateralRelease num.Decimal
}

type MarginLevels struct {
	MaintenanceMargin      *num.Uint
	SearchLevel            *num.Uint
	InitialMargin          *num.Uint
	CollateralReleaseLevel *num.Uint
	Party                  string
	MarketID               string
	Asset                  string
	Timestamp              int64
}

type RiskFactor struct {
	Market string
	Short  num.Decimal
	Long   num.Decimal
}

type RiskResult struct {
	UpdatedTimestamp         int64
	RiskFactors              map[string]*RiskFactor
	NextUpdateTimestamp      int64
	PredictedNextRiskFactors map[string]*RiskFactor
}

func (m MarginLevels) IntoProto() *proto.MarginLevels {
	return &proto.MarginLevels{
		MaintenanceMargin:      num.UintToUint64(m.MaintenanceMargin),
		SearchLevel:            num.UintToUint64(m.SearchLevel),
		InitialMargin:          num.UintToUint64(m.InitialMargin),
		CollateralReleaseLevel: num.UintToUint64(m.CollateralReleaseLevel),
		PartyId:                m.Party,
		MarketId:               m.MarketID,
		Asset:                  m.Asset,
		Timestamp:              m.Timestamp,
	}
}

func (m MarginLevels) String() string {
	return m.IntoProto().String()
}

func (r RiskResult) IntoProto() *proto.RiskResult {
	pr := &proto.RiskResult{
		UpdatedTimestamp:         r.UpdatedTimestamp,
		RiskFactors:              make(map[string]*proto.RiskFactor, len(r.RiskFactors)),
		NextUpdateTimestamp:      r.NextUpdateTimestamp,
		PredictedNextRiskFactors: make(map[string]*proto.RiskFactor, len(r.PredictedNextRiskFactors)),
	}
	for k, f := range r.RiskFactors {
		pr.RiskFactors[k] = f.IntoProto()
	}
	for k, f := range r.PredictedNextRiskFactors {
		pr.PredictedNextRiskFactors[k] = f.IntoProto()
	}
	return pr
}

func (r RiskResult) String() string {
	return r.IntoProto().String()
}

func (r RiskFactor) IntoProto() *proto.RiskFactor {
	short, _ := r.Short.Float64()
	long, _ := r.Long.Float64()
	return &proto.RiskFactor{
		Market: r.Market,
		Short:  short,
		Long:   long,
	}
}

func (r RiskFactor) String() string {
	return r.IntoProto().String()
}

func (m MarginCalculator) IntoProto() *proto.MarginCalculator {
	return &proto.MarginCalculator{
		ScalingFactors: m.ScalingFactors.IntoProto(),
	}
}

func (m MarginCalculator) String() string {
	return m.IntoProto().String()
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
	return s.IntoProto().String()
}

func (s *ScalingFactors) Reset() {
	*s = ScalingFactors{}
}

type TradableInstrumentSimpleRiskModel struct {
	SimpleRiskModel *SimpleRiskModel
}

type SimpleRiskModel struct {
	Params *SimpleModelParams
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

func SimpleModelParamsFromProto(p *proto.SimpleModelParams) *SimpleModelParams {
	return &SimpleModelParams{
		FactorLong:           num.DecimalFromFloat(p.FactorLong),
		FactorShort:          num.DecimalFromFloat(p.FactorShort),
		MaxMoveUp:            num.DecimalFromFloat(p.MaxMoveUp),
		MinMoveDown:          num.DecimalFromFloat(p.MinMoveDown),
		ProbabilityOfTrading: num.DecimalFromFloat(p.ProbabilityOfTrading),
	}
}

func (t TradableInstrumentSimpleRiskModel) IntoProto() *proto.TradableInstrument_SimpleRiskModel {
	return &proto.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: t.SimpleRiskModel.IntoProto(),
	}
}

func (TradableInstrumentSimpleRiskModel) isTRM() {}

func (t TradableInstrumentSimpleRiskModel) trmIntoProto() interface{} {
	return t.IntoProto()
}

func (TradableInstrumentSimpleRiskModel) rmType() rmType {
	return SIMPLE_RISK_MODEL
}

func (s SimpleRiskModel) IntoProto() *proto.SimpleRiskModel {
	return &proto.SimpleRiskModel{
		Params: s.Params.IntoProto(),
	}
}

func (s SimpleRiskModel) String() string {
	return s.IntoProto().String()
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
	return s.IntoProto().String()
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
