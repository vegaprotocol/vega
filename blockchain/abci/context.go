package abci

import "context"

var (
	txKey int
)

func TxToContext(ctx context.Context, tx Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

func TxFromContext(ctx context.Context) Tx {
	return ctx.Value(txKey).(Tx)
}
