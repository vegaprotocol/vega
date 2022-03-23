package abci_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/txn"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
)

type testTx struct {
	payload     []byte
	pubkey      []byte
	hash        []byte
	signature   []byte
	command     txn.Command
	validateFn  func() error
	blockHeight uint64
	powNonce    uint64
	powTxID     string
}

func (tx *testTx) Unmarshal(interface{}) error { return nil }
func (tx *testTx) GetPoWTID() string           { return tx.powTxID }
func (tx *testTx) GetVersion() uint32          { return 2 }
func (tx *testTx) GetPoWNonce() uint64         { return tx.powNonce }
func (tx *testTx) Signature() []byte           { return tx.signature }
func (tx *testTx) Payload() []byte             { return tx.payload }
func (tx *testTx) PubKey() []byte              { return tx.pubkey }
func (tx *testTx) PubKeyHex() string           { return hex.EncodeToString(tx.pubkey) }
func (tx *testTx) Party() string               { return hex.EncodeToString(tx.pubkey) }
func (tx *testTx) Hash() []byte                { return tx.hash }
func (tx *testTx) Command() txn.Command        { return tx.command }
func (tx *testTx) BlockHeight() uint64         { return tx.blockHeight }
func (tx *testTx) GetCmd() interface{}         { return nil }
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
	testCommandA = txn.Command(0x01)
	testCommandB = txn.Command(0x02)
)

type testCtxKey int

var testKey testCtxKey

func TestABCICheckTx(t *testing.T) {
	cdc := newTestCodec()

	app := abci.New(cdc).
		HandleCheckTx(testCommandA, func(ctx context.Context, tx abci.Tx) error {
			require.Equal(t, "val", ctx.Value(testKey))
			return nil
		}).
		HandleCheckTx(testCommandB, func(ctx context.Context, tx abci.Tx) error {
			require.Equal(t, "val", ctx.Value(testKey))
			return errors.New("boom")
		})

	app.OnCheckTx = func(ctx context.Context, req types.RequestCheckTx, _ abci.Tx) (context.Context, types.ResponseCheckTx) {
		resp := types.ResponseCheckTx{}
		return context.WithValue(ctx, testKey, "val"), resp
	}

	t.Run("CommandWithNoError", func(t *testing.T) {
		tx := []byte("tx1")
		cdc.addTx(tx, &testTx{
			command: testCommandA,
		})

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsOK())
	})

	t.Run("CommandWithError", func(t *testing.T) {
		tx := []byte("tx2")
		cdc.addTx(tx, &testTx{
			command: testCommandB,
		})

		req := types.RequestCheckTx{Tx: tx}
		resp := app.CheckTx(req)
		require.True(t, resp.IsErr())
		require.Equal(t, abci.AbciTxnInternalError, resp.Code)
	})

	t.Run("TxValidationError", func(t *testing.T) {
		tx := []byte("tx3")
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
