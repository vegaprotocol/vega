// Copyright (c) 2022 Gobalsky Labs Limited
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
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/libs/crypto"
)

// JSONAdaptor is an oracle Adaptor for simple oracle data broadcasting.
// Link: https://compound.finance/docs/prices
type JSONAdaptor struct{}

// NewJSONAdaptor creates a new JSONAdaptor.
func NewJSONAdaptor() *JSONAdaptor {
	return &JSONAdaptor{}
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
