package gql

import (
	"context"

	vega "code.vegaprotocol.io/vega/protos/vega"
)

type ledgerEntryResolver VegaResolverRoot

func (le ledgerEntryResolver) FromAccountID(ctx context.Context, obj *vega.LedgerEntry) (*vega.AccountDetails, error) {
	return obj.FromAccount, nil
}

func (le ledgerEntryResolver) ToAccountID(ctx context.Context, obj *vega.LedgerEntry) (*vega.AccountDetails, error) {
	return obj.ToAccount, nil
}
