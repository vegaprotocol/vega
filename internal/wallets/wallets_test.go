package wallets_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/wallets"
	"code.vegaprotocol.io/vega/internal/wallets/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	traderID   = "sometrader"
	traderID2  = "sometrader2"
	asset      = "ETH"
	otherAsset = "BTC"
)

type testWallets struct {
	wallets *wallets.Wallets
	ctrl    *gomock.Controller
	buffer  *mocks.MockBuffer
}

func getTestWallets(t *testing.T) *testWallets {
	ctrl := gomock.NewController(t)
	buf := mocks.NewMockBuffer(ctrl)
	wallets := wallets.New(logging.NewTestLogger(), buf)
	return &testWallets{
		wallets: wallets,
		ctrl:    ctrl,
		buffer:  buf,
	}
}

func TestWallets(t *testing.T) {
	t.Run("create wallet", testWalletCreate)
	t.Run("error on non-existing wallet", testErrorIfNotExist)
	t.Run("withdraw success", testWithdrawSuccess)
	t.Run("withdraw failure", testWithdrawFailure)
	t.Run("credit success", testCreditSuccess)
	t.Run("move success", testMoveSuccess)
	// t.Run("move failure", test.MoveFailure)
}

func testMoveSuccess(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(5).Return()

	// create first trader
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, asset)
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID2, asset)

	// credit trader1
	// by default when created the wallet has no monies
	balance, err := tw.wallets.Credit(wallets.TraderWalletType, traderID, asset, 100)
	assert.Equal(t, int64(100), balance)
	assert.Nil(t, err)

	// check the balance now
	balance, err = tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, balance, int64(100))
	assert.Nil(t, err)

	// now move monies from trader to trader2
	// trader1 should have 60, trader2 should have 40
	fromBalance, toBalance, err := tw.wallets.Move(
		wallets.TraderWalletType, traderID, // from
		wallets.TraderWalletType, traderID2, // to
		asset, 40)
	assert.Equal(t, fromBalance, int64(60))
	assert.Equal(t, toBalance, int64(40))
	assert.Nil(t, err)
}

func testWithdrawSuccess(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	// create the wallet an add monies
	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).Return()
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, asset)
	_, _ = tw.wallets.Credit(wallets.TraderWalletType, traderID, asset, 100)

	// now try to remove 60, left should be 40
	balance, err := tw.wallets.Withdraw(wallets.TraderWalletType, traderID, asset, 60)
	assert.Equal(t, balance, int64(40))
	assert.Nil(t, err)

	// check the balance now
	balance, err = tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, balance, int64(40))
	assert.Nil(t, err)
}

func testWithdrawFailure(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	// test with non existing wallet
	balance, err := tw.wallets.Withdraw(wallets.TraderWalletType, traderID, asset, 100)
	assert.Equal(t, int64(0), balance)
	assert.NotNil(t, err)
	assert.Equal(t, err, wallets.ErrNoAccountForTrader)

	// test with an existing wallet
	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return()
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, asset)

	// by default when created the wallet has no monies
	balance, err = tw.wallets.Withdraw(wallets.TraderWalletType, traderID, asset, 100)
	assert.Equal(t, int64(0), balance)
	assert.NotNil(t, err)
	assert.Equal(t, err, wallets.ErrTraderInsufficientFunds)

}

func testCreditSuccess(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return()
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, asset)

	// by default when created the wallet has no monies
	balance, err := tw.wallets.Credit(wallets.TraderWalletType, traderID, asset, 100)
	assert.Equal(t, int64(100), balance)
	assert.Nil(t, err)

	// check the balance now
	balance, err = tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, balance, int64(100))
	assert.Nil(t, err)
}

func testWalletCreate(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return()
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, asset)

	balance, err := tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, int64(0), balance)
	assert.Nil(t, err)
}
func testErrorIfNotExist(t *testing.T) {
	tw := getTestWallets(t)
	defer tw.ctrl.Finish()

	balance, err := tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, int64(0), balance)
	assert.NotNil(t, err)
	assert.Equal(t, err, wallets.ErrNoAccountForTrader)

	tw.buffer.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return()
	// create a wallet in another asset
	// so the trader account exists but not for this asset
	_ = tw.wallets.GetCreate(wallets.TraderWalletType, traderID, otherAsset)

	balance, err = tw.wallets.GetBalance(wallets.TraderWalletType, traderID, asset)
	assert.Equal(t, int64(0), balance)
	assert.NotNil(t, err)
	assert.Equal(t, err, wallets.ErrTraderNoAccountForAsset)
}
