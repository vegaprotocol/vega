// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/require"
)

var (
	err error
	out []byte
)

type CommandSuite struct{}

// RunMain simulates a CLI execution. It formats a cmd invocation given a format and its args and overwrites os.Args.
// The output of the command is captured and returned.
func (suite *CommandSuite) RunMain(ctx context.Context, format string, args ...interface{}) ([]byte, error) {
	stdout := os.Stdout
	stderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	cmd := fmt.Sprintf(format, args...)
	fmt.Fprintf(stdout, "-> %s\n", cmd)
	os.Args = append([]string{"vega"}, strings.Fields(cmd)...)
	err := Main(ctx)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	fmt.Fprintf(stdout, "<- %s\n", out)
	os.Stdout = stdout
	os.Stderr = stderr

	return out, err
}

// PrepareSandbox creates a sandbox directory where to run a command.
// It returns the path of the new created directory and a closer function.
func (suite *CommandSuite) PrepareSandbox(t *testing.T) (string, string, func()) {
	t.Helper()
	path := filepath.Join("/tmp", "vega-tests", "test-sandbox", vgrand.RandomStr(10))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}

	pass := filepath.Join(path, "passphrase")
	f, err := os.Create(pass)
	require.NoError(t, err)

	_, err = f.WriteString("the password")
	require.NoError(t, err)
	f.Close()

	return path, pass, func() {
		os.RemoveAll(path)
	}
}

func TestSuite(t *testing.T) {
	s := &CommandSuite{}

	t.Run("Faucet", s.TestFaucet)
}
