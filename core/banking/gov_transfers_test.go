package banking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCalculateGovernanceTransferAmount(t *testing.T) {
	e := getTestEngine(t)

	e.OnMaxAmountChanged(context.Background(), num.DecimalFromInt64(1000000))
	e.OnMaxFractionChanged(context.Background(), num.MustDecimalFromString("0.5"))

	e.col.EXPECT().GetSystemAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).Return(num.NewUint(1000000), nil).AnyTimes()
	balance, err := e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(10000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING)
	require.NoError(t, err)

	// max amount allowed by max fraction = 500k
	// max amount = 1000k
	// max amount by transfer = 10k
	// amount by transfer fraction = 200k
	// => amount to be transferred = min(500k, 1000k, 10k, 200k) = 10k which is fine for all or nothing
	require.Equal(t, num.NewUint(10000), balance)
	balance, err = e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(400000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING)
	require.NoError(t, err)

	// max amount allowed by max fraction = 500k
	// max amount = 1000k
	// max amount by transfer = 400k
	// amount by transfer fraction = 200k
	// => amount to be transferred = min(500k, 1000k, 400k, 200k) = 200k which is fine for all or nothing
	require.Equal(t, num.NewUint(200000), balance)

	e.OnMaxAmountChanged(context.Background(), num.DecimalFromInt64(100000))
	balance, err = e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(400000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING)

	// max amount allowed by max fraction = 500k
	// max amount = 100k
	// max amount by transfer = 400k
	// amount by transfer fraction = 200k
	// => amount to be transferred = min(500k, 100k, 400k, 200k) = 100k which is not fine for all or nothing
	require.Nil(t, balance)
	require.Equal(t, "invalid transfer amount for transfer type all or nothing", err.Error())

	// same settings with best effort would give 50k
	balance, err = e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(400000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_BEST_EFFORT)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100000), balance)

	e.OnMaxAmountChanged(context.Background(), num.DecimalFromInt64(1000000))
	e.OnMaxFractionChanged(context.Background(), num.MustDecimalFromString("0.05"))

	balance, err = e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(400000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING)

	// max amount allowed by max fraction = 50k
	// max amount = 100k
	// max amount by transfer = 400k
	// amount by transfer fraction = 200k
	// => amount to be transferred = min(50k, 100k, 400k, 200k) = 50k which is not fine for all or nothing
	require.Nil(t, balance)
	require.Equal(t, "invalid transfer amount for transfer type all or nothing", err.Error())

	// same settings with best effort would give 50k
	balance, err = e.CalculateGovernanceTransferAmount("asset", "", vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, num.DecimalFromFloat(0.2), num.NewUint(400000), vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_BEST_EFFORT)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(50000), balance)
}
