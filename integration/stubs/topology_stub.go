package stubs

import (
	"sort"
)

type TopologyStub struct {
	validators map[string]string
}

func NewTopologyStub() *TopologyStub {
	return &TopologyStub{
		validators: map[string]string{},
	}
}

func (ts *TopologyStub) IsValidatorNode(nodeID string) bool {
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

func (ts *TopologyStub) AddValidator(node string, pubkey string) {
	ts.validators[node] = pubkey
}
