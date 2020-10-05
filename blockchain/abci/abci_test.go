package abci_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	proto "github.com/tendermint/tendermint/proto/types"
)

type testTx struct {
	payload     []byte
	pubkey      []byte
	hash        []byte
	command     blockchain.Command
	blockHeight uint64
	validateFn  func() error
}

func (tx *testTx) Payload() []byte             { return tx.payload }
func (tx *testTx) PubKey() []byte              { return tx.pubkey }
func (tx *testTx) Hash() []byte                { return tx.hash }
func (tx *testTx) Command() blockchain.Command { return tx.command }
func (tx *testTx) BlockHeight() uint64         { return tx.blockHeight }
func (tx *testTx) Validate() error {
	if fn := tx.validateFn; fn != nil {
		return fn()
	}
	return nil
}

type testCodec struct {
	txs map[string]abci.Tx
}

func newTestCodec() *testCodec {
	return &testCodec{
		txs: map[string]abci.Tx{},
	}
}

func (c *testCodec) addTx(in []byte, tx abci.Tx) *testCodec {
	c.txs[string(in)] = tx
	return c
}

func (c *testCodec) Decode(in []byte) (abci.Tx, error) {
	tx, ok := c.txs[string(in)]
	if !ok {
		return nil, errors.New("tx not defined")
	}
	return tx, nil
}

const (
	testCommandA = blockchain.Command(0x01)
	testCommandB = blockchain.Command(0x02)
	testCommandC = blockchain.Command(0x03)
)

func TestABCICheckTx(t *testing.T) {
	cdc := newTestCodec()

	app := abci.New(cdc).
		HandleCheckTx(testCommandA, func(ctx context.Context, tx abci.Tx) error {
			require.Equal(t, "val", ctx.Value("key"))
			return nil
		}).
		HandleCheckTx(testCommandB, func(ctx context.Context, tx abci.Tx) error {
			require.Equal(t, "val", ctx.Value("key"))
			return errors.New("boom")
		})

	app.OnCheckTx = func(ctx context.Context, req types.RequestCheckTx, _ abci.Tx) (context.Context, types.ResponseCheckTx) {
		resp := types.ResponseCheckTx{}
		return context.WithValue(ctx, "key", "val"), resp
	}

	t.Run("CommandWithNoError", func(t *testing.T) {
		tx := []byte("tx")
		cdc.addTx(tx, &testTx{
			command: testCommandA,
		})

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsOK())
	})

	t.Run("CommandWithError", func(t *testing.T) {
		tx := []byte("tx")
		cdc.addTx(tx, &testTx{
			command: testCommandB,
		})

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsErr())
		require.Equal(t, abci.AbciTxnInternalError, resp.Code)
	})

	t.Run("TxValidationError", func(t *testing.T) {
		tx := []byte("tx")
		cdc.addTx(tx, &testTx{
			command:    testCommandA,
			validateFn: func() error { return errors.New("invalid tx") },
		})

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsErr())
		require.Equal(t, abci.AbciTxnValidationFailure, resp.Code)
	})

	t.Run("TxDecodingError", func(t *testing.T) {
		tx := []byte("tx-not-registered-on-the-codec")

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsErr())
		require.Equal(t, abci.AbciTxnDecodingFailure, resp.Code)
	})
}

// beginBlockN is a helper function that will move the blockchain to a given
// block number by calling BeginBlock with the right parameter.
func beginBlockN(app *abci.App, n int) {
	header := proto.Header{Height: int64(n)}
	app.BeginBlock(types.RequestBeginBlock{
		Header: header,
	})
}

func TestReplayProtectionByDistance(t *testing.T) {
	cdc := newTestCodec()
	tx := []byte("tx")
	cdc.addTx(tx, &testTx{
		blockHeight: 100,
		command:     testCommandA,
	})

	tests := []struct {
		name        string
		height      int
		expectError bool
	}{
		{"within distance: low", 91, false},
		{"within distance: high", 109, false},

		{"same heights", 100, false},

		{"higher distance - short", 110, true},
		{"higher distance - long", 200, true},
	}

	for _, test := range tests {
		app := abci.New(cdc,
			// reject Txs with blockHeight further away than 10 blocks.
			abci.ReplayProtection(10),
		)

		// forward to a given block
		beginBlockN(app, test.height)

		// perform the request (all of them uses blockHeight 100)
		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		t.Run(test.name, func(t *testing.T) {
			if test.expectError {
				require.True(t, resp.IsErr())
				require.Equal(t, abci.AbciTxnValidationFailure, resp.Code)
				require.NotEmpty(t, resp.Info)
			} else {
				require.True(t, resp.IsOK())
			}
		})
	}
}

func TestReplayProtectionByCache(t *testing.T) {
	cdc := newTestCodec()
	tx := []byte("tx")
	cdc.addTx(tx, &testTx{
		blockHeight: 1,
		command:     testCommandA,
	})

	app := abci.New(cdc,
		// reject Txs with blockHeight further away than 10 blocks.
		abci.ReplayProtection(2),
	)

	// forward to a given block
	beginBlockN(app, 0)
	req := types.RequestCheckTx{Tx: tx}
	resp1 := app.CheckTx(req)
	resp2 := app.CheckTx(req)

	require.True(t, resp1.IsOK())
	require.True(t, resp2.IsErr())
	require.Equal(t, abci.ErrTxAlreadyInCache.Error(), resp2.Info)

	beginBlockN(app, 1)
	beginBlockN(app, 2)
	beginBlockN(app, 3)
	resp3 := app.CheckTx(req)
	require.True(t, resp3.IsErr())
	require.Equal(t, abci.ErrTxStaled.Error(), resp3.Info)
}
