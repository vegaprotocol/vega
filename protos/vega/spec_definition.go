package vega

import (
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const (
	DataSourceDefinitionTypeUspec = 0
	DataSourceDefinitionTypeInt   = 1
	DataSourceDefinitionTypeExt   = 2
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

	if d.SourceType != nil {
		switch tp := d.SourceType.(type) {
		case *DataSourceDefinition_External:
			signers = tp.External.GetOracle().Signers
		case *DataSourceDefinition_Internal:

		}
	}

	return signers
}

func (d DataSourceDefinition) GetFilters() []*datapb.Filter {
	filters := []*datapb.Filter{}

	if d.SourceType != nil {
		switch tp := d.SourceType.(type) {
		case *DataSourceDefinition_External:
			if tp.External != nil {
				if tp.External.SourceType != nil {
					switch etp := tp.External.SourceType.(type) {
					case *DataSourceDefinitionExternal_Oracle:
						if etp.Oracle != nil {
							filters = etp.Oracle.Filters
						}
					}
				}
			}

		case *DataSourceDefinition_Internal:
			if tp.Internal != nil {
				if tp.Internal.SourceType != nil {
					switch itp := tp.Internal.SourceType.(type) {
					case *DataSourceDefinitionInternal_Time:
						if itp.Time != nil {
							if len(itp.Time.Conditions) > 0 {
								filters = append(filters,
									&datapb.Filter{
										Key: &datapb.PropertyKey{
											Name: "vegaprotocol.builtin.timestamp",
											Type: datapb.PropertyKey_TYPE_TIMESTAMP,
										},
										Conditions: []*datapb.Condition{
											itp.Time.Conditions[0],
										},
									},
								)
							}
						}
					}
				}
			}

		}
	}

	return filters
}

func NewDataSourceDefinition(tp int) *DataSourceDefinition {
	ds := &DataSourceDefinition{}

	switch tp {
	case DataSourceDefinitionTypeInt:
		ds.SourceType = &DataSourceDefinition_Internal{
			Internal: &DataSourceDefinitionInternal{
				SourceType: &DataSourceDefinitionInternal_Time{
					Time: &DataSourceSpecConfigurationTime{
						Conditions: []*datapb.Condition{},
					},
				},
			},
		}

	case DataSourceDefinitionTypeExt:
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

	}

	return ds
}

// SetOracleConfig sets a given oracle config in the receiver.
// If the receiver is not external oracle type of data source - it is not changed.
// This method does not care about object previous contents - use with caution (currenty needed only for testing purposes).
func (s *DataSourceDefinition) SetOracleConfig(oc *DataSourceSpecConfiguration) *DataSourceDefinition {
	if s.SourceType != nil {
		switch def := s.SourceType.(type) {
		// For the case the definition source type is vegapb.DataSourceDefinition_External
		case *DataSourceDefinition_External:
			if def.External != nil {
				if def.External.SourceType != nil {
					switch def.External.SourceType.(type) {
					// For the case the vegapb.DataSourceDefinitionExternal is not nill
					// and its embedded object is of type vegapb.DataSourceDefinitionExternal_Oracle
					case *DataSourceDefinitionExternal_Oracle:
						// Set the new config only in this case
						ds := &DataSourceDefinition{
							SourceType: &DataSourceDefinition_External{
								External: &DataSourceDefinitionExternal{
									SourceType: &DataSourceDefinitionExternal_Oracle{
										Oracle: oc,
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

	return s
}

// SetTimeTriggerConditionConfig sets a condition to the time triggered receiver.
// If the receiver is not a time triggered data source - it does not set anything to it.
// This method does not care about object previous contents - use with caution (currenty needed only for testing purposes).
func (s *DataSourceDefinition) SetTimeTriggerConditionConfig(c []*datapb.Condition) *DataSourceDefinition {
	if s.SourceType != nil {
		switch def := s.SourceType.(type) {
		// For the case the definition source type is vegapb.DataSourceDefinition_Internal
		case *DataSourceDefinition_Internal:
			if def.Internal != nil {
				if def.Internal.SourceType != nil {
					switch def.Internal.SourceType.(type) {
					// For the case the vegapb.DataSourceDefinitionInternal is not nill
					// and its embedded object is of type vegapb.DataSourceDefinitionInternal_Time
					case *DataSourceDefinitionInternal_Time:
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
		}
	}

	return s
}
