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

type ByXY []*pb.Delegation

func (o ByXY) Len() int      { return len(o) }
func (o ByXY) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ByXY) Less(i, j int) bool {
	if o[i].Amount == o[j].Amount {
		if o[i].EpochSeq == o[j].EpochSeq {
			return o[i].Party < o[j].Party
		}
		return o[i].EpochSeq < o[j].EpochSeq
	}

	return o[i].Amount < o[j].Amount
}

func TestNodes(t *testing.T) {
	a := assert.New(t)

	nodeStore := storage.NewNode(logging.NewTestLogger(), storage.NewDefaultConfig(""))

	n, err := nodeStore.GetByID("pub_key_1")
	a.Nil(n)
	a.Error(err, errors.New("node 1 not found"))

	testNode := pb.Node{
		Id:       "tm_pub_key_1",
		PubKey:   "pub_key_1",
		InfoUrl:  "http://info-node-1.vega",
		Location: "UK",
		Status:   pb.NodeStatus_NODE_STATUS_VALIDATOR,
	}

	expectedNode := &pb.Node{
		Id:                "tm_pub_key_1",
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

	actualNode, err := nodeStore.GetByID("tm_pub_key_1")
	a.NoError(err)
	a.Equal(expectedNode, actualNode)

	delegations := []*pb.Delegation{
		{
			Party:    "1",
			NodeId:   "tm_pub_key_1",
			Amount:   "20",
			EpochSeq: "1",
		},
		{
			Party:    "pub_key_1",
			NodeId:   "tm_pub_key_1",
			Amount:   "10",
			EpochSeq: "1",
		},
		{
			Party:    "2",
			NodeId:   "tm_pub_key_1",
			Amount:   "5",
			EpochSeq: "1",
		},
	}

	nodeStore.AddDelegation(*delegations[0])
	nodeStore.AddDelegation(*delegations[1])
	nodeStore.AddDelegation(*delegations[2])

	actualNode, err = nodeStore.GetByID("tm_pub_key_1")
	a.NoError(err)
	assertNode(a, actualNode, delegations, "10", "25", "35")

	nodeStore.AddNode(pb.Node{
		Id:       "tm_pub_key_2",
		PubKey:   "pub_key_2",
		InfoUrl:  "http://info-node-2.vega",
		Location: "UK",
		Status:   pb.NodeStatus_NODE_STATUS_VALIDATOR,
	})

	delegations = []*pb.Delegation{
		{
			Party:    "3",
			NodeId:   "tm_pub_key_2",
			Amount:   "10",
			EpochSeq: "1",
		},
		{
			Party:    "4",
			NodeId:   "tm_pub_key_2",
			Amount:   "50",
			EpochSeq: "1",
		},
		{
			Party:    "4",
			NodeId:   "tm_pub_key_2",
			Amount:   "50",
			EpochSeq: "2",
		},
		{
			Party:    "3",
			NodeId:   "tm_pub_key_2",
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

	node, err := nodeStore.GetByID("tm_pub_key_2")
	a.NoError(err)
	assertNode(a, node, delegations, "0", "170", "170")

	nodes := nodeStore.GetAll()
	a.Equal(2, len(nodes))

	a.Equal(2, nodeStore.GetTotalNodesNumber())
	a.Equal(2, nodeStore.GetValidatingNodesNumber())

	a.Equal("95", nodeStore.GetStakedTotal("1"))
	a.Equal("110", nodeStore.GetStakedTotal("2"))
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

	sort.Sort(ByXY(delegations))
	sort.Sort(ByXY(node.Delagations))

	a.Equal(len(delegations), len(node.Delagations))

	for i := range delegations {
		a.Equal(delegations[i], node.Delagations[i])
	}
}
