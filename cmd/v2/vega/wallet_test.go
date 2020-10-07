package main

import (
	"context"
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/vega/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *CommandSuite) TestWallet(t *testing.T) {
	path, closer := suite.PrepareSandbox(t)
	defer closer()
	ctx := context.Background()

	// Generate a Key pair
	_, err = suite.RunMain(ctx, "wallet genkey -r %s -p %s/passphrase --name test", path, path)
	require.NoError(t, err)

	// List the wallet and keep it
	w := suite.WalletList(t, path)
	require.NotEmpty(t, w.Keypairs)
	pub := w.Keypairs[0].Pub

	t.Run("Sign/Verify", func(t *testing.T) {
		// Sign and retrieve the signature (base64 encoded)
		out, err = suite.RunMain(ctx, "wallet sign -r %s -p %s/passphrase --name test -m aG9sYQo= -k %s", path, path, pub)
		require.NoError(t, err)
		sig := out

		// Verify
		t.Run("Verify", func(t *testing.T) {
			out, err = suite.RunMain(ctx, "wallet verify -r %s -p %s/passphrase --name test -m aG9sYQo= -k %s -s %s", path, path, pub, sig)
			require.NoError(t, err)
			require.Equal(t, "true\n", string(out))
		})
	})

	// Meta
	t.Run("Meta", func(t *testing.T) {
		_, err = suite.RunMain(ctx, "wallet meta -r %s -p %s/passphrase --name test -k %s -m primary:true;asset:BTC", path, path, pub)
		require.NoError(t, err)
		w := suite.WalletList(t, path)
		require.NotEmpty(t, w.Keypairs)

		meta := w.Keypairs[0].Meta
		require.Len(t, meta, 2)

		assert.Equal(t, meta[0].Key, "primary")
		assert.Equal(t, meta[0].Value, "true")

		assert.Equal(t, meta[1].Key, "asset")
		assert.Equal(t, meta[1].Value, "BTC")
	})
}

// WalletList runs the `wallet list` command and returns the wallet.
func (suite *CommandSuite) WalletList(t *testing.T, path string) *wallet.Wallet {
	ctx := context.Background()

	out, err := suite.RunMain(ctx, "wallet list -r %s -p %s/passphrase --name test", path, path)
	require.NoError(t, err)

	var w wallet.Wallet
	require.NoError(t, json.Unmarshal(out, &w))

	return &w
}
