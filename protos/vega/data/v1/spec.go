package v1

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/libs/crypto"
)

func (DataSourceSpec) IsEvent() {}

func NewDataSourceSpec(sc *DataSourceSpecConfiguration) *DataSourceSpec {
	return &DataSourceSpec{
		Id:     NewID(sc.Signers, sc.Filters),
		Config: sc,
	}
}

func NewID(signers []*Signer, filters []*Filter) string {
	buf := []byte{}
	for _, filter := range filters {
		s := filter.Key.Name + filter.Key.Type.String()
		for _, c := range filter.Conditions {
			s += c.Operator.String() + c.Value
		}

		buf = append(buf, []byte(s)...)
	}

	return hex.EncodeToString(crypto.Hash(buf))
}

func NewOracleSpec(d *DataSourceSpec) *OracleSpec {
	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{
			Spec: d,
		},
	}
}

func (*OracleSpec) IsEvent() {}
