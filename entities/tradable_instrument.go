package entities

import (
	"encoding/json"
	"errors"

	"code.vegaprotocol.io/protos/vega"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
)

type TradableInstrument struct {
	Instrument       Instrument       `json:"instrument"`
	MarginCalculator MarginCalculator `json:"marginCalculator"`
	RiskModel        RiskModel        `json:"riskModel"`
}

func tradableInstrumentFromProto(ti *vega.TradableInstrument) (TradableInstrument, error) {
	if ti == nil {
		return TradableInstrument{}, errors.New("tradable instrument cannot be nil")
	}

	var riskModel RiskModel

	switch rm := ti.RiskModel.(type) {
	case *vega.TradableInstrument_SimpleRiskModel:
		riskModel = tradableInstrumentSimpleRiskModelFromProto(rm)
	case *vega.TradableInstrument_LogNormalRiskModel:
		riskModel = tradableInstrumentLogNormalRiskModelFromProto(rm)
	}

	return TradableInstrument{
		Instrument:       instrumentFromProto(ti.Instrument),
		MarginCalculator: marginCalculatorFromProto(ti.MarginCalculator),
		RiskModel:        riskModel,
	}, nil
}

func (ti TradableInstrument) ToProto() *vega.TradableInstrument {
	switch rm := ti.RiskModel.(type) {
	case TradableInstrumentSimpleRiskModel:
		return &vega.TradableInstrument{
			Instrument:       instrumentToProto(ti.Instrument),
			MarginCalculator: marginCalculatorToProto(ti.MarginCalculator),
			RiskModel:        tradableInstrumentSimpleRiskModelToProto(rm),
		}
	case TradableInstrumentLogNormalRiskModel:
		return &vega.TradableInstrument{
			Instrument:       instrumentToProto(ti.Instrument),
			MarginCalculator: marginCalculatorToProto(ti.MarginCalculator),
			RiskModel:        tradableInstrumentLogNormalRiskModelToProto(rm),
		}
	default:
		return &vega.TradableInstrument{
			Instrument:       instrumentToProto(ti.Instrument),
			MarginCalculator: marginCalculatorToProto(ti.MarginCalculator),
			RiskModel:        nil,
		}
	}
}

func (ti *TradableInstrument) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	if err := json.Unmarshal(b, &objMap); err != nil {
		return err
	}

	riskModelRaw := *objMap["riskModel"]
	instrument := &Instrument{}

	if err := instrument.UnmarshalJSON(*objMap["instrument"]); err != nil {
		return err
	}

	ti.Instrument = *instrument

	if err := json.Unmarshal(*objMap["marginCalculator"], &ti.MarginCalculator); err != nil {
		return err
	}

	var simpleRiskModel TradableInstrumentSimpleRiskModel
	var logNormalRiskModel TradableInstrumentLogNormalRiskModel

	if err := json.Unmarshal(riskModelRaw, &simpleRiskModel); err == nil {
		ti.RiskModel = simpleRiskModel
		return nil
	}

	if err := json.Unmarshal(riskModelRaw, &logNormalRiskModel); err == nil {
		ti.RiskModel = logNormalRiskModel
		return nil
	}

	return errors.New("tradable instrument contains invalid risk model")
}

type Instrument struct {
	Id       string             `json:"id"`
	Code     string             `json:"code"`
	Name     string             `json:"name"`
	Metadata InstrumentMetadata `json:"metadata"`
	Product  Product            `json:"product"`
}

func (i *Instrument) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	if err := json.Unmarshal(b, &objMap); err != nil {
		return err
	}

	if err := json.Unmarshal(*objMap["id"], &i.Id); err != nil {
		return err
	}
	if err := json.Unmarshal(*objMap["code"], &i.Code); err != nil {
		return err
	}
	if err := json.Unmarshal(*objMap["name"], &i.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(*objMap["metadata"], &i.Metadata); err != nil {
		return err
	}

	var instrumentFuture InstrumentFuture
	if err := json.Unmarshal(*objMap["product"], &instrumentFuture); err != nil {
		return err
	}

	i.Product = instrumentFuture

	return nil
}

