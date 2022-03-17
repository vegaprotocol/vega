package integration_test

import "testing"

func TestEpochs(t *testing.T) {
	queries := map[string]string{
		"CurrentEpoch":    `{ epoch{ id timestamps{ start, expiry, end } } }`,
		"EpochDelgations": `{ epoch { delegations { node { id }, party {id}, amount} } }`,
		"SpecificEpoch":   `{ epoch(id:"10") { id timestamps{ start, expiry, end } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Markets []Market }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
