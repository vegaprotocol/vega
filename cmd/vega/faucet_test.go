package main

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
