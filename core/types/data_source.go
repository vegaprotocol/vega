package types

import (
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// Some stuff to be tested with this message

var (
	// ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList is returned when filters with same key names exists inside a single list.
	ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList = errors.New("multiple keys with same name found in filter list")
	// ErrDataSourceSpecHasInvalidTimeCondition is returned when timestamp value is used with 'LessThan'
	// or 'LessThanOrEqual' condition operator value.
	ErrDataSourceSpecHasInvalidTimeCondition = errors.New("data source spec time value is used with 'less than' or 'less than equal' condition")
)

type dataSourceType interface {
	String() string
	DeepClone() dataSourceType
	ToDataSourceDefinitionProto() (*vegapb.DataSourceDefinition, error)
}

type DataSourceDefinition struct {
	dataSourceType
}

func NewDataSourceDefinitionWith(dst dataSourceType) *DataSourceDefinition {
	if dst == nil {
		return &DataSourceDefinition{}
	}
	return &DataSourceDefinition{
		dataSourceType: dst.DeepClone(),
	}
}

// NewDataSourceDefinition creates a new EMPTY DataSourceDefinition object.
// TODO: eth oracle type too.
func NewDataSourceDefinition(tp int) *DataSourceDefinition {
	ds := &DataSourceDefinition{}
	switch tp {
	case vegapb.DataSourceDefinitionTypeInt:
		return NewDataSourceDefinitionWith(
			DataSourceSpecConfigurationTime{
				Conditions: []*DataSourceSpecCondition{},
			})
	case vegapb.DataSourceDefinitionTypeExt:
		return NewDataSourceDefinitionWith(
			DataSourceSpecConfiguration{
				Signers: []*Signer{},
				Filters: []*DataSourceSpecFilter{},
			})
	}
	return ds
}

// IntoProto returns the proto object from DataSourceDefinition
// that is - vegapb.DataSourceDefinition that may have external or internal SourceType.
// Returns the whole proto object.
func (s *DataSourceDefinition) IntoProto() *vegapb.DataSourceDefinition {
	if s.dataSourceType == nil {
		return &vegapb.DataSourceDefinition{}
	}
	proto, err := s.ToDataSourceDefinitionProto()
	if err != nil {
		// TODO: bubble error
		return &vegapb.DataSourceDefinition{}
	}

	return proto
}

// DeepClone returns a clone of the DataSourceDefinition object.
func (s *DataSourceDefinition) DeepClone() DataSourceDefinition {
	return DataSourceDefinition{
		dataSourceType: s.dataSourceType.DeepClone(),
	}
}

func (s *DataSourceDefinition) Content() interface{} {
	return s.dataSourceType
}

// DataSourceDefinitionFromProto tries to build the DataSourceDfiniition object
// from the given proto object.
func DataSourceDefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) *DataSourceDefinition {
	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinition_External:
				dst, err := dataSourceDefinitionExternalFromProto(tp.External)
				if err != nil {
					// todo: bubble error
					return &DataSourceDefinition{}
				}
				return &DataSourceDefinition{dataSourceType: dst}
			case *vegapb.DataSourceDefinition_Internal:
				dst, err := dataSourceDefinitionInternalFromProto(tp.Internal)
				if err != nil {
					// todo: bubble error
					return &DataSourceDefinition{}
				}
				return NewDataSourceDefinitionWith(dst)
			}
		}
	}

	return &DataSourceDefinition{}
}

// GetSigners tries to get the signers from the DataSourceDefinition if they exist.
func (s *DataSourceDefinition) GetSigners() []*Signer {
	signers := []*Signer{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case DataSourceSpecConfiguration:
			signers = tp.Signers
		}
	}

	return signers
}

// GetFilters tries to get the filters from the DataSourceDefinition if they exist.
func (s *DataSourceDefinition) GetFilters() []*DataSourceSpecFilter {
	filters := []*DataSourceSpecFilter{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case DataSourceSpecConfiguration:
			filters = tp.Filters

		case DataSourceSpecConfigurationTime:
			// For the case the internal data source is time based
			// (as of OT https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed)
			// We add the filter key values manually to match a time based data source
			// Ensure only a single filter has been created, that holds the first condition
			if len(tp.Conditions) > 0 {
				filters = append(
					filters,
					&DataSourceSpecFilter{
						Key: &DataSourceSpecPropertyKey{
							Name: "vegaprotocol.builtin.timestamp",
							Type: datapb.PropertyKey_TYPE_TIMESTAMP,
						},
						Conditions: []*DataSourceSpecCondition{
							tp.Conditions[0],
						},
					},
				)
			}
		}
	}

	return filters
}

