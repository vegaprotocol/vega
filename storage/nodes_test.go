package storage_test

import (
	"errors"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

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

func TestCleanupOldEpochsFromNodes(t *testing.T) {
	nodeStore := storage.NewNode(logging.NewTestLogger(), storage.NewDefaultConfig())
	testNode1 := pb.Node{
		Id:               "pub_key_1",
		PubKey:           "pub_key_1",
		TmPubKey:         "tm_pub_key_1",
		EthereumAdddress: "eth_pub_key_1",
		InfoUrl:          "http://info-node-1.vega",
		Location:         "UK",
		Status:           pb.NodeStatus_NODE_STATUS_VALIDATOR,
	}
	testNode2 := pb.Node{
		Id:               "pub_key_2",
		PubKey:           "pub_key_2",
		TmPubKey:         "tm_pub_key_2",
		EthereumAdddress: "eth_pub_key_2",
		InfoUrl:          "http://info-node-2.vega",
		Location:         "UK",
		Status:           pb.NodeStatus_NODE_STATUS_VALIDATOR,
	}
	nodeStore.AddNode(testNode1)
	nodeStore.AddNode(testNode2)
	for i := 0; i < 30; i++ {
		nodeStore.AddNodeRankingScore("pub_key_1", strconv.Itoa(i), pb.RankingScore{})
		nodeStore.AddNodeRankingScore("pub_key_2", strconv.Itoa(i), pb.RankingScore{})
		nodeStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_1",
			EpochSeq: strconv.Itoa(i),
			Amount:   "100",
		})
		nodeStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_2",
			EpochSeq: strconv.Itoa(i),
			Amount:   "200",
		})
		epochSeq := strconv.Itoa(i)
		node1, err := nodeStore.GetByID("pub_key_1", epochSeq)
		require.NoError(t, err)
		require.Equal(t, "100", node1.StakedByDelegates)

		node2, err := nodeStore.GetByID("pub_key_2", epochSeq)
		require.NoError(t, err)
		require.Equal(t, "200", node2.StakedByDelegates)
	}
	for i := 30; i < 40; i++ {
		nodeStore.AddNodeRankingScore("pub_key_1", strconv.Itoa(i), pb.RankingScore{})
		nodeStore.AddNodeRankingScore("pub_key_2", strconv.Itoa(i), pb.RankingScore{})
		nodeStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_1",
			EpochSeq: strconv.Itoa(i),
			Amount:   "100",
		})
		nodeStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_2",
			EpochSeq: strconv.Itoa(i),
			Amount:   "200",
		})
		// we don't have delegations for the 31st past epoch
		epochSeqMinus30 := strconv.Itoa(i - 30)
		node1, _ := nodeStore.GetByID("pub_key_1", epochSeqMinus30)
		require.Equal(t, "0", node1.StakedByDelegates)

		node2, _ := nodeStore.GetByID("pub_key_2", epochSeqMinus30)
		require.Equal(t, "0", node2.StakedByDelegates)

		// we have delegation for the past 30 epochs
		for j := 0; j < 30; j++ {
			epochSeq := strconv.Itoa(i - j)
			node1, _ := nodeStore.GetByID("pub_key_1", epochSeq)
			require.Equal(t, "100", node1.StakedByDelegates)

			node2, _ := nodeStore.GetByID("pub_key_2", epochSeq)
			require.Equal(t, "200", node2.StakedByDelegates)
		}
	}
}

