package market

import (
	"embed"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	//go:embed defaults/risk-model/simple/*.json
	defaultSimpleRiskModels         embed.FS
	defaultSimpleRiskModelFileNames = []string{
		"defaults/risk-model/simple/default-simple-risk-model.json",
		"defaults/risk-model/simple/default-simple-risk-model-2.json",
		"defaults/risk-model/simple/default-simple-risk-model-3.json",
		"defaults/risk-model/simple/default-simple-risk-model-4.json",
	}

	//go:embed defaults/risk-model/log-normal/*.json
	defaultLogNormalRiskModels         embed.FS
	defaultLogNormalRiskModelFileNames = []string{
		"defaults/risk-model/log-normal/default-log-normal-risk-model.json",
	}
)

type riskModels struct {
	simple    map[string]*types.TradableInstrument_SimpleRiskModel
	logNormal map[string]*types.TradableInstrument_LogNormalRiskModel
}

func newRiskModels(unmarshaler *defaults.Unmarshaler) *riskModels {
	models := &riskModels{
		simple:    map[string]*types.TradableInstrument_SimpleRiskModel{},
		logNormal: map[string]*types.TradableInstrument_LogNormalRiskModel{},
	}

	simpleRiskModelReaders := defaults.ReadAll(defaultSimpleRiskModels, defaultSimpleRiskModelFileNames)
	for name, contentReader := range simpleRiskModelReaders {
		instrument, err := unmarshaler.UnmarshalRiskModel(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default risk model %s: %v", name, err))
		}
		if err := models.AddSimple(name, instrument.RiskModel.(*types.TradableInstrument_SimpleRiskModel)); err != nil {
			panic(fmt.Errorf("failed to add default simple risk model %s: %v", name, err))
		}
	}

	logNormalRiskModelReaders := defaults.ReadAll(defaultLogNormalRiskModels, defaultLogNormalRiskModelFileNames)
	for name, contentReader := range logNormalRiskModelReaders {
		instrument, err := unmarshaler.UnmarshalRiskModel(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default risk model %s: %v", name, err))
		}
		if err := models.AddLogNormal(name, instrument.RiskModel.(*types.TradableInstrument_LogNormalRiskModel)); err != nil {
			panic(fmt.Errorf("failed to add default simple risk model %s: %v", name, err))
		}
	}

	return models
}

func (r *riskModels) AddSimple(name string, model *types.TradableInstrument_SimpleRiskModel) error {
	_, okLogNormal := r.logNormal[name]
	if okLogNormal {
		return fmt.Errorf("risk model \"%s\" already registered as log normal risk model", name)
	}
	r.simple[name] = model
	return nil
}

func (r *riskModels) AddLogNormal(name string, model *types.TradableInstrument_LogNormalRiskModel) error {
	_, okSimple := r.simple[name]
	if okSimple {
		return fmt.Errorf("risk model \"%s\" already registered as simple risk model", name)
	}
	r.logNormal[name] = model
	return nil
}

func (r riskModels) LoadModel(name string, instrument *types.TradableInstrument) error {
	simpleModel, okSimple := r.simple[name]
	if okSimple {
		// Copy to avoid modification between tests.
		copyConfig := &types.TradableInstrument_SimpleRiskModel{}
		if err := copier.Copy(copyConfig, simpleModel); err != nil {
			panic(fmt.Errorf("failed to deep copy simple risk model: %v", err))
		}
		instrument.RiskModel = copyConfig
		return nil
	}

	logNormalModel, okLogNormal := r.logNormal[name]
	if okLogNormal {
		// Copy to avoid modification between tests.
		copyConfig := &types.TradableInstrument_LogNormalRiskModel{}
		if err := copier.Copy(copyConfig, logNormalModel); err != nil {
			panic(fmt.Errorf("failed to deep copy log normal risk model: %v", err))
		}
		instrument.RiskModel = logNormalModel
		return nil
	}

	return fmt.Errorf("no risk model \"%s\" registered", name)
}
