package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceDefinitionExternal struct {
	SourceType oracleSourceType
}

func (e *DataSourceDefinitionExternal) isOracleSourceType() {}

func (e *DataSourceDefinitionExternal) oneOfProto() interface{} {
	return e.IntoProto()
}

// /
// IntoProto tries to return the base proto object from DataSourceDefinitionExternal.
func (e *DataSourceDefinitionExternal) IntoProto() *vegapb.DataSourceDefinitionExternal {
	ds := &vegapb.DataSourceDefinitionExternal{}

	if e.SourceType != nil {
		switch dsn := e.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			ds = &vegapb.DataSourceDefinitionExternal{
				SourceType: dsn,
			}
		case **vegapb.DataSourceDefinitionExternal_Eth:
			// ...
		}
	}

	return ds
}

func (e *DataSourceDefinitionExternal) String() string {
	if e.SourceType != nil {
		return e.SourceType.String()
	}

	return ""
}

func (e *DataSourceDefinitionExternal) DeepClone() oracleSourceType {
	if e.SourceType != nil {
		return e.SourceType.DeepClone()
	}

	return nil
}

// /
// DataSourceDefinitionExternalFromProto tries to build the DataSourceDefinitionExternal object
// from the given proto object..
func DataSourceDefinitionExternalFromProto(protoConfig *vegapb.DataSourceDefinitionExternal) *DataSourceDefinitionExternal {
	ds := &DataSourceDefinitionExternal{
		SourceType: &DataSourceDefinitionExternalOracle{},
	}

	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinitionExternal_Oracle:
				ds.SourceType = DataSourceDefinitionExternalOracleFromProto(tp)
			}
		}
	}

	return ds
}
