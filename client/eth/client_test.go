package eth_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/client/eth/mocks"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNullChain(t *testing.T) {
	t.Run("test valid hash", testValidHash)
	t.Run("test mismatch hash", testMismatchHash)
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
