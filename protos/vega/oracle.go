package vega

func NewOracleSpec(d *DataSourceSpec) *OracleSpec {
	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{
			Spec: d,
		},
	}
}

func (*OracleSpec) IsEvent() {}
