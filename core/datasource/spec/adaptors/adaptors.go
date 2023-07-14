// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
