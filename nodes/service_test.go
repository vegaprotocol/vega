// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nodes_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/nodes/mocks"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
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
			Delegations: []*pb.Delegation{
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
			Delegations: []*pb.Delegation{
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

	testService.epochStore.EXPECT().GetEpochSeq().Return("1").Times(1)
	testService.nodeStore.EXPECT().GetAll("1").Return(expectedNodes).Times(1)

	nodes, err := testService.GetNodes(testService.ctx)
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
			Delegations: []*pb.Delegation{
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

		testService.epochStore.EXPECT().GetEpochSeq().Return("1").Times(1)
		testService.nodeStore.EXPECT().GetByID("node_1", "1").Return(expectedNode, nil).Times(1)

		node, err := testService.GetNodeByID(testService.ctx, "node_1")
		a.NoError(err)
		a.Equal(expectedNode, node)
	})

	t.Run("returns error", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		testService.epochStore.EXPECT().GetEpochSeq().Return("1").Times(1)
		testService.nodeStore.EXPECT().GetByID("non_existing", "1").Return(nil, fmt.Errorf("node not found")).Times(1)

		node, err := testService.GetNodeByID(testService.ctx, "non_existing")
		a.EqualError(err, "node not found")
		a.Nil(node)
	})
}

func TestNodesService_GetAllPubKeyRotations(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		expectedRotations := []*protoapi.KeyRotation{
			{
				NodeId:      "node_1",
				OldPubKey:   "old_node_1",
				NewPubKey:   "new_node_1",
				BlockHeight: 10,
			},
			{
				NodeId:      "node_2",
				OldPubKey:   "old_node_2",
				NewPubKey:   "new_node_2",
				BlockHeight: 11,
			},
		}

		testService.nodeStore.EXPECT().GetAllPubKeyRotations().Return(expectedRotations).Times(1)

		rotations, err := testService.GetAllPubKeyRotations(testService.ctx)
		a.NoError(err)
		a.Equal(expectedRotations, rotations)
	})
}

func TestNodesService_GetPubKeyRotationsPerNode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		expectedRotations := []*protoapi.KeyRotation{
			{
				NodeId:      "node_1",
				OldPubKey:   "old_node_1",
				NewPubKey:   "new_node_2",
				BlockHeight: 10,
			},
			{
				NodeId:      "node_1",
				OldPubKey:   "old_node_2",
				NewPubKey:   "new_node_3",
				BlockHeight: 22,
			},
		}

		testService.nodeStore.EXPECT().GetPubKeyRotationsPerNode("node_1").Return(expectedRotations).Times(1)

		rotations, err := testService.GetPubKeyRotationsPerNode(testService.ctx, "node_1")
		a.NoError(err)
		a.Equal(expectedRotations, rotations)
	})
}

func TestNodesService_GetNodeData(t *testing.T) {
	a := assert.New(t)
	testService := getTestService(t)
	defer testService.Finish()

	expectedData := &pb.NodeData{
		StakedTotal:     "40",
		TotalNodes:      10,
		ValidatingNodes: 5,
		Uptime:          float32(time.Duration(10 * time.Hour).Minutes()),
	}

	testService.epochStore.EXPECT().GetTotalNodesUptime().Return(10 * time.Hour).Times(1)
	testService.epochStore.EXPECT().GetEpochSeq().Return("epoch_1").Times(1)
	testService.nodeStore.EXPECT().GetStakedTotal("epoch_1").Return("40").Times(1)
	testService.nodeStore.EXPECT().GetTotalNodesNumber(gomock.Any()).Return(10).Times(1)
	testService.nodeStore.EXPECT().GetValidatingNodesNumber(gomock.Any()).Return(5).Times(1)

	data, err := testService.GetNodeData(testService.ctx)
	a.NoError(err)
	a.Equal(expectedData, data)
}

func (t *testService) Finish() {
	t.cancel()
	_ = t.log.Sync()
	t.ctrl.Finish()
}
