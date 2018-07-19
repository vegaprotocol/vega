package core

import (
	"encoding/binary"
	"os/exec"
	"vega/proto"
)

type RiskEngine interface {
	Assess(*msg.Order)
}

type riskEngine struct {
	Command RiskCommand
}

type RiskCommand interface {
	Output() ([]byte, error)
}

func newRiskEngine() *riskEngine {
	return &riskEngine{
		Command: &ExecCommand{
			command: "python",
			args:    []string{"-c", "20"},
		},
	}
}

type ExecCommand struct {
	command string
	args    []string
}

func (ec ExecCommand) Output() ([]byte, error) {
	return exec.Command(ec.command, ec.args...).Output()
}

func (re riskEngine) Assess(order *msg.Order) {
	returnValue, _ := re.Command.Output()
	order.RiskFactor, _ = binary.Uvarint(returnValue)
}
