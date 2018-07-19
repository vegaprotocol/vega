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
	Output(command string, args ...string) ([]byte, error)
}

type ExecCommand struct{}

func (ec ExecCommand) Output(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).Output()
}

func (re riskEngine) Assess(order *msg.Order) {
	returnValue, _ := re.Command.Output("echo", "20")
	order.RiskFactor, _ = binary.Uvarint(returnValue)
}
