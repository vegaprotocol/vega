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

var (
	// ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList is returned when filters with same key names exists inside a single list.
	ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList = errors.New("multiple keys with same name found in filter list")
	// ErrDataSourceSpecHasInvalidTimeCondition is returned when timestamp value is used with 'LessThan'
	// or 'LessThanOrEqual' condition operator value.
	ErrDataSourceSpecHasInvalidTimeCondition = errors.New("data source spec time value is used with 'less than' or 'less than equal' condition")
)

type DataSourceType = vegapb.DataSourceDefinition_Type

const (
	// DataSourceDefinitionInvalid represents invalid data source definition type
	DataSourceDefinitionInvalid DataSourceType = vegapb.DataSourceDefinition_TYPE_DATA_SOURCE_INVALID

	// DataSourceDefinitionExtOracle is the default oracle data type
	DataSourceDefinitionExtOracle DataSourceType = vegapb.DataSourceDefinition_TYPE_DATA_SOURCE_EXT_ORACLE

	// DataSourceDefinitionExtEthOracle is the Ethereum oracle data type
	DataSourceDefinitionExtEthOracle DataSourceType = vegapb.DataSourceDefinition_TYPE_DATA_SOURCE_EXT_ETHEREUM_ORACLE

	// DataSourceDefinitionIntTimeTrigger is the internal time trigger data source finition type
	DataSourceDefinitionIntTimeTrigger DataSourceType = vegapb.DataSourceDefinition_TYPE_DATA_SOURCE_INT_TIME_TRIGGER
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
		// definitions are located in core/vega/protos/ and do not access the local interface
		// and accessing it will lead to cycle import.
		return fmt.Sprintf("internal(%s)", s.Internal.String())
	}

	return ""
}

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
			case *vegapb.DataSourceDefinitionExternal_EthOracle:
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

type dataSourceType interface {
	isDataSourceType()
	oneOfProto() interface{}

	String() string
	DeepClone() dataSourceType
}

type DataSourceDefinition struct {
	SourceType dataSourceType
}

// IntoProto returns the proto object from DataSourceDefinition
// that is - vegapb.DataSourceDefinition that may have external or internal SourceType.
// Returns the whole proto object.
func (s DataSourceDefinition) IntoProto() *vegapb.DataSourceDefinition {
	ds := &vegapb.DataSourceDefinition{}

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			if dsn.External != nil {
				if dsn.External.SourceType != nil {
					switch dsn.External.SourceType.(type) {
					case *vegapb.DataSourceDefinitionExternal_Oracle, *vegapb.DataSourceDefinitionExternal_EthOracle:
						ds = &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_External{
								External: &vegapb.DataSourceDefinitionExternal{
									// This will return the external data source oracle object that satisfies the interface
									SourceType: dsn.External.GetSourceType(),
								},
							},
						}
					}
				}
			}

		case *vegapb.DataSourceDefinition_Internal:
			if dsn.Internal != nil {
				if dsn.Internal.SourceType != nil {
					switch dsn.Internal.SourceType.(type) {
					case *vegapb.DataSourceDefinitionInternal_Time:
						ds = &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_Internal{
								Internal: &vegapb.DataSourceDefinitionInternal{
									// This will return the internal data source time object that satisfies the interface
									SourceType: dsn.Internal.GetSourceType(),
								},
							},
						}
						// More types of internal sources that will come in the future - will go here.
					}
				}
			}
		}
	}

	return ds
}

// String returns the data source definition content as a string.
func (s DataSourceDefinition) String() string {
	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			s := ""
			if dsn.External != nil {
				s = dsn.External.String()
			}
			return fmt.Sprintf("external(%s)", s)

		case *vegapb.DataSourceDefinition_Internal:
			s := ""
			if dsn.Internal != nil {
				s = dsn.Internal.String()
			}
			return fmt.Sprintf("internal(%s)", s)
		}
	}

	return ""
}

