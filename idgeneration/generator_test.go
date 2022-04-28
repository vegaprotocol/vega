package idgeneration_test

import (
	"testing"

	"code.vegaprotocol.io/vega/idgeneration"
	"github.com/stretchr/testify/assert"
)

func TestGeneratorCreationFailsWithInvalidRootId(t *testing.T) {
}

func TestOrderIdGeneration(t *testing.T) {
	detId := "e1152cf235f6200ed0eb4598706821031d57403462c31a80b3cdd6b209bff2e6"
	gen := idgeneration.New(detId)

	assert.Equal(t, detId, gen.NextID())
	assert.NotEqual(t, detId, gen.NextID())
}
