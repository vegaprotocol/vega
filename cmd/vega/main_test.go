package main

import (
	"context"
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

// RunMain simulates a CLI execution. It formats a cmd invocation given a format and its args and overwrites os.Args.
// The output of the command is captured and returned.
func (suite *CommandSuite) RunMain(ctx context.Context, format string, args ...interface{}) ([]byte, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := fmt.Sprintf(format, args...)
	fmt.Fprintf(old, "-> %s\n", cmd)
	os.Args = append([]string{"vega"}, strings.Fields(cmd)...)
	err := Main(ctx)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	fmt.Fprintf(old, "<- %s\n", out)
	os.Stdout = old

	return out, err
}

// PrepareSandbox creates a sandbox directory where to run a command.
// It returns the path of the new created directory and a closer function.
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
	t.Run("Faucet", s.TestFaucet)
}
