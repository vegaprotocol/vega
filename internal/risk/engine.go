package risk

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"vega/msg"
	"fmt"
	"path/filepath"
)

type Engine interface {
	AddNewMarket(market *msg.Market)
	RecalculateRisk()
	GetRiskFactors(marketName string) (float64, float64, error)
}

type riskEngine struct {
	*Config
	pyRiskModels map[string]string
	riskFactors map[string]*msg.RiskFactor
}

func NewRiskEngine() Engine {
	config := NewConfig()
	return &riskEngine{
		Config: config,
		riskFactors: make(map[string]*msg.RiskFactor, 0),
		pyRiskModels: make(map[string]string, 0),

	}
}

func NewRiskFactor(market *msg.Market) *msg.RiskFactor {
	return &msg.RiskFactor{Market: market.Name}
}

func (re *riskEngine) AddNewMarket(market *msg.Market) {
	// todo: will need to re-arch this when we have multiple markets/risk models/instrument definitions.
	// todo: load the default for now for all markets (./risk-model.py)
	re.pyRiskModels[market.Name] = re.PyRiskModelDefaultFileName
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
		return 0, 0, errors.New(fmt.Sprintf("risk factors for market %s do not exist", marketName))
	}
	return re.riskFactors[marketName].Long, re.riskFactors[marketName].Short, nil
}

func (re riskEngine) RecalculateRisk() {
	for marketName, _ := range re.riskFactors {
		if err := re.Assess(re.riskFactors[marketName]); err != nil {
			re.log.Errorf("error during risk assessment at market %s", marketName)
		}
	}
}

func (re *riskEngine) Assess(riskFactor *msg.RiskFactor) error {
	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}

	re.log.Debugf("executable at: %s", ex)
	re.log.Debugf("Running risk calculations: ")
	re.log.Debugf("sigma %f", re.getSigma())
	re.log.Debugf("lambda %f", re.getLambda())
	re.log.Debugf("interestRate %d", re.getInterestRate())
	re.log.Debugf("calculationFrequency %d", re.getCalculationFrequency())

	// Using the vega binary location, we load the external risk script (risk-model.py)
	// - users can specify either relative paths or absolute paths, via configuration.
	pyPath := re.pyRiskModels[riskFactor.Market]
	if !re.PyRiskModelAbsolutePath {
		pyPath = filepath.Dir(ex) + re.pyRiskModels[riskFactor.Market]
	}

	re.log.Debugf("pyPath: %s\n", pyPath)

	cmd := exec.Command("python", pyPath)
	stdout, err := cmd.Output()
	re.log.Debugf("python stdout: %s\n", stdout)
	if err != nil {
		re.log.Infof("error calling out to python", err.Error())

		// Fail-safe return canned byte array, :(
		stdout = []byte("0.00553|0.00550")
	}

	s := strings.Split(string(stdout), "|")
	if len(s) != 2 {
		re.log.Infof("Could not get risk factors from python model -> using defaults [%d]", len(s))
		return errors.New("unable to get risk factor")
	}

	// Currently the risk script spec is to just print the int64 value '0.00553' on stdout
	riskFactorShort, err := strconv.ParseFloat(s[re.PyRiskModelShortIndex], 64)
	if err != nil {
		re.log.Errorf("error calculating short risk factor", err.Error())
		return err
	}

	riskFactorLong, err := strconv.ParseFloat(s[re.PyRiskModelLongIndex], 64)
	if err != nil {
		re.log.Errorf("error calculating long risk factor", err.Error())
		return err
	}

	riskFactor.Short = riskFactorShort
	riskFactor.Long = riskFactorLong

	re.log.Debugf("Risk Factors obtained from risk model: ")
	re.log.Debugf("Short: %f", riskFactor.Short)
	re.log.Debugf("Long: %f", riskFactor.Long)

	return nil
}

