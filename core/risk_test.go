package core

import (
	"encoding/binary"
	"testing"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)

type MockCommand struct {
	desiredValue string
}

func (mc MockCommand) Output(command string, args ...string) ([]byte, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, 20)
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
