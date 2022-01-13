package storage_test

import (
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	pb "code.vegaprotocol.io/protos/vega"
)

type delegationTest struct {
	ds *storage.Delegations

	delegation1 pb.Delegation
	delegation2 pb.Delegation
	delegation3 pb.Delegation
	delegation4 pb.Delegation
}

type ByX []*pb.Delegation

func (o ByX) Len() int      { return len(o) }
func (o ByX) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ByX) Less(i, j int) bool {
	if o[i].Party == o[j].Party {
		return o[i].NodeId < o[j].NodeId
	} else {
		return o[i].Party < o[j].Party
	}
}

func setup(t *testing.T) *delegationTest {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	delegationStore := storage.NewDelegations(logging.NewTestLogger(), config)
	testService := delegationTest{
		ds: delegationStore,
	}

	testService.delegation1 = pb.Delegation{
		Party:    "party1",
		NodeId:   "node1",
		EpochSeq: "1",
		Amount:   "10",
	}

	testService.delegation2 = pb.Delegation{
		Party:    "party1",
		NodeId:   "node2",
		EpochSeq: "1",
		Amount:   "20",
	}
	testService.delegation3 = pb.Delegation{
		Party:    "party2",
		NodeId:   "node1",
		EpochSeq: "1",
		Amount:   "30",
	}
	testService.delegation4 = pb.Delegation{
		Party:    "party3",
		NodeId:   "node2",
		EpochSeq: "2",
		Amount:   "40",
	}

	// Added in reverse order so we can check our sorting
	testService.ds.AddDelegation(testService.delegation4)
	testService.ds.AddDelegation(testService.delegation3)
	testService.ds.AddDelegation(testService.delegation2)
	testService.ds.AddDelegation(testService.delegation1)

	return &testService
}

func TestClearOldEpochs(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	delegationStore := storage.NewDelegations(logging.NewTestLogger(), config)
	testService := delegationTest{
		ds: delegationStore,
	}

	for i := 0; i < 100; i++ {
		testService.ds.AddDelegation(pb.Delegation{
			Party:    "party1",
			NodeId:   "node1",
			EpochSeq: strconv.Itoa(i),
			Amount:   "100",
		})
		delegations, _ := testService.ds.GetAllDelegations(0, 0, false)

		if i < 30 {
			require.Equal(t, i+1, len(delegations))
		} else {
			require.Equal(t, 30, len(delegations))
		}
	}
}

func TestGetAllDelegations(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetAllDelegations(0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 4, len(delegations))

	sort.Sort(ByX(delegations))

	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation2, *delegations[1])
	require.Equal(t, testService.delegation3, *delegations[2])
	require.Equal(t, testService.delegation4, *delegations[3])
}

func TestGetAllDelegationsOnEpoch(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetAllDelegationsOnEpoch("1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 3, len(delegations))

	sort.Sort(ByX(delegations))

	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation2, *delegations[1])
	require.Equal(t, testService.delegation3, *delegations[2])

	delegations, err = testService.ds.GetAllDelegationsOnEpoch("2", 0, 0, false)
	sort.Sort(ByX(delegations))

	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])
}

func TestPagination(t *testing.T) {
	testService := setup(t)

	// Get the first one
	delegations, err := testService.ds.GetAllDelegations(0, 1, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])

	// Get two more
	delegations, err = testService.ds.GetAllDelegations(1, 2, false)
	require.Nil(t, err)

	sort.Sort(ByX(delegations))

	require.Equal(t, 2, len(delegations))
	require.Equal(t, testService.delegation2, *delegations[0])
	require.Equal(t, testService.delegation3, *delegations[1])
}

func TestPaginationSorting(t *testing.T) {
	testService := setup(t)

	// Check we sort by epoch, then party, then node
	delegations, err := testService.ds.GetAllDelegations(0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 4, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation2, *delegations[1])
	require.Equal(t, testService.delegation3, *delegations[2])
	require.Equal(t, testService.delegation4, *delegations[3])

	// And backwards
	delegations, err = testService.ds.GetAllDelegations(0, 0, true)
	require.Nil(t, err)
	require.Equal(t, 4, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])
	require.Equal(t, testService.delegation3, *delegations[1])
	require.Equal(t, testService.delegation2, *delegations[2])
	require.Equal(t, testService.delegation1, *delegations[3])
}

func TestGetNodeDelegations(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetNodeDelegations("node1", 0, 0, false)
	sort.Sort(ByX(delegations))

	require.Nil(t, err)
	require.Equal(t, 2, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation3, *delegations[1])

	delegations, err = testService.ds.GetNodeDelegations("node2", 0, 0, false)
	sort.Sort(ByX(delegations))

	require.Nil(t, err)
	require.Equal(t, 2, len(delegations))
	require.Equal(t, testService.delegation2, *delegations[0])
	require.Equal(t, testService.delegation4, *delegations[1])
}

func TestGetNodeDelegationsOnEpoch(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetNodeDelegationsOnEpoch("node1", "1", 0, 0, false)
	sort.Sort(ByX(delegations))

	require.Nil(t, err)
	require.Equal(t, 2, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation3, *delegations[1])

	delegations, err = testService.ds.GetNodeDelegationsOnEpoch("node2", "1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation2, *delegations[0])

	delegations, err = testService.ds.GetNodeDelegationsOnEpoch("node2", "2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])
}

func TestGetPartyDelegations(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetPartyDelegations("party1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(delegations))
	sort.Sort(ByX(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation2, *delegations[1])

	delegations, err = testService.ds.GetPartyDelegations("party2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation3, *delegations[0])

	delegations, err = testService.ds.GetPartyDelegations("party3", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])
}

func TestGetPartyDelegationsOnEpoch(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetPartyDelegationsOnEpoch("party1", "1", 0, 0, false)
	require.Nil(t, err)
	sort.Sort(ByX(delegations))
	require.Equal(t, 2, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])
	require.Equal(t, testService.delegation2, *delegations[1])

	delegations, err = testService.ds.GetPartyDelegationsOnEpoch("party2", "1", 0, 0, false)
	require.Nil(t, err)
	sort.Sort(ByX(delegations))
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation3, *delegations[0])

	delegations, err = testService.ds.GetPartyDelegationsOnEpoch("party3", "1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyDelegationsOnEpoch("party1", "2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyDelegationsOnEpoch("party3", "2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])
}

func TestGetPartyNodeDelegations(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetPartyNodeDelegations("party1", "node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegations("party1", "node2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation2, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegations("party1", "node3", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyNodeDelegations("party2", "node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation3, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegations("party2", "node2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyNodeDelegations("party3", "node2", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegations("party3", "node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))
}

func TestGetPartyNodeDelegationsOnEpoch(t *testing.T) {
	testService := setup(t)

	delegations, err := testService.ds.GetPartyNodeDelegationsOnEpoch("party1", "node1", "1")
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation1, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party1", "node2", "1")
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation2, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party1", "node1", "2")
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party1", "node2", "2")
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party2", "node1", "1")
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation3, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party2", "node1", "2")
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party3", "node2", "2")
	require.Nil(t, err)
	require.Equal(t, 1, len(delegations))
	require.Equal(t, testService.delegation4, *delegations[0])

	delegations, err = testService.ds.GetPartyNodeDelegationsOnEpoch("party3", "node2", "1")
	require.Nil(t, err)
	require.Equal(t, 0, len(delegations))
}
