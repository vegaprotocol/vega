package clef_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/nodewallets/eth/clef/mocks"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var testAddress = ethCommon.HexToAddress("0x1Ff482D42D1237258A1686102Fa4ba925C23Bc42")

func TestNewWallet(t *testing.T) {
	t.Run("Success", testNewWalletSuccess)
	t.Run("Returns an error if account is not found", testNewWalletAccountNotFound)
	t.Run("Returns an error on RPC call failure", testNewWalletRPCError)
}

func testNewWalletSuccess(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_list", gomock.Any()).
		Times(1).
		DoAndReturn(func(_ context.Context, accs *[]ethCommon.Address, s string, _ ...interface{}) error {
			*accs = append(*accs, testAddress)

			return nil
		})

	wallet, err := clef.NewWallet(clientMock, "http://127.0.0.1:8580", testAddress)
	a.NoError(err)
	a.NotNil(wallet)
}

func testNewWalletAccountNotFound(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_list", gomock.Any()).
		Times(1).
		Return(nil)

	wallet, err := clef.NewWallet(clientMock, "http://127.0.0.1:8580", testAddress)
	a.EqualError(err, "account not found: wallet does not contain account \"0x1fF482d42D1237258a1686102FA4bA925c23bc42\"")
	a.Nil(wallet)
}

func testNewWalletRPCError(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_list", gomock.Any()).
		Times(1).
		Return(fmt.Errorf("something went wrong"))

	wallet, err := clef.NewWallet(clientMock, "http://127.0.0.1:8580", testAddress)
	a.EqualError(err, "account not found: failed to list accounts: failed to call client: something went wrong")
	a.Nil(wallet)
}

func TestGenerateNewWallet(t *testing.T) {
	t.Run("Success", testGenerateNewWalletSuccess)
	t.Run("Returns an error on RPC call failure", testGenerateRPCError)
}

func testGenerateNewWalletSuccess(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_new", gomock.Any()).
		Times(1).
		DoAndReturn(func(_ context.Context, addr *string, _ interface{}, _ ...interface{}) error {
			*addr = testAddress.String()

			return nil
		})

	wallet, err := clef.GenerateNewWallet(clientMock, "http://127.0.0.1:8580")
	a.NoError(err)
	a.NotNil(wallet)
}

func testGenerateRPCError(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_new", gomock.Any()).
		Times(1).
		Return(fmt.Errorf("something went wrong"))

	wallet, err := clef.GenerateNewWallet(clientMock, "http://127.0.0.1:8580")
	a.EqualError(err, "failed to generate account: failed to call client: something went wrong")
	a.Nil(wallet)
}

func TestVersion(t *testing.T) {
	t.Run("Success", testVersionSuccess)
}

func testVersionSuccess(t *testing.T) {
	a := assert.New(t)

	ctrl := gomock.NewController(t)
	clientMock := mocks.NewMockClient(ctrl)

	testVersion := "v1.0.1"

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_list", gomock.Any()).
		Times(1).
		DoAndReturn(func(_ interface{}, accs *[]ethCommon.Address, _ interface{}, _ ...interface{}) error {
			*accs = append(*accs, testAddress)

			return nil
		})

	clientMock.EXPECT().
		CallContext(gomock.Any(), gomock.Any(), "account_version", gomock.Any()).
		Times(1).
		DoAndReturn(func(_ context.Context, version *string, _ interface{}, _ ...interface{}) error {
			*version = testVersion

			return nil
		})

	wallet, err := clef.NewWallet(clientMock, "http://127.0.0.1:8580", testAddress)
	a.NoError(err)
	a.NotNil(wallet)

	v, err := wallet.Version()
	a.NoError(err)
	a.Equal(testVersion, v)
}
