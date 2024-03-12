package protos_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/protos"
)

func Test_CoreBindings(t *testing.T) {
	t.Run("CoreBindings should return the core http bindings", func(t *testing.T) {
		bindings, err := protos.CoreBindings()
		require.NoError(t, err)

		assert.Len(t, bindings.HTTP.Rules, 21)

		postCount := 0
		getCount := 0

		for _, rule := range bindings.HTTP.Rules {
			if rule.Post == nil && rule.Get == nil {
				continue
			}

			if rule.Post != nil {
				postCount++
			}

			if rule.Get != nil {
				getCount++
			}
		}

		assert.Equal(t, 4, postCount)
		assert.Equal(t, 17, getCount)

		assert.True(t, bindings.HasRoute("GET", "/statistics"))
		assert.True(t, bindings.HasRoute("POST", "/transactions"))
	})
}

func Test_DataNodeBindings(t *testing.T) {
	t.Run("CoreBindings should return the core http bindings", func(t *testing.T) {
		bindings, err := protos.DataNodeBindings()
		require.NoError(t, err)
		wantCount := 114

		assert.Len(t, bindings.HTTP.Rules, wantCount)

		postCount := 0
		getCount := 0

		for _, rule := range bindings.HTTP.Rules {
			if rule.Post == nil && rule.Get == nil {
				continue
			}

			if rule.Post != nil {
				postCount++
			}

			if rule.Get != nil {
				getCount++
			}
		}

		assert.Equal(t, 0, postCount)
		assert.Equal(t, wantCount, getCount)

		assert.True(t, bindings.HasRoute("GET", "/api/v2/oracle/data"))
		assert.True(t, bindings.HasRoute("GET", "/api/v2/stream/markets/data"))
	})
}
