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

type DataSourceContentType int32

const (
	DataSourceContentTypeInvalid DataSourceContentType = iota
	DataSourceContentTypeOracle
	DataSourceContentTypeEthOracle
	DataSourceContentTypeInternalTimeTermination
)

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
func NewDataSourceDefinition(tp DataSourceContentType) *DataSourceDefinition {
	ds := &DataSourceDefinition{}
	switch tp {
	case DataSourceContentTypeOracle:
		return NewDataSourceDefinitionWith(
			DataSourceSpecConfiguration{
				Signers: []*Signer{},
				Filters: []*DataSourceSpecFilter{},
			})
	case DataSourceContentTypeEthOracle:
		return NewDataSourceDefinitionWith(
			EthCallSpec{
				AbiJson:  []byte{},
				ArgsJson: []string{},
				Trigger:  &EthTimeTrigger{},
				Filters:  DataSourceSpecFilters{},
			})
	case DataSourceContentTypeInternalTimeTermination:
		return NewDataSourceDefinitionWith(
			DataSourceSpecConfigurationTime{
				Conditions: []*DataSourceSpecCondition{},
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
func (s DataSourceDefinition) DeepClone() dataSourceType {
	if s.dataSourceType != nil {
		return DataSourceDefinition{
			dataSourceType: s.dataSourceType.DeepClone(),
		}
	}
	return nil // ?
}

func (s DataSourceDefinition) String() string {
	if s.dataSourceType != nil {
		return s.dataSourceType.String()
	}
	return ""
}

func (s *DataSourceDefinition) Content() interface{} {
	return s.dataSourceType
}

// DataSourceDefinitionFromProto tries to build the DataSourceDfiniition object
// from the given proto object.
func DataSourceDefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) (dataSourceType, error) {
	if protoConfig != nil {
		data := protoConfig.Content()
		switch dtp := data.(type) {
		case *vegapb.DataSourceSpecConfiguration:
			return DataSourceSpecConfigurationFromProto(dtp), nil

		case *vegapb.EthCallSpec:
			return EthCallSpecFromProto(dtp)

		case *vegapb.DataSourceSpecConfigurationTime:
			return DataSourceSpecConfigurationTimeFromProto(dtp), nil
		}
	}

	return &DataSourceDefinition{}, nil
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

		case EthCallSpec:
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

// GetEthCallSpec returns the base object - EthCallSpec
// from the DataSourceDefinition.
func (s *DataSourceDefinition) GetEthCallSpec() EthCallSpec {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case EthCallSpec:
			return tp
		}
	}

	return EthCallSpec{}
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

	case EthCallSpec:
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
	case EthCallSpec:
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
func (s *DataSourceDefinition) SetOracleConfig(oc dataSourceType) *DataSourceDefinition {
	if _, ok := s.dataSourceType.(DataSourceSpecConfiguration); ok {
		s.dataSourceType = oc.DeepClone()
	}

	if _, ok := s.dataSourceType.(EthCallSpec); ok {
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

func (s *DataSourceDefinition) GetDataSourceSpecConfigurationTime() DataSourceSpecConfigurationTime {
	data := s.Content()
	switch tp := data.(type) {
	case DataSourceSpecConfigurationTime:
		return tp
	}

	return DataSourceSpecConfigurationTime{}
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

func (s *DataSourceDefinition) Type() (DataSourceContentType, bool) {
	switch s.dataSourceType.(type) {
	case DataSourceSpecConfiguration:
		return DataSourceContentTypeOracle, true
	case EthCallSpec:
		return DataSourceContentTypeEthOracle, true
	case DataSourceSpecConfigurationTime:
		return DataSourceContentTypeInternalTimeTermination, false
	}
	return DataSourceContentTypeInvalid, false
}
