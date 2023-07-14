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

package adaptors_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/datasource/spec/adaptors"
	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

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
		Payload: dummyOraclePayload(t),
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
				Payload: dummyOraclePayload(t),
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

func dummyOraclePayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"field_1": "value_1",
		"field_2": "value_2",
	})
	if err != nil {
		t.Fatalf("failed to generate random oracle payload in tests: %s", err)
	}

	return payload
}

func internalOraclePayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		spec.BuiltinTimestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("failed to generate internal oracle payload in tests: %s", err)
	}

	return payload
}

type dummyOracleAdaptor struct{}

func (d *dummyOracleAdaptor) Normalise(pk crypto.PublicKey, payload []byte) (*common.Data, error) {
	var data map[string]string
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	return &common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString(pk.Hex(), common.SignerTypePubKey),
		},
		Data: data,
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
				Payload: dummyOraclePayload(t),
			}

			// when
			adaptor := stubbedAdaptors()
			normalisedData, err := adaptor.Normalise(pubKey, rawData)

			// then
			require.NoError(tt, err)
			assert.NotNil(tt, normalisedData)
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
				Payload: internalOraclePayload(tt),
			}

			// when
			adaptor := stubbedAdaptors()
			normalisedData, err := adaptor.Normalise(pubKey, rawData)

			// then
			require.Error(tt, err)
			assert.Nil(tt, normalisedData)
		})
	}
}
