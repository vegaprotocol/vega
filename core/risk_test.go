package core

import (
	"testing"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)

type MockCommand struct {
	desiredValue string
}

func (mc MockCommand) Output() ([]byte, error) {
	//buf := make([]byte, binary.MaxVarintLen64)
	//binary.PutUvarint(buf, 20)
	buf := []byte("20\n")
	return buf, nil
}

func TestAssess(t *testing.T) {

	riskEngine := &riskEngine{
		Command: &MockCommand{},
	}

	order := &msg.Order{}

	riskEngine.Assess(order)

	assert.Equal(t, uint64(20), order.RiskFactor)
}
