package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNotary_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockNotaryStore(ctrl)

	store.EXPECT().Add(gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewNotary(store, logging.NewTestLogger())
	err := subscriber.Push(
		events.NewNodeSignatureEvent(context.Background(),
			v1.NodeSignature{
				Id:   "someid",
				Sig:  []byte("somesig"),
				Kind: v1.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL,
			},
		),
	)
	require.NoError(t, err)
}

func TestNotary_PushWrongEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockNotaryStore(ctrl)
	subscriber := sqlsubscribers.NewNotary(store, logging.NewTestLogger())
	err := subscriber.Push(events.NewOracleDataEvent(context.Background(), oraclespb.OracleData{}))
	require.Error(t, err)
}
