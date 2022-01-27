package adaptors_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/adaptors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptors(t *testing.T) {
	t.Run("Creating adaptors succeeds", testCreatingAdaptorsSucceeds)
	t.Run("Normalising data from unknown oracle fails", testAdaptorsNormalisingDataFromUnknownOracleFails)
	t.Run("Normalising data from known oracle succeeds", testAdaptorsNormalisingDataFromKnownOracleSucceeds)
	t.Run("Validating data should pass if validators return no errors", testAdaptorValidationSuccess)
	t.Run("Validating data should fail if any validator returns an error", testAdaptorValidationFails)
}

func testCreatingAdaptorsSucceeds(t *testing.T) {
	// when
	as := adaptors.New()

	// then
	assert.NotNil(t, as)
}

func testAdaptorsNormalisingDataFromUnknownOracleFails(t *testing.T) {
	// given
	pubKeyB := []byte("0xdeadbeef")
	pubKey := crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
	rawData := commandspb.OracleDataSubmission{
		Source:  commandspb.OracleDataSubmission_ORACLE_SOURCE_UNSPECIFIED,
		Payload: dummyOraclePayload(),
	}

	// when
	normalisedData, err := stubbedAdaptors().Normalise(pubKey, rawData)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, adaptors.ErrUnknownOracleSource.Error())
	assert.Nil(t, normalisedData)
}

func testAdaptorsNormalisingDataFromKnownOracleSucceeds(t *testing.T) {
	tcs := []struct {
		name   string
		source commandspb.OracleDataSubmission_OracleSource
	}{
		{
			name:   "with Open Oracle source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE,
		}, {
			name:   "with JSON source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			pubKeyB := []byte("0xdeadbeef")
			pubKey := crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
			rawData := commandspb.OracleDataSubmission{
				Source:  tc.source,
				Payload: dummyOraclePayload(),
			}

			// when
			normalisedData, err := stubbedAdaptors().Normalise(pubKey, rawData)

			// then
			require.NoError(t, err)
			assert.NotNil(t, normalisedData)
		})
	}
}

func stubbedAdaptors() *adaptors.Adaptors {
	return &adaptors.Adaptors{
		Adaptors: map[commandspb.OracleDataSubmission_OracleSource]adaptors.Adaptor{
			commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE: &dummyOracleAdaptor{},
			commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON:        &dummyOracleAdaptor{},
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

func internalOraclePayload() []byte {
	payload, err := json.Marshal(map[string]string{
		oracles.BuiltinOracleTimestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	})
	if err != nil {
		panic("failed to generate internal oracle payload in tests")
	}

	return payload
}

type dummyOracleAdaptor struct{}

func (d *dummyOracleAdaptor) Normalise(pk crypto.PublicKey, payload []byte) (*oracles.OracleData, error) {
	var data map[string]string
	err := json.Unmarshal(payload, &data)
	if err != nil {
		return nil, err
	}

	return &oracles.OracleData{
		PubKeys: []string{pk.Hex()},
		Data:    data,
	}, nil
}

func testAdaptorValidationSuccess(t *testing.T) {
	tcs := []struct {
		name   string
		source commandspb.OracleDataSubmission_OracleSource
	}{
		{
			name:   "with Open Oracle source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE,
		}, {
			name:   "with JSON source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			pubKeyB := []byte("0xdeadbeef")
			pubKey := crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
			rawData := commandspb.OracleDataSubmission{
				Source:  tc.source,
				Payload: dummyOraclePayload(),
			}

			// when
			adaptor := stubbedAdaptors()
			normalisedData, err := adaptor.Normalise(pubKey, rawData)

			// then
			require.NoError(t, err)
			assert.NotNil(t, normalisedData)
		})
	}
}

func testAdaptorValidationFails(t *testing.T) {
	tcs := []struct {
		name   string
		source commandspb.OracleDataSubmission_OracleSource
	}{
		{
			name:   "with Open Oracle source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE,
		}, {
			name:   "with JSON source",
			source: commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			pubKeyB := []byte("0xdeadbeef")
			pubKey := crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
			rawData := commandspb.OracleDataSubmission{
				Source:  tc.source,
				Payload: internalOraclePayload(),
			}

			// when
			adaptor := stubbedAdaptors()
			normalisedData, err := adaptor.Normalise(pubKey, rawData)

			// then
			require.Error(t, err)
			assert.Nil(t, normalisedData)
		})
	}
}
