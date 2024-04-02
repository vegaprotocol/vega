// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package eth_test

import (
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/client/eth"
	localMocks "code.vegaprotocol.io/vega/core/client/eth/mocks"
	"code.vegaprotocol.io/vega/core/staking/mocks"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestEthereumConfirmations(t *testing.T) {
	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockEthereumClientConfirmations(ctrl)
	tim := localMocks.NewMockTime(ctrl)
	cfg := eth.NewDefaultConfig()
	cfg.RetryDelay.Duration = 15 * time.Second
	ethCfns := eth.NewEthereumConfirmations(cfg, ethClient, tim)
	defer ctrl.Finish()

	ethCfns.UpdateConfirmations(30)

	tim.EXPECT().Now().Times(1).Return(time.Unix(10, 0))
	// start a block 10
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(10)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.Check(50), eth.ErrMissingConfirmations)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(15, 0))

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.Check(50), eth.ErrMissingConfirmations)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(26, 0))
	// do block 50 == requested block
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(50)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.Check(50), eth.ErrMissingConfirmations)

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(1).Return(time.Unix(42, 0))
	// do block 79 > requested block < confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(79)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.Check(50), eth.ErrMissingConfirmations)

	// request again but before buf size
	tim.EXPECT().Now().Times(2).Return(time.Unix(58, 0))
	// do block 80 > requested block == confirmations, and also block is seen as finalized
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(2).
		Return(&ethtypes.Header{Number: big.NewInt(80)}, nil)

	assert.NoError(t, ethCfns.Check(50))

	// request again but before buf size
	// no request to eth
	tim.EXPECT().Now().Times(2).Return(time.Unix(1000, 0))
	// do block 80 > requested block == confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(2).
		Return(&ethtypes.Header{Number: big.NewInt(100)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.NoError(t, ethCfns.Check(50))
}

func TestBlockFinalisation(t *testing.T) {
	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockEthereumClientConfirmations(ctrl)
	tim := localMocks.NewMockTime(ctrl)
	cfg := eth.NewDefaultConfig()
	cfg.RetryDelay.Duration = 15 * time.Second
	ethCfns := eth.NewEthereumConfirmations(cfg, ethClient, tim)
	defer ctrl.Finish()

	ethCfns.UpdateConfirmations(10)

	// testing with block 50 where we need 10 confirmations
	// current Ethereum block is 70 so we have enough confirmations, but finalized block is 49
	tim.EXPECT().Now().Times(2).Return(time.Unix(10, 0))
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(70)}, nil)
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(49)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.Check(50), eth.ErrBlockNotFinalized)

	// now we are passed enough confirmations AND the block has been finalized
	tim.EXPECT().Now().Times(2).Return(time.Unix(60, 0))
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(70)}, nil)
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(50)}, nil)
	assert.NoError(t, ethCfns.Check(50))
}

func TestCheckRequiredConfirmations(t *testing.T) {
	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockEthereumClientConfirmations(ctrl)
	tim := localMocks.NewMockTime(ctrl)
	cfg := eth.NewDefaultConfig()
	cfg.RetryDelay.Duration = 15 * time.Second
	ethCfns := eth.NewEthereumConfirmations(cfg, ethClient, tim)
	defer ctrl.Finish()

	tim.EXPECT().Now().Times(1).Return(time.Unix(10, 0))
	// start a block 10
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(10)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.ErrorIs(t, ethCfns.CheckRequiredConfirmations(50, 30), eth.ErrMissingConfirmations)

	tim.EXPECT().Now().Times(1).Return(time.Unix(58, 0))
	// do block 80 > requested block == confirmations
	ethClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Times(1).
		Return(&ethtypes.Header{Number: big.NewInt(80)}, nil)

	// block 10, request 50, we are in the past, return err
	assert.NoError(t, ethCfns.CheckRequiredConfirmations(50, 30))

	tim.EXPECT().Now().Times(1).Return(time.Unix(59, 0))
	assert.ErrorIs(t, ethCfns.CheckRequiredConfirmations(50, 40), eth.ErrMissingConfirmations)
}
