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
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/client/eth/mocks"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
)

func TestNullChain(t *testing.T) {
	t.Run("test valid hash", testValidHash)
	t.Run("test mismatch hash", testMismatchHash)
	t.Run("test current block", testCurrentBlock)
}

func testValidHash(t *testing.T) {
	contractAddress := "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	byteCode := "BC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	contractCode, _ := hex.DecodeString(byteCode + "a2640033")
	ethAddress := ethcommon.HexToAddress(contractAddress)

	c := getTestClient(t)
	defer c.ctrl.Finish()

	c.mockEthClient.EXPECT().CodeAt(gomock.Any(), ethAddress, gomock.Any()).Times(1).Return(contractCode, nil)

	// get expected hash
	asBytes, _ := hex.DecodeString(byteCode)
	err := c.client.VerifyContract(context.Background(), ethAddress, hex.EncodeToString(vgcrypto.Hash(asBytes)))
	assert.NoError(t, err)
}

func testMismatchHash(t *testing.T) {
	contractAddress := "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	byteCode := "BC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	contractCode, _ := hex.DecodeString(byteCode + "a2640033")

	c := getTestClient(t)
	defer c.ctrl.Finish()

	c.mockEthClient.EXPECT().CodeAt(gomock.Any(), ethcommon.HexToAddress(contractAddress), gomock.Any()).Times(1).Return(contractCode, nil)

	err := c.client.VerifyContract(context.Background(), ethcommon.HexToAddress(contractAddress), "iamnotthehashyouarelookingfor")
	assert.ErrorIs(t, err, eth.ErrUnexpectedContractHash)
}

func testCurrentBlock(t *testing.T) {
	number := big.NewInt(19)
	c := getTestClient(t)
	c.mockEthClient.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Return(&types.Header{Number: number}, nil).AnyTimes()

	defer c.ctrl.Finish()

	got, err := c.client.CurrentHeight(context.Background())
	if !assert.NoError(t, err, fmt.Sprintf("CurrentHeight()")) {
		return
	}
	assert.Equal(t, number.Uint64(), got, "CurrentHeight()")
}

type testClient struct {
	ctrl          *gomock.Controller
	client        *eth.Client
	mockEthClient *mocks.MockETHClient
}

func getTestClient(t *testing.T) *testClient {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockEthClient := mocks.NewMockETHClient(ctrl)
	c := &eth.Client{ETHClient: mockEthClient}
	mockEthClient.EXPECT().ChainID(gomock.Any()).Return(big.NewInt(1), nil).AnyTimes()

	return &testClient{
		ctrl:          ctrl,
		client:        c,
		mockEthClient: mockEthClient,
	}
}