// DeepClone returns a clone of the DataSourceDefinition object.
func (s DataSourceDefinition) DeepClone() DataSourceDefinition {
	cpy := &DataSourceDefinition{}

	d := s.Content()
	switch tp := d.(type) {
	case *DataSourceSpecConfiguration:
		cpy = &DataSourceDefinition{
			SourceType: &DataSourceDefinitionExternalx{
				External: &DataSourceDefinitionExternal{
					SourceType: &DataSourceSpecConfiguration{
						Signers: tp.Signers,
						Filters: tp.Filters,
					},
				},
			},
		}

	case *EthCallSpec:
		cpy = &DataSourceDefinition{
			SourceType: &DataSourceDefinitionExternalx{
				External: &DataSourceDefinitionExternal{
					SourceType: &EthCallSpec{
						Address:               tp.Address,
						AbiJson:               tp.AbiJson,
						Method:                tp.Method,
						ArgsJson:              tp.ArgsJson,
						Trigger:               tp.Trigger,
						RequiredConfirmations: tp.RequiredConfirmations,
						Filter:                tp.Filter,
						Normaliser:            tp.Normaliser,
					},
				},
			},
		}
	case *DataSourceSpecConfigurationTime:
		cpy = &DataSourceDefinition{
			SourceType: &DataSourceDefinitionInternalx{
				Internal: &DataSourceDefinitionInternal{
					SourceType: &DataSourceSpecConfigurationTime{
						Conditions: tp.Conditions,
					},
				},
			},
		}
	}

	return *cpy
}

func (s *DataSourceDefinition) Content() interface{} {
	if s != nil {
		if s.SourceType != nil {
			switch tp := s.SourceType.(type) {
			case *DataSourceDefinitionExternalx:
				if tp.External != nil {
					if tp.External.SourceType != nil {
						switch extTp := tp.External.SourceType.(type) {
						case *DataSourceDefinitionExternalOracle:
							if extTp.Oracle != nil {
								return extTp.Oracle
							}

						case *DataSourceDefinitionExternalEthOracle:
							if extTp.EthOracle != nil {
								return extTp.EthOracle
							}
						}
					}
				}

			case *DataSourceDefinitionInternalx:
				if tp.Internal != nil {
					if tp.Internal.SourceType != nil {
						switch intTp := tp.Internal.SourceType.(type) {
						case *DataSourceDefinitionInternalTime:
							if intTp.Time != nil {
								return intTp.Time
							}

							// The rest of the internal type sources cases will go here later.
						}
					}
				}
			}
		}
	}

	return nil
}

// DataSourceDefinitionFromProto tries to build the DataSourceDfiniition object
// from the given proto object.
func DataSourceDefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) *DataSourceDefinition {
	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinition_External:
				return &DataSourceDefinition{
					SourceType: &DataSourceDefinitionExternalx{
						// Checking if the tp.External is nil is made in the `DataSourceDefinitionExternalFromProto` step
						External: DataSourceDefinitionExternalFromProto(tp.External),
					},
				}

			case *vegapb.DataSourceDefinition_Internal:
				return &DataSourceDefinition{
					SourceType: &DataSourceDefinitionInternalx{
						// Checking if the tp.Internal is nil is made in the `DataSourceDefinitionInternalFromProto` step
						Internal: DataSourceDefinitionInternalFromProto(tp.Internal),
					},
				}
			}
		}
	}

	return &DataSourceDefinition{}
}

// GetSigners tries to get the signers from the DataSourceDefinition if they exist.
func (s DataSourceDefinition) GetSigners() []*Signer {
	signers := []*Signer{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *DataSourceSpecConfiguration:
			signers = tp.Signers
		}
	}

	return signers
}

// GetFilters tries to get the filters from the DataSourceDefinition if they exist.
func (s DataSourceDefinition) GetFilters() []*DataSourceSpecFilter {
	filters := []*DataSourceSpecFilter{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *DataSourceSpecConfiguration:
			filters = tp.Filters

		case *EthCallSpec:
			if tp.Filter != nil {
				filters = tp.Filter.Filters
			}

		case *DataSourceSpecConfigurationTime:
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
func (s DataSourceDefinition) GetDataSourceSpecConfiguration() *DataSourceSpecConfiguration {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case *DataSourceSpecConfiguration:
			return tp
		}
	}

	return &DataSourceSpecConfiguration{}
}

