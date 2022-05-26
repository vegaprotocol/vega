package storage_test

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	pb "code.vegaprotocol.io/protos/vega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupOldEpochsFromEpoch(t *testing.T) {
	log := logging.NewTestLogger()
	c := storage.NewDefaultConfig()
	startTime := time.Date(2020, time.December, 25, 12, 0, 0, 0, time.UTC)
	expiryTime := time.Date(2020, time.December, 25, 12, 23, 59, 30, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	nodeStore := storage.NewNode(log, c)
	epochStore := storage.NewEpoch(log, nodeStore, c)
	for i := 0; i < 30; i++ {
		epochStore.AddEpoch(uint64(i), startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

		epochStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_1",
			EpochSeq: strconv.Itoa(i),
			Amount:   "100",
		})
	}

	for i := 30; i < 40; i++ {
		epochStore.AddEpoch(uint64(i), startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())
		epochStore.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "pub_key_1",
			EpochSeq: strconv.Itoa(i),
			Amount:   "100",
		})

		// we don't have delegations for the 31st past epoch
		epochSeqMinus30 := strconv.Itoa(i - 30)
		epoch, _ := epochStore.GetEpochByID(epochSeqMinus30)
		require.Equal(t, 0, len(epoch.Delegations))

		// we have delegation for the past 30 epochs
		for j := 0; j < 30; j++ {
			epochSeq := strconv.Itoa(i - j)
			epoch, _ := epochStore.GetEpochByID(epochSeq)
			require.Equal(t, "100", epoch.Delegations[0].Amount)
		}
	}
}

func TestEpochs(t *testing.T) {
	a := assert.New(t)

	log := logging.NewTestLogger()
	c := storage.NewDefaultConfig()

	nodeStore := storage.NewNode(log, c)
	epochStore := storage.NewEpoch(log, nodeStore, c)

	epoch, err := epochStore.GetEpoch()
	a.EqualError(err, "no epoch present")
	a.Nil(epoch)

	epoch, err = epochStore.GetEpochByID("epoch_id")
	a.EqualError(err, "epoch epoch_id not found")
	a.Nil(epoch)

	startTime := time.Date(2020, time.December, 25, 12, 0, 0, 0, time.UTC)
	expiryTime := time.Date(2020, time.December, 25, 12, 23, 59, 30, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	epochStore.AddEpoch(1, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	epoch, err = epochStore.GetEpochByID("1")
	a.NoError(err)
	assertEpoch(a, epoch, []*pb.Delegation{}, []*pb.Node{}, 1, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	epoch, err = epochStore.GetEpoch()
	a.NoError(err)
	assertEpoch(a, epoch, []*pb.Delegation{}, []*pb.Node{}, 1, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	delegations := []*pb.Delegation{
		{
			EpochSeq: "1",
			Party:    "party_1",
			NodeId:   "node_1",
			Amount:   "10",
		},
		{
			EpochSeq: "1",
			Party:    "party_2",
			NodeId:   "node_2",
			Amount:   "5",
		},
	}

	// Add delegations to existing epoch
	epochStore.AddDelegation(*delegations[0])
	epochStore.AddDelegation(*delegations[1])

	epoch, err = epochStore.GetEpoch()
	a.NoError(err)
	assertEpoch(a, epoch, delegations, []*pb.Node{}, 1, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	// Add delegations to epoch that hasn't arrived yet
	delegations[0].EpochSeq = "2"
	delegations[1].EpochSeq = "2"

	epochStore.AddDelegation(*delegations[0])
	epochStore.AddDelegation(*delegations[1])

	epoch, err = epochStore.GetEpochByID("2")
	a.NoError(err)
	assertEpoch(a, epoch, delegations, []*pb.Node{}, 2, 0, 0, 0)

	// Add epoch that already holds delegations - this will update the epoch
	startTime = startTime.Add(24 * time.Hour)
	expiryTime = expiryTime.Add(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour)
	epochStore.AddEpoch(2, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	epoch, err = epochStore.GetEpochByID("2")
	a.NoError(err)
	assertEpoch(a, epoch, delegations, []*pb.Node{}, 2, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())
	epoch, err = epochStore.GetEpoch()
	a.NoError(err)
	assertEpoch(a, epoch, delegations, []*pb.Node{}, 2, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	uptime := epochStore.GetTotalNodesUptime()
	a.Equal(48*time.Hour, uptime)

	var nodes []*pb.Node
	for i := 0; i < 2; i++ {
		nodes = append(nodes, &pb.Node{
			Id:                fmt.Sprintf("%d", i),
			PubKey:            fmt.Sprintf("pub_key_%d", i),
			InfoUrl:           fmt.Sprintf("node-%d.xyz.vega/info", i),
			Location:          "GB",
			Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
			StakedByOperator:  "0",
			StakedByDelegates: "0",
			StakedTotal:       "0",
			PendingStake:      "0",
			Delegations:       nil,
			RankingScore:      &pb.RankingScore{},
		})
	}

	// Test epoch returns nodes
	nodeStore.AddNode(*nodes[0], true, 3)
	nodeStore.AddNode(*nodes[1], true, 3)
	nodeStore.AddNodeRankingScore("0", "3", pb.RankingScore{})
	nodeStore.AddNodeRankingScore("1", "3", pb.RankingScore{})

	startTime = startTime.Add(24 * time.Hour)
	expiryTime = expiryTime.Add(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour)

	epochStore.AddEpoch(3, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())
	delegations[0].EpochSeq = "3"
	delegations[1].EpochSeq = "3"
	epochStore.AddDelegation(*delegations[0])
	epochStore.AddDelegation(*delegations[1])

	epoch, err = epochStore.GetEpoch()
	a.NoError(err)
	assertEpoch(a, epoch, delegations, nodes, 3, startTime.UnixNano(), expiryTime.UnixNano(), endTime.UnixNano())

	a.Equal(72*time.Hour, epochStore.GetTotalNodesUptime())
}

func assertEpoch(
	a *assert.Assertions,
	epoch *pb.Epoch,
	delegations []*pb.Delegation,
	nodes []*pb.Node,
	seq uint64,
	startTime, expiryTime, endTime int64,
) {
	a.Equal(epoch.Seq, seq)
	a.Equal(epoch.Timestamps.StartTime, startTime)
	a.Equal(epoch.Timestamps.ExpiryTime, expiryTime)
	a.Equal(epoch.Timestamps.EndTime, endTime)

	a.Equal(len(delegations), len(epoch.Delegations))

	sort.Sort(ByXY(delegations))
	sort.Sort(ByXY(epoch.Delegations))

	for i := range delegations {
		a.Equal(delegations[i], epoch.Delegations[i])
	}

	a.Equal(len(nodes), len(epoch.Validators))

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Id < nodes[j].Id
	})
	sort.Slice(epoch.Validators, func(i, j int) bool {
		return epoch.Validators[i].Id < epoch.Validators[j].Id
	})

	for i := range nodes {
		a.Equal(nodes[i], epoch.Validators[i])
	}
}
