package vega

// Add any additional types related External Data sources specifications here.

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
