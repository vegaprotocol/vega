package collateral_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollateralTransferFunds(t *testing.T) {
	t.Run("invalid number of parameters", testInvalidNumberOfParameters)
	t.Run("general to general", testTransferFundsFromGeneralToGeneral)
	t.Run("general to reward pool", testTransferFundsFromGeneralToRewardPool)
	t.Run("take from general do not distribute", testTakeFromGeneralDoNotDistribute)
	t.Run("distribute only from pending transfer", testDistributeScheduleFunds)
}

func testDistributeScheduleFunds(t *testing.T) {
	e := getTestEngine(t, "test-market")
	defer e.Finish()

	party1 := "party1"
	initialBalance := num.NewUint(90)

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	pendingTransfersAcc := e.GetPendingTransfersAccount(testMarketAsset)
	assert.Equal(t, pendingTransfersAcc.Balance, num.Zero())

	err := e.UpdateBalance(context.Background(), pendingTransfersAcc.ID, initialBalance)
	assert.Nil(t, err)

	e.broker.EXPECT().Send(gomock.Any()).Times(4)

	// now we create the transfers:
	resps, err := e.TransferFunds(
		context.Background(),
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90), // we assume the transfer to the other party is 90
				},
				Type:      types.TransferTypeTransferFundsDistribute,
				MinAmount: num.NewUint(90),
			},
		},
		// 1 general accounts
		[]types.AccountType{
			types.AccountTypeGeneral,
		},
		[]string{
			"pending-transfer-account-to-party1",
		},
		// no fees, they are paid before
		[]*types.Transfer{},
		[]types.AccountType{},
	)

	assert.NoError(t, err)
	assert.Len(t, resps, 1)

	// ensure balances now
	acc1, err := e.GetPartyGeneralAccount(party1, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, acc1.Balance, num.NewUint(90))

	pendingTransfersAcc = e.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, pendingTransfersAcc.Balance, num.Zero())
}

func testTakeFromGeneralDoNotDistribute(t *testing.T) {
	e := getTestEngine(t, "test-market")
	defer e.Finish()

	party1 := "party1"
	initialBalance := num.NewUint(100)

	// create party1 account + top up
	// no need to create party2 account, this account
	// should be created on the go
	e.broker.EXPECT().Send(gomock.Any()).Times(3)

	p1AccID, err := e.CreatePartyGeneralAccount(
		context.Background(), party1, testMarketAsset)
	require.NoError(t, err)

	err = e.UpdateBalance(context.Background(), p1AccID, initialBalance)
	assert.Nil(t, err)

	e.broker.EXPECT().Send(gomock.Any()).Times(4)

	// now we create the transfers:
	resps, err := e.TransferFunds(
		context.Background(),
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90), // we assume the transfer to the other party is 90
				},
				Type:      types.TransferTypeTransferFundsSend,
				MinAmount: num.NewUint(90),
			},
		},
		// 1 general accounts
		[]types.AccountType{
			types.AccountTypeGeneral,
		},
		[]string{
			"party1-to-party2",
		},
		// fee transfer
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(10), // we should have just enough to pay the fee
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(10),
			},
		},
		[]types.AccountType{
			types.AccountTypeGeneral, // only one account, the general account of party
		},
	)

	assert.NoError(t, err)
	assert.Len(t, resps, 2)

	// ensure balances now
	acc1, err := e.GetPartyGeneralAccount(party1, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, acc1.Balance, num.Zero())

	pendingTransfersAcc := e.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, pendingTransfersAcc.Balance, num.NewUint(90))
}

