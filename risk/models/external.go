package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

// External represent an external risk model
// connection is done throug a unix domain socket
type External struct {
	log    *logging.Logger
	client *http.Client

	isSetup bool
	mu      sync.Mutex
	Name    string
	Socket  string
	Config  map[string]string
}

// SetupRequest represent the request used to setup the risk model
type SetupRequest struct {
	Config map[string]string `json:"config"`
}

// CalculationIntervalResponse is the response sent by the risk model
// when requested with the calculation interval
type CalculationIntervalResponse struct {
	DurationNano uint64 `json:"duration_nano"`
}

// CalculateRiskFactorsRequest is the request send to the risk model
// in order to instruct them to calculate new risk factors
type CalculateRiskFactorsRequest struct {
	Current *types.RiskResult `json:"current"`
}

// CalculateRiskFactorsResponse is the type use by the risk model
// in order to return the newly calculated risk factors
type CalculateRiskFactorsResponse struct {
	WasUpdated bool              `json:"was_calculated"`
	Result     *types.RiskResult `json:"result"`
}

// NewExternal instantiate a new connection with an external risk model
func NewExternal(log *logging.Logger, pe *types.ExternalRiskModel) (*External, error) {
	tr := &http.Transport{
		Dial: func(proto, addr string) (conn net.Conn, err error) {
			return net.Dial("unix", pe.Socket)
		},
	}

	client := &http.Client{Transport: tr}
	return &External{
		log:    log,
		client: client,

		Name:   pe.Name,
		Socket: pe.Socket,
		Config: pe.Config,
	}, nil
}

func (e *External) getIsSetup() bool {
	e.mu.Lock()
	b := e.isSetup
	e.mu.Unlock()
	return b
}

func (e *External) setIsSetup(b bool) {
	e.mu.Lock()
	e.isSetup = b
	e.mu.Unlock()
}

func (e *External) setup() error {
	req := SetupRequest{e.Config}

	buf, err := json.Marshal(req)
	if err != nil {
		return err
	}

	before := vegatime.Now()
	resp, err := e.client.Post("http://d/setup", "encoding/json", bytes.NewBuffer(buf))
	if err != nil {
		return err
	}
	e.log.Info(
		"external call to /calculationInterval succeed",
		logging.String("risk-model", e.Name),
		logging.String("time-taken", vegatime.Now().Sub(before).String()),
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.New("invalid status code")
	}

	e.setIsSetup(true)
	e.log.Info(
		"external risk model setted up with success",
		logging.String("risk-model", e.Name),
	)
	return nil
}

// CalculationInterval returns the calculation interval of the risk model
func (e *External) CalculationInterval() time.Duration {
	if !e.getIsSetup() {
		if err := e.setup(); err != nil {
			e.log.Error(
				"unable to setup external risk model",
				logging.Error(err),
				logging.String("risk-model", e.Name),
			)
			return time.Duration(0)
		}
	}

	before := vegatime.Now()
	resp, err := e.client.Get("http://d/calculationInterval")
	if err != nil {
		e.log.Error(
			"unable to call external risk model /calculationInterval",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		e.setIsSetup(false)
		return time.Duration(0)
	}
	e.log.Info(
		"external call to /calculationInterval succeed",
		logging.String("risk-model", e.Name),
		logging.String("time-taken", vegatime.Now().Sub(before).String()),
	)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e.log.Error(
			"unable to read body for risk model /calculationInterval",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		return time.Duration(0)
	}

	ciresp := CalculationIntervalResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		e.log.Error(
			"unable to unmarshal for risk model /calculationInterval",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		return time.Duration(0)
	}

	return time.Duration(ciresp.DurationNano)
}

// CalculateRiskFactors calls the risk model in order to retrieve
// new risk factors
func (e *External) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	if !e.getIsSetup() {
		if err := e.setup(); err != nil {
			e.log.Error(
				"unable to setup external risk model",
				logging.Error(err),
				logging.String("risk-model", e.Name),
			)
			return false, current
		}
	}

	req := CalculateRiskFactorsRequest{
		Current: current,
	}
	buf, err := json.Marshal(req)
	if err != nil {
		e.log.Error(
			"unable to marshal risk model /calculateRiskFactor",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		return false, current
	}

	before := vegatime.Now()
	resp, err := e.client.Post("http://d/calculateRiskFactors", "encoding/json", bytes.NewBuffer(buf))
	if err != nil {
		e.log.Error(
			"unable to call external risk model /calculateRiskFactor",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		e.setIsSetup(false)
		return false, current
	}
	e.log.Info(
		"external call to /calculateRiskFactor succeed",
		logging.String("risk-model", e.Name),
		logging.String("time-taken", vegatime.Now().Sub(before).String()),
	)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e.log.Error(
			"unable to read body for risk model /calculateRiskFactor",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		return false, current
	}

	crfresp := CalculateRiskFactorsResponse{}
	err = json.Unmarshal(body, &crfresp)
	if err != nil {
		e.log.Error(
			"unable to unmarshal for risk model /calculationInterval",
			logging.Error(err),
			logging.String("risk-model", e.Name),
		)
		return false, current
	}

	e.log.Info("was calculated", logging.Bool("ok", crfresp.WasUpdated))

	return crfresp.WasUpdated, crfresp.Result
}

// PriceRange returns currentPrice twice as the mocn mindPrice/maxPrice calculation implementation
//TODO (WG 02/10/2020): Mock interface implementaiton to avoid additional validation on risk model, this should be implemented properly unless a decision to remove the external model is made.
//See: https://github.com/vegaprotocol/vega/issues/2337
func (e *External) PriceRange(currentPrice float64, yearFraction float64, probabilityLevel float64) (minPrice float64, maxPrice float64) {
	minPrice, maxPrice = currentPrice, currentPrice
	return
}
