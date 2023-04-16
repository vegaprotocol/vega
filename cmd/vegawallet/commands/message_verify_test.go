package cmd_test

import (
	"encoding/base64"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyMessageFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testVerifyMessageFlagsValidFlagsSucceeds)
	t.Run("Missing public key fails", testVerifyMessageFlagsMissingPubKeyFails)
	t.Run("Missing message fails", testVerifyMessageFlagsMissingMessageFails)
	t.Run("Malformed message fails", testVerifyMessageFlagsMalformedMessageFails)
	t.Run("Missing signature fails", testVerifyMessageFlagsMissingSignatureFails)
	t.Run("Malformed signature fails", testVerifyMessageFlagsMalformedSignatureFails)
}

func testVerifyMessageFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	pubKey := vgrand.RandomStr(20)
	decodedMessage := []byte(vgrand.RandomStr(20))
	decodedSignature := []byte(vgrand.RandomStr(20))

	f := &cmd.VerifyMessageFlags{
		PubKey:    pubKey,
		Message:   base64.StdEncoding.EncodeToString(decodedMessage),
		Signature: base64.StdEncoding.EncodeToString(decodedSignature),
	}

	expectedReq := api.AdminVerifyMessageParams{
		PublicKey:        pubKey,
		EncodedMessage:   f.Message,
		EncodedSignature: f.Signature,
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testVerifyMessageFlagsMissingPubKeyFails(t *testing.T) {
	// given
	f := newVerifyMessageFlags(t)
	f.PubKey = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("pubkey"))
	assert.Empty(t, req)
}

func testVerifyMessageFlagsMissingMessageFails(t *testing.T) {
	// given
	f := newVerifyMessageFlags(t)
	f.Message = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("message"))
	assert.Empty(t, req)
}

func testVerifyMessageFlagsMalformedMessageFails(t *testing.T) {
	// given
	f := newVerifyMessageFlags(t)
	f.Message = vgrand.RandomStr(5)

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBase64EncodedError("message"))
	assert.Empty(t, req)
}

func testVerifyMessageFlagsMissingSignatureFails(t *testing.T) {
	// given
	f := newVerifyMessageFlags(t)
	f.Signature = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("signature"))
	assert.Empty(t, req)
}

func testVerifyMessageFlagsMalformedSignatureFails(t *testing.T) {
	// given
	f := newVerifyMessageFlags(t)
	f.Signature = vgrand.RandomStr(5)

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBase64EncodedError("signature"))
	assert.Empty(t, req)
}

func newVerifyMessageFlags(t *testing.T) *cmd.VerifyMessageFlags {
	t.Helper()

	pubKey := vgrand.RandomStr(20)
	decodedMessage := []byte(vgrand.RandomStr(20))
	decodedSignature := []byte(vgrand.RandomStr(20))

	return &cmd.VerifyMessageFlags{
		PubKey:    pubKey,
		Message:   base64.StdEncoding.EncodeToString(decodedMessage),
		Signature: base64.StdEncoding.EncodeToString(decodedSignature),
	}
}
