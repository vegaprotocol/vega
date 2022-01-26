package adaptors

import (
	"errors"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/validation"
)

// ErrUnknownOracleSource is used when the input data is originated from an
// unknown, unsupported or unspecified oracle source.
var ErrUnknownOracleSource = errors.New("unknown oracle source")

// Adaptor represents an oracle adaptor that consumes and normalises data from
// a specific type of oracle.
type Adaptor interface {
	Normalise(crypto.PublicKey, []byte) (*oracles.OracleData, error)
}

// Adaptors normalises the input data into an oracles.OracleData according to
// its source.
type Adaptors struct {
	// Adaptors holds all the supported Adaptors sorted by source.
	Adaptors map[commandspb.OracleDataSubmission_OracleSource]Adaptor
}

// New creates an Adaptors with all the supported oracle Adaptor.
func New() *Adaptors {
	return &Adaptors{
		Adaptors: map[commandspb.OracleDataSubmission_OracleSource]Adaptor{
			commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE: NewOpenOracleAdaptor(),
			commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON:        NewJSONAdaptor(),
		},
	}
}

// Normalise normalises the input data into an oracles.OracleData based on its source.
func (a *Adaptors) Normalise(txPubKey crypto.PublicKey, data commandspb.OracleDataSubmission) (*oracles.OracleData, error) {
	adaptor, ok := a.Adaptors[data.Source]
	if !ok {
		return nil, ErrUnknownOracleSource
	}

	oracleData, err := adaptor.Normalise(txPubKey, data.Payload)
	if err != nil {
		return nil, err
	}

	err = validation.CheckForInternalOracle(oracleData.Data)

	if err != nil {
		return nil, err
	}

	return oracleData, nil
}
