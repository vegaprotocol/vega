package abci

type Option func(*App)

// ReplayProtection sets a optional toleration thershold between the
// current block in the chain and the block heigh specified in the Tx.
// Tx with blocks height >= than (chain's height - distance) are rejected with a AbciTxnRejected.
func ReplayProtection(distance uint64) Option {
	fn := func(app *App) {
		app.replayMaxDistance = distance
	}

	return fn
}
