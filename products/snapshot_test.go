package products_test

import (
	"bytes"
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/products/mocks"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const marketID = "mktID"

func TestSnapshotFutures(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	oe.EXPECT().Subscribe(ctx, gomock.Any(), gomock.Any()).AnyTimes()

	protoInstrument := getValidInstrumentProto()
	prod, err := products.New(ctx, logging.NewTestLogger(), protoInstrument.Product, oe, marketID)
	require.Nil(t, err)

	// Cast back into a future so we can call future specific functions
	f, ok := prod.(*products.Future)
	require.True(t, ok)

	// Check hashes change as state changes
	h1, err := f.GetHash(marketID)
	require.Nil(t, err)
	require.NotNil(t, h1)

	err = f.SetSettlementPrice(ctx, "prices.ETH.value", 12)
	require.Nil(t, err)

	h2, err := f.GetHash(marketID)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))

	err = f.SetTradingTerminated(ctx, true)
	require.Nil(t, err)

	h3, err := f.GetHash(marketID)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h3))
	require.False(t, bytes.Equal(h2, h3))

	state, err := f.GetState(marketID)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapProd, _ := products.New(ctx, logging.NewTestLogger(), protoInstrument.Product, oe, marketID)
	fSnap, _ := snapProd.(*products.Future)

	// Load into a fresh future and check the values have returned
	fSnap.LoadState(types.PayloadFromProto(snap))

	p1, _ := f.SettlementPrice()
	p2, _ := fSnap.SettlementPrice()
	require.Equal(t, p1, p2)
	require.Equal(t, f.IsTradingTerminated(), fSnap.IsTradingTerminated())
}