func instrumentFromProto(instrument *vega.Instrument) Instrument {
	var product Product

	switch p := instrument.Product.(type) {
	case *vega.Instrument_Future:
		product = instrumentFutureFromProto(p)
	default:
		product = InstrumentFuture{}
	}

	return Instrument{
		Id:   instrument.Id,
		Code: instrument.Code,
		Name: instrument.Name,
		Metadata: InstrumentMetadata{
			Tags: instrument.Metadata.Tags,
		},
		Product: product,
	}
}

func instrumentToProto(instrument Instrument) *vega.Instrument {
	switch p := instrument.Product.(type) {
	case InstrumentFuture:
		return &vega.Instrument{
			Id:   instrument.Id,
			Code: instrument.Code,
			Name: instrument.Name,
			Metadata: &vega.InstrumentMetadata{
				Tags: instrument.Metadata.Tags,
			},
			Product: instrumentFutureToProto(p),
		}

	default:
		return &vega.Instrument{
			Id:   instrument.Id,
			Code: instrument.Code,
			Name: instrument.Name,
			Metadata: &vega.InstrumentMetadata{
				Tags: instrument.Metadata.Tags,
			},
			Product: nil,
		}

	}
}

type MarginCalculator struct {
	ScalingFactors ScalingFactors `json:"scalingFactors"`
}

func marginCalculatorFromProto(calc *vega.MarginCalculator) MarginCalculator {
	if calc == nil || calc.ScalingFactors == nil {
		return MarginCalculator{}
	}

	return MarginCalculator{
		ScalingFactors: ScalingFactors{
			SearchLevel:       calc.ScalingFactors.SearchLevel,
			InitialMargin:     calc.ScalingFactors.InitialMargin,
			CollateralRelease: calc.ScalingFactors.CollateralRelease,
		},
	}
}

func marginCalculatorToProto(calc MarginCalculator) *vega.MarginCalculator {
	return &vega.MarginCalculator{
		ScalingFactors: &vega.ScalingFactors{
			SearchLevel:       calc.ScalingFactors.SearchLevel,
			InitialMargin:     calc.ScalingFactors.InitialMargin,
			CollateralRelease: calc.ScalingFactors.CollateralRelease,
		},
	}
}

type RiskModel interface {
	RiskModel()
}

type TradableInstrumentSimpleRiskModel struct {
	SimpleRiskModel SimpleRiskModel `json:"simpleRiskModel"`
}

func tradableInstrumentSimpleRiskModelToProto(model TradableInstrumentSimpleRiskModel) *vega.TradableInstrument_SimpleRiskModel {
	return &vega.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: &vega.SimpleRiskModel{
			Params: &vega.SimpleModelParams{
				FactorLong:           model.SimpleRiskModel.Params.FactorLong,
				FactorShort:          model.SimpleRiskModel.Params.FactorShort,
				MaxMoveUp:            model.SimpleRiskModel.Params.MaxMoveUp,
				MinMoveDown:          model.SimpleRiskModel.Params.MinMoveDown,
				ProbabilityOfTrading: model.SimpleRiskModel.Params.ProbabilityOfTrading,
			},
		},
	}
}

func tradableInstrumentSimpleRiskModelFromProto(model *vega.TradableInstrument_SimpleRiskModel) TradableInstrumentSimpleRiskModel {
	if model == nil {
		return TradableInstrumentSimpleRiskModel{}
	}

	return TradableInstrumentSimpleRiskModel{
		SimpleRiskModel: SimpleRiskModel{
			Params: SimpleModelParams{
				FactorLong:           model.SimpleRiskModel.Params.FactorLong,
				FactorShort:          model.SimpleRiskModel.Params.FactorShort,
				MaxMoveUp:            model.SimpleRiskModel.Params.MaxMoveUp,
				MinMoveDown:          model.SimpleRiskModel.Params.MinMoveDown,
				ProbabilityOfTrading: model.SimpleRiskModel.Params.ProbabilityOfTrading,
			},
		},
	}

}

type TradableInstrumentLogNormalRiskModel struct {
	LogNormalRiskModel LogNormalRiskModel `json:"logNormalRiskModel"`
}

