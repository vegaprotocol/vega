package risk

import (
	"os/exec"
	"vega/msg"
	"strconv"
	"os"
	"path/filepath"
)

const riskModelFileName = "/risk-model.py"

type RiskEngine interface {
	Assess(*msg.Order) error
}

type riskEngine struct {}

func New() RiskEngine {
	return &riskEngine{}
}

func (re riskEngine) Assess(order *msg.Order) error {
	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	// Using the vega binary location, we load the external risk script (risk-model.py)
	pyPath := filepath.Dir(ex) + riskModelFileName
	cmd := exec.Command("python", pyPath)
	stdout, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return err
	}
	// Currently the risk script spec is to just print the int64 value '20' on stdout
	n, err := strconv.ParseInt(string(stdout), 10, 64)
	if err != nil {
		println(err.Error())
		return err
	}
	order.RiskFactor = uint64(n)

	return nil
}
