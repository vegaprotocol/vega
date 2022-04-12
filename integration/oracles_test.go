package integration_test

import "testing"

func TestOracles(t *testing.T) {
	queries := map[string]string{
		"OracleSpecs": `{ oracleSpecs { id, createdAt, updatedAt, pubKeys, filters { key { name, type }, conditions { operator, value } }, status } }`,
		"OracleData":  `{ oracleSpecs { id, data { pubKeys, data { name, value } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ OracleSpecs []OracleSpec }
			assertGraphQLQueriesReturnSameIgnoreErrors(t, query, &new, &old)
		})
	}
}
