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

package market

import (
	"embed"
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/steps/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/risk-model/simple/*.json
	defaultSimpleRiskModels         embed.FS
	defaultSimpleRiskModelFileNames = []string{
		"defaults/risk-model/simple/default-simple-risk-model.json",
		"defaults/risk-model/simple/default-simple-risk-model-2.json",
		"defaults/risk-model/simple/default-simple-risk-model-3.json",
		"defaults/risk-model/simple/default-simple-risk-model-4.json",
		"defaults/risk-model/simple/system-test-risk-model.json",
	}

	//go:embed defaults/risk-model/log-normal/*.json
	defaultLogNormalRiskModels         embed.FS
	defaultLogNormalRiskModelFileNames = []string{
		"defaults/risk-model/log-normal/default-log-normal-risk-model.json",
		"defaults/risk-model/log-normal/default-st-risk-model.json",
		"defaults/risk-model/log-normal/closeout-st-risk-model.json",
	}
)

type riskModels struct {
	simple    map[string]*vegapb.TradableInstrument_SimpleRiskModel
	logNormal map[string]*vegapb.TradableInstrument_LogNormalRiskModel
}

func newRiskModels(unmarshaler *defaults.Unmarshaler) *riskModels {
	models := &riskModels{
		simple:    map[string]*vegapb.TradableInstrument_SimpleRiskModel{},
		logNormal: map[string]*vegapb.TradableInstrument_LogNormalRiskModel{},
	}

	simpleRiskModelReaders := helpers.ReadAll(defaultSimpleRiskModels, defaultSimpleRiskModelFileNames)
	for name, contentReader := range simpleRiskModelReaders {
		instrument, err := unmarshaler.UnmarshalRiskModel(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default risk model %s: %v", name, err))
		}
		if err := models.AddSimple(name, instrument.RiskModel.(*vegapb.TradableInstrument_SimpleRiskModel)); err != nil {
			panic(fmt.Errorf("failed to add default simple risk model %s: %v", name, err))
		}
	}

	logNormalRiskModelReaders := helpers.ReadAll(defaultLogNormalRiskModels, defaultLogNormalRiskModelFileNames)
	for name, contentReader := range logNormalRiskModelReaders {
		instrument, err := unmarshaler.UnmarshalRiskModel(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default risk model %s: %v", name, err))
		}
		if err := models.AddLogNormal(name, instrument.RiskModel.(*vegapb.TradableInstrument_LogNormalRiskModel)); err != nil {
			panic(fmt.Errorf("failed to add default simple risk model %s: %v", name, err))
		}
	}

	return models
}

func (r *riskModels) AddSimple(name string, model *vegapb.TradableInstrument_SimpleRiskModel) error {
	if _, okLogNormal := r.logNormal[name]; okLogNormal {
		return fmt.Errorf("risk model \"%s\" already registered as log normal risk model", name)
	}
	r.simple[name] = model
	return nil
}

func (r *riskModels) AddLogNormal(name string, model *vegapb.TradableInstrument_LogNormalRiskModel) error {
	if _, okSimple := r.simple[name]; okSimple {
		return fmt.Errorf("risk model \"%s\" already registered as simple risk model", name)
	}
	r.logNormal[name] = model
	return nil
}

func (r riskModels) LoadModel(name string, instrument *vegapb.TradableInstrument) error {
	simpleModel, okSimple := r.simple[name]
	if okSimple {
		// Copy to avoid modification between tests.
		copyConfig := &vegapb.TradableInstrument_SimpleRiskModel{}
		if err := copier.Copy(copyConfig, simpleModel); err != nil {
			panic(fmt.Errorf("failed to deep copy simple risk model: %v", err))
		}
		instrument.RiskModel = copyConfig
		return nil
	}

	logNormalModel, okLogNormal := r.logNormal[name]
	if okLogNormal {
		// Copy to avoid modification between tests.
		copyConfig := &vegapb.TradableInstrument_LogNormalRiskModel{}
		if err := copier.Copy(copyConfig, logNormalModel); err != nil {
			panic(fmt.Errorf("failed to deep copy log normal risk model: %v", err))
		}
		instrument.RiskModel = logNormalModel
		return nil
	}

	return fmt.Errorf("no risk model \"%s\" registered", name)
}
