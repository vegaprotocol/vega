package risk

import (
	"github.com/golang/go/src/pkg/fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"vega/log"
	"vega/msg"
	"github.com/golang/go/src/pkg/strings"
)

const (
	riskModelFileName = "/risk-model.py"
	shortIndex = 0
	longIndex = 1
)


type RiskEngine interface {
	AddNewMarket(market *msg.Market)
	CalibrateRiskModel()
	GetRiskFactors(marketName string) (float64, float64, error)
}

type riskEngine struct {
	riskFactors map[string]*RiskFactor
}

type RiskFactor struct {
	Market            *msg.Market
	RiskModelFileName string
	Short             float64
	Long              float64
}

func New() RiskEngine {
	return &riskEngine{riskFactors: make(map[string]*RiskFactor)}
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

func (re riskEngine) GetRiskFactors(marketName string) (float64, float64, error) {
	if _, exist := re.riskFactors[marketName]; !exist {
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

	fmt.Printf("executable at: %s\n", ex)
	fmt.Printf("Running risk calculations: \n")
	fmt.Printf("sigma %f\n", re.getSigma())
	fmt.Printf("lambda %f\n", re.getLambda())
	fmt.Printf("interestRate %d\n", re.getInterestRate())
	fmt.Printf("calculationFrequency %d\n", re.getCalculationFrequency())

	// Using the vega binary location, we load the external risk script (risk-model.py)
	pyPath := filepath.Dir(ex) + riskFactor.RiskModelFileName

	fmt.Printf("pyPath: %s\n", pyPath)
	cmd := exec.Command("python", pyPath)
	stdout, err := cmd.Output()
	fmt.Printf("python stdout: %s\n", stdout)
	if err != nil {
		println(err.Error())
		// SHORT|LONG
		stdout = []byte("0.00553|0.00550")
	}

	s := strings.Split(string(stdout), "|")
	if len(s) != 2 {
		fmt.Printf("error: unable to get risk factor")
		return fmt.Errorf("unable to get risk factor")
	}

	// Currently the risk script spec is to just print the int64 value '0.00553' on stdout
	riskFactorShort, err := strconv.ParseFloat(s[shortIndex], 64)
	if err != nil {
		println(err.Error())
		return err
	}

	riskFactorLong, err := strconv.ParseFloat(s[longIndex], 64)
	if err != nil {
		println(err.Error())
		return err
	}

	riskFactor.Short = riskFactorShort
	riskFactor.Long = riskFactorLong
	fmt.Printf("Risk Factors obtained from risk model: \n")
	fmt.Printf("Short: %f\n", riskFactor.Short)
	fmt.Printf("Long: %f\n", riskFactor.Long)

	return nil
}
