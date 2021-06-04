package v1

func (o *OracleSpecConfiguration) ToOracleSpec() *OracleSpec {
	return NewOracleSpec(o.PubKeys, o.Filters)
}
