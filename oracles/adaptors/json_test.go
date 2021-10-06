package adaptors_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/oracles/adaptors"
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
	// given
	pubKeyB := []byte("0xdeadbeef")
	hexPubKey := hex.EncodeToString(pubKeyB)
	pubKey := crypto.NewPublicKey(hexPubKey, pubKeyB)
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
	assert.Equal(t, []string{hexPubKey}, normalisedData.PubKeys)
	assert.Equal(t, oracleData["BTC"], normalisedData.Data["BTC"])
	assert.Equal(t, oracleData["ETH"], normalisedData.Data["ETH"])
}