func TestNodes(t *testing.T) {
	a := assert.New(t)

	nodeStore := storage.NewNode(logging.NewTestLogger(), storage.NewDefaultConfig())

	n, err := nodeStore.GetByID("pub_key_1", "1")
	a.Nil(n)
	a.Error(err, errors.New("node 1 not found"))

	testNode := pb.Node{
		Id:               "pub_key_1",
		PubKey:           "pub_key_1",
		TmPubKey:         "tm_pub_key_1",
		EthereumAdddress: "eth_pub_key_1",
		InfoUrl:          "http://info-node-1.vega",
		Location:         "UK",
		Status:           pb.NodeStatus_NODE_STATUS_VALIDATOR,
	}

	expectedNode := &pb.Node{
		Id:                "pub_key_1",
		PubKey:            "pub_key_1",
		TmPubKey:          "tm_pub_key_1",
		EthereumAdddress:  "eth_pub_key_1",
		InfoUrl:           "http://info-node-1.vega",
		Location:          "UK",
		Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
		StakedByOperator:  "0",
		StakedByDelegates: "0",
		StakedTotal:       "0",
		PendingStake:      "0",
		RankingScore:      &pb.RankingScore{},
	}

	nodeStore.AddNode(testNode)
	nodeStore.AddNodeRankingScore("pub_key_1", "1", pb.RankingScore{})

	actualNode, err := nodeStore.GetByID("pub_key_1", "1")
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

	actualNode, err = nodeStore.GetByID("pub_key_1", "1")

	a.NoError(err)
	assertNode(a, actualNode, delegations, "10", "25", "35", nil)

	nodeStore.AddNode(pb.Node{
		Id:               "pub_key_2",
		PubKey:           "pub_key_2",
		TmPubKey:         "tm_pub_key_2",
		EthereumAdddress: "eth_pub_key_2",
		InfoUrl:          "http://info-node-2.vega",
		Location:         "UK",
		Status:           pb.NodeStatus_NODE_STATUS_VALIDATOR,
	})

	rs1 := pb.RewardScore{
		RawValidatorScore: "20",
		PerformanceScore:  "0.89",
		MultisigScore:     "1",
		ValidatorScore:    "25",
		NormalisedScore:   "0.8",
		ValidatorStatus:   pb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT,
	}

	nodeStore.AddNodeRewardScore("pub_key_2", "1", rs1)

	rs2 := pb.RewardScore{
		RawValidatorScore: "30",
		PerformanceScore:  "0.9",
		ValidatorScore:    "40",
		MultisigScore:     "1",
		ValidatorStatus:   pb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ,
	}
	nodeStore.AddNodeRewardScore("pub_key_2", "2", rs2)

	rankScore1 := pb.RankingScore{
		Status: pb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT,
	}
	nodeStore.AddNodeRankingScore("pub_key_1", "2", rankScore1)
	rankScore2 := pb.RankingScore{
		Status: pb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ,
	}
	nodeStore.AddNodeRankingScore("pub_key_2", "2", rankScore2)
	nodeStore.AddNodeRankingScore("pub_key_2", "1", rankScore2)

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
			Party:    "3",
			NodeId:   "pub_key_2",
			Amount:   "10",
			EpochSeq: "2",
		},
		{
			Party:    "4",
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
	delegations[1].Amount = "60"
	nodeStore.AddDelegation(*delegations[1])

	// Get node in first epoch
	node, err := nodeStore.GetByID("pub_key_2", "1")
	a.NoError(err)
	assertNode(a, node, delegations[0:2], "0", "70", "70", &rs1)

	// Get node in second epoch
	node, err = nodeStore.GetByID("pub_key_2", "2")
	a.NoError(err)
	assertNode(a, node, delegations[2:], "0", "60", "60", &rs2)

	nodes := nodeStore.GetAll("1")
	a.Equal(2, len(nodes))

	nodes = nodeStore.GetAll("2")
	a.Equal(2, len(nodes))

	a.Equal(2, nodeStore.GetTotalNodesNumber("2"))
	a.Equal(1, nodeStore.GetValidatingNodesNumber("2"))

	a.Equal("105", nodeStore.GetStakedTotal("1"))
	a.Equal("60", nodeStore.GetStakedTotal("2"))

	// test key change
	node, err = nodeStore.GetByID("pub_key_2", "2")
	assert.NoError(t, err)
	assert.Equal(t, "pub_key_2", node.PubKey)

	// when
	nodeStore.PublickKeyChanged("pub_key_2", "pub_key_2", "new_vega_pub_key", 10)

	// then
	node, err = nodeStore.GetByID("pub_key_2", "2")
	assert.NoError(t, err)
	assert.Equal(t, "new_vega_pub_key", node.PubKey)

	allKeyRotations := nodeStore.GetAllPubKeyRotations()
	assert.Len(t, allKeyRotations, 1)
	assert.Equal(t, allKeyRotations, nodeStore.GetPubKeyRotationsPerNode("pub_key_2"))
}

func assertNode(
	a *assert.Assertions,
	node *pb.Node,
	delegations []*pb.Delegation,
	stakedByOperator, stakedByDelegates, stakedTotal string,
	rewardScore *pb.RewardScore,
) {
	a.Equal(stakedByOperator, node.StakedByOperator)
	a.Equal(stakedByDelegates, node.StakedByDelegates)
	a.Equal(stakedTotal, node.StakedTotal)
	if rewardScore == nil {
		a.Nil(node.RewardScore)
	} else {
		a.Equal(rewardScore.ValidatorScore, node.RewardScore.ValidatorScore)
		a.Equal(rewardScore.NormalisedScore, node.RewardScore.NormalisedScore)
		a.Equal(rewardScore.MultisigScore, node.RewardScore.MultisigScore)
		a.Equal(rewardScore.PerformanceScore, node.RewardScore.PerformanceScore)
		a.Equal(rewardScore.ValidatorStatus, node.RewardScore.ValidatorStatus)
		a.Equal(rewardScore.RawValidatorScore, node.RewardScore.RawValidatorScore)
	}
	sort.Sort(ByXY(delegations))
	sort.Sort(ByXY(node.Delegations))

	a.Equal(len(delegations), len(node.Delegations))

	for i := range delegations {
		a.Equal(delegations[i], node.Delegations[i])
	}
}
