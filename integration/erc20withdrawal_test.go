package integration_test

import "testing"

func TestERC20WithdrawalApproval(t *testing.T) {
	queries := map[string]string{
		"ERC20WithdrawalApproval": `{ erc20WithdrawalApproval(withdrawalId:"7ee15f2fc0d49687df4a791fce246d82a0b82c420d02a562e7d4bcc430e9a8c7") { assetSource amount nonce signatures targetAddress } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ ERC20WithdrawalApproval ERC20WithdrawalApproval }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
