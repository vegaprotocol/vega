//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

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
