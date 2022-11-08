package vega

import datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

// Add any additional types related Internal Data sources specifications here.

func (x DataSourceSpecConfigurationTime) DeepClone() *DataSourceSpecConfigurationTime {
	cpy := &DataSourceSpecConfigurationTime{
		Conditions: []*datapb.Condition{},
	}

	for _, c := range x.Conditions {
		cpy.Conditions = append(cpy.Conditions, c.DeepClone())
	}

	return cpy
}

func (x DataSourceDefinitionInternal_Time) DeepClone() *DataSourceDefinitionInternal_Time {
	cpy := &DataSourceDefinitionInternal_Time{}
	if x.Time != nil {
		cpy.Time = x.Time.DeepClone()
	}

	return cpy
}

func (x DataSourceDefinitionInternal) DeepClone() *DataSourceDefinitionInternal {
	cpy := &DataSourceDefinitionInternal{}

	if x.GetSourceType() != nil {
		switch t := x.GetSourceType().(type) {
		case *DataSourceDefinitionInternal_Time:
			cpy.SourceType = t.DeepClone()
		}
	}

	return cpy
}

func (s DataSourceDefinition_Internal) DeepClone() *DataSourceDefinition_Internal {
	ds := &DataSourceDefinition_Internal{}
	if s.Internal != nil {
		ds.Internal = s.Internal.DeepClone()
	}

	return ds
}
