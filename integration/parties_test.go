package integration_test

import "testing"

func TestParties(t *testing.T) {
	queries := map[string]string{
		"Delegations": "{ parties{ delegations{ node { id }, party{ id }, epoch, amount } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Markets []Market }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
