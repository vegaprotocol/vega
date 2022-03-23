package integration_test

import "testing"

func TestParties(t *testing.T) {
	queries := map[string]string{
		"Deposits":      "{ parties { deposits{ id, party { id }, amount, asset { id }, status, createdTimestamp, creditedTimestamp, txHash } } }",
		"Delegations":   "{ parties{ id delegations{ node { id }, party{ id }, epoch, amount } } }",
		"Proposals":     "{ parties{ id proposals{ id votes{ yes { totalNumber } no { totalNumber } } } } }",
		"Votes":         "{ parties{ id votes{ proposalId vote{ value } } } }",
		"Margin Levels": "{ parties { margins { market { id }, asset { id }, party { id }, maintenanceLevel, searchLevel, initialLevel, collateralReleaseLevel, timestamp } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Party []Party }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
