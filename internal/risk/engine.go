package risk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"vega/internal/logging"
	types "vega/proto"

	"github.com/pkg/errors"
)

type Engine interface {
	AddNewMarket(market *types.Market)
	RecalculateRisk()
	GetRiskFactors(marketName string) (float64, float64, error)
}

type riskEngine struct {
	*Config
	pyRiskModels map[string]string
	riskFactors  map[string]*types.RiskFactor
}

func NewRiskEngine(config *Config) Engine {
	return &riskEngine{
		Config:       config,
		riskFactors:  make(map[string]*types.RiskFactor, 0),
		pyRiskModels: make(map[string]string, 0),
	}
}

func NewRiskFactor(market *types.Market) *types.RiskFactor {
	return &types.RiskFactor{Market: market.Name}
}

func (re *riskEngine) AddNewMarket(market *types.Market) {
	// We will need to re-arch this when we have multiple markets/risk models/instrument definitions.
	// Just load the default for now for all markets (./risk-model.py)
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
			re.log.Error(fmt.Sprintf("error during risk assessment at market %s", marketName))
		}
	}
}

func (re *riskEngine) Assess(riskFactor *types.RiskFactor) error {
	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}

	re.log.Debug("Assess risk for risk factor",
		logging.String("market-id", riskFactor.Market),
		logging.Float64("long", riskFactor.Long),
		logging.Float64("short", riskFactor.Short),
		logging.String("executable", ex),
		logging.Float64("sigma", re.getSigma()),
		logging.Float64("lambda", re.getLambda()),
		logging.Int64("interestRate", re.getInterestRate()),
		logging.Int64("calculationFrequency", re.getCalculationFrequency()))

	// Using the vega binary location, we load the external risk script (risk-model.py)
	// - users can specify either relative paths or absolute paths, via configuration.
	pyPath := re.pyRiskModels[riskFactor.Market]
	if !re.PyRiskModelAbsolutePath {
		pyPath = filepath.Dir(ex) + re.pyRiskModels[riskFactor.Market]
	}

	re.log.Debug(fmt.Sprintf("pyPath: %s", pyPath))

	cmd := exec.Command("python", pyPath)
	stdout, err := cmd.Output()
	re.log.Debug(fmt.Sprintf("python stdout: %s", stdout))
	if err != nil {
		re.log.Error("Failure calling out to python, using defaults (0.00553|0.00550)", logging.Error(err))

		// Fail-safe return canned byte array, :(
		stdout = []byte("0.00553|0.00550")
	}

	s := strings.Split(string(stdout), "|")
	if len(s) != 2 {
		re.log.Error(fmt.Sprintf("Could not get risk factors from python model -> using defaults [%d]", len(s)))
		return errors.New("unable to get risk factor")
	}

	// Currently the risk script spec is to just print the int64 value '0.00553' on stdout
	riskFactorShort, err := strconv.ParseFloat(s[re.PyRiskModelShortIndex], 64)
	if err != nil {
		return errors.Wrap(err, "Failure calculating short risk factor")
	}

	riskFactorLong, err := strconv.ParseFloat(s[re.PyRiskModelLongIndex], 64)
	if err != nil {
		return errors.Wrap(err, "Failure calculating long risk factor")
	}

	riskFactor.Short = riskFactorShort
	riskFactor.Long = riskFactorLong

	re.log.Debug("Risk Factors obtained from risk model",
		logging.String("market-id", riskFactor.Market),
		logging.Float64("long", riskFactor.Long),
		logging.Float64("short", riskFactor.Short),
		logging.String("executable", ex),
		logging.Float64("sigma", re.getSigma()),
		logging.Float64("lambda", re.getLambda()),
		logging.Int64("interestRate", re.getInterestRate()),
		logging.Int64("calculationFrequency", re.getCalculationFrequency()))

	return nil
}
