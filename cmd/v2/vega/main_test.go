package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

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

func TestSuite(t *testing.T) {
	s := &CommandSuite{}

	t.Run("Wallet", s.TestWallet)
}
