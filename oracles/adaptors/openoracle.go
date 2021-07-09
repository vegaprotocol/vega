package adaptors

import (
	"code.vegaprotocol.io/vega/oracles"

	"code.vegaprotocol.io/oracles-relay/openoracle"
)

// OpenOracleAdaptor is a specific oracle Adaptor for Open Oracle / Open Price Feed
// standard.
// Link: https://compound.finance/docs/prices
type OpenOracleAdaptor struct {
}

// NewOpenOracleAdaptor creates a new OpenOracleAdaptor.
func NewOpenOracleAdaptor() *OpenOracleAdaptor {
	return &OpenOracleAdaptor{}
}

// Normalise normalises an Open Oracle / Open Price Feed payload into an oracles.OracleData.
func (a *OpenOracleAdaptor) Normalise(data []byte) (*oracles.OracleData, error) {
	oresp, err := openoracle.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	pubKeys, kvs, err := openoracle.Verify(*oresp)
	if err != nil {
		return nil, err
	}

	return &oracles.OracleData{
		PubKeys: pubKeys,
		Data:    kvs,
	}, nil
}
