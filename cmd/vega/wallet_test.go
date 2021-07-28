package main

import (
	"context"
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/go-wallet/wallet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *CommandSuite) TestWallet(t *testing.T) {
	path, closer := suite.PrepareSandbox(t)
	defer closer()
	ctx := context.Background()

	// Initialise the wallet
	_, err = suite.RunMain(ctx, "wallet init --no-version-check --root-path %s", path)
	require.NoError(t, err)

	// Generate a Key pair
	_, err = suite.RunMain(ctx, "wallet key generate --no-version-check --root-path %s --passphrase %s/passphrase --name test", path, path)
	require.NoError(t, err)

	// List the wallet and keep it
	keyPairs := suite.ListKeyPairs(t, path)
	require.NotEmpty(t, keyPairs)
	pub := keyPairs[0].PublicKey()

	t.Run("Sign/Verify", func(t *testing.T) {
		// Sign and retrieve the signature (base64 encoded)
		out, err = suite.RunMain(ctx, "wallet sign --no-version-check --root-path %s --passphrase %s/passphrase --name test -m aG9sYQo= -k %s", path, path, pub)
		require.NoError(t, err)
		sig := out

		// Verify
		t.Run("Verify", func(t *testing.T) {
			out, err = suite.RunMain(ctx, "wallet verify --no-version-check --root-path %s --passphrase %s/passphrase --name test -m aG9sYQo= -k %s -s %s", path, path, pub, sig)
			require.NoError(t, err)
			require.Equal(t, "true\n", string(out))
		})
	})

	// Meta
	t.Run("Meta", func(t *testing.T) {
		_, err = suite.RunMain(ctx, "wallet key meta --no-version-check --root-path %s --passphrase %s/passphrase --name test -k %s -m primary:true;asset:BTC", path, path, pub)
		require.NoError(t, err)
		keyPairs := suite.ListKeyPairs(t, path)
		require.NotEmpty(t, keyPairs)

		meta := keyPairs[0].Meta()
		require.Len(t, meta, 2)

		assert.Equal(t, meta[0].Key, "primary")
		assert.Equal(t, meta[0].Value, "true")

		assert.Equal(t, meta[1].Key, "asset")
		assert.Equal(t, meta[1].Value, "BTC")
	})
}

func (suite *CommandSuite) ListKeyPairs(t *testing.T, path string) []wallet.HDKeyPair {
	ctx := context.Background()

	out, err := suite.RunMain(ctx, "wallet key list --no-version-check --root-path %s --passphrase %s/passphrase --name test", path, path)
	require.NoError(t, err)

	w := []wallet.HDKeyPair{}
	// out[23:] is done to skip the `List of all your keys:` prefixing the key
	// pairs list. It will be remove when the CLI support JSON output.
	require.NoError(t, json.Unmarshal(out[23:], &w))

	return w
}
