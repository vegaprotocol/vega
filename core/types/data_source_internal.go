package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceDefinitionInternal struct {
	SourceType dataSourceType
}

func (i *DataSourceDefinitionInternal) isDataSourceType() {}

func (i *DataSourceDefinitionInternal) oneOfProto() interface{} {
	return i.IntoProto()
}

func (i *DataSourceDefinitionInternal) IntoProto() interface{} {
	ds := &vegapb.DataSourceDefinitionInternal{}

	if i.SourceType != nil {
		switch dsn := i.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinitionInternal_Time:
			ds = &vegapb.DataSourceDefinitionInternal{
				SourceType: dsn,
			}
		}
	}

	return ds
}

func (i *DataSourceDefinitionInternal) String() string {
	if i.SourceType != nil {
		return i.SourceType.String()
	}

	return ""
}

func (i *DataSourceDefinitionInternal) DeepClone() dataSourceType {
	if i.SourceType != nil {
		return i.SourceType.DeepClone()
	}

	return nil
}

// //
// DataSourceDefinitionInternalFromProto tries to build the DataSourceDefinitionInternal object
// from the given proto configuration.
func DataSourceDefinitionInternalFromProto(protoConfig *vegapb.DataSourceDefinitionInternal) *DataSourceDefinitionInternal {
	ds := &DataSourceDefinitionInternal{
		SourceType: &DataSourceDefinitionInternalTime{},
	}

	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinitionInternal_Time:
				ds.SourceType = DataSourceDefinitionInternalTimeFromProto(tp)
			}
		}
	}

	return ds
}
