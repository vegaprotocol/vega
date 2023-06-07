package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceDefinitionExternal struct {
	SourceType dataSourceType
}

func (e *DataSourceDefinitionExternal) isDataSourceType() {}

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
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			ds = &vegapb.DataSourceDefinitionExternal{
				SourceType: dsn,
			}
		}
	}

	return ds
}

func (e *DataSourceDefinitionExternal) String() string {
	if e.SourceType != nil {
		switch dsn := e.SourceType.oneOfProto().(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			if dsn.Oracle != nil {
				return dsn.Oracle.String()
			}
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			if dsn.EthOracle != nil {
				return dsn.EthOracle.String()
			}
		}
	}

	return ""
}

func (e *DataSourceDefinitionExternal) DeepClone() dataSourceType {
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
		// SourceType:
	}

	if protoConfig != nil {
		if protoConfig.SourceType != nil {
			switch tp := protoConfig.SourceType.(type) {
			case *vegapb.DataSourceDefinitionExternal_Oracle:
				ds.SourceType = DataSourceDefinitionExternalOracleFromProto(tp)

			case *vegapb.DataSourceDefinitionExternal_EthOracle:
				ds.SourceType = EthCallSpecFromProto(tp)
			}
		}
	}

	return ds
}
