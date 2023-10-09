// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package adaptors

import (
	"errors"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec/validation"
	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

// ErrUnknownOracleSource is used when the input data is originated from an
// unknown, unsupported or unspecified oracle source.
var ErrUnknownOracleSource = errors.New("unknown oracle source")

// Adaptor represents a adaptor that consumes and normalises data from
// a specific type of source.
type Adaptor interface {
	Normalise(crypto.PublicKey, []byte) (*common.Data, error)
}

// Adaptors normalises the input data into an common.Data according to
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

// Normalise normalises the input data into an common.Data based on its source.
func (a *Adaptors) Normalise(txPubKey crypto.PublicKey, data commandspb.OracleDataSubmission) (*common.Data, error) {
	adaptor, ok := a.Adaptors[data.Source]
	if !ok {
		return nil, ErrUnknownOracleSource
	}

	oracleData, err := adaptor.Normalise(txPubKey, data.Payload)
	if err != nil {
		return nil, err
	}

	if err = validation.CheckForInternalOracle(oracleData.Data); err != nil {
		return nil, err
	}

	return oracleData, err
}
