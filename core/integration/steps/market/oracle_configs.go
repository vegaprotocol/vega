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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
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
	futures     *oConfig[*vegapb.DataSourceSpecToFutureBinding]
	perps       *oConfig[*vegapb.DataSourceSpecToPerpetualBinding]
	fullPerps   map[string]*vegapb.Perpetual
	fullFutures map[string]*vegapb.Future
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

func newOracleSpecs(unmarshaler *defaults.Unmarshaler) *oracleConfigs {
	configs := &oracleConfigs{
		futures:     newOConfig[*vegapb.DataSourceSpecToFutureBinding](),
		perps:       newOConfig[*vegapb.DataSourceSpecToPerpetualBinding](),
		fullPerps:   map[string]*vegapb.Perpetual{},
		fullFutures: map[string]*vegapb.Future{},
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

func (c *oracleConfigs) futureOracleSpecs(unmarshaler *defaults.Unmarshaler) {
	contentReaders := defaults.ReadAll(defaultOracleConfigs, defaultOracleConfigFileNames)
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
	contentReaders := defaults.ReadAll(defaultOraclePerpsConfigs, defaultPerpsOracleConfigFileNames)
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

// *vegapb.DataSourceSpecToFutureBinding | *vegapb.DataSourceSpecToPerpetualBinding
func (c *oracleConfigs) GetFuture(name, specType string) (*OracleConfig[*vegapb.DataSourceSpecToFutureBinding], error) {
	return c.futures.Get(name, specType)
}

func (c *oracleConfigs) GetPerps(name, specType string) (*OracleConfig[*vegapb.DataSourceSpecToPerpetualBinding], error) {
	return c.perps.Get(name, specType)
}

func (c *oracleConfigs) GetFullPerp(name string) (*vegapb.Perpetual, error) {
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
