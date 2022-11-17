package types

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceDefinitionInternalx struct {
	Internal *DataSourceDefinitionInternal
}

func (s *DataSourceDefinitionInternalx) isDataSourceType() {}

func (s *DataSourceDefinitionInternalx) oneOfProto() interface{} {
	return s.IntoProto()
}

// IntoProto returns the proto object from DataSourceDefinitionInternalx.
// This method is not called from anywhere.
func (s *DataSourceDefinitionInternalx) IntoProto() *vegapb.DataSourceDefinition_Internal {
	ds := &vegapb.DataSourceDefinition_Internal{
		Internal: &vegapb.DataSourceDefinitionInternal{},
	}

	if s.Internal != nil {
		if s.Internal.SourceType != nil {
			switch dsn := s.Internal.SourceType.oneOfProto().(type) {
			case *vegapb.DataSourceDefinitionInternal_Time:
				ds = &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: dsn,
					},
				}
			}
		}
	}

	return ds
}

// DeepClone returns a clone of the DataSourceDefinitionInternalx object.
func (s *DataSourceDefinitionInternalx) DeepClone() dataSourceType {
	cpy := &DataSourceDefinitionInternalx{
		Internal: &DataSourceDefinitionInternal{
			SourceType: s.Internal.SourceType.DeepClone(),
		},
	}
	return cpy
}

// String returns the DataSourceDefinitionInternalx content as a string.
func (s *DataSourceDefinitionInternalx) String() string {
	if s.Internal != nil {
		// Does not return the type of the internal data source, becase the base object
		// definitions are located in core/vega/protos/ and do not access the local intrface
		// and accessing it will lead to cycle import.
		return fmt.Sprintf("external(%s)", s.Internal.String())
	}

	return ""
}

//func (s *DataSourceDefinitionInternalx) ToDataSourceSpec() *DataSourceSpec {
//	return nil
//}

type DataSourceDefinitionExternalx struct {
	External *DataSourceDefinitionExternal
}

func (s *DataSourceDefinitionExternalx) isDataSourceType() {}

func (s *DataSourceDefinitionExternalx) oneOfProto() interface{} {
	return s.IntoProto()
}

// IntoProto returns the proto object from DataSourceDefinitionInternalx
// This method is not called from anywhere.
func (s *DataSourceDefinitionExternalx) IntoProto() *vegapb.DataSourceDefinition_External {
	ds := &vegapb.DataSourceDefinition_External{
		External: &vegapb.DataSourceDefinitionExternal{},
	}

	if s.External != nil {
		if s.External.SourceType != nil {
			switch dsn := s.External.SourceType.oneOfProto().(type) {
			case *vegapb.DataSourceDefinitionExternal_Oracle:
				ds = &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: dsn,
					},
				}
			}
		}
	}

	return ds
}

func (s *DataSourceDefinitionExternalx) DeepClone() dataSourceType {
	cpy := &DataSourceDefinitionExternalx{
		External: &DataSourceDefinitionExternal{
			SourceType: s.External.SourceType.DeepClone(),
		},
	}
	return cpy
}

// String returns the DataSourceDefinitionExternalx content as a string.
func (s *DataSourceDefinitionExternalx) String() string {
	if s.External != nil {
		// Does not return the type of the external data source, becase the base object
		// definitions are located in core/vega/protos/ and do not access the local intrface
		// and accessing it will lead to cycle import.
		return fmt.Sprintf("external(%s)", s.External.String())
	}

	return ""
}

//func (s *DataSourceDefinitionExternalx) ToDataSourceSpec() *DataSourceSpec {
//	return nil
//}

type dataSourceType interface {
	isDataSourceType()
	oneOfProto() interface{}

	String() string
	DeepClone() dataSourceType
	// ToDataSourceSpec() *DataSourceSpec
}

type DataSourceDefinition struct {
	SourceType dataSourceType
}

// /
// IntoProto returns the proto object from DataSourceDefinition
// that is - vegapb.DataSourceDefinition that may have external or internal SourceType.
// Returns the whole proto object.
func (s DataSourceDefinition) IntoProto() *vegapb.DataSourceDefinition {
	ds := &vegapb.DataSourceDefinition{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			ds = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: dsn.External.GetSourceType(),
					},
				},
			}

		case *vegapb.DataSourceDefinition_Internal:
			ds = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: dsn.Internal.GetSourceType(),
					},
				},
			}
		}
	}

	return ds
}

// /
// String returns the data source definition content as a string.
func (s DataSourceDefinition) String() string {
	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			return fmt.Sprintf("external(%s)", dsn.External.String())

		case *vegapb.DataSourceDefinition_Internal:
			return fmt.Sprintf("internal(%s)", dsn.Internal.String())
		}
	}

	return ""
}

// DeepClone returns a clone of the DataSourceDefinition object.
func (s DataSourceDefinition) DeepClone() DataSourceDefinition {
	cpy := &DataSourceDefinition{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			cpy = &DataSourceDefinition{
				SourceType: &DataSourceDefinitionExternalx{
					External: &DataSourceDefinitionExternal{
						SourceType: &DataSourceSpecConfiguration{
							Signers: SignersFromProto(dsn.External.GetOracle().Signers),
							Filters: DataSourceSpecFiltersFromProto(dsn.External.GetOracle().Filters),
						},
					},
				},
			}

		case *vegapb.DataSourceDefinition_Internal:
			cpy = &DataSourceDefinition{
				SourceType: s.SourceType.DeepClone(),
			}
		}
	}

	return *cpy
}

