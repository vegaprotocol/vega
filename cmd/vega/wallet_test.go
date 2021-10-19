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
	path, pass, closer := suite.PrepareSandbox(t)
	defer closer()
	ctx := context.Background()

	// Initialise the wallet
	_, err = suite.RunMain(ctx, "wallet init --output json --no-version-check --home %s", path)
	require.NoError(t, err)

	// Generate a Key pair
	_, err = suite.RunMain(ctx, "wallet key generate --output json --no-version-check --home %s --passphrase-file %s --wallet test", path, pass)
	require.NoError(t, err)

	// List the wallet and keep it
	keyPairs := suite.ListKeyPairs(t, path, pass)
	require.NotEmpty(t, keyPairs)

	pub := keyPairs[0].PublicKey()

	t.Run("Sign/Verify", func(t *testing.T) {
		// Sign and retrieve the signature (base64 encoded)
		out, err = suite.RunMain(ctx, "wallet sign --output json --no-version-check --home %s --passphrase-file %s --wallet test -m aG9sYQo= -k %s", path, pass, pub)
		require.NoError(t, err)
		sig := struct {
			Signature string `json:"signature"`
		}{}
		err := json.Unmarshal(out, &sig)
		require.NoError(t, err)

		// Verify
		t.Run("Verify", func(t *testing.T) {
			out, err = suite.RunMain(ctx, "wallet verify --output json --no-version-check --home %s -m aG9sYQo= -k %s -s %s", path, pub, sig.Signature)
			require.NoError(t, err)

			verify := struct {
				IsValid bool `json:"isValid"`
			}{}
			err := json.Unmarshal(out, &verify)
			require.NoError(t, err)
			require.True(t, verify.IsValid)
		})
	})

	// Meta
	t.Run("Meta", func(t *testing.T) {
		_, err = suite.RunMain(ctx, "wallet key annotate --output json --no-version-check --home %s --passphrase-file %s --wallet test -k %s -m primary:true;asset:BTC", path, pass, pub)
		require.NoError(t, err)
		keyPairs := suite.ListKeyPairs(t, path, pass)
		require.NotEmpty(t, keyPairs)

		meta := keyPairs[0].Meta()
		require.Len(t, meta, 2)

		assert.Equal(t, meta[0].Key, "primary")
		assert.Equal(t, meta[0].Value, "true")

		assert.Equal(t, meta[1].Key, "asset")
		assert.Equal(t, meta[1].Value, "BTC")
	})
}

func (suite *CommandSuite) ListKeyPairs(t *testing.T, path, pass string) []wallet.HDKeyPair {
	t.Helper()
	ctx := context.Background()

	out, err := suite.RunMain(ctx, "wallet key list --output json --no-version-check --home %s --passphrase-file %s --wallet test", path, pass)
	require.NoError(t, err)

	w := []wallet.HDKeyPair{}
	require.NoError(t, json.Unmarshal(out, &w))

	return w
}
