package eth_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	t.Run("test unknown address", testUnknownAddress)
}

func testValidHash(t *testing.T) {
	contractAddress := "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	contractCode := make([]byte, 64)
	rand.Read(contractCode)

	c := getTestClient(t)
	defer c.ctrl.Finish()

	c.mockEthClient.EXPECT().CodeAt(gomock.Any(), ethcommon.HexToAddress(contractAddress), gomock.Any()).Times(1).Return(contractCode, nil)

	// Inject address map
	eth.ContractHashes = map[string]string{
		contractAddress: hex.EncodeToString(vgcrypto.Hash(contractCode)),
	}

	err := c.client.VerifyContract(context.Background(), contractAddress)
	assert.NoError(t, err)
}

func testMismatchHash(t *testing.T) {
	contractAddress := "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8"
	contractCode := make([]byte, 64)
	rand.Read(contractCode)

	c := getTestClient(t)
	defer c.ctrl.Finish()

	c.mockEthClient.EXPECT().CodeAt(gomock.Any(), ethcommon.HexToAddress(contractAddress), gomock.Any()).Times(1).Return(contractCode, nil)

	// Inject address map
	eth.ContractHashes = map[string]string{
		contractAddress: "someinvalidhash",
	}

	err := c.client.VerifyContract(context.Background(), contractAddress)
	assert.ErrorIs(t, err, eth.ErrContractHashMismatch)
}

func testUnknownAddress(t *testing.T) {
	c := getTestClient(t)
	defer c.ctrl.Finish()

	err := c.client.VerifyContract(context.Background(), "HELLO")
	assert.ErrorIs(t, err, eth.ErrUnexpectedContractAddress)
}

type testClient struct {
	ctrl          *gomock.Controller
	client        *eth.Client
	mockEthClient *mocks.MockETHClient
}

func getTestClient(t *testing.T) *testClient {
	ctrl := gomock.NewController(t)
	mockEthClient := mocks.NewMockETHClient(ctrl)
	c := &eth.Client{ETHClient: mockEthClient}

	return &testClient{
		ctrl:          ctrl,
		client:        c,
		mockEthClient: mockEthClient,
	}
}
