package integration_test

import "testing"

func TestBalances(t *testing.T) {
	queries := map[string]string{
		"Positions": `{ parties{ id accounts { type asset{ id } market{ id } balance } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Parties []Party }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
