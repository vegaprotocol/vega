package v1

func (o *DataSourceSpecConfiguration) ToDataSpec() *DataSourceSpec {
	return NewDataSourceSpec(o)
}

func (o *DataSourceSpecConfiguration) ToOracleSpec(d *DataSourceSpec) *OracleSpec {
	return NewOracleSpec(d)
}
