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

	delegationOne := pb.Delegation{
		Party:    "1",
		NodeId:   "pub_key_1",
		Amount:   "20",
		EpochSeq: "1",
	}

	delegationTwo := pb.Delegation{
		Party:    "pub_key_1",
		NodeId:   "pub_key_1",
		Amount:   "10",
		EpochSeq: "1",
	}

	delegationThree := pb.Delegation{
		Party:    "2",
		NodeId:   "pub_key_1",
		Amount:   "5",
		EpochSeq: "1",
	}

	nodeStore.AddDelegation(delegationOne)
	nodeStore.AddDelegation(delegationTwo)
	nodeStore.AddDelegation(delegationThree)

	expectedDelegations := []*pb.Delegation{
		&delegationOne,
		&delegationTwo,
		&delegationThree,
	}

	actualNode, err = nodeStore.GetByID("pub_key_1")
	a.NoError(err)
	assertNode(a, actualNode, expectedDelegations, "10", "25", "35")

	nodeStore.AddNode(pb.Node{
		Id:       "2",
		PubKey:   "pub_key_2",
		InfoUrl:  "http://info-node-2.vega",
		Location: "UK",
		Status:   pb.NodeStatus_NODE_STATUS_VALIDATOR,
	})

	delegationOne = pb.Delegation{
		Party:    "3",
		NodeId:   "pub_key_2",
		Amount:   "10",
		EpochSeq: "1",
	}

	delegationTwo = pb.Delegation{
		Party:    "4",
		NodeId:   "pub_key_2",
		Amount:   "50",
		EpochSeq: "1",
	}

	nodeStore.AddDelegation(delegationOne)
	nodeStore.AddDelegation(delegationTwo)

	expectedDelegations = []*pb.Delegation{
		&delegationOne,
		&delegationTwo,
	}

	node, err := nodeStore.GetByID("pub_key_2")
	a.NoError(err)
	assertNode(a, node, expectedDelegations, "0", "60", "60")

	nodes := nodeStore.GetAll()
	a.Equal(2, len(nodes))

	a.Equal(2, nodeStore.GetTotalNodesNumber())
	a.Equal(2, nodeStore.GetValidatingNodesNumber())
	a.Equal("95", nodeStore.GetStakedTotal())
}

func assertNode(
	a *assert.Assertions,
	node *pb.Node,
	expectedDelegations []*pb.Delegation,
	stakedByOperator, stakedByDelegates, stakedTotal string,
) {
	a.Equal(node.StakedByOperator, stakedByOperator)
	a.Equal(node.StakedByDelegates, stakedByDelegates)
	a.Equal(node.StakedTotal, stakedTotal)

	sort.Slice(expectedDelegations, func(i, j int) bool {
		return expectedDelegations[i].Amount < expectedDelegations[j].Amount
	})

	sort.Slice(node.Delagations, func(i, j int) bool {
		return node.Delagations[i].Amount < node.Delagations[j].Amount
	})

	a.Equal(len(expectedDelegations), len(node.Delagations))

	for i := range expectedDelegations {
		a.Equal(expectedDelegations[i], node.Delagations[i])
	}
}
