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
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/openoracle"
	"code.vegaprotocol.io/vega/libs/crypto"
)

// OpenOracleAdaptor is a specific oracle Adaptor for Open Oracle / Open Price Feed
// standard.
// Link: https://compound.finance/docs/prices
type OpenOracleAdaptor struct{}

// NewOpenOracleAdaptor creates a new OpenOracleAdaptor.
func NewOpenOracleAdaptor() *OpenOracleAdaptor {
	return &OpenOracleAdaptor{}
}

// Normalise normalises an Open Oracle / Open Price Feed payload into an signed.Data.
// The public key from the transaction is not used, only those from the Open
// Oracle data.
func (a *OpenOracleAdaptor) Normalise(_ crypto.PublicKey, data []byte) (*common.Data, error) {
	response, err := openoracle.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal Open Oracle data: %w", err)
	}

	pubKeys, kvs, err := openoracle.Verify(*response)
	if err != nil {
		return nil, fmt.Errorf("invalid Open Oracle response: %w", err)
	}

	pubKeysSigners := make([]*common.Signer, len(pubKeys))
	for i, pk := range pubKeys {
		pubKeysSigners[i] = common.CreateSignerFromString(pk, common.SignerTypeEthAddress)
	}
	return &common.Data{
		Signers: pubKeysSigners,
		Data:    kvs,
	}, nil
}
