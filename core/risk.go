package core

import (
	"encoding/binary"
	"fmt"
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
	returnValue, err := re.Command.Output()

	if err != nil {
		// TODO - log this
		fmt.Println(err)
	}

	order.RiskFactor, _ = binary.Uvarint(returnValue)
}
