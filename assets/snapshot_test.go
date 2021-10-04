package assets_test

import (
	"bytes"
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func testAssets(t *testing.T) *assets.Service {
	conf := assets.NewDefaultConfig()
	logger := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	ts := mocks.NewMockTimeService(ctrl)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	as := assets.New(logger, conf, nil, nil, ts)
	return as
}

// test round trip of active snapshot hash and serialisation
func TestActiveSnapshotRoundTrip(t *testing.T) {
	activeKey := (&types.PayloadActiveAssets{}).Key()
	for i := 0; i < 10; i++ {
		as := testAssets(t)
		_, err := as.NewAsset("asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable("asset1")
		require.Nil(t, err)
		_, err = as.NewAsset("asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		err = as.Enable("asset2")
		require.Nil(t, err)

		// get the has and serialised state
		hash, err := as.GetHash(activeKey)
		require.Nil(t, err)
		state, err := as.GetState(activeKey)
		require.Nil(t, err)

		// verify hash is consistent in the absence of change
		hashNoChange, err := as.GetHash(activeKey)
		require.Nil(t, err)
		stateNoChange, err := as.GetState(activeKey)
		require.Nil(t, err)

		require.True(t, bytes.Equal(hash, hashNoChange))
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var active snapshot.ActiveAssets
		proto.Unmarshal(state, &active)

		payload := &types.Payload{
			Data: &types.PayloadActiveAssets{
				ActiveAssets: types.ActiveAssetsFromProto(&active),
			},
		}

		err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		hashPostReload, _ := as.GetHash(activeKey)
		require.True(t, bytes.Equal(hash, hashPostReload))
		statePostReload, _ := as.GetState(activeKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}

}

// test round trip of active snapshot hash and serialisation
func TestPendingSnapshotRoundTrip(t *testing.T) {
	pendingKey := (&types.PayloadPendingAssets{}).Key()

	for i := 0; i < 10; i++ {
		as := testAssets(t)
		_, err := as.NewAsset("asset1", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)
		_, err = as.NewAsset("asset2", &types.AssetDetails{
			Source: &types.AssetDetailsBuiltinAsset{},
		})
		require.Nil(t, err)

		// get the has and serialised state
		hash, err := as.GetHash(pendingKey)
		require.Nil(t, err)
		state, err := as.GetState(pendingKey)
		require.Nil(t, err)

		// verify hash is consistent in the absence of change
		hashNoChange, err := as.GetHash(pendingKey)
		require.Nil(t, err)
		stateNoChange, err := as.GetState(pendingKey)
		require.Nil(t, err)

		require.True(t, bytes.Equal(hash, hashNoChange))
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var pending snapshot.PendingAssets
		proto.Unmarshal(state, &pending)

		payload := &types.Payload{
			Data: &types.PayloadPendingAssets{
				PendingAssets: types.PendingAssetsFromProto(&pending),
			},
		}

		err = as.LoadState(context.Background(), payload)
		require.Nil(t, err)
		hashPostReload, _ := as.GetHash(pendingKey)
		require.True(t, bytes.Equal(hash, hashPostReload))
		statePostReload, _ := as.GetState(pendingKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}
