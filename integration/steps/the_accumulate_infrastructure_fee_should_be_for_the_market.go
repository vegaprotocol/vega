package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheAccumulatedInfrastructureFeesShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	amountStr, asset string,
) error {
	amount, err := U64(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	acc, err := broker.GetMarketInfrastructureFeePoolAccount(asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != amount {
		return errInvalidAmountInInfraFee(asset, amount, stringToU64(acc.Balance))
	}

	return nil
}

func errInvalidAmountInInfraFee(asset string, expected, got uint64) error {
	return fmt.Errorf("invalid amount in infrastructure fee pool for asset %s want %d got %d", asset, expected, got)
}
