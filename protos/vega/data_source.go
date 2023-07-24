package vega

import (
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

func (o *DataSourceSpecConfiguration) ToOracleSpec(d *DataSourceSpec) *OracleSpec {
	return NewOracleSpec(d)
}

func (DataSourceSpec) IsEvent() {}

// Used only for testing purposes at the moment.
func NewDataSourceSpec(sc *DataSourceDefinition) *DataSourceSpec {
	ds := &DataSourceSpec{}
	tp := sc.GetSourceType()
	if tp != nil {
		switch sc.SourceType.(type) {
		case *DataSourceDefinition_External:
			ext := sc.GetExternal()
			if ext != nil {
				switch ext.SourceType.(type) {
				case *DataSourceDefinitionExternal_Oracle:
					o := ext.GetOracle()
					if o != nil {
						ds.Id = datapb.NewID(o.Signers, o.Filters)
					}

				case *DataSourceDefinitionExternal_EthOracle:
					o := ext.GetEthOracle()
					if o != nil {
						ds.Id = datapb.NewID(nil, o.Filters)
					}
				}
			}
		case *DataSourceDefinition_Internal:
			in := sc.GetInternal()
			if in != nil {
				switch in.SourceType.(type) {
				case *DataSourceDefinitionInternal_Time:
					// t := in.GetTime()

				case *DataSourceDefinitionInternal_TimeTrigger:
					// t := in.GetTimeTrigger()
				}
				t := in.GetTime()
				if t != nil {
					//
				}
			}
		}
	}

	ds.Data = sc
	return ds
}
