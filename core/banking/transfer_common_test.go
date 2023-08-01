package banking_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCheckTransfer(t *testing.T) {
	e := getTestEngine(t)

	transfer := &types.TransferBase{
		From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
		FromAccountType: types.AccountTypeGeneral,
		To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           "eth",
		Amount:          num.NewUint(10),
		Reference:       "someref",
	}

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))

	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Return(&types.Account{Balance: num.NewUint(200)}, nil).AnyTimes()

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(2).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	require.EqualError(t,
		e.CheckTransfer(transfer),
		"could not transfer funds, less than minimal amount requested to transfer",
	)

	// decrease quantum multiple
	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(0.01))
	require.NoError(t, e.CheckTransfer(transfer))

	// invalid asset
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, errors.New("asset does not exist"))
	require.EqualError(t,
		e.CheckTransfer(transfer),
		"could not transfer funds, asset does not exist",
	)

	// invalid amount
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	transfer.Amount = num.UintZero()
	require.EqualError(t,
		e.CheckTransfer(transfer),
		"could not transfer funds, cannot transfer zero funds",
	)

	e.OnTransferFeeFactorUpdate(context.Background(), num.DecimalFromFloat(0.01))
	e.assets.EXPECT().Get(gomock.Any()).Times(2).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	// sufficient balance to cover fees
	transfer.Amount = num.NewUint(100)
	require.NoError(t, e.CheckTransfer(transfer))

	// insufficient balance to cover fees
	transfer.Amount = num.NewUint(200)
	require.EqualError(t, e.CheckTransfer(transfer), "could not transfer funds, not enough funds to transfer")
}
