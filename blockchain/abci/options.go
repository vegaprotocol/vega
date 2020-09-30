package abci

type Option func(*App)

// ReplayProtection sets a optional toleration thershold between the
// current block in the chain and the block heigh specified in the Tx.
// Tx with blocks height >= than (chain's height - distance) are rejected with a AbciTxnRejected.
func ReplayProtection(distance uint) Option {
	fn := func(app *App) {
		// ReplayProtection is a ring buffer that keeps track of seen Txs
		app.replayProtector = NewReplayProtector(distance)
		app.replayMaxDistance = distance
	}

	return fn
}
