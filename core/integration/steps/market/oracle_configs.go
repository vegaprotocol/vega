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

package market

import (
	"embed"
	"errors"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	//go:embed defaults/oracle-config/*.json
	defaultOracleConfigs         embed.FS
	defaultOracleConfigFileNames = []string{
		"defaults/oracle-config/default-eth-for-future.json",
		"defaults/oracle-config/default-usd-for-future.json",
		"defaults/oracle-config/default-dai-for-future.json",
		"defaults/oracle-config/default-eth-for-perps.json",
	}
)

type oracleConfigs struct {
	configForSettlementData    map[string]*OracleConfig
	configFoTradingTermination map[string]*OracleConfig
	settlementDataDecimals     map[string]uint32
}

type OracleConfig struct {
	Spec    *vegapb.OracleSpec
	Binding *types.DataSourceSpecToFutureBinding
}

func newOracleSpecs(unmarshaler *defaults.Unmarshaler) *oracleConfigs {
	specs := &oracleConfigs{
		configForSettlementData:    map[string]*OracleConfig{},
		configFoTradingTermination: map[string]*OracleConfig{},
		settlementDataDecimals:     map[string]uint32{},
	}

	contentReaders := defaults.ReadAll(defaultOracleConfigs, defaultOracleConfigFileNames)
	for name, contentReader := range contentReaders {
		future, err := unmarshaler.UnmarshalDataSourceConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default data source config %s: %v", name, err))
		}
		if err := specs.Add(name, "settlement data", future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding); err != nil {
			panic(fmt.Errorf("failed to add default data source config %s: %v", name, err))
		}
		if err := specs.Add(name, "trading termination", future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding); err != nil {
			panic(fmt.Errorf("failed to add default data source config %s: %v", name, err))
		}
	}

	return specs
}

func (f *oracleConfigs) SetSettlementDataDP(name string, decimals uint32) {
	f.settlementDataDecimals[name] = decimals
}

func (f *oracleConfigs) GetSettlementDataDP(name string) uint32 {
	dp, ok := f.settlementDataDecimals[name]
	if ok {
		return dp
	}
	return 0
}

func (f *oracleConfigs) Add(
	name string,
	specType string,
	spec *vegapb.DataSourceSpec,
	binding *types.DataSourceSpecToFutureBinding,
) error {
	if specType == "settlement data" {
		f.configForSettlementData[name] = &OracleConfig{
			Spec: &vegapb.OracleSpec{
				ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
					Spec: spec,
				},
			},
			Binding: binding,
		}
	} else if specType == "trading termination" {
		f.configFoTradingTermination[name] = &OracleConfig{
			Spec: &vegapb.OracleSpec{
				ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
					Spec: spec,
				},
			},
			Binding: binding,
		}
	} else {
		return errors.New("unknown oracle spec type definition - expecting settlement data or trading termination")
	}

	return nil
}

func (f *oracleConfigs) Get(name string, specType string) (*OracleConfig, error) {
	var cfg map[string]*OracleConfig

	if specType == "settlement data" {
		cfg = f.configForSettlementData
	} else if specType == "trading termination" {
		cfg = f.configFoTradingTermination
	} else {
		return nil, errors.New("unknown oracle spec type definition - expecting settlement data or trading termination")
	}

	config, ok := cfg[name]
	if !ok {
		return config, fmt.Errorf("no oracle spec \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &OracleConfig{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy oracle config: %v", err))
	}
	return copyConfig, nil
}
