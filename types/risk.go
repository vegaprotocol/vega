//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

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

type TradableInstrument_LogNormalRiskModel struct {
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

func (l LogNormalRiskModel) IntoProto() *proto.LogNormalRiskModel {
	ra, _ := l.RiskAversionParameter.Float64()
	t, _ := l.Tau.Float64()
	return &proto.LogNormalRiskModel{
		RiskAversionParameter: ra,
		Tau:                   t,
		Params:                l.Params.IntoProto(),
	}
}

func (t TradableInstrument_LogNormalRiskModel) IntoProto() *proto.TradableInstrument_LogNormalRiskModel {
	return &proto.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: t.LogNormalRiskModel.IntoProto(),
	}
}

func (TradableInstrument_LogNormalRiskModel) isTRM() {}

func (t TradableInstrument_LogNormalRiskModel) trmIntoProto() interface{} {
	return t.IntoProto()
}

func MarginCalculatorFromProto(p *proto.MarginCalculator) *MarginCalculator {
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
	PartyId                string
	MarketId               string
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
		MaintenanceMargin:      m.MaintenanceMargin.Uint64(),
		SearchLevel:            m.SearchLevel.Uint64(),
		InitialMargin:          m.InitialMargin.Uint64(),
		CollateralReleaseLevel: m.CollateralReleaseLevel.Uint64(),
		PartyId:                m.PartyId,
		MarketId:               m.MarketId,
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

type TradableInstrument_SimpleRiskModel struct {
	SimpleRiskModel *SimpleRiskModel `protobuf:"bytes,101,opt,name=simple_risk_model,json=simpleRiskModel,proto3,oneof"`
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
		return TradableInstrumentLogNoramlFromProto(tirm)
	}
	return nil
}

func LogNormalParamsFromProto(p *proto.LogNormalModelParams) *LogNormalModelParams {
	return &LogNormalModelParams{
		Mu:    num.DecimalFromFloat(p.Mu),
		R:     num.DecimalFromFloat(p.R),
		Sigma: num.DecimalFromFloat(p.Sigma),
	}
}

func TradableInstrumentLogNoramlFromProto(p *proto.TradableInstrument_LogNormalRiskModel) *TradableInstrument_LogNormalRiskModel {
	return &TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormalRiskModel.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormalRiskModel.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormalRiskModel.Params),
		},
	}
}

func TradableInstrumentSimpleFromProto(p *proto.TradableInstrument_SimpleRiskModel) *TradableInstrument_SimpleRiskModel {
	return &TradableInstrument_SimpleRiskModel{
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

func (t TradableInstrument_SimpleRiskModel) IntoProto() *proto.TradableInstrument_SimpleRiskModel {
	return &proto.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: t.SimpleRiskModel.IntoProto(),
	}
}

func (TradableInstrument_SimpleRiskModel) isTRM() {}

func (t TradableInstrument_SimpleRiskModel) trmIntoProto() interface{} {
	return t.IntoProto()
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
