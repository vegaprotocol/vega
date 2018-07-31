package risk

import (
	"github.com/golang/go/src/pkg/fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"vega/log"
	"vega/msg"
)

const riskModelFileName = "/risk-model.py"

type RiskEngine interface {
	AddNewMarket(market *msg.Market)
	CalibrateRiskModel()
	GetRiskFactors(marketName string) (int64, int64, error)
}

type riskEngine struct {
	riskFactors map[string]*RiskFactor
}

type RiskFactor struct {
	Market            *msg.Market
	RiskModelFileName string
	Short             int64
	Long              int64
}

func New() RiskEngine {
	return &riskEngine{}
}

func NewRiskFactor(market *msg.Market) *RiskFactor {
	return &RiskFactor{Market: market, RiskModelFileName: riskModelFileName}
}

func (re *riskEngine) AddNewMarket(market *msg.Market) {
	re.riskFactors[market.Name] = NewRiskFactor(market)
	re.Assess(re.riskFactors[market.Name])
}

func (re riskEngine) getSigma() float64 {
	return 0.8
}

func (re riskEngine) getLambda() float64 {
	return 0.01
}

func (re riskEngine) getInterestRate() int64 {
	return 0
}

func (re riskEngine) getCalculationFrequency() int64 {
	return 5
}

func (re riskEngine) GetRiskFactors(marketName string) (int64, int64, error) {
	if _, exist := re.riskFactors[marketName]; exist {
		return 0, 0, fmt.Errorf("NOT FOUND")
	}
	return re.riskFactors[marketName].Long, re.riskFactors[marketName].Short, nil
}

func (re riskEngine) CalibrateRiskModel() {
	for marketName, _ := range re.riskFactors {
		if err := re.Assess(re.riskFactors[marketName]); err != nil {
			log.Errorf("Error during risk assessment at market %s", marketName)
		}
	}
}

func (re *riskEngine) Assess(riskFactor *RiskFactor) error {
	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	log.Infof("Running risk calculations: \n")
	log.Infof("sigma %f\n", re.getSigma())
	log.Infof("lambda %f\n", re.getLambda())
	log.Infof("interestRate %d\n", re.getInterestRate())
	log.Infof("calculationFrequency %d\n", re.getCalculationFrequency())

	// Using the vega binary location, we load the external risk script (risk-model.py)
	pyPath := filepath.Dir(ex) + riskFactor.RiskModelFileName
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

	riskFactor.Long = n
	riskFactor.Short = n
	log.Infof("Risk Factors obtained from risk model: \n")
	log.Infof("Long: %d\n", riskFactor.Long)
	log.Infof("Short: %d\n", riskFactor.Short)

	return nil
}
