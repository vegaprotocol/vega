package core

import (
	"testing"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)

func TestAssess(t *testing.T) {
	order := &msg.Order{}

	Assess(order)

	assert.Equal(t, uint64(20), order.RiskFactor)
}
