package validators_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/bytes"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func tmTestPubKey() testPubKey {
	return testPubKey{bytes: []byte("test-pub-key")}
}

type testTop struct {
	*validators.Topology
	ctrl *gomock.Controller
	bc   *mocks.MockBlockchainClient
}

func getTestTop(t *testing.T) *testTop {
	ctrl := gomock.NewController(t)
	bc := mocks.NewMockBlockchainClient(ctrl)

	ch := make(chan struct{}, 2)
	bc.EXPECT().GetStatus(gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context) (*tmctypes.ResultStatus, error) {
			defer func() { ch <- struct{}{} }()
			return &tmctypes.ResultStatus{
				ValidatorInfo: tmctypes.ValidatorInfo{
					Address: bytes.HexBytes([]byte("addresstm")),
					PubKey:  tmTestPubKey(),
				},
			}, nil
		},
	)
	bc.EXPECT().GenesisValidators().Times(1).DoAndReturn(
		func() ([]*tmtypes.Validator, error) {
			defer func() { ch <- struct{}{} }()
			return []*tmtypes.Validator{
				&tmtypes.Validator{
					Address: bytes.HexBytes([]byte("addresstm")),
					PubKey:  tmTestPubKey(),
				},
			}, nil
		},
	)

	top := validators.NewTopology(logging.NewTestLogger(), bc)

	_ = <-ch
	_ = <-ch

	return &testTop{
		Topology: top,
		ctrl:     ctrl,
		bc:       bc,
	}
}

func TestValidatorTopology(t *testing.T) {
	t.Run("add node registration - success", testAddNodeRegistrationSuccess)
	t.Run("add node registration - failure", testAddNodeRegistrationFailure)
	t.Run("get len ", testGetLen)
	t.Run("get self tm", testGetSelfTM)
	t.Run("exists", testExists)
	t.Run("ready", testReady)
}

func testAddNodeRegistrationSuccess(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()
	nr := types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)
}

func testReady(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()

	assert.False(t, top.Ready())

	nr := types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	assert.True(t, top.Ready())
}

func testAddNodeRegistrationFailure(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()
	nr := types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	nr = types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key-2"),
	}
	err = top.AddNodeRegistration(&nr)
	assert.Error(t, err)
}

func testGetLen(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()

	// first len is 0
	assert.Equal(t, 0, top.Len())

	nr := types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	assert.Equal(t, 1, top.Len())
}

func testGetSelfTM(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()

	key := top.SelfChainPubKey()
	assert.NotNil(t, key)
}

func testExists(t *testing.T) {
	top := getTestTop(t)
	defer top.ctrl.Finish()

	assert.False(t, top.Exists([]byte("vega-key")))

	nr := types.NodeRegistration{
		ChainPubKey: tmTestPubKey().bytes,
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	assert.True(t, top.Exists([]byte("vega-key")))

}

type testPubKey struct {
	addr  crypto.Address
	bytes []byte
}

func (t testPubKey) Address() crypto.Address { return t.addr }

func (t testPubKey) Bytes() []byte                           { return t.bytes }
func (t testPubKey) VerifyBytes(msg []byte, sig []byte) bool { return true }
func (t testPubKey) Equals(crypto.PubKey) bool               { return false }
