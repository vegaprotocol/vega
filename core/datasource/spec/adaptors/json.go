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
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/libs/crypto"
)

// JSONAdaptor is a Adaptor for simple data broadcasting.
// Link: https://compound.finance/docs/prices
type JSONAdaptor struct{}

// NewJSONAdaptor creates a new JSONAdaptor.
func NewJSONAdaptor() *JSONAdaptor {
	return &JSONAdaptor{}
}

// Normalise normalises a JSON payload into an common.Data.
func (a *JSONAdaptor) Normalise(txPubKey crypto.PublicKey, data []byte) (*common.Data, error) {
	kvs := map[string]string{}
	err := json.Unmarshal(data, &kvs)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal JSON data: %w", err)
	}

	return &common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString(txPubKey.Hex(), common.SignerTypePubKey),
		},
		Data: kvs,
	}, nil
}
