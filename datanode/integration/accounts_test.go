package integration_test

import "testing"

func TestAccounts(t *testing.T) {
	queries := map[string]string{
		"PartyAccounts":       `{ partiesConnection{ edges { node { id accountsConnection{ edges { node { asset{ id } market { id } type balance } } } } } } }`,
		"MarketAccounts":      `{ marketsConnection{ edges { node { id accountsConnection{ edges { node { asset{ id } market { id } type balance } } } } } } }`,
		"AssetFeeAccounts":    `{ assetsConnection{ edges { node { id infrastructureFeeAccount{ asset{ id } market { id } type balance } } } } }`,
		"AssetRewardAccounts": `{ assetsConnection{ edges { node { id globalRewardPoolAccount{ asset{ id } market { id } type balance } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
