package integration_test

import "testing"

func TestNodeSignatures(t *testing.T) {
	queries := map[string]string{
		"NodeSignatures": `{ nodeSignatures(resourceId:"7ee15f2fc0d49687df4a791fce246d82a0b82c420d02a562e7d4bcc430e9a8c7") { id signature kind } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ NodeSignatures []NodeSignature }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
