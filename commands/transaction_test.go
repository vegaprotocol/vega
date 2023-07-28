package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestCheckTransaction(t *testing.T) {
	t.Run("Submitting valid transaction succeeds", testSubmittingInvalidSignature)
	t.Run("Submitting valid transaction succeeds", testSubmittingValidTransactionSucceeds)
	t.Run("Submitting empty transaction fails", testSubmittingEmptyTransactionFails)
	t.Run("Submitting nil transaction fails", testSubmittingNilTransactionFails)
	t.Run("Submitting transaction without version fails", testSubmittingTransactionWithoutVersionFails)
	t.Run("Submitting transaction with unsupported version fails", testSubmittingTransactionWithUnsupportedVersionFails)
	t.Run("Submitting transaction without input data fails", testSubmittingTransactionWithoutInputDataFails)
	t.Run("Submitting transaction without signature fails", testSubmittingTransactionWithoutSignatureFails)
	t.Run("Submitting transaction without signature value fails", testSubmittingTransactionWithoutSignatureValueFails)
	t.Run("Submitting transaction without signature algo fails", testSubmittingTransactionWithoutSignatureAlgoFails)
	t.Run("Submitting transaction without from fails", testSubmittingTransactionWithoutFromFails)
	t.Run("Submitting transaction without public key fails", testSubmittingTransactionWithoutPubKeyFromFails)
	t.Run("Submitting transaction with invalid encoding of value fails", testSubmittingTransactionWithInvalidEncodingOfValueFails)
	t.Run("Submitting transaction with invalid encoding of public key fails", testSubmittingTransactionWithInvalidEncodingOfPubKeyFails)
}

func testSubmittingInvalidSignature(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Signature = &commandspb.Signature{
		Value:   crypto.RandomHash(),
		Algo:    "vega/ed25519",
		Version: 1,
	}
	err := checkTransaction(tx)
	require.Error(t, err)
	require.Equal(t, commands.ErrInvalidSignature, err["tx.signature.value"][0])
}

func testSubmittingValidTransactionSucceeds(t *testing.T) {
	tx := newValidTransactionV2(t)

	err := checkTransaction(tx)

	assert.True(t, err.Empty(), err.Error())
}

func testSubmittingEmptyTransactionFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{})

	assert.Error(t, err)
}

func testSubmittingNilTransactionFails(t *testing.T) {
	err := checkTransaction(nil)

	assert.Contains(t, err.Get("tx"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutVersionFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Version = 0

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.version"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithUnsupportedVersionFails(t *testing.T) {
	tcs := []struct {
		name    string
		version uint32
	}{
		{
			name:    "version 1",
			version: 1,
		}, {
			name:    "version 4",
			version: 4,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			tx := newValidTransactionV2(tt)
			tx.Version = commandspb.TxVersion(tc.version)

			err := checkTransaction(tx)

			assert.Contains(tt, err.Get("tx.version"), commands.ErrIsNotSupported)
		})
	}
}

func testSubmittingTransactionWithoutInputDataFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.InputData = []byte{}

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.input_data"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Signature = nil

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.signature"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureValueFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Signature.Value = ""

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.signature.value"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureAlgoFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Signature.Algo = ""

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.signature.algo"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutFromFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.From = nil

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.from"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutPubKeyFromFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.From = &commandspb.Transaction_PubKey{
		PubKey: "",
	}

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.from.pub_key"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithInvalidEncodingOfValueFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.Signature.Value = "invalid-hex-encoding"

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.signature.value"), commands.ErrShouldBeHexEncoded, err.Error())
}

func testSubmittingTransactionWithInvalidEncodingOfPubKeyFails(t *testing.T) {
	tx := newValidTransactionV2(t)
	tx.From = &commandspb.Transaction_PubKey{
		PubKey: "my-pub-key",
	}

	err := checkTransaction(tx)

	assert.Contains(t, err.Get("tx.from.pub_key"), commands.ErrShouldBeAValidVegaPublicKey)
}

func checkTransaction(cmd *commandspb.Transaction) commands.Errors {
	_, err := commands.CheckTransaction(cmd, "testnet")

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}

func newValidTransactionV2(t *testing.T) *commandspb.Transaction {
	t.Helper()

	inputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "USD/BTC",
				OrderId:  "7fa6d9f6a9dfa9f66fada",
			},
		},
	}

	rawInputData, err := proto.Marshal(inputData)
	if err != nil {
		t.Fatal(err)
	}

	return &commandspb.Transaction{
		InputData: rawInputData,
		Signature: &commandspb.Signature{
			Algo:    "vega/ed25519",
			Value:   "876e46defc40030391b5feb2c9bb0b6b68b2d95a6b5fd17a730a46ea73f3b1808420c8c609be6f1c6156e472ecbcd09202f750da000dee41429947a4b7eca00b",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
		},
		Version: 2,
	}
}
