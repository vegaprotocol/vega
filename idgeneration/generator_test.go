package idgeneration_test

import (
	"testing"

	"code.vegaprotocol.io/vega/idgeneration"
	"github.com/stretchr/testify/assert"
)

func TestGeneratorCreationFailsWithInvalidRootId(t *testing.T) {
}

func TestOrderIdGeneration(t *testing.T) {
	detId := "E1152CF235F6200ED0EB4598706821031D57403462C31A80B3CDD6B209BFF2E6"
	gen := idgeneration.New(detId)

	assert.Equal(t, detId, gen.NextID())
	assert.NotEqual(t, detId, gen.NextID())
}
