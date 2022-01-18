package delegations_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/delegations/mocks"
	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testService struct {
	svc             *delegations.Service
	ctx             context.Context
	cfunc           context.CancelFunc
	log             *logging.Logger
	ctrl            *gomock.Controller
	delegationStore *mocks.MockDelegationStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	delegationStore := mocks.NewMockDelegationStore(ctrl)
	log := logging.NewTestLogger()
	ctx, cfunc := context.WithCancel(context.Background())
	svc := delegations.NewService(
		log,
		delegations.NewDefaultConfig(),
		delegationStore,
	)
	return &testService{
		svc:             svc,
		ctx:             ctx,
		cfunc:           cfunc,
		log:             log,
		ctrl:            ctrl,
		delegationStore: delegationStore,
	}
}

func TestGetAllDelegations(t *testing.T) {
	testService := getTestService(t)
	// empty delegations
	testService.delegationStore.EXPECT().GetAllDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)

	res, err := testService.svc.GetAllDelegations(0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party2", NodeId: "node2"}
	testService.delegationStore.EXPECT().GetAllDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{
		del1, del2,
	}, nil)

	res, err = testService.svc.GetAllDelegations(0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}

func TestGetAllDelegationsOnEpoch(t *testing.T) {
	testService := getTestService(t)

	// no delegations for epoch
	testService.delegationStore.EXPECT().GetAllDelegationsOnEpoch(gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetAllDelegationsOnEpoch("1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party2", NodeId: "node2"}
	testService.delegationStore.EXPECT().GetAllDelegationsOnEpoch(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{
		del1, del2,
	}, nil)

	res, err = testService.svc.GetAllDelegationsOnEpoch("1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}
func TestGetPartyDelegations(t *testing.T) {
	testService := getTestService(t)

	// no delegations for party
	testService.delegationStore.EXPECT().GetPartyDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetPartyDelegations("party1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations for party1
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party1", NodeId: "node2"}
	testService.delegationStore.EXPECT().GetPartyDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{
		del1, del2,
	}, nil)

	res, err = testService.svc.GetPartyDelegations("party1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}
func TestGetPartyDelegationsOnEpoch(t *testing.T) {
	testService := getTestService(t)

	// no delegations for epoch
	testService.delegationStore.EXPECT().GetPartyDelegationsOnEpoch(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetPartyDelegationsOnEpoch("party1", "1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party1", NodeId: "node2"}
	testService.delegationStore.EXPECT().GetPartyDelegationsOnEpoch(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{del1, del2}, nil)

	res, err = testService.svc.GetPartyDelegationsOnEpoch("party1", "1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}

func TestGetPartyNodeDelegations(t *testing.T) {
	testService := getTestService(t)

	// no delegations for epoch
	testService.delegationStore.EXPECT().GetPartyNodeDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetPartyNodeDelegations("party1", "node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1", EpochSeq: "1"}
	del2 := &pb.Delegation{Party: "party1", NodeId: "node2", EpochSeq: "2"}
	testService.delegationStore.EXPECT().GetPartyNodeDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{
		del1, del2,
	}, nil)

	res, err = testService.svc.GetPartyNodeDelegations("party1", "node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}
func TestGetPartyNodeDelegationsOnEpoch(t *testing.T) {
	testService := getTestService(t)

	// no delegations for epoch
	testService.delegationStore.EXPECT().GetPartyNodeDelegationsOnEpoch(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetPartyNodeDelegationsOnEpoch("party1", "node1", "1234")
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegation
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1", EpochSeq: "1234"}
	testService.delegationStore.EXPECT().GetPartyNodeDelegationsOnEpoch(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*pb.Delegation{
		del1,
	}, nil)

	res, err = testService.svc.GetPartyNodeDelegationsOnEpoch("party1", "node1", "1234")
	require.Nil(t, err)
	require.Equal(t, 1, len(res))
	require.Equal(t, *del1, *res[0])
}
func TestGetNodeDelegations(t *testing.T) {
	testService := getTestService(t)

	// no delegations for node1
	testService.delegationStore.EXPECT().GetNodeDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetNodeDelegations("node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations for node1
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party2", NodeId: "node1"}
	testService.delegationStore.EXPECT().GetNodeDelegations(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{
		del1, del2,
	}, nil)

	res, err = testService.svc.GetNodeDelegations("node1", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}
func TestGetNodeDelegationsOnEpoch(t *testing.T) {
	testService := getTestService(t)

	// no delegations for node for epoch
	testService.delegationStore.EXPECT().GetNodeDelegationsOnEpoch(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{}, nil)
	res, err := testService.svc.GetNodeDelegationsOnEpoch("node1", "1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 0, len(res))

	// some delegations
	del1 := &pb.Delegation{Party: "party1", NodeId: "node1"}
	del2 := &pb.Delegation{Party: "party2", NodeId: "node1"}
	testService.delegationStore.EXPECT().GetNodeDelegationsOnEpoch(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return([]*pb.Delegation{del1, del2}, nil)

	res, err = testService.svc.GetNodeDelegationsOnEpoch("node1", "1234", 0, 0, false)
	require.Nil(t, err)
	require.Equal(t, 2, len(res))
	require.Equal(t, *del1, *res[0])
	require.Equal(t, *del2, *res[1])
}

func (t *testService) Finish() {
	t.cfunc()
	_ = t.log.Sync()
	t.ctrl.Finish()
}
