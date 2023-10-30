// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands

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

type CommandSuite struct{}

// RunMain simulates a CLI execution. It formats a cmd invocation given a format and its args and overwrites os.Args.
// The output of the command is captured and returned.
func (suite *CommandSuite) RunMain(ctx context.Context, format string, args ...interface{}) ([]byte, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := fmt.Sprintf(format, args...)
	fmt.Fprintf(old, "-> %s\n", cmd)
	os.Args = append([]string{"vega"}, strings.Fields(cmd)...)
	err := Execute(ctx)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	fmt.Fprintf(old, "<- %s\n", out)
	os.Stdout = old

	return out, err
}

// PrepareSandbox creates a sandbox directory where to run a command.
// It returns the path of the new created directory and a closer function.
func (suite *CommandSuite) PrepareSandbox(t *testing.T) (string, func()) {
	t.Helper()
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
