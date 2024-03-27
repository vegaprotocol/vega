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
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/client/eth/mocks"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
	if !assert.NoError(t, err, "CurrentHeight()") {
		return
	}
	assert.Equal(t, number.Uint64(), got, "CurrentHeight()")
}

type testClient struct {
	ctrl          *gomock.Controller
	client        *eth.PrimaryClient
	mockEthClient *mocks.MockETHClient
}

func getTestClient(t *testing.T) *testClient {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockEthClient := mocks.NewMockETHClient(ctrl)
	c := &eth.PrimaryClient{ETHClient: mockEthClient}
	mockEthClient.EXPECT().ChainID(gomock.Any()).Return(big.NewInt(1), nil).AnyTimes()

	return &testClient{
		ctrl:          ctrl,
		client:        c,
		mockEthClient: mockEthClient,
	}
}
