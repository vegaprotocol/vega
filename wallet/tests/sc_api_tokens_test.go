package tests_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAPITokens(t *testing.T) {
	// given
	home := t.TempDir()
	_, tokensPassphraseFilePath := NewPassphraseFile(t, home)
	_, walletPassphraseFilePath := NewPassphraseFile(t, home)
	walletName := vgrand.RandomStr(5)

	// when
	err := InitAPIToken(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
	})

	// then
	require.NoError(t, err)

	// when
	listTokensResp1, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp1)
	require.Len(t, listTokensResp1.Tokens, 0)

	// when
	createWalletResp, err := WalletCreate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet", walletName,
		"--passphrase-file", walletPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertCreateWallet(t, createWalletResp).
		WithName(walletName).
		LocatedUnder(home)

	// when
	generateAPITokenResp1, err := APITokenGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet-name", walletName,
		"--wallet-passphrase-file", walletPassphraseFilePath,
		"--tokens-passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertGenerateAPIToken(t, generateAPITokenResp1)

	// when
	generateAPITokenResp2, err := APITokenGenerate(t, []string{
		"--home", home,
		"--output", "json",
		"--wallet-name", walletName,
		"--wallet-passphrase-file", walletPassphraseFilePath,
		"--tokens-passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	AssertGenerateAPIToken(t, generateAPITokenResp2)

	// when
	listTokensResp2, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp2)
	require.Len(t, listTokensResp2.Tokens, 2)
	assert.Equal(t, generateAPITokenResp1.Token, listTokensResp2.Tokens[0].Token)
	assert.Equal(t, generateAPITokenResp2.Token, listTokensResp2.Tokens[1].Token)

	// when
	err = APITokenDelete(t, []string{
		"--home", home,
		"--token", generateAPITokenResp1.Token,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
		"--force",
	})

	// then
	require.NoError(t, err)

	// when
	listTokensResp3, err := APITokensList(t, []string{
		"--home", home,
		"--passphrase-file", tokensPassphraseFilePath,
		"--output", "json",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, listTokensResp3)
	require.Len(t, listTokensResp3.Tokens, 1)
	assert.Equal(t, generateAPITokenResp2.Token, listTokensResp3.Tokens[0].Token)
}
