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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *CommandSuite) TestFaucet(t *testing.T) {
	path, pass, _ := suite.PrepareSandbox(t)
	// defer closer()
	ctx, cancel := context.WithCancel(context.Background())

	_, err = suite.RunMain(ctx, "init --output json --home %s --nodewallet-passphrase-file %s validator", path, pass)
	require.NoError(t, err)

	_, err = suite.RunMain(ctx, "faucet init --output json --home %s -p %s", path, pass)
	require.NoError(t, err)

	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	out, err = suite.RunMain(ctx, "faucet run --home %s -p %s --ip=127.0.0.1 --port=11790", path, pass)
	require.NoError(t, err)

	assert.Contains(t, string(out), "starting faucet server")
	assert.Contains(t, string(out), "127.0.0.1:11790")
	assert.Contains(t, string(out), "server stopped")
}
