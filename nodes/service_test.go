package nodes_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/nodes/mocks"
	pb "code.vegaprotocol.io/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*nodes.Service
	ctx        context.Context
	cancel     context.CancelFunc
	log        *logging.Logger
	ctrl       *gomock.Controller
	epochStore *mocks.MockEpochStore
	nodeStore  *mocks.MockNodeStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	epoch := mocks.NewMockEpochStore(ctrl)
	node := mocks.NewMockNodeStore(ctrl)
	log := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())

	svc := nodes.NewService(
		log,
		nodes.NewDefaultConfig(),
		node,
		epoch,
	)

	return &testService{
		Service:    svc,
		ctx:        ctx,
		cancel:     cancel,
		log:        log,
		ctrl:       ctrl,
		epochStore: epoch,
		nodeStore:  node,
	}
}

func TestNodesService_GetAll(t *testing.T) {
	a := assert.New(t)
	testService := getTestService(t)
	defer testService.Finish()

	expectedNodes := []*pb.Node{
		{
			Id:                "node_1",
			PubKey:            "pub_key",
			InfoUrl:           "node-1.xyz.vega",
			Location:          "GB",
			StakedByOperator:  "10",
			StakedByDelegates: "20",
			StakedTotal:       "30",
			Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
			Delagations: []*pb.Delegation{
				{
					Party:    "1",
					NodeId:   "node_1",
					Amount:   "20",
					EpochSeq: "1",
				},
				{
					Party:    "1",
					NodeId:   "node_1",
					Amount:   "10",
					EpochSeq: "1",
				},
			},
		},
		{
			Id:                "node_2",
			PubKey:            "pub_key",
			InfoUrl:           "node-2.xyz.vega",
			Location:          "GB",
			StakedByOperator:  "10",
			StakedByDelegates: "20",
			StakedTotal:       "30",
			Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
			Delagations: []*pb.Delegation{
				{
					Party:    "1",
					NodeId:   "node_2",
					Amount:   "20",
					EpochSeq: "1",
				},
				{
					Party:    "1",
					NodeId:   "node_2",
					Amount:   "10",
					EpochSeq: "1",
				},
			},
		},
	}

	testService.nodeStore.EXPECT().GetAll().Return(expectedNodes).Times(1)

	nodes, err := testService.GetNodes(context.Background())
	a.NoError(err)
	a.Equal(expectedNodes, nodes)
}

func TestNodesService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		expectedNode := &pb.Node{
			Id:                "node_1",
			PubKey:            "pub_key",
			InfoUrl:           "node-1.xyz.vega",
			Location:          "GB",
			StakedByOperator:  "10",
			StakedByDelegates: "20",
			StakedTotal:       "30",
			Status:            pb.NodeStatus_NODE_STATUS_VALIDATOR,
			Delagations: []*pb.Delegation{
				{
					Party:    "1",
					NodeId:   "node_1",
					Amount:   "20",
					EpochSeq: "1",
				},
				{
					Party:    "1",
					NodeId:   "node_1",
					Amount:   "10",
					EpochSeq: "1",
				},
			},
		}

		testService.nodeStore.EXPECT().GetByID("node_1").Return(expectedNode, nil).Times(1)

		node, err := testService.GetNodeByID(context.Background(), "node_1")
		a.NoError(err)
		a.Equal(expectedNode, node)
	})

	t.Run("returns error", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		testService.nodeStore.EXPECT().GetByID("non_existing").Return(nil, fmt.Errorf("node not found")).Times(1)

		node, err := testService.GetNodeByID(context.Background(), "non_existing")
		a.EqualError(err, "node not found")
		a.Nil(node)
	})
}

func TestNodesService_GetNodeData(t *testing.T) {
	a := assert.New(t)
	testService := getTestService(t)
	defer testService.Finish()

	expectedData := &pb.NodeData{
		StakedTotal:     "40",
		TotalNodes:      10,
		ValidatingNodes: 10,
		Uptime:          float32(time.Duration(10 * time.Hour).Minutes()),
	}

	testService.epochStore.EXPECT().GetTotalNodesUptime().Return(10 * time.Hour).Times(1)
	testService.epochStore.EXPECT().GetEpochSeq().Return("epoch_1").Times(1)
	testService.nodeStore.EXPECT().GetStakedTotal("epoch_1").Return("40").Times(1)
	testService.nodeStore.EXPECT().GetTotalNodesNumber().Return(10).Times(1)
	testService.nodeStore.EXPECT().GetValidatingNodesNumber().Return(10).Times(1)

	data, err := testService.GetNodeData(context.Background())
	a.NoError(err)
	a.Equal(expectedData, data)
}

func (t *testService) Finish() {
	t.cancel()
	_ = t.log.Sync()
	t.ctrl.Finish()
}
