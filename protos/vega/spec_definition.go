package vega

import (
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type DataSourceContentType int32

const (
	DataSurceContentTypeInvalid DataSourceContentType = iota
	DataSourceContentTypeOracle
	DataSourceContentTypeEthOracle
	DataSourceContentTypeInternalTimeTermination
)

func (d DataSourceDefinition) DeepClone() *DataSourceDefinition {
	cpy := &DataSourceDefinition{}

	if d.SourceType != nil {
		switch t := d.SourceType.(type) {
		case *DataSourceDefinition_External:
			cpy.SourceType = t.DeepClone()
		case *DataSourceDefinition_Internal:
			cpy.SourceType = t.DeepClone()
		}
	}

	return cpy
}

// GetSigners tries to get the Signers from the DataSourceDefinition object.
func (d DataSourceDefinition) GetSigners() []*datapb.Signer {
	signers := []*datapb.Signer{}
	cnt := d.Content()
	if cnt != nil {
		switch tp := cnt.(type) {
		case *DataSourceSpecConfiguration:
			signers = tp.Signers
		}
	}

	return signers
}

func (d DataSourceDefinition) GetFilters() []*datapb.Filter {
	filters := []*datapb.Filter{}
	cnt := d.Content()
	if cnt != nil {
		switch tp := cnt.(type) {
		case *DataSourceSpecConfiguration:
			if tp != nil {
				return tp.Filters
			}

		case *EthCallSpec:
			if tp != nil {
				if tp.Filters != nil {
					return tp.Filters
				}
			}

		case *DataSourceSpecConfigurationTime:
			if tp != nil { // wtf?
				if len(tp.Conditions) > 0 {
					filters = append(filters,
						&datapb.Filter{
							Key: &datapb.PropertyKey{
								Name: "vegaprotocol.builtin.timestamp",
								Type: datapb.PropertyKey_TYPE_TIMESTAMP,
							},
							Conditions: []*datapb.Condition{
								tp.Conditions[0],
							},
						},
					)
				}
			}
		}
	}

	return filters
}

func NewDataSourceDefinitionWith(dst isDataSourceDefinition_SourceType) *DataSourceDefinition {
	return &DataSourceDefinition{SourceType: dst}
}

func NewDataSourceDefinition(tp DataSourceContentType) *DataSourceDefinition {
	ds := &DataSourceDefinition{}

	switch tp {
	case DataSourceContentTypeOracle:
		ds.SourceType = &DataSourceDefinition_External{
			External: &DataSourceDefinitionExternal{
				SourceType: &DataSourceDefinitionExternal_Oracle{
					Oracle: &DataSourceSpecConfiguration{
						Signers: []*datapb.Signer{},
						Filters: []*datapb.Filter{},
					},
				},
			},
		}

	case DataSourceContentTypeEthOracle:
		ds.SourceType = &DataSourceDefinition_External{
			External: &DataSourceDefinitionExternal{
				SourceType: &DataSourceDefinitionExternal_EthOracle{
					EthOracle: &EthCallSpec{
						Abi:     &structpb.ListValue{},
						Args:    []*structpb.Value{},
						Trigger: &EthCallTrigger{},
						Filters: []*datapb.Filter{},
					},
				},
			},
		}

	case DataSourceContentTypeInternalTimeTermination:
		ds.SourceType = &DataSourceDefinition_Internal{
			Internal: &DataSourceDefinitionInternal{
				SourceType: &DataSourceDefinitionInternal_Time{
					Time: &DataSourceSpecConfigurationTime{
						Conditions: []*datapb.Condition{},
					},
				},
			},
		}
	}

	return ds
}

// SetOracleConfig sets a given oracle config in the receiver.
// If the receiver is not external oracle type of data source - it is not changed.
// This method does not care about object previous contents - use with caution (currenty needed only for testing purposes).
func (s *DataSourceDefinition) SetOracleConfig(oc isDataSourceDefinitionExternal_SourceType) *DataSourceDefinition {
	if oc != nil {
		cnt := NewDataSourceDefinitionWith(
			&DataSourceDefinition_External{
				External: &DataSourceDefinitionExternal{
					SourceType: oc,
				},
			}).Content()
		if s.SourceType != nil {
			switch te := s.SourceType.(type) {
			case *DataSourceDefinition_External:
				if te.External != nil {
					if te.External.SourceType != nil {
						switch te.External.SourceType.(type) {
						case *DataSourceDefinitionExternal_Oracle:
							switch tp := cnt.(type) {
							case *DataSourceSpecConfiguration:
								ds := &DataSourceDefinition{
									SourceType: &DataSourceDefinition_External{
										External: &DataSourceDefinitionExternal{
											SourceType: &DataSourceDefinitionExternal_Oracle{
												Oracle: tp,
											},
										},
									},
								}
								*s = *ds
							}

						case *DataSourceDefinitionExternal_EthOracle:
							switch tp := cnt.(type) {
							case *EthCallSpec:
								ds := &DataSourceDefinition{
									SourceType: &DataSourceDefinition_External{
										External: &DataSourceDefinitionExternal{
											SourceType: &DataSourceDefinitionExternal_EthOracle{
												EthOracle: tp,
											},
										},
									},
								}
								*s = *ds
							}
						}
					}
				}
			}
		}
	}
	return s
}

// SetTimeTriggerConditionConfig sets a condition to the time triggered receiver.
// If the receiver is not a time triggered data source - it does not set anything to it.
// This method does not care about object previous contents - use with caution (currenty needed only for testing purposes).
func (s *DataSourceDefinition) SetTimeTriggerConditionConfig(c []*datapb.Condition) *DataSourceDefinition {
	if c != nil {
		cnt := s.Content()
		if cnt != nil {
			switch cnt.(type) {
			// For the case the vegapb.DataSourceDefinitionInternal is not nill
			// and its embedded object is of type vegapb.DataSourceDefinitionInternal_Time
			case *DataSourceSpecConfigurationTime:
				// Set the new condition only in this case
				cond := []*datapb.Condition{}
				if len(c) > 0 {
					cond = append(cond, c[0])
				}
				ds := &DataSourceDefinition{
					SourceType: &DataSourceDefinition_Internal{
						Internal: &DataSourceDefinitionInternal{
							SourceType: &DataSourceDefinitionInternal_Time{
								Time: &DataSourceSpecConfigurationTime{
									Conditions: cond,
								},
							},
						},
					},
				}
				*s = *ds
			}
		}
	}

	return s
}

func (s *DataSourceDefinition) Content() interface{} {
	if s != nil {
		if s.SourceType != nil {
			switch tp := s.SourceType.(type) {
			case *DataSourceDefinition_External:
				if tp.External != nil {
					if tp.External.SourceType != nil {
						switch extTp := tp.External.SourceType.(type) {
						case *DataSourceDefinitionExternal_Oracle:
							if extTp.Oracle != nil {
								return extTp.Oracle
							}

						case *DataSourceDefinitionExternal_EthOracle:
							if extTp.EthOracle != nil {
								return extTp.EthOracle
							}
						}
					}
				}

			case *DataSourceDefinition_Internal:
				if tp.Internal != nil {
					if tp.Internal.SourceType != nil {
						switch intTp := tp.Internal.SourceType.(type) {
						case *DataSourceDefinitionInternal_Time:
							if intTp.Time != nil {
								return intTp.Time
							}

							// The rest of the internal type sources will go here.
						}
					}
				}
			}
		}
	}

	return nil
}
