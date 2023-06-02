package entities

import (
	"code.vegaprotocol.io/vega/core/types"
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
			signers, err := SerializeSigners(types.SignersFromProto(tp.GetSigners()))
			if err != nil {
				return nil, err
			}
			ds.Signers = signers
			ds.Filters = FiltersFromProto(tp.GetFilters())
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
			signers, err = SerializeSigners(types.SignersFromProto(tp.GetSigners()))
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
			filters = FiltersFromProto(tp.GetFilters())
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
