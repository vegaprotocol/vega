package stubs

import (
	"sort"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/validators"
)

type TopologyStub struct {
	validators map[string]string
	nodeID     string
}

func NewTopologyStub(nodeID string) *TopologyStub {
	return &TopologyStub{
		validators: map[string]string{},
		nodeID:     nodeID,
	}
}

func (ts *TopologyStub) Len() int {
	return len(ts.validators)
}

func (ts *TopologyStub) ValidatorPerformanceScore(nodeID string) num.Decimal {
	return num.DecimalFromFloat(1)
}

func (ts *TopologyStub) SelfNodeID() string {
	return ts.nodeID
}

func (ts *TopologyStub) SelfVegaPubKey() string {
	return ts.nodeID
}

func (ts *TopologyStub) IsValidator() bool {
	return true
}

func (ts *TopologyStub) IsValidatorVegaPubKey(pubKey string) bool {
	return true
}

func (ts *TopologyStub) IsValidatorNodeID(nodeID string) bool {
	_, ok := ts.validators[nodeID]
	return ok
}

func (ts *TopologyStub) AllNodeIDs() []string {
	nodes := make([]string, 0, len(ts.validators))
	for n := range ts.validators {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	return nodes
}

func (ts *TopologyStub) AllVegaPubKeys() []string {
	nodes := make([]string, 0, len(ts.validators))
	for _, pk := range ts.validators {
		nodes = append(nodes, pk)
	}
	sort.Strings(nodes)
	return nodes
}

func (ts *TopologyStub) Get(key string) *validators.ValidatorData {
	if data, ok := ts.validators[key]; ok {
		return &validators.ValidatorData{
			ID:         key,
			VegaPubKey: data,
		}
	}

	return nil
}

func (ts *TopologyStub) AddValidator(node string, pubkey string) {
	ts.validators[node] = pubkey
}
