package helpers

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/require"
)

func AddTestMarket(t *testing.T, ctx context.Context, ms *sqlstore.Markets, block entities.Block) entities.Market {
	t.Helper()
	market := entities.Market{
		ID:       entities.MarketID(GenerateID()),
		VegaTime: block.VegaTime,
	}

	err := ms.Upsert(ctx, &market)
	require.NoError(t, err)
	return market
}

func GenerateMarkets(t *testing.T, ctx context.Context, numMarkets int, block entities.Block, ms *sqlstore.Markets) []entities.Market {
	t.Helper()
	markets := make([]entities.Market, numMarkets)
	for i := 0; i < numMarkets; i++ {
		markets[i] = AddTestMarket(t, ctx, ms, block)
	}
	return markets
}