func testTransferFundsFromGeneralToGeneral(t *testing.T) {
	e := getTestEngine(t, "test-market")
	defer e.Finish()

	party1 := "party1"
	party2 := "party2"
	initialBalance := num.NewUint(100)

	// create party1 account + top up
	// no need to create party2 account, this account
	// should be created on the go
	e.broker.EXPECT().Send(gomock.Any()).Times(3)

	p1AccID, err := e.CreatePartyGeneralAccount(
		context.Background(), party1, testMarketAsset)
	require.NoError(t, err)

	err = e.UpdateBalance(context.Background(), p1AccID, initialBalance)
	assert.Nil(t, err)

	e.broker.EXPECT().Send(gomock.Any()).Times(8)

	// now we create the transfers:
	resps, err := e.TransferFunds(
		context.Background(),
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90), // we assume the transfer to the other party is 90
				},
				Type:      types.TransferTypeTransferFundsSend,
				MinAmount: num.NewUint(90),
			},
			{
				Owner: party2,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90),
				},
				Type:      types.TransferTypeTransferFundsDistribute,
				MinAmount: num.NewUint(90),
			},
		},
		// 2 general accounts
		[]types.AccountType{
			types.AccountTypeGeneral,
			types.AccountTypeGeneral,
		},
		[]string{
			"party1-to-party2",
			"party1-to-party2",
		},
		// fee transfer
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(10), // we should have just enough to pay the fee
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(10),
			},
		},
		[]types.AccountType{
			types.AccountTypeGeneral, // only one account, the general account of party
		},
	)

	assert.NoError(t, err)
	assert.Len(t, resps, 3)

	// ensure balances now
	acc1, err := e.GetPartyGeneralAccount(party1, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, acc1.Balance, num.Zero())

	acc2, err := e.GetPartyGeneralAccount(party2, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, acc2.Balance, num.NewUint(90))

	pendingTransfersAcc := e.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, pendingTransfersAcc.Balance, num.Zero())
}

func testTransferFundsFromGeneralToRewardPool(t *testing.T) {
	e := getTestEngine(t, "test-market")
	defer e.Finish()

	party1 := "party1"
	initialBalance := num.NewUint(100)

	// create party1 account + top up
	// no need to create party2 account, this account
	// should be created on the go
	e.broker.EXPECT().Send(gomock.Any()).Times(3)

	p1AccID, err := e.CreatePartyGeneralAccount(
		context.Background(), party1, testMarketAsset)
	require.NoError(t, err)

	err = e.UpdateBalance(context.Background(), p1AccID, initialBalance)
	assert.Nil(t, err)

	e.broker.EXPECT().Send(gomock.Any()).Times(6)

	// now we create the transfers:
	resps, err := e.TransferFunds(
		context.Background(),
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90), // we assume the transfer to the other party is 90
				},
				Type:      types.TransferTypeTransferFundsSend,
				MinAmount: num.NewUint(90),
			},
			{
				Owner: "",
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(90),
				},
				Type:      types.TransferTypeTransferFundsDistribute,
				MinAmount: num.NewUint(90),
			},
		},
		// 2 general accounts
		[]types.AccountType{
			types.AccountTypeGeneral,
			types.AccountTypeGlobalReward,
		},
		[]string{
			"party1-to-reward",
			"party1-to-reward",
		},
		// fee transfer
		[]*types.Transfer{
			{
				Owner: party1,
				Amount: &types.FinancialAmount{
					Asset:  testMarketAsset,
					Amount: num.NewUint(10), // we should have just enough to pay the fee
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(10),
			},
		},
		[]types.AccountType{
			types.AccountTypeGeneral, // only one account, the general account of party
		},
	)

	assert.NoError(t, err)
	assert.Len(t, resps, 3)

	// ensure balances now
	acc1, err := e.GetPartyGeneralAccount(party1, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, acc1.Balance, num.Zero())

	rewardAcc, err := e.GetGlobalRewardAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, rewardAcc.Balance, num.NewUint(90))

	pendingTransfersAcc := e.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, pendingTransfersAcc.Balance, num.Zero())
}

func testInvalidNumberOfParameters(t *testing.T) {
	e := getTestEngine(t, "test-market")
	defer e.Finish()

	assert.Panics(t, func() {
		e.TransferFunds(
			context.Background(),
			make([]*types.Transfer, 3),
			make([]types.AccountType, 2),
			make([]string, 3),
			make([]*types.Transfer, 1),
			make([]types.AccountType, 1),
		)
	})
	assert.Panics(t, func() {
		e.TransferFunds(
			context.Background(),
			make([]*types.Transfer, 3),
			make([]types.AccountType, 3),
			make([]string, 1),
			make([]*types.Transfer, 1),
			make([]types.AccountType, 1),
		)
	})
	assert.Panics(t, func() {
		e.TransferFunds(
			context.Background(),
			make([]*types.Transfer, 3),
			make([]types.AccountType, 3),
			make([]string, 3),
			make([]*types.Transfer, 2),
			make([]types.AccountType, 1),
		)
	})
}
