package entities

import (
	"fmt"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/protos/vega"

	"google.golang.org/protobuf/encoding/protojson"
)

type DataSourceDefinition struct {
	*vega.DataSourceDefinition
}

func (s DataSourceDefinition) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(s)
}

func (s *DataSourceDefinition) UnmarshalJSON(b []byte) error {
	s.DataSourceDefinition = &vega.DataSourceDefinition{}
	return protojson.Unmarshal(b, s)
}

func (s *DataSourceDefinition) GetOracle() (*DataSourceSpecConfiguration, error) {
	ds := &DataSourceSpecConfiguration{
		Signers: Signers{},
		Filters: []Filter{},
	}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.DataSourceSpecConfiguration:
			signers, err := SerializeSigners(dstypes.SignersFromProto(tp.GetSigners()))
			if err != nil {
				return nil, err
			}
			ds.Signers = signers
			ds.Filters = FiltersFromProto(tp.GetFilters())
		}
	}

	return ds, nil
}

func (s *DataSourceDefinition) GetEthOracle() (*EthCallSpec, error) {
	ds := &EthCallSpec{}
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.EthCallSpec:
			ds.Address = tp.Address
			abi := tp.GetAbi()
			ds.Abi = []byte(abi)
			ds.Method = tp.Method
			args := tp.GetArgs()
			for _, arg := range args {
				jsonArg, err := arg.MarshalJSON()
				if err != nil {
					return nil, err // TODO: Fix all of the errors
				}
				ds.ArgsJson = append(ds.ArgsJson, string(jsonArg))
			}
			trigger, err := ethcallcommon.TriggerFromProto(tp.Trigger)
			if err != nil {
				return nil, fmt.Errorf("failed to get trigger from proto: %w", err)
			}
			ds.Trigger = EthCallTrigger{Trigger: trigger}
			ds.RequiredConfirmations = tp.RequiredConfirmations
			ds.Filters = s.GetFilters()
		}
	}

	return ds, nil
}

func (s *DataSourceDefinition) GetInternalTimeTrigger() *DataSourceSpecConfigurationTime {
	ds := &DataSourceSpecConfigurationTime{
		Conditions: []Condition{},
	}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.DataSourceSpecConfigurationTime:
			for _, c := range tp.Conditions {
				ds.Conditions = append(ds.Conditions, ConditionFromProto(c))
			}
		}
	}

	return ds
}

func (s *DataSourceDefinition) GetSigners() (Signers, error) {
	signers := Signers{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.DataSourceSpecConfiguration:
			var err error
			signers, err = SerializeSigners(dstypes.SignersFromProto(tp.GetSigners()))
			if err != nil {
				return nil, err
			}
		}
	}

	return signers, nil
}

func (s *DataSourceDefinition) GetFilters() []Filter {
	filters := []Filter{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.DataSourceSpecConfiguration:
			filters = FiltersFromProto(tp.Filters)
		case *vega.EthCallSpec:
			filters = FiltersFromProto(tp.Filters)
		}
	}

	return filters
}

func (s *DataSourceDefinition) GetConditions() []Condition {
	conditions := []Condition{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *vega.DataSourceSpecConfigurationTime:
			for _, c := range tp.Conditions {
				conditions = append(conditions, ConditionFromProto(c))
			}
		}
	}

	return conditions
}

func DataSourceDefinitionFromProto(dsp *vega.DataSourceDefinition) DataSourceDefinition {
	return DataSourceDefinition{dsp}
}

// DataSourceSpecConfiguration is a simplified version of the oracle content.
// In the future it is intended to be part of an interface, not a hardcoded objcet.
type DataSourceSpecConfiguration struct {
	Signers Signers
	Filters []Filter
}

type EthCallTrigger struct {
	ethcallcommon.Trigger
}

type EthCallSpec struct {
	Address               string
	Abi                   []byte
	Method                string
	ArgsJson              []string
	Trigger               EthCallTrigger
	RequiredConfirmations uint64
	Filters               []Filter
}

func (es *EthCallSpec) GetFilters() []Filter {
	if es != nil {
		return es.Filters
	}

	return []Filter{}
}

func (es *EthCallSpec) GetAddress() string {
	if es != nil {
		return es.Address
	}

	return ""
}

func (es *EthCallSpec) GetAbi() []byte {
	if es != nil {
		return es.Abi
	}

	return nil
}

func (es *EthCallSpec) GetMethod() string {
	if es != nil {
		return es.Method
	}

	return ""
}

func (es *EthCallSpec) GetArgs() []string {
	if es != nil {
		return es.ArgsJson
	}

	return []string{}
}

func (es *EthCallSpec) GetTrigger() EthCallTrigger {
	if es != nil {
		return es.Trigger
	}

	return EthCallTrigger{}
}

func (es *EthCallSpec) GetRequiredConfirmations() uint64 {
	if es != nil {
		return es.RequiredConfirmations
	}

	return uint64(0)
}

// DataSourceSpecConfigurationTime is a simplified version of the internal time
// termination data source; only for internal use;
// New internal types will be created for Cosmic Elevator new internal terminations.
type DataSourceSpecConfigurationTime struct {
	Conditions []Condition
}

func (ds *DataSourceSpecConfiguration) GetSigners() Signers {
	if ds != nil {
		return ds.Signers
	}
	return Signers{}
}

func (ds *DataSourceSpecConfiguration) GetFilters() []Filter {
	if ds != nil {
		return ds.Filters
	}
	return []Filter{}
}

func (d *DataSourceSpecConfigurationTime) GetConditions() []Condition {
	if d != nil {
		return d.Conditions
	}
	return []Condition{}
}
