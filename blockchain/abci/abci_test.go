package abci_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
)

type testTx struct {
	payload    []byte
	pubkey     []byte
	command    blockchain.Command
	validateFn func() error
}

func (tx *testTx) Payload() []byte             { return tx.payload }
func (tx *testTx) PubKey() []byte              { return tx.pubkey }
func (tx *testTx) Command() blockchain.Command { return tx.command }
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

	app.OnCheckTx = func(ctx context.Context, req types.RequestCheckTx) (context.Context, types.ResponseCheckTx) {
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
