package staking_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/staking/mocks"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestEthereumConfirmations(t *testing.T) {
	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockEthereumClientConfirmations(ctrl)
	tim := mocks.NewMockTime(ctrl)
	ethCfns := staking.NewEthereumConfirmations(ethClient, tim)
	defer ctrl.Finish()

	ethCfns.OnEthereumConfigUpdate(context.Background(), &vgproto.EthereumConfig{
		Confirmations: 30,
	})

	tim.EXPECT().Now().Times(1).Return(time.Unix(10, 0))
	// start a block 10
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(10)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.EqualError(t,
		ethCfns.Check(50),
		staking.ErrMissingConfirmations.Error(),
	)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(15, 0))

	// block 10, request 50, we are in the past, return err
	assert.EqualError(t,
		ethCfns.Check(50),
		staking.ErrMissingConfirmations.Error(),
	)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(26, 0))
	// do block 50 == requested block
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(50)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.EqualError(t,
		ethCfns.Check(50),
		staking.ErrMissingConfirmations.Error(),
	)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(42, 0))
	// do block 79 > requested block < confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(79)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.EqualError(t,
		ethCfns.Check(50),
		staking.ErrMissingConfirmations.Error(),
	)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(58, 0))
	// do block 80 > requested block == confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(80)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.NoError(t, ethCfns.Check(50))

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(1000, 0))
	// do block 80 > requested block == confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(100)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.NoError(t, ethCfns.Check(50))

}
