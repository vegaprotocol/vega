package adaptors

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/oracles"
)

// JSONAdaptor is an oracle Adaptor for simple oracle data broadcasting.
// Link: https://compound.finance/docs/prices
type JSONAdaptor struct {
	validators []ValidatorFunc
}

// NewJSONAdaptor creates a new JSONAdaptor.
func NewJSONAdaptor(validators ...ValidatorFunc) *JSONAdaptor {
	return &JSONAdaptor{
		validators: validators,
	}
}

// Normalise normalises a JSON payload into an oracles.OracleData.
func (a *JSONAdaptor) Normalise(txPubKey crypto.PublicKey, data []byte) (*oracles.OracleData, error) {
	kvs := map[string]string{}
	err := json.Unmarshal(data, &kvs)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal JSON data: %w", err)
	}

	return &oracles.OracleData{
		PubKeys: []string{txPubKey.Hex()},
		Data:    kvs,
	}, nil
}

// Validate checks the data using the validators that are passed when the adapter is instantiated
// to ensure the data complies with the rules of each validator
func (a *JSONAdaptor) Validate(data *oracles.OracleData) error {
	return runValidation(data, a.validators)
}
