package adaptors

import (
	"errors"

	"code.vegaprotocol.io/vega/oracles"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

var (
	// ErrUnknownOracleSource is used when the input data is originated from an
	// unknown, unsupported or unspecified oracle source.
	ErrUnknownOracleSource = errors.New("unknown oracle source")
)

// Adaptor represents an oracle adaptor that consumes and normalises data from
// a specific type of oracle.
type Adaptor interface {
	Normalise([]byte) (*oracles.OracleData, error)
}

// Adaptors normalises the input data into an oracles.OracleData according to
// its source.
type Adaptors struct {
	// holds all the supported Adaptor⸱s by source.
	adaptors map[commandspb.OracleDataSubmission_OracleSource]Adaptor
}

// New creates an Adaptors with all the supported oracle Adaptor.
func New() *Adaptors {
	return &Adaptors{
		adaptors: map[commandspb.OracleDataSubmission_OracleSource]Adaptor{
			commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE: NewOpenOracleAdaptor(),
		},
	}
}

// Normalise normalises the input data into an oracles.OracleData based on its source.
func (a *Adaptors) Normalise(data commandspb.OracleDataSubmission) (*oracles.OracleData, error) {
	adaptor, ok := a.adaptors[data.Source]
	if !ok {
		return nil, ErrUnknownOracleSource
	}

	return adaptor.Normalise(data.Payload)
}
