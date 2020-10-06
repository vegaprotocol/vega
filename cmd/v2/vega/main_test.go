package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/wallet"

	"github.com/stretchr/testify/require"
)

var (
	err error
	out []byte
)

type CommandSuite struct {
}

func (suite *CommandSuite) RunMain(line string, args ...interface{}) ([]byte, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := fmt.Sprintf(line, args...)
	err := Main(strings.Split(cmd, " ")...)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = old

	fmt.Printf("-> %s\n", cmd)
	fmt.Printf("<- %s\n", out)
	return out, err
}

func (suite *CommandSuite) PrepareSandbox(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir(".", "test-sandbox-*")
	require.NoError(t, err)

	pass := path.Join(dir, "passphrase")
	f, err := os.Create(pass)
	require.NoError(t, err)

	_, err = f.WriteString("the password")
	require.NoError(t, err)
	f.Close()

	return dir, func() {
		os.RemoveAll(dir)
	}
}

func (suite *CommandSuite) TestWallet(t *testing.T) {
	path, closer := suite.PrepareSandbox(t)
	defer closer()

	// Generate a Key pair
	_, err = suite.RunMain("wallet genkey -r ./%s -p ./%s/passphrase --name test", path, path)
	require.NoError(t, err)

	// List the wallet and keep it
	out, err = suite.RunMain("wallet list -r ./%s -p ./%s/passphrase --name test", path, path)
	require.NoError(t, err)
	var w wallet.Wallet
	require.NoError(t, json.Unmarshal(out, &w))
	require.NotEmpty(t, w.Keypairs)

	// Sign and retrieve the signature (base64 encoded)
	out, err = suite.RunMain("wallet sign -r ./%s -p ./%s/passphrase --name test -m aG9sYQo= -k %s", path, path, w.Keypairs[0].Pub)
	require.NoError(t, err)
	sig := out

	// Verify
	out, err = suite.RunMain("wallet verify -r ./%s -p ./%s/passphrase --name test -m aG9sYQo= -k %s -s %s", path, path, w.Keypairs[0].Pub, sig)
	require.NoError(t, err)
	require.Equal(t, "true\n", string(out))
}

func TestSuite(t *testing.T) {
	s := &CommandSuite{}

	t.Run("Wallet", s.TestWallet)
}
