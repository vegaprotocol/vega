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

package adaptors_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/spec/adaptors"
	"code.vegaprotocol.io/vega/libs/crypto"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONAdaptor(t *testing.T) {
	t.Run("Normalising incompatible data fails", testJSONAdaptorNormalisingIncompatibleDataFails)
	t.Run("Normalising compatible and valid data succeeds", testJSONAdaptorNormalisingCompatibleAndValidDataSucceeds)
}

func testJSONAdaptorNormalisingIncompatibleDataFails(t *testing.T) {
	// given
	pubKeyB := []byte("0xdeadbeef")
	pubKey := crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
	rawData, _ := json.Marshal(struct {
		Prices       string
		MarketNumber uint
	}{
		Prices:       "42",
		MarketNumber: 1337,
	})

	// when
	normalisedData, err := adaptors.NewJSONAdaptor().Normalise(pubKey, rawData)

	// then
	assert.Error(t, err)
	assert.Nil(t, normalisedData)
}

func testJSONAdaptorNormalisingCompatibleAndValidDataSucceeds(t *testing.T) {
	pubKeyB := &datapb.Signer_PubKey{
		PubKey: &datapb.PubKey{
			Key: "0xdeadbeef",
		},
	}

	hexPubKey := hex.EncodeToString([]byte(pubKeyB.PubKey.Key))
	pubKey := crypto.NewPublicKey(hexPubKey, []byte(pubKeyB.PubKey.Key))
	oracleData := map[string]string{
		"BTC": "37371.725",
		"ETH": "1412.67",
	}
	rawData, _ := json.Marshal(oracleData)

	// when
	normalisedData, err := adaptors.NewJSONAdaptor().Normalise(pubKey, rawData)

	// then
	require.NoError(t, err)
	assert.NotNil(t, normalisedData)
	assert.Equal(t, fmt.Sprintf("signerPubKey(pubKey(%s))", hexPubKey), normalisedData.Signers[0].Signer.String())
	assert.Equal(t, oracleData["BTC"], normalisedData.Data["BTC"])
	assert.Equal(t, oracleData["ETH"], normalisedData.Data["ETH"])
}