// NewDataSourceDefinition creates a new EMPTY DataSourceDefinition object.
func NewDataSourceDefinition(tp DataSourceType) *DataSourceDefinition {
	ds := &DataSourceDefinition{}
	switch tp {
	case DataSourceDefinitionExtOracle:
		ds.SourceType = &DataSourceDefinitionExternalx{
			External: &DataSourceDefinitionExternal{
				SourceType: &DataSourceDefinitionExternalOracle{
					Oracle: &DataSourceSpecConfiguration{
						Signers: []*Signer{},
						Filters: []*DataSourceSpecFilter{},
					},
				},
			},
		}

	case DataSourceDefinitionExtEthOracle:
		ds.SourceType = &DataSourceDefinitionExternalx{
			External: &DataSourceDefinitionExternal{
				SourceType: &DataSourceDefinitionExternalEthOracle{
					EthOracle: &EthCallSpec{
						Filter:     &EthFilter{},
						Normaliser: &Normaliser{},
					},
				},
			},
		}

	case DataSourceDefinitionIntTimeTrigger:
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
	}

	return ds
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

	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			if dsn.External != nil {
				if dsn.External.SourceType != nil {
					switch dsn.External.SourceType.(type) {
					case *vegapb.DataSourceDefinitionExternal_Oracle:
						o := dsn.External.GetOracle()
						signers := []*datapb.Signer{}
						if o != nil {
							signers = o.Signers
						}

						ds := &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_External{
								External: &vegapb.DataSourceDefinitionExternal{
									SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
										Oracle: &vegapb.DataSourceSpecConfiguration{
											// We do not care if return empty lists for signers and filters here
											Filters: DataSourceSpecFilters(filters).IntoProto(),
											Signers: signers,
										},
									},
								},
							},
						}

						dsd := DataSourceDefinitionFromProto(ds)
						if dsd.SourceType != nil {
							*s = *dsd
						}

					case *vegapb.DataSourceDefinitionExternal_EthOracle:
						o := dsn.External.GetEthOracle()
						o.Filter = &vegapb.EthFilter{
							Filters: DataSourceSpecFilters(filters).IntoProto(),
						}

						ds := &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_External{
								External: &vegapb.DataSourceDefinitionExternal{
									SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
										EthOracle: o,
									},
								},
							},
						}

						dsd := DataSourceDefinitionFromProto(ds)
						if dsd.SourceType != nil {
							*s = *dsd
						}
					}
				}
			}

		case *vegapb.DataSourceDefinition_Internal:
			if dsn.Internal != nil {
				if dsn.Internal.SourceType != nil {
					switch dsn.Internal.SourceType.(type) {
					case *vegapb.DataSourceDefinitionInternal_Time:
						// The data source definition is an internal time based source
						// For this case we take only the first item from the list of filters
						// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
						c := []*datapb.Condition{}
						if len(filters) > 0 {
							if len(filters[0].Conditions) > 0 {
								c = append(c, filters[0].IntoProto().Conditions[0])
							}
						}
						ds := &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_Internal{
								Internal: &vegapb.DataSourceDefinitionInternal{
									SourceType: &vegapb.DataSourceDefinitionInternal_Time{
										Time: &vegapb.DataSourceSpecConfigurationTime{
											Conditions: c,
										},
									},
								},
							},
						}

						dsd := DataSourceDefinitionFromProto(ds)
						if dsd.SourceType != nil {
							*s = *dsd
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *DataSourceDefinition) SetFilterDecimals(d uint64) *DataSourceDefinition {
	if s.SourceType != nil {
		switch dsn := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			if dsn.External != nil {
				if dsn.External.SourceType != nil {
					switch dsn.External.SourceType.(type) {
					case *vegapb.DataSourceDefinitionExternal_Oracle:
						filters := dsn.External.GetOracle().Filters
						for i := range filters {
							filters[i].Key.NumberDecimalPlaces = &d
						}

						ds := &vegapb.DataSourceDefinition{
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

						dsd := DataSourceDefinitionFromProto(ds)
						if dsd.SourceType != nil {
							*s = *dsd
						}
					}
				}
			}
		}
	}

	return s
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
// If the receiver is not external oracle type data source - it is not changed.
// This method does not care about object previous contents.
func (s *DataSourceDefinition) SetOracleConfig(ds *DataSourceDefinitionExternal) *DataSourceDefinition {
	if s.SourceType != nil {
		switch def := s.SourceType.oneOfProto().(type) {
		// For the case the definition source type is vegapb.DataSourceDefinition_External
		case *vegapb.DataSourceDefinition_External:
			if def.External != nil {
				if def.External.SourceType != nil {
					switch def.External.SourceType.(type) {
					// For the case the vegapb.DataSourceDefinitionExternal is not nill
					// and its embedded object is of type vegapb.DataSourceDefinitionExternal_Oracle
					case *vegapb.DataSourceDefinitionExternal_Oracle:
						if ds != nil {
							if ds.SourceType != nil {
								switch et := ds.SourceType.(type) {
								case *DataSourceDefinitionExternalOracle:
									newDs := &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: et.IntoProto().Oracle,
												},
											},
										},
									}

									dsd := DataSourceDefinitionFromProto(newDs)
									if dsd.SourceType != nil {
										*s = *dsd
									}
								}
							}
						}

					case *vegapb.DataSourceDefinitionExternal_EthOracle:
						if ds != nil {
							if ds.SourceType != nil {
								switch et := ds.SourceType.(type) {
								case *DataSourceDefinitionExternalEthOracle:
									newDs := &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
													EthOracle: et.IntoProto().EthOracle,
												},
											},
										},
									}
									dsd := DataSourceDefinitionFromProto(newDs)
									if dsd.SourceType != nil {
										*s = *dsd
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return s
}

// SetTimeTriggerConditionConfig sets a given conditions config in the receiver.
// If the receiver is not a time triggered data source - it does not set anything to it.
// This method does not care about object previous contents.
func (s *DataSourceDefinition) SetTimeTriggerConditionConfig(c []*DataSourceSpecCondition) *DataSourceDefinition {
	if s.SourceType != nil {
		switch def := s.SourceType.oneOfProto().(type) {
		// For the case the definition source type is vegapb.DataSourceDefinition_Internal
		case *vegapb.DataSourceDefinition_Internal:
			if def.Internal != nil {
				if def.Internal.SourceType != nil {
					switch def.Internal.SourceType.(type) {
					// For the case the vegapb.DataSourceDefinitionInternal is not nill
					// and its embedded object is of type vegapb.DataSourceDefinitionInternal_Time
					case *vegapb.DataSourceDefinitionInternal_Time:
						// Set the new first condition only in this case
						cond := []*datapb.Condition{}
						if len(c) > 0 {
							cond = append(cond, c[0].IntoProto())
						}

						ds := &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_Internal{
								Internal: &vegapb.DataSourceDefinitionInternal{
									SourceType: &vegapb.DataSourceDefinitionInternal_Time{
										Time: &vegapb.DataSourceSpecConfigurationTime{
											// We do not care if we return an empty list of conditions in this place
											Conditions: cond,
										},
									},
								},
							},
						}

						dsd := DataSourceDefinitionFromProto(ds)
						if dsd.SourceType != nil {
							*s = *dsd
						}
					}
				}
			}
		}
	}

	return s
}

func (s *DataSourceDefinition) IsExternal() (bool, error) {
	if s.SourceType != nil {
		switch s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			return true, nil
		}

		return false, nil
	}

	return false, errors.New("unknown type of data source provided")
}

func (s *DataSourceDefinition) Type() (DataSourceType, bool) {
	if s.SourceType != nil {
		switch tp := s.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinition_External:
			if tp.External != nil {
				switch tp.External.SourceType.(type) {
				case *vegapb.DataSourceDefinitionExternal_Oracle:
					return DataSourceDefinitionExtOracle, true
				case *vegapb.DataSourceDefinitionExternal_EthOracle:
					return DataSourceDefinitionExtEthOracle, true
				}
			}

		case *vegapb.DataSourceDefinition_Internal:
			if tp.Internal != nil {
				switch tp.Internal.SourceType.(type) {
				case *vegapb.DataSourceDefinitionInternal_Time:
					return DataSourceDefinitionIntTimeTrigger, false
				}
			}
		}
	}

	return DataSourceDefinitionInvalid, false
}

func (s *DataSourceDefinition) GetDataSourceSpecConfigurationTime() *DataSourceSpecConfigurationTime {
	data := s.Content()
	switch tp := data.(type) {
	case *DataSourceSpecConfigurationTime:
		return tp
	}

	return &DataSourceSpecConfigurationTime{}
}
