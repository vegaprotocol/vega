package v1

import (
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/crypto"
)

func (OracleSpec) IsEvent() {}

func NewOracleSpec(pubKeys []string, filters []*Filter) *OracleSpec {
	return &OracleSpec{
		Id:      newID(pubKeys, filters),
		PubKeys: pubKeys,
		Filters: filters,
	}
}

func newID(pubKeys []string, filters []*Filter) string {
	buf := []byte{}
	for _, filter := range filters {
		buf = append(buf, []byte(filter.String())...)
	}
	buf = append(buf, []byte(strings.Join(pubKeys, ""))...)

	return hex.EncodeToString(crypto.Hash(buf))
}
