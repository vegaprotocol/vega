package vega

func NewOracleSpec(d *DataSourceSpec) *OracleSpec {
	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{
			Spec: d,
		},
	}
}

func (*OracleSpec) IsEvent() {}

func (o OracleSpec) DeepClone() *OracleSpec {
	if o.ExternalDataSourceSpec != nil {
		return &OracleSpec{
			ExternalDataSourceSpec: o.ExternalDataSourceSpec.DeepClone(),
		}
	}

	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{
			Spec: &DataSourceSpec{},
		},
	}
}
