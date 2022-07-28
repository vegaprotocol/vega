package integration_test

import "testing"

func TestAccounts(t *testing.T) {
	queries := map[string]string{
		"PartyAccounts":       `{ parties{ id accounts{ asset{ id } market { id } type balance } } }`,
		"MarketAccounts":      `{ markets{ id accounts{ asset{ id } market { id } type balance } } }`,
		"AssetFeeAccounts":    `{ assets{ id infrastructureFeeAccount{ asset{ id } market { id } type balance } } }`,
		"AssetRewardAccounts": `{ assets{ id globalRewardPoolAccount{ asset{ id } market { id } type balance } }}`,
	}

	t.Run("PartyAccounts", func(t *testing.T) {
		assertGraphQLQueriesReturnSame[struct{ Parties []Party }](t, queries["PartyAccounts"])
	})

	t.Run("MarketAccounts", func(t *testing.T) {
		assertGraphQLQueriesReturnSame[struct{ Markets []Market }](t, queries["MarketAccounts"])
	})

	t.Run("AssetFeeAccounts", func(t *testing.T) {
		assertGraphQLQueriesReturnSame[struct{ Assets []Asset }](t, queries["AssetFeeAccounts"])
	})

	t.Run("AssetRewardAccounts", func(t *testing.T) {
		assertGraphQLQueriesReturnSame[struct{ Assets []Asset }](t, queries["AssetRewardAccounts"])
	})

}
