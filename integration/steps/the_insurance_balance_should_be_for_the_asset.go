package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheInsurancePoolBalanceShouldBeForTheAsset(
	broker *stubs.BrokerStub,
	rawAmount, asset string,
) error {

	amount := parseExpectedInsurancePoolBalance(rawAmount)

	acc, err := broker.GetAssetInsurancePoolAccount(asset)
	if err != nil {
		return errCannotGetInsurancePoolAccountForAsset(asset, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidAssetInsurancePoolBalance(amount, acc)
	}
	return nil
}

func errCannotGetInsurancePoolAccountForAsset(asset string, err error) error {
	return fmt.Errorf("couldn't get insurance pool account for asset(%s): %s", asset, err.Error())
}

func errInvalidAssetInsurancePoolBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for asset insurance pool, expected %v, got %v",
		amount, acc.Balance,
	)
}
