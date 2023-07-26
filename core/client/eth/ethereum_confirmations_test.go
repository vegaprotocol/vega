// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
