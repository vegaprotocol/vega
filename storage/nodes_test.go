package storage_test

import (
	"errors"
	"sort"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	pb "code.vegaprotocol.io/protos/vega"
	"github.com/stretchr/testify/assert"
)

func TestNodes(t *testing.T) {
	a := assert.New(t)

	nodeStore := storage.NewNode(logging.NewTestLogger(), storage.NewDefaultConfig(""))

	n, err := nodeStore.GetByID("pub_key_1")
	a.Nil(n)
	a.Error(err, errors.New("node 1 not found"))

	testNode := pb.Node{
		Id:       "1",
		PubKey:   "pub_key_1",
		InfoUrl:  "http://info-node-1.vega",
		Location: "UK",
		Status:   pb.NodeStatus_NODE_STATUS_VALIDATOR,
	}

	expectedNode := &pb.Node{
		Id:                "1",
		PubKey:            "pub_key_1",
		InfoUrl:           "http://info-node-1.vega",
		Location:          "UK",
		Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
		StakedByOperator:  "0",
		StakedByDelegates: "0",
		StakedTotal:       "0",
		Delagations:       []*pb.Delegation{},
	}

	nodeStore.AddNode(testNode)

	actualNode, err := nodeStore.GetByID("pub_key_1")
	a.NoError(err)
	a.Equal(expectedNode, actualNode)

	delegations := []*pb.Delegation{
		{
			Party:    "1",
			NodeId:   "pub_key_1",
			Amount:   "20",
			EpochSeq: "1",
		},
		{
			Party:    "pub_key_1",
			NodeId:   "pub_key_1",
			Amount:   "10",
			EpochSeq: "1",
		},
		{
			Party:    "2",
			NodeId:   "pub_key_1",
			Amount:   "5",
			EpochSeq: "1",
		},
	}

	nodeStore.AddDelegation(*delegations[0])
	nodeStore.AddDelegation(*delegations[1])
	nodeStore.AddDelegation(*delegations[2])

	actualNode, err = nodeStore.GetByID("pub_key_1")
	a.NoError(err)
	assertNode(a, actualNode, delegations, "10", "25", "35")

	nodeStore.AddNode(pb.Node{
		Id:       "2",
		PubKey:   "pub_key_2",
		InfoUrl:  "http://info-node-2.vega",
		Location: "UK",
		Status:   pb.NodeStatus_NODE_STATUS_VALIDATOR,
	})

	delegations = []*pb.Delegation{
		{
			Party:    "3",
			NodeId:   "pub_key_2",
			Amount:   "10",
			EpochSeq: "1",
		},
		{
			Party:    "4",
			NodeId:   "pub_key_2",
			Amount:   "50",
			EpochSeq: "1",
		},
		{
			Party:    "4",
			NodeId:   "pub_key_2",
			Amount:   "50",
			EpochSeq: "2",
		},
		{
			Party:    "3",
			NodeId:   "pub_key_2",
			Amount:   "50",
			EpochSeq: "2",
		},
	}

	nodeStore.AddDelegation(*delegations[0])
	nodeStore.AddDelegation(*delegations[1])
	nodeStore.AddDelegation(*delegations[2])
	nodeStore.AddDelegation(*delegations[3])

	// This delegation should just replace previous one in the epoch - only increase the amount
	delegations[3].Amount = "60"
	nodeStore.AddDelegation(*delegations[3])

	node, err := nodeStore.GetByID("pub_key_2")
	a.NoError(err)
	assertNode(a, node, delegations, "0", "170", "170")

	nodes := nodeStore.GetAll()
	a.Equal(2, len(nodes))

	a.Equal(2, nodeStore.GetTotalNodesNumber())
	a.Equal(2, nodeStore.GetValidatingNodesNumber())
	a.Equal("205", nodeStore.GetStakedTotal())
}

func assertNode(
	a *assert.Assertions,
	node *pb.Node,
	delegations []*pb.Delegation,
	stakedByOperator, stakedByDelegates, stakedTotal string,
) {
	a.Equal(stakedByOperator, node.StakedByOperator)
	a.Equal(stakedByDelegates, node.StakedByDelegates)
	a.Equal(stakedTotal, node.StakedTotal)

	sort.Slice(delegations, func(i, j int) bool {
		return delegations[i].Amount < delegations[j].Amount
	})

	sort.Slice(node.Delagations, func(i, j int) bool {
		return node.Delagations[i].Amount < node.Delagations[j].Amount
	})

	a.Equal(len(delegations), len(node.Delagations))

	for i := range delegations {
		a.Equal(delegations[i], node.Delagations[i])
	}
}
