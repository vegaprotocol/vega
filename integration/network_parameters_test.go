package integration_test

import "testing"

func TestNetParams(t *testing.T) {
	queries := map[string]string{
		"Network Parameters": "{ networkParameters{ key, value } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ NetworkParameters []NetworkParameter }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
