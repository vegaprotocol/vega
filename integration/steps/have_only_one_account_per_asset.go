package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func HaveOnlyOneAccountPerAsset(broker *stubs.BrokerStub, owner string) error {
	assets := map[string]struct{}{}

	accs := broker.GetAccounts()
	data := make([]types.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			if _, ok := assets[acc.Asset]; ok {
				return fmt.Errorf("trader=%v have multiple account for asset=%v", owner, acc.Asset)
			}
			assets[acc.Asset] = struct{}{}
		}
	}
	return nil
}