// /
// DataSourceDefinitionFromProto tries to build the DataSourceDfiniition object
// from the given proto object.
func DataSourceDefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) *DataSourceDefinition {
	ds := &DataSourceDefinition{}

	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinition_External:
				ds.SourceType = &DataSourceDefinitionExternalx{
					External: DataSourceDefinitionExternalFromProto(tp.External),
				}

			case *vegapb.DataSourceDefinition_Internal:
				ds.SourceType = &DataSourceDefinitionInternalx{
					Internal: DataSourceDefinitionInternalFromProto(tp.Internal),
				}
			}
		}
	}

	return ds
}

// /
// GetSigners tries to get the signers from the DataSourceDefinition if they exist.
func (s DataSourceDefinition) GetSigners() []*Signer {
	signers := []*Signer{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			signers = SignersFromProto(dsn.External.GetOracle().Signers)
		}
	}

	return signers
}

// /
// GetFilters tries to get the filters from the DataSourceDefinition if they exist.
func (s DataSourceDefinition) GetFilters() []*DataSourceSpecFilter {
	filters := []*DataSourceSpecFilter{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			filters = DataSourceSpecFiltersFromProto(dsn.External.GetOracle().Filters)
		}
	}

	return filters
}

// GetDataSourceSpecConfiguration returns the base object - DataSourceSpecConfiguration
// from the DataSourceDefinition.
func (s DataSourceDefinition) GetDataSourceSpecConfiguration() *vegapb.DataSourceSpecConfiguration {
	if s.SourceType != nil {
		switch tp := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			return tp.External.GetOracle()
		}
	}

	return nil
}

// NewDataSourceDefinition creates a new empty DataSourceDefinition object.
func NewDataSourceDefinition(tp int) *DataSourceDefinition {
	ds := &DataSourceDefinition{}
	switch tp {
	case vegapb.DataSourceDefinitionTypeInt:
		ds.SourceType = &DataSourceDefinitionInternalx{
			Internal: &DataSourceDefinitionInternal{
				// Create internal type definition with time for now.
				SourceType: &DataSourceDefinitionInternalTime{
					Time: &DataSourceSpecConfigurationTime{
						Conditions: []*DataSourceSpecCondition{},
					},
				},
			},
		}

	case vegapb.DataSourceDefinitionTypeExt:
		ds.SourceType = &DataSourceDefinitionExternalx{
			External: &DataSourceDefinitionExternal{
				// Create external definition for oracles for now.
				// Extened when needed.
				SourceType: &DataSourceDefinitionExternalOracle{
					Oracle: &DataSourceSpecConfiguration{
						Signers: []*Signer{},
						Filters: []*DataSourceSpecFilter{},
					},
				},
			},
		}
	}

	return ds
}

// /
// UpdateFilters updates the DataSourceDefinition Filters.
func (s *DataSourceDefinition) UpdateFilters(filters []*DataSourceSpecFilter) {
	ds := &vegapb.DataSourceDefinition{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			ds = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Filters: DataSourceSpecFilters(filters).IntoProto(),
								Signers: dsn.External.GetOracle().Signers,
							},
						},
					},
				},
			}
		}
	}

	dsd := DataSourceDefinitionFromProto(ds)
	*s = *dsd
}

func (s *DataSourceDefinition) SetFilterDecimals(d uint64) {
	ds := &vegapb.DataSourceDefinition{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			filters := dsn.External.GetOracle().Filters
			for i, _ := range filters {
				filters[i].Key.NumberDecimalPlaces = uint32(d)
			}

			ds = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Filters: filters,
								Signers: dsn.External.GetOracle().Signers,
							},
						},
					},
				},
			}
		}
	}

	dsd := DataSourceDefinitionFromProto(ds)
	*s = *dsd
}

func (s DataSourceDefinition) ToDataSourceSpec() *DataSourceSpec {
	bytes, _ := proto.Marshal(s.IntoProto())
	specID := hex.EncodeToString(crypto.Hash(bytes))
	return &DataSourceSpec{
		ID:   specID,
		Data: &s,
	}
}

func (s *DataSourceDefinition) ToExternalDataSourceSpec() *ExternalDataSourceSpec {
	return &ExternalDataSourceSpec{
		Spec: s.ToDataSourceSpec(),
	}
}

// SetOracleConfig sets a given oracle config in the receiver.
// This method does not care about object previous contents.
func (s *DataSourceDefinition) SetOracleConfig(oc *DataSourceSpecConfiguration) *DataSourceDefinition {
	ds := &vegapb.DataSourceDefinition{}

	if s.SourceType != nil {
		switch s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			ds = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: oc.IntoProto(),
						},
					},
				},
			}
		}
	}

	*s = *DataSourceDefinitionFromProto(ds)
	return s
}
