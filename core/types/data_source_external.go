package types

/*
func dataSourceDefinitionExternalFromProto(proto *vegapb.DataSourceDefinitionExternal) (dataSourceType, error) {
	if proto == nil {
		return nil, fmt.Errorf("data source definition external proto is nil")
	}
	switch st := proto.SourceType.(type) {
	case *vegapb.DataSourceDefinitionExternal_Oracle:
		return DataSourceSpecConfigurationFromProto(st.Oracle), nil
	case *vegapb.DataSourceDefinitionExternal_EthOracle:
		return EthCallSpecFromProto(st.EthOracle)
	}
	return nil, fmt.Errorf("unknown data source type %T", proto.SourceType)
}
*/
