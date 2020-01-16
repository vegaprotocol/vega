package positions_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/plugins/positions"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	p := positions.NewPositionsStore(ctx)
	mkt, trader := "test-market", "trader1"
	data := []types.Position{
		{
			MarketID: mkt,
			PartyID:  trader,
		},
		{
			MarketID: mkt,
			PartyID:  "trader2",
		},
		{
			MarketID: "market-2",
			PartyID:  trader,
		},
	}
	for _, pos := range data {
		p.Add(pos)
	}
	pos, err := p.Get(mkt, trader)
	assert.NoError(t, err)
	assert.Equal(t, data[0], *pos)
	p.Remove(data[0])
	_, err = p.Get(mkt, trader)
	assert.Error(t, err)
}