// GetDataSourceSpecConfiguration returns the base object - DataSourceSpecConfiguration
// from the DataSourceDefinition.
func (s *DataSourceDefinition) GetDataSourceSpecConfiguration() DataSourceSpecConfiguration {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case DataSourceSpecConfiguration:
			return tp
		}
	}

	return DataSourceSpecConfiguration{}
}

// UpdateFilters updates the DataSourceDefinition Filters.
func (s *DataSourceDefinition) UpdateFilters(filters []*DataSourceSpecFilter) error {
	fTypeCheck := map[*DataSourceSpecFilter]struct{}{}
	fNameCheck := map[string]struct{}{}
	for _, f := range filters {
		if _, ok := fTypeCheck[f]; ok {
			return ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList
		}
		if f.Key != nil {
			if _, ok := fNameCheck[f.Key.Name]; ok {
				return ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList
			}
			fNameCheck[f.Key.Name] = struct{}{}
		}
		fTypeCheck[f] = struct{}{}
	}

	// maybe todo - enforce that it's never nil
	if s.dataSourceType == nil {
		return nil
	}

	switch content := s.dataSourceType.DeepClone().(type) {
	case DataSourceSpecConfiguration:
		content.Filters = filters
		s.dataSourceType = content
	case DataSourceSpecConfigurationTime:
		// The data source definition is an internal time based source
		// For this case we take only the first item from the list of filters
		// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
		c := []*DataSourceSpecCondition{}
		if len(filters) > 0 {
			if len(filters[0].Conditions) > 0 {
				c = append(c, filters[0].Conditions[0])
			}
		}
		content.Conditions = c
		s.dataSourceType = content
	default:
		return fmt.Errorf("unable to set filters on data source type: %T", content)
	}
	return nil
}

func (s *DataSourceDefinition) SetFilterDecimals(d uint64) *DataSourceDefinition {
	switch content := s.dataSourceType.DeepClone().(type) {
	case DataSourceSpecConfiguration:
		for i := range content.Filters {
			content.Filters[i].Key.NumberDecimalPlaces = &d
		}
		s.dataSourceType = content
	default:
		// we should really be returning an error here but this method is only used in the integration tests
		panic(fmt.Sprintf("unable to set filter decimals on data source type: %T", content))
	}
	return s
}

func (s *DataSourceDefinition) ToDataSourceSpec() *DataSourceSpec {
	bytes, _ := proto.Marshal(s.IntoProto())
	specID := hex.EncodeToString(crypto.Hash(bytes))
	return &DataSourceSpec{
		ID:   specID,
		Data: s,
	}
}

func (s *DataSourceDefinition) ToExternalDataSourceSpec() *ExternalDataSourceSpec {
	return &ExternalDataSourceSpec{
		Spec: s.ToDataSourceSpec(),
	}
}

// SetOracleConfig sets a given oracle config in the receiver.
// If the receiver is not external oracle type data source - it is not changed.
// This method does not care about object previous contents.
func (s *DataSourceDefinition) SetOracleConfig(oc *DataSourceSpecConfiguration) *DataSourceDefinition {
	if _, ok := s.dataSourceType.(DataSourceSpecConfiguration); ok {
		s.dataSourceType = oc.DeepClone()
	}

	return s
}

// SetTimeTriggerConditionConfig sets a given conditions config in the receiver.
// If the receiver is not a time triggered data source - it does not set anything to it.
// This method does not care about object previous contents.
func (s *DataSourceDefinition) SetTimeTriggerConditionConfig(c []*DataSourceSpecCondition) *DataSourceDefinition {
	if _, ok := s.dataSourceType.(DataSourceSpecConfigurationTime); ok {
		s.dataSourceType = DataSourceSpecConfigurationTime{
			Conditions: c,
		}
	}
	return s
}

func (s *DataSourceDefinition) IsExternal() (bool, error) {
	switch s.dataSourceType.(type) {
	case DataSourceSpecConfiguration:
		return true, nil
	case EthCallSpec:
		return true, nil
	case DataSourceSpecConfigurationTime:
		return false, nil
	}
	return false, errors.New("unknown type of data source provided")
}

func (s *DataSourceDefinition) GetDataSourceSpecConfigurationTime() DataSourceSpecConfigurationTime {
	data := s.Content()
	switch tp := data.(type) {
	case DataSourceSpecConfigurationTime:
		return tp
	}

	return DataSourceSpecConfigurationTime{}
}
