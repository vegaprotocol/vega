package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"code.vegaprotocol.io/vega/internal/riskmodels"
	types "code.vegaprotocol.io/vega/proto"
)

type RiskModel struct {
	*http.ServeMux
}

func NewRiskModel() *RiskModel {
	rm := &RiskModel{http.NewServeMux()}
	rm.HandleFunc("/setup", rm.setup)
	rm.HandleFunc("/calculationInterval", rm.calculationInterval)
	rm.HandleFunc("/calculateRiskFactors", rm.calculateRiskFactors)
	return rm
}

func (r *RiskModel) setup(w http.ResponseWriter, req *http.Request) {
	log.Printf("/setup called")
	w.WriteHeader(http.StatusNoContent)
}

func (rm *RiskModel) calculateRiskFactors(w http.ResponseWriter, r *http.Request) {
	log.Printf("/calculateRiskFactors called")

	w.Header().Add("content-type", "encoding/json")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("unable to read body (/calculationInterval), %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := riskmodels.CalculateRiskFactorsRequest{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("unable to unmarshal body (/calculationInterval), %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var current = req.Current
	if current == nil {
		current = &types.RiskResult{
			RiskFactors: map[string]*types.RiskFactor{
				"Ethereum/Ether": &types.RiskFactor{
					Long:  0.15,
					Short: 0.25,
				},
			},
			PredictedNextRiskFactors: map[string]*types.RiskFactor{
				"Ethereum/Ether": &types.RiskFactor{
					Long:  0.15,
					Short: 0.25,
				},
			},
		}
	}

	res := riskmodels.CalculateRiskFactorsResponse{
		WasUpdated: true,
		Result:     current,
	}

	buf, err := json.Marshal(&res)
	if err != nil {
		log.Printf("unable to marshal response: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}

func (rm *RiskModel) calculationInterval(w http.ResponseWriter, r *http.Request) {
	log.Printf("/calculationInterval called")

	resp := riskmodels.CalculationIntervalResponse{
		DurationNano: 50,
	}

	buf, _ := json.Marshal(&resp)
	w.Header().Add("content-type", "encoding/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}
