package execution_test

import (
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneratorCreationFailsWithInvalidRootId(t *testing.T) {

}

func TestOrderIdGeneration(t *testing.T) {

	detId := "E1152CF235F6200ED0EB4598706821031D57403462C31A80B3CDD6B209BFF2E6"
	gen, err := execution.NewDeterministicIDGenerator(detId)
	if err != nil {
		t.Fatalf("failed to create generator:%s", err)
	}

	order := &types.Order{}
	gen.SetID(order)
	assert.Equal(t, detId, order.ID)
	gen.SetID(order)
	assert.NotEqual(t, detId, order.ID)
}
