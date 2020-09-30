package abci

type Option func(*App)

// ReplayProtection protects the node against replay attacks.  It sets a
// toleration thershold between the current block in the chain and the block
// heigh specified in the Tx.  Tx with blocks height >= than (chain's height -
// distance) are rejected with a AbciTxnRejected.  It also keeps a ring-buffer
// to cache seen Tx. The Ring buffer size defines the number of block to cache,
// each block can hold an unlimited number of Txs.
func ReplayProtection(tolerance uint) Option {
	fn := func(app *App) {
		rp := NewReplayProtector(tolerance)
		app.replayProtector = rp
	}

	return fn
}