func tradableInstrumentLogNormalRiskModelFromProto(model *vega.TradableInstrument_LogNormalRiskModel) TradableInstrumentLogNormalRiskModel {
	if model == nil {
		return TradableInstrumentLogNormalRiskModel{}
	}

	return TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: LogNormalRiskModel{
			RiskAversionParameter: model.LogNormalRiskModel.RiskAversionParameter,
			Tau:                   model.LogNormalRiskModel.Tau,
			Params: LogNormalModelParams{
				Mu:    model.LogNormalRiskModel.Params.Mu,
				R:     model.LogNormalRiskModel.Params.R,
				Sigma: model.LogNormalRiskModel.Params.Sigma,
			},
		},
	}
}

func tradableInstrumentLogNormalRiskModelToProto(model TradableInstrumentLogNormalRiskModel) *vega.TradableInstrument_LogNormalRiskModel {
	return &vega.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &vega.LogNormalRiskModel{
			RiskAversionParameter: model.LogNormalRiskModel.RiskAversionParameter,
			Tau:                   model.LogNormalRiskModel.Tau,
			Params: &vega.LogNormalModelParams{
				Mu:    model.LogNormalRiskModel.Params.Mu,
				R:     model.LogNormalRiskModel.Params.R,
				Sigma: model.LogNormalRiskModel.Params.Sigma,
			},
		},
	}
}

func (rm TradableInstrumentSimpleRiskModel) RiskModel() {}

func (rm *TradableInstrumentSimpleRiskModel) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("received unexpected data for TradableInstrumentSimpleRiskModel")
	}
	return json.Unmarshal(b, &rm)
}

func (rm TradableInstrumentLogNormalRiskModel) RiskModel() {}

func (rm *TradableInstrumentLogNormalRiskModel) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("received unexpected data for TradableInstrumentLogNormalRiskModel")
	}
	return json.Unmarshal(b, &rm)
}

type SimpleRiskModel struct {
	Params SimpleModelParams `json:"params"`
}

type SimpleModelParams struct {
	FactorLong           float64 `json:"factorLong"`
	FactorShort          float64 `json:"factorShort"`
	MaxMoveUp            float64 `json:"maxMoveUp"`
	MinMoveDown          float64 `json:"maxMoveDown"`
	ProbabilityOfTrading float64 `json:"probabilityOfTrading"`
}

type LogNormalRiskModel struct {
	RiskAversionParameter float64              `json:"RiskAversionParameter"`
	Tau                   float64              `json:"tau"`
	Params                LogNormalModelParams `json:"params"`
}

type LogNormalModelParams struct {
	Mu    float64 `json:"mu"`
	R     float64 `json:"r"`
	Sigma float64 `json:"sigma"`
}

type ScalingFactors struct {
	SearchLevel       float64 `json:"searchLevel"`
	InitialMargin     float64 `json:"initialMargin"`
	CollateralRelease float64 `json:"collateralRelease"`
}

type Product interface {
	Product()
}

type InstrumentFuture struct {
	Future Future `json:"future"`
}

func (i InstrumentFuture) Product() {}

func instrumentFutureFromProto(instrument *vega.Instrument_Future) InstrumentFuture {
	return InstrumentFuture{
		Future: futureFromProto(instrument.Future),
	}
}

func instrumentFutureToProto(instrument InstrumentFuture) *vega.Instrument_Future {
	return &vega.Instrument_Future{
		Future: futureToProto(instrument.Future),
	}
}

type Future struct {
	SettlementAsset                 string                    `json:"settlementAsset"`
	QuoteName                       string                    `json:"quoteName"`
	OracleSpecForSettlementPrice    OracleSpec                `json:"oracleSpecForSettlementPrice"`
	OracleSpecForTradingTermination OracleSpec                `json:"oracleSpecForTradingTermination"`
	OracleSpecBinding               OracleSpecToFutureBinding `json:"oracleSpecBinding"`
}

