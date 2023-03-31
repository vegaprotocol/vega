package vega

import datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

// Add any additional types related External Data sources specifications here.

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

func (x DataSourceDefinitionExternal_Oracle) DeepClone() *DataSourceDefinitionExternal_Oracle {
	cpy := &DataSourceDefinitionExternal_Oracle{}
	if x.Oracle != nil {
		cpy.Oracle = x.Oracle.DeepClone()
	}

	return cpy
}

func (x DataSourceDefinitionExternal) DeepClone() *DataSourceDefinitionExternal {
	cpy := &DataSourceDefinitionExternal{}

	if x.GetSourceType() != nil {
		switch t := x.GetSourceType().(type) {
		case *DataSourceDefinitionExternal_Oracle:
			cpy.SourceType = t.DeepClone()
		}
	}

	return cpy
}

func (s DataSourceDefinition_External) DeepClone() *DataSourceDefinition_External {
	ds := &DataSourceDefinition_External{}
	if s.External != nil {
		ds.External = s.External.DeepClone()
	}

	return ds
}

func (s DataSourceSpec) DeepClone() *DataSourceSpec {
	data := &DataSourceDefinition{}
	if s.Data != nil {
		data = s.Data.DeepClone()
	}
	return &DataSourceSpec{
		Id:        s.Id,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Data:      data,
		Status:    s.Status,
	}
}

func (s ExternalDataSourceSpec) DeepClone() *ExternalDataSourceSpec {
	if s.Spec != nil {
		spec := s.Spec.DeepClone()
		return &ExternalDataSourceSpec{
			Spec: spec,
		}
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{},
	}
}
