package main

import (
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/vega/wallet"
	"github.com/stretchr/testify/require"
)

func (suite *CommandSuite) TestWallet(t *testing.T) {
	path, closer := suite.PrepareSandbox(t)
	defer closer()

	// Generate a Key pair
	_, err = suite.RunMain("wallet genkey -r %s -p %s/passphrase --name test", path, path)
	require.NoError(t, err)

	// List the wallet and keep it
	out, err = suite.RunMain("wallet list -r %s -p %s/passphrase --name test", path, path)
	require.NoError(t, err)
	var w wallet.Wallet
	require.NoError(t, json.Unmarshal(out, &w))
	require.NotEmpty(t, w.Keypairs)

	// Sign and retrieve the signature (base64 encoded)
	out, err = suite.RunMain("wallet sign -r %s -p %s/passphrase --name test -m aG9sYQo= -k %s", path, path, w.Keypairs[0].Pub)
	require.NoError(t, err)
	sig := out

	// Verify
	out, err = suite.RunMain("wallet verify -r %s -p %s/passphrase --name test -m aG9sYQo= -k %s -s %s", path, path, w.Keypairs[0].Pub, sig)
	require.NoError(t, err)
	require.Equal(t, "true\n", string(out))
}
