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
