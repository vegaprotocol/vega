package types

/*
func dataSourceDefinitionInternalFromProto(proto *vegapb.DataSourceDefinitionInternal) (dataSourceType, error) {
	if proto == nil {
		return nil, fmt.Errorf("data source definition internal proto is nil")
	}
	switch st := proto.SourceType.(type) {
	case *vegapb.DataSourceDefinitionInternal_Time:
		return DataSourceSpecConfigurationTimeFromProto(st.Time), nil
	}
	return nil, fmt.Errorf("unknown data source type %T", proto.SourceType)
}
*/
