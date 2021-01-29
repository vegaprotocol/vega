package adaptors

import (
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestAdaptors(t *testing.T) {
	t.Run("Normalising data from unknown oracle fails", testAdaptorsNormalisingDataFromUnknownOracleFails)
	t.Run("Normalising data from known oracle succeeds", testAdaptorsNormalisingDataFromKnownOracleSucceeds)
}

func testAdaptorsNormalisingDataFromUnknownOracleFails(t *testing.T) {
	// given
	rawData := types.OracleDataSubmission{
		Source:  types.OracleDataSubmission_ORACLE_SOURCE_UNSPECIFIED,
		Payload: dummyOraclePayload(),
	}

	// when
	normalisedData, err := stubbedAdaptors().Normalise(rawData)

	// then
	if assert.Error(t, err) {
		assert.Equal(t, "unknown oracle source", err.Error())
	}
	assert.Nil(t, normalisedData)
}

func testAdaptorsNormalisingDataFromKnownOracleSucceeds(t *testing.T) {
	// given
	rawData := types.OracleDataSubmission{
		Source:  types.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE,
		Payload: dummyOraclePayload(),
	}

	// when
	normalisedData, err := stubbedAdaptors().Normalise(rawData)

	// then
	assert.NoError(t, err)
	assert.NotNil(t, normalisedData)
}

func stubbedAdaptors() *Adaptors {
	return &Adaptors{
		adaptors: map[types.OracleDataSubmission_OracleSource]Adaptor{
			types.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE: &dummyOracleAdaptor{},
		},
	}
}

func dummyOraclePayload() []byte {
	payload, err := json.Marshal(map[string]string{
		"field_1": "value_1",
		"field_2": "value_2",
	})
	if err != nil {
		panic("failed to generate random oracle payload in tests")
	}

	return payload
}

type dummyOracleAdaptor struct {
}

func (d *dummyOracleAdaptor) Normalise(payload []byte) (*oracles.OracleData, error) {
	data := &oracles.OracleData{}
	err := json.Unmarshal(payload, data)
	return data, err
}