func futureFromProto(future *vega.Future) Future {
	return Future{
		SettlementAsset:                 future.SettlementAsset,
		QuoteName:                       future.QuoteName,
		OracleSpecForSettlementPrice:    oracleSpecFromProto(future.OracleSpecForSettlementPrice),
		OracleSpecForTradingTermination: oracleSpecFromProto(future.OracleSpecForTradingTermination),
		OracleSpecBinding: OracleSpecToFutureBinding{
			SettlementPriceProperty:    future.OracleSpecBinding.SettlementPriceProperty,
			TradingTerminationProperty: future.OracleSpecBinding.TradingTerminationProperty,
		},
	}
}

func futureToProto(future Future) *vega.Future {
	return &vega.Future{
		SettlementAsset:                 future.SettlementAsset,
		QuoteName:                       future.QuoteName,
		OracleSpecForSettlementPrice:    oracleSpecToProto(future.OracleSpecForSettlementPrice),
		OracleSpecForTradingTermination: oracleSpecToProto(future.OracleSpecForTradingTermination),
		OracleSpecBinding:               &vega.OracleSpecToFutureBinding{},
	}
}

type OracleSpec struct {
	Id        string               `json:"id"`
	CreatedAt int64                `json:"createdAt"`
	UpdatedAt int64                `json:"updatedAt"`
	PubKeys   []string             `json:"pubKeys"`
	Filters   []Filter             `json:"filters"`
	Status    v1.OracleSpec_Status `json:"status"`
}

func oracleSpecFromProto(spec *v1.OracleSpec) OracleSpec {
	return OracleSpec{
		Id:        spec.Id,
		CreatedAt: spec.CreatedAt,
		UpdatedAt: spec.UpdatedAt,
		PubKeys:   spec.PubKeys,
		Filters:   filtersFromProto(spec.Filters),
		Status:    spec.Status,
	}
}

func oracleSpecToProto(spec OracleSpec) *v1.OracleSpec {
	return &v1.OracleSpec{
		Id:        spec.Id,
		CreatedAt: spec.CreatedAt,
		UpdatedAt: spec.UpdatedAt,
		PubKeys:   spec.PubKeys,
		Filters:   filtersToProto(spec.Filters),
		Status:    spec.Status,
	}
}

func (spec OracleSpec) ToProto() *v1.OracleSpec {
	return &v1.OracleSpec{
		Id:        spec.Id,
		CreatedAt: spec.CreatedAt,
		UpdatedAt: spec.UpdatedAt,
		PubKeys:   spec.PubKeys,
		Filters:   filtersToProto(spec.Filters),
		Status:    0,
	}
}

func filtersFromProto(filters []*v1.Filter) []Filter {
	if len(filters) == 0 {
		return nil
	}

	results := make([]Filter, 0, len(filters))
	for _, filter := range filters {
		conditions := make([]Condition, 0, len(filter.Conditions))

		for _, condition := range filter.Conditions {
			conditions = append(conditions, Condition{
				Operator: condition.Operator,
				Value:    condition.Value,
			})
		}

		results = append(results, Filter{
			Key: PropertyKey{
				Name: filter.Key.Name,
				Type: filter.Key.Type,
			},
			Conditions: conditions,
		})
	}

	return results
}

func filtersToProto(filters []Filter) []*v1.Filter {
	if len(filters) == 0 {
		return nil
	}

	results := make([]*v1.Filter, 0, len(filters))
	for _, filter := range filters {
		conditions := make([]*v1.Condition, 0, len(filter.Conditions))
		for _, condition := range filter.Conditions {
			conditions = append(conditions, &v1.Condition{
				Operator: condition.Operator,
				Value:    condition.Value,
			})
		}

		results = append(results, &v1.Filter{
			Key: &v1.PropertyKey{
				Name: filter.Key.Name,
				Type: filter.Key.Type,
			},
			Conditions: conditions,
		})
	}

	return results
}

type OracleSpecToFutureBinding struct {
	SettlementPriceProperty    string `json:"settlementPriceProperty"`
	TradingTerminationProperty string `json:"tradingTerminationProperty"`
}

type Filter struct {
	Key        PropertyKey `json:"key"`
	Conditions []Condition `json:"conditions"`
}

type PropertyKey struct {
	Name string `json:"name"`
	Type v1.PropertyKey_Type
}

type Condition struct {
	Operator v1.Condition_Operator
	Value    string
}

type InstrumentMetadata struct {
	Tags []string
}
