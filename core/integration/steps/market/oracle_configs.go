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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/steps/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/oracle-config/*future.json
	defaultOracleConfigs         embed.FS
	defaultOracleConfigFileNames = []string{
		"defaults/oracle-config/default-eth-for-future.json",
		"defaults/oracle-config/default-usd-for-future.json",
		"defaults/oracle-config/default-dai-for-future.json",
	}
	//go:embed defaults/oracle-config/*perps.json
	defaultOraclePerpsConfigs         embed.FS
	defaultPerpsOracleConfigFileNames = []string{
		"defaults/oracle-config/default-eth-for-perps.json",
		"defaults/oracle-config/default-usd-for-perps.json",
		"defaults/oracle-config/default-dai-for-perps.json",
	}

	// swap out the oracle names for future with perp oracles.
	perpsSwapMapping = map[string]string{
		"default-eth-for-future": "default-eth-for-perps",
		"default-usd-for-future": "default-usd-for-perps",
		"default-dai-for-future": "default-dai-for-perps",
	}
)

type Binding interface {
	Descriptor() ([]byte, []int)
	GetSettlementDataProperty() string
	ProtoMessage()
}

type BindType interface {
	*vegapb.DataSourceSpecToFutureBinding | *vegapb.DataSourceSpecToPerpetualBinding
}

type oracleConfigs struct {
	futures               *oConfig[*vegapb.DataSourceSpecToFutureBinding]
	perps                 *oConfig[*vegapb.DataSourceSpecToPerpetualBinding]
	fullPerps             map[string]*vegapb.Perpetual
	fullFutures           map[string]*vegapb.Future
	perpSwap              bool
	compositePriceOracles map[string]CompositePriceOracleConfig
	timeTriggers          map[string]*vegapb.DataSourceSpec
}

type oConfig[T BindType] struct {
	configForSettlementData    map[string]*OracleConfig[T]
	configFoTradingTermination map[string]*OracleConfig[T]
	configForSchedule          map[string]*OracleConfig[T]
	settlementDataDecimals     map[string]uint32
}

type OracleConfig[T BindType] struct {
	Spec    *vegapb.OracleSpec
	Binding T
}

type CompositePriceOracleConfig struct {
	Spec    *vegapb.OracleSpec
	Binding *vegapb.SpecBindingForCompositePrice
}

func newOracleSpecs(unmarshaler *defaults.Unmarshaler) *oracleConfigs {
	configs := &oracleConfigs{
		futures:               newOConfig[*vegapb.DataSourceSpecToFutureBinding](),
		perps:                 newOConfig[*vegapb.DataSourceSpecToPerpetualBinding](),
		fullPerps:             map[string]*vegapb.Perpetual{},
		fullFutures:           map[string]*vegapb.Future{},
		compositePriceOracles: map[string]CompositePriceOracleConfig{},
		timeTriggers:          map[string]*vegapb.DataSourceSpec{},
	}
	configs.futureOracleSpecs(unmarshaler)
	configs.perpetualOracleSpecs(unmarshaler)
	return configs
}

func newOConfig[T BindType]() *oConfig[T] {
	return &oConfig[T]{
		configForSettlementData:    map[string]*OracleConfig[T]{},
		configFoTradingTermination: map[string]*OracleConfig[T]{},
		configForSchedule:          map[string]*OracleConfig[T]{},
		settlementDataDecimals:     map[string]uint32{},
	}
}

func (c *oracleConfigs) SwapToPerps() {
	c.perpSwap = true
}

func (c *oracleConfigs) CheckName(name string) string {
	if !c.perpSwap {
		return name
	}
	if alt, ok := perpsSwapMapping[name]; ok {
		return alt
	}
	return name
}

func (c *oracleConfigs) futureOracleSpecs(unmarshaler *defaults.Unmarshaler) {
	contentReaders := helpers.ReadAll(defaultOracleConfigs, defaultOracleConfigFileNames)
	for name, contentReader := range contentReaders {
		future, err := unmarshaler.UnmarshalDataSourceConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default data source config %s: %v", name, err))
		}
		if err := c.AddFuture(name, future); err != nil {
			panic(fmt.Errorf("failed to add default data source config %s: %v", name, err))
		}
	}
}

func (c *oracleConfigs) perpetualOracleSpecs(unmarshaler *defaults.Unmarshaler) {
	contentReaders := helpers.ReadAll(defaultOraclePerpsConfigs, defaultPerpsOracleConfigFileNames)
	for name, contentReader := range contentReaders {
		perp, err := unmarshaler.UnmarshalPerpsDataSourceConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default data source config %s: %v", name, err))
		}
		if err := c.AddPerp(name, perp); err != nil {
			panic(fmt.Errorf("failed to add default data source config %s: %v", name, err))
		}
	}
}

func (c *oracleConfigs) SetSettlementDataDP(name string, decimals uint32) {
	// for now, wing it and set for both
	c.futures.SetSettlementDataDP(name, decimals)
	c.perps.SetSettlementDataDP(name, decimals)
}

func (f *oConfig[T]) SetSettlementDataDP(name string, decimals uint32) {
	f.settlementDataDecimals[name] = decimals
}

func (c *oracleConfigs) GetSettlementDataDP(name string) uint32 {
	fd, ok := c.futures.GetSettlementDataDP(name)
	if !ok {
		fd, _ = c.perps.GetSettlementDataDP(name)
	}
	return fd
}

func (f *oConfig[T]) GetSettlementDataDP(name string) (uint32, bool) {
	dp, ok := f.settlementDataDecimals[name]
	if ok {
		return dp, ok
	}
	return 0, ok
}

func (c *oracleConfigs) AddFuture(name string, future *vegapb.Future) error {
	if err := c.futures.Add(name, "settlement data", future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding); err != nil {
		return err
	}
	if err := c.futures.Add(name, "trading termination", future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding); err != nil {
		return err
	}
	c.fullFutures[name] = future
	return nil
}

func (c *oracleConfigs) AddCompositePriceOracle(name string, spec *vegapb.DataSourceSpec, binding *vegapb.SpecBindingForCompositePrice) {
	c.compositePriceOracles[name] = CompositePriceOracleConfig{
		Spec: &vegapb.OracleSpec{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: spec,
			},
		},
		Binding: binding,
	}
}

func (c *oracleConfigs) AddPerp(name string, perp *vegapb.Perpetual) error {
	if err := c.perps.Add(name, "settlement data", perp.DataSourceSpecForSettlementData, perp.DataSourceSpecBinding); err != nil {
		return err
	}
	if err := c.perps.Add(name, "settlement schedule", perp.DataSourceSpecForSettlementSchedule, perp.DataSourceSpecBinding); err != nil {
		return err
	}
	c.fullPerps[name] = perp
	return nil
}

func (c *oracleConfigs) AddTimeTrigger(name string, spec *vegapb.DataSourceSpec) error {
	c.timeTriggers[name] = spec
	return nil
}

func (c *oracleConfigs) Add(name, specType string, spec *vegapb.DataSourceSpec, binding Binding) error {
	switch bt := binding.(type) {
	case *vegapb.DataSourceSpecToPerpetualBinding:
		return c.perps.Add(name, specType, spec, bt)
	case *vegapb.DataSourceSpecToFutureBinding:
		return c.futures.Add(name, specType, spec, bt)
	default:
		panic("unsupported binding type")
	}
	return nil
}

func (f *oConfig[T]) Add(
	name string,
	specType string,
	spec *vegapb.DataSourceSpec,
	binding T,
) error {
	if specType == "settlement data" {
		f.configForSettlementData[name] = &OracleConfig[T]{
			Spec: &vegapb.OracleSpec{
				ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
					Spec: spec,
				},
			},
			Binding: binding,
		}
		for _, filter := range spec.GetData().GetFilters() {
			if filter.Key.NumberDecimalPlaces != nil {
				f.settlementDataDecimals[name] = uint32(*filter.Key.NumberDecimalPlaces)
				break
			}
		}
	} else if specType == "trading termination" {
		f.configFoTradingTermination[name] = &OracleConfig[T]{
			Spec: &vegapb.OracleSpec{
				ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
					Spec: spec,
				},
			},
			Binding: binding,
		}
	} else if specType == "settlement schedule" {
		f.configForSchedule[name] = &OracleConfig[T]{
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

func (c *oracleConfigs) GetFullInstrument(name string) (*vegapb.Instrument, error) {
	var instrument vegapb.Instrument
	if fut, ok := c.fullFutures[name]; ok {
		instrument.Product = &vegapb.Instrument_Future{
			Future: fut,
		}
		return &instrument, nil
	}
	if perp, ok := c.fullPerps[name]; ok {
		instrument.Product = &vegapb.Instrument_Perpetual{
			Perpetual: perp,
		}
		return &instrument, nil
	}
	return nil, fmt.Errorf("no products with name %s found", name)
}

func (c *oracleConfigs) GetOracleDefinitionForCompositePrice(name string) (*vegapb.OracleSpec, *vegapb.SpecBindingForCompositePrice, error) {
	if oc, found := c.compositePriceOracles[name]; !found {
		return nil, nil, fmt.Errorf("oracle config for composite price not found")
	} else {
		return oc.Spec, oc.Binding, nil
	}
}

func (c *oracleConfigs) GetFuture(name, specType string) (*OracleConfig[*vegapb.DataSourceSpecToFutureBinding], error) {
	return c.futures.Get(name, specType)
}

func (c *oracleConfigs) GetPerps(name, specType string) (*OracleConfig[*vegapb.DataSourceSpecToPerpetualBinding], error) {
	return c.perps.Get(name, specType)
}

func (c *oracleConfigs) GetFullPerp(name string) (*vegapb.Perpetual, error) {
	// if we're swapping to perps, ensure we have the correct oracle name
	if c.perpSwap {
		name = c.CheckName(name)
	}
	perp, ok := c.fullPerps[name]
	if !ok {
		return nil, fmt.Errorf("perpetual product with name %s not found", name)
	}
	copyConfig := &vegapb.Perpetual{}
	if err := copier.Copy(copyConfig, perp); err != nil {
		panic(fmt.Errorf("failed to deep copy oracle config: %v", err))
	}
	return copyConfig, nil
}

func (c *oracleConfigs) GetFullFuture(name string) (*vegapb.Future, error) {
	future, ok := c.fullFutures[name]
	if !ok {
		return nil, fmt.Errorf("future product with name %s not found", name)
	}
	copyConfig := &vegapb.Future{}
	if err := copier.Copy(copyConfig, future); err != nil {
		panic(fmt.Errorf("failed to deep copy oracle config: %v", err))
	}
	return copyConfig, nil
}

func (f *oConfig[T]) Get(name string, specType string) (*OracleConfig[T], error) {
	var cfg map[string]*OracleConfig[T]

	if specType == "settlement data" {
		cfg = f.configForSettlementData
	} else if specType == "trading termination" {
		cfg = f.configFoTradingTermination
	} else if specType == "settlement schedule" {
		cfg = f.configForSchedule
	} else {
		return nil, errors.New("unknown oracle spec type definition - expecting settlement data or trading termination")
	}

	config, ok := cfg[name]
	if !ok {
		return config, fmt.Errorf("no oracle spec \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &OracleConfig[T]{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy oracle config: %v", err))
	}
	return copyConfig, nil
}

func (c *oracleConfigs) GetTimeTrigger(name string) (*vegapb.DataSourceSpec, error) {
	ds, ok := c.timeTriggers[name]
	if !ok {
		return nil, fmt.Errorf("oracle name not found")
	}
	return ds, nil
}
