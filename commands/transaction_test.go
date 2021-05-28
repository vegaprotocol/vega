package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/stretchr/testify/assert"
)

func TestCheckTransaction(t *testing.T) {
	t.Run("Submitting empty transaction fails", testSubmittingEmptyTransactionFails)
	t.Run("Submitting transaction without input data fails", testSubmittingTransactionWithoutInputDataFails)
	t.Run("Submitting transaction with input data succeeds", testSubmittingTransactionWithInputDataSucceeds)
	t.Run("Submitting transaction without signature fails", testSubmittingTransactionWithoutSignatureFails)
	t.Run("Submitting transaction with signature succeeds", testSubmittingTransactionWithSignatureSucceeds)
	t.Run("Submitting transaction without signature bytes fails", testSubmittingTransactionWithoutSignatureBytesFails)
	t.Run("Submitting transaction with signature bytes succeeds", testSubmittingTransactionWithSignatureBytesSucceeds)
	t.Run("Submitting transaction without signature algo fails", testSubmittingTransactionWithoutSignatureAlgoFails)
	t.Run("Submitting transaction with signature algo succeeds", testSubmittingTransactionWithSignatureAlgoSucceeds)
	t.Run("Submitting transaction without from fails", testSubmittingTransactionWithoutFromFails)
	t.Run("Submitting transaction with from succeeds", testSubmittingTransactionWithFromSucceeds)
	t.Run("Submitting transaction without public key fails", testSubmittingTransactionWithoutPubKeyFromFails)
	t.Run("Submitting transaction with public key succeeds", testSubmittingTransactionWithPubKeySucceeds)
	t.Run("Submitting transaction with unsupported algo fails", testSubmittingTransactionWithUnsupportedAlgoFails)
	t.Run("Submitting transaction with unsupported algo succeeds", testSubmittingTransactionWithSupportedAlgoSucceeds)
	t.Run("Submitting transaction with invalid encoding of bytes fails", testSubmittingTransactionWithInvalidEncodingOfBytesFails)
	t.Run("Submitting transaction with valid encoding of bytes succeeds", testSubmittingTransactionWithValidEncodingOfBytesSucceeds)
	t.Run("Submitting transaction with invalid encoding of bytes fails", testSubmittingTransactionWithInvalidEncodingOfPubKeyFails)
	t.Run("Submitting transaction with valid encoding of bytes succeeds", testSubmittingTransactionWithValidEncodingOfPubKeySucceeds)
	t.Run("Submitting transaction with invalid signature fails", testSubmittingTransactionWithInvalidSignatureFails)
	t.Run("Submitting transaction with valid signature succeeds", testSubmittingTransactionWithValidSignatureSucceeds)
}

func testSubmittingEmptyTransactionFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{})
	assert.EqualError(t, err, "tx.from (is required), tx.input_data (is required), tx.signature (is required)")
}

func testSubmittingTransactionWithoutInputDataFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{})

	assert.Contains(t, err.Get("tx.input_data"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithInputDataSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("hello"),
	})

	assert.NotContains(t, err.Get("tx.input_data"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{})

	assert.Contains(t, err.Get("tx.signature"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithSignatureSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		Signature: &commandspb.Signature{},
	})

	assert.NotContains(t, err.Get("tx.signature"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureBytesFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		Signature: &commandspb.Signature{},
	})

	assert.Contains(t, err.Get("tx.signature.bytes"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithSignatureBytesSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		Signature: &commandspb.Signature{
			Bytes: "hello",
		},
	})

	assert.NotContains(t, err.Get("tx.signature.bytes"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutSignatureAlgoFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		Signature: &commandspb.Signature{},
	})

	assert.Contains(t, err.Get("tx.signature.algo"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithSignatureAlgoSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		Signature: &commandspb.Signature{
			Algo: "some-algo",
		},
	})

	assert.NotContains(t, err.Get("tx.signature.algo"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutFromFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{})

	assert.Contains(t, err.Get("tx.from"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithFromSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		From: &commandspb.Transaction_Address{},
	})

	assert.NotContains(t, err.Get("tx.from"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithoutPubKeyFromFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		From: &commandspb.Transaction_PubKey{},
	})

	assert.Contains(t, err.Get("tx.from.pub_key"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithPubKeySucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.NotContains(t, err.Get("tx.from.pub_key"), commands.ErrIsRequired)
}

func testSubmittingTransactionWithUnsupportedAlgoFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "unsupported-algo",
			Bytes: "some-bytes",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.Contains(t, err.Get("tx.signature.algo"), crypto.ErrUnsupportedSignatureAlgorithm)
}

func testSubmittingTransactionWithSupportedAlgoSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "some-bytes",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.NotContains(t, err.Get("tx.signature.algo"), crypto.ErrUnsupportedSignatureAlgorithm)
}

func testSubmittingTransactionWithInvalidEncodingOfBytesFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "invalid-hex-encoding",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.Contains(t, err.Get("tx.signature.bytes"), commands.ErrCannotDecodeSignature)
}

func testSubmittingTransactionWithValidEncodingOfBytesSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "6C6F6F6B206174207468697320637572696F757320666F78",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.NotContains(t, err.Get("tx.signature.bytes"), commands.ErrCannotDecodeSignature)
}

func testSubmittingTransactionWithInvalidEncodingOfPubKeyFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "invalid-hex-encoding",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "my-pub-key",
		},
	})

	assert.Contains(t, err.Get("tx.from.pub_key"), commands.ErrCannotDecodeSignature)
}

func testSubmittingTransactionWithValidEncodingOfPubKeySucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte("some-data"),
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "6C6F6F6B206174207468697320637572696F757320666F78",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "6C6F6F6B206174207468697320637572696F757320666F78",
		},
	})

	assert.NotContains(t, err.Get("tx.from.pub_key"), commands.ErrCannotDecodeSignature)
}

func testSubmittingTransactionWithInvalidSignatureFails(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte{8, 178, 211, 130, 220, 159, 158, 160, 128, 80, 210, 62, 0},
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "8ea1c9baab2919a73b6acd3dae15f515c9d9b191ac2a2cd9e7d7a2f9750da0793a88c8ee96a640e0de64c91d81770299769d4d4d93f81208e17573c836e3a810",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "b82756d3a3c5beff01152d3565e0c5c2235ccbe9c9d29ea4e760d981f53db7c6",
		},
		Version: 1,
	})

	assert.Contains(t, err.Get("tx.signature"), commands.ErrInvalidSignature)
}

func testSubmittingTransactionWithValidSignatureSucceeds(t *testing.T) {
	err := checkTransaction(&commandspb.Transaction{
		InputData: []byte{8, 178, 211, 130, 220, 159, 158, 160, 128, 80, 210, 62, 0},
		Signature: &commandspb.Signature{
			Algo: "vega/ed25519",
			Bytes: "8ea1c9baab2919a73b6acd3dae15f515c9d9b191ac2a2cd9e7d7a2f9750da0793a88c8ee96a640e0de64c91d81770299769d4d4d93f81208e17573c836e3a80d",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "b82756d3a3c5beff01152d3565e0c5c2235ccbe9c9d29ea4e760d981f53db7c6",
		},
		Version: 1,
	})

	assert.NotContains(t, err.Get("tx.signature"), commands.ErrInvalidSignature)
}

func checkTransaction(cmd *commandspb.Transaction) commands.Errors {
	err := commands.CheckTransaction(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
