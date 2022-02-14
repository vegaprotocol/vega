package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheNetworkTreasuryBalanceShouldBeForTheAsset(
	broker *stubs.BrokerStub,
	rawAmount, asset string,
) error {
	amount := parseExpectedInsurancePoolBalance(rawAmount)

	acc, err := broker.GetAssetNetworkTreasuryAccount(asset)
	if err != nil {
		return errCannotGetRewardPoolAccountForAsset(asset, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidAssetRewardPoolBalance(amount, acc)
	}
	return nil
}

func errCannotGetRewardPoolAccountForAsset(asset string, err error) error {
	return fmt.Errorf("couldn't get reward pool account for asset(%s): %s", asset, err.Error())
}

func errInvalidAssetRewardPoolBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for asset reward pool, expected %v, got %v",
		amount, acc.Balance,
	)
}
