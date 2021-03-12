package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func HaveOnlyOneAccountPerAsset(
	broker *stubs.BrokerStub,
	owner string,
) error {
	assets := map[string]struct{}{}

	accounts := getAccounts(broker)

	for _, acc := range accounts {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			if _, ok := assets[acc.Asset]; ok {
				return errMultipleGeneralAccountForAsset(owner, acc)
			}
			assets[acc.Asset] = struct{}{}
		}
	}
	return nil
}

func errMultipleGeneralAccountForAsset(owner string, acc types.Account) error {
	return fmt.Errorf("trader=%v have multiple account for asset=%v", owner, acc.Asset)
}
