package vega

import (
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const (
	DataSourceDefinitionTypeUspec = 0
	DataSourceDefinitionTypeInt   = 1
	DataSourceDefinitionTypeExt   = 2
)

func (s DataSourceSpecConfiguration) DeepClone() *DataSourceSpecConfiguration {
	if len(s.Signers) > 0 {
		sgns := s.Signers
		s.Signers = make([]*datapb.Signer, len(sgns))
		for i, sig := range sgns {
			s.Signers[i] = sig.DeepClone()
		}
	}

	if len(s.Filters) > 0 {
		filters := s.Filters
		s.Filters = make([]*datapb.Filter, len(filters))
		for i, f := range filters {
			s.Filters[i] = f.DeepClone()
		}
	}

	return &DataSourceSpecConfiguration{
		Signers: s.Signers,
		Filters: s.Filters,
	}
}

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

//
func (d DataSourceDefinition) GetFilters() []*datapb.Filter {
	filters := []*datapb.Filter{}

	if d.SourceType != nil {
		switch tp := d.SourceType.(type) {
		case *DataSourceDefinition_External:
			filters = tp.External.GetOracle().Filters
		case *DataSourceDefinition_Internal:

		}
	}

	return filters
}

//
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

///
// SetOracleConfig sets a given oracle config in the receiver.
// This method does not care about object previous contents - use with caution (currenty needed on ly for testing purposes).
func (s *DataSourceDefinition) SetOracleConfig(oc *DataSourceSpecConfiguration) *DataSourceDefinition {
	ds := &DataSourceDefinition{}

	if s.SourceType != nil {
		switch s.SourceType.(type) {
		case *DataSourceDefinition_External:
			ds = &DataSourceDefinition{
				SourceType: &DataSourceDefinition_External{
					External: &DataSourceDefinitionExternal{
						SourceType: &DataSourceDefinitionExternal_Oracle{
							Oracle: oc,
						},
					},
				},
			}
		}
	}

	*s = *ds
	return s
}
