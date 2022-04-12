package integration_test

import "testing"

func TestPositions(t *testing.T) {
	queries := map[string]string{
		"Positions": `{
			parties {
			  id
			  positions{
				market{id}
				openVolume
				realisedPNL
				unrealisedPNL
				averageEntryPrice
				updatedAt
			  }
			}
		  }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Parties []Party }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
