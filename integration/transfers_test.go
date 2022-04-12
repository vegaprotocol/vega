package integration_test

import "testing"

func TestTransfers(t *testing.T) {
	queries := map[string]string{
		"Transfers": "{ transfers(pubkey : \"test\", isFrom : false, isTo: false){id,from,fromAccountType,to,toAccountType,amount,reference, status,asset{id}}}",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Transfers []Transfer }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
