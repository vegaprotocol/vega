package cmd_test

import (
	"fmt"
	"testing"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendTxFlags(t *testing.T) {
	t.Run("Valid flags succeeds", testSendTxFlagsValidFlagsSucceeds)
	t.Run("Missing log level fails", testSendTxFlagsMissingLogLevelFails)
	t.Run("Unsupported log level fails", testSendTxFlagsUnsupportedLogLevelFails)
	t.Run("Missing network and node address fails", testSendTxFlagsMissingNetworkAndNodeAddressFails)
	t.Run("Both network and node address specified fails", testSendTxFlagsBothNetworkAndNodeAddressSpecifiedFails)
	t.Run("Missing tx fails", testSendTxFlagsMissingTxFails)
	t.Run("Malformed tx fails", testSendTxFlagsMalformedTxFails)
}

func testSendTxFlagsValidFlagsSucceeds(t *testing.T) {
	// given
	network := vgrand.RandomStr(10)

	encodedTx := "ChwIxZXB58qn4K06EMC2BPI+CwoHc29tZS1pZBACEpMBCoABMTM1ZDdmN2Q4MjhkMjg3ZDMyNDQzYjQ2NGEyZDQwNTkyZjQ1OTgwMGQ0MGZmMzY5Y2VhMGFkZDUzZmZjNjYzYzlkZmU2YTI4MGIxZWI4MjdiOTJmYmY2NTY3NzI3MjgwYzMwODBiNjg5NGYyMjYzZmJlYmFkN2I2M2VhN2M4MGYSDHZlZ2EvZWQyNTUxORgBgH0B0j5AZjM4MTc5NjljZDMxNmQ1NmMzN2EzYzE5MjVjMDMyOWM5ZTMxMDQ0ODI5OGZmNzYyMjMwMTVjN2QyY2RiOTFiOQ=="
	f := &cmd.SendRawTransactionFlags{
		Network:     network,
		NodeAddress: "",
		Retries:     10,
		LogLevel:    "debug",
		RawTx:       encodedTx,
	}

	expectedReq := api.AdminSendRawTransactionParams{
		Network:            network,
		NodeAddress:        "",
		Retries:            10,
		EncodedTransaction: encodedTx,
		SendingMode:        "TYPE_ASYNC",
	}

	// when
	req, err := f.Validate()

	// then
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, expectedReq, req)
}

func testSendTxFlagsMissingLogLevelFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.LogLevel = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MustBeSpecifiedError("level"))
	assert.Empty(t, req)
}

func testSendTxFlagsUnsupportedLogLevelFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.LogLevel = vgrand.RandomStr(5)

	// when
	req, err := f.Validate()

	// then
	assert.EqualError(t, err, fmt.Sprintf("unsupported log level %q, supported levels: debug, info, warn, error", f.LogLevel))
	assert.Empty(t, req)
}

func testSendTxFlagsMissingNetworkAndNodeAddressFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.Network = ""
	f.NodeAddress = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.OneOfFlagsMustBeSpecifiedError("network", "node-address"))
	assert.Empty(t, req)
}

func testSendTxFlagsBothNetworkAndNodeAddressSpecifiedFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.Network = vgrand.RandomStr(10)
	f.NodeAddress = vgrand.RandomStr(10)

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.MutuallyExclusiveError("network", "node-address"))
	assert.Empty(t, req)
}

func testSendTxFlagsMissingTxFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.RawTx = ""

	// when
	req, err := f.Validate()

	// then
	assert.ErrorIs(t, err, flags.ArgMustBeSpecifiedError("transaction"))
	assert.Empty(t, req)
}

func testSendTxFlagsMalformedTxFails(t *testing.T) {
	// given
	f := newSendTxFlags(t)
	f.RawTx = vgrand.RandomStr(5)

	// when
	req, err := f.Validate()

	// then
	assert.Error(t, err)
	assert.Empty(t, req)
}

func newSendTxFlags(t *testing.T) *cmd.SendRawTransactionFlags {
	t.Helper()

	networkName := vgrand.RandomStr(10)

	return &cmd.SendRawTransactionFlags{
		Network:     networkName,
		NodeAddress: "",
		Retries:     10,
		LogLevel:    "debug",
		RawTx:       "ChsItbycz7nhsO4/EPZ38j4LCgdzb21lLWlkEAISkwEKgAE4NjNjY2NhZGU5OTM5NTU5NWFmMmRkYjc4MTRiM2Q0NTE4NTllNDljNGRkZjUwYjRkZTJkOGUwNTBhY2U2YTQzOTM4OGJmMmFiN2E0N2NhZDM3MjQ3YWEwNzU1Yzk5NmMxZDJmMDY4MTI1YzY5NGVlNGNiMmU4ZWEyZmE2YmYwNRIMdmVnYS9lZDI1NTE5GAGAfQHSPkBmMzgxNzk2OWNkMzE2ZDU2YzM3YTNjMTkyNWMwMzI5YzllMzEwNDQ4Mjk4ZmY3NjIyMzAxNWM3ZDJjZGI5MWI5",
	}
}
