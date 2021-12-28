package adaptors

import (
	"fmt"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/oracles"

	"code.vegaprotocol.io/oracles-relay/openoracle"
)

// OpenOracleAdaptor is a specific oracle Adaptor for Open Oracle / Open Price Feed
// standard.
// Link: https://compound.finance/docs/prices
type OpenOracleAdaptor struct{}

// NewOpenOracleAdaptor creates a new OpenOracleAdaptor.
func NewOpenOracleAdaptor() *OpenOracleAdaptor {
	return &OpenOracleAdaptor{}
}

// Normalise normalises an Open Oracle / Open Price Feed payload into an oracles.OracleData.
// The public key from the transaction is not used, only those from the Open
// Oracle data.
func (a *OpenOracleAdaptor) Normalise(_ crypto.PublicKey, data []byte) (*oracles.OracleData, error) {
	response, err := openoracle.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal Open Oracle data: %w", err)
	}

	pubKeys, kvs, err := openoracle.Verify(*response)
	if err != nil {
		return nil, fmt.Errorf("invalid Open Oracle response: %w", err)
	}

	return &oracles.OracleData{
		PubKeys: pubKeys,
		Data:    kvs,
	}, nil
}
