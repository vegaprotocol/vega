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

package epochs_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/epochs/mocks"
	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*epochs.Service
	ctx        context.Context
	cancel     context.CancelFunc
	log        *logging.Logger
	ctrl       *gomock.Controller
	epochStore *mocks.MockEpochStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	epoch := mocks.NewMockEpochStore(ctrl)
	log := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())

	svc := epochs.NewService(
		log,
		epochs.NewDefaultConfig(),
		epoch,
	)

	return &testService{
		Service:    svc,
		ctx:        ctx,
		cancel:     cancel,
		log:        log,
		ctrl:       ctrl,
		epochStore: epoch,
	}
}

func TestNodesService_GetEpoch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		expectedEpoch := &pb.Epoch{
			Seq: 1,
			Timestamps: &pb.EpochTimestamps{
				StartTime: time.Now().Unix(),
				EndTime:   time.Now().Unix(),
			},
			Validators: []*pb.Node{
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
			},
		}

		testService.epochStore.EXPECT().GetEpoch().Return(expectedEpoch, nil).Times(1)

		epoch, err := testService.GetEpoch(testService.ctx)
		a.NoError(err)
		a.Equal(expectedEpoch, epoch)
	})

	t.Run("returns error", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		testService.epochStore.EXPECT().GetEpoch().Return(nil, fmt.Errorf("something went wrong")).Times(1)

		epoch, err := testService.GetEpoch(testService.ctx)
		a.EqualError(err, "something went wrong")
		a.Nil(epoch)
	})
}

func TestNodesService_GetEpochByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		expectedEpoch := &pb.Epoch{
			Seq: 1,
			Timestamps: &pb.EpochTimestamps{
				StartTime: time.Now().Unix(),
				EndTime:   time.Now().Unix(),
			},
			Validators: []*pb.Node{
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
			},
		}

		testService.epochStore.EXPECT().GetEpochByID("1").Return(expectedEpoch, nil).Times(1)

		epoch, err := testService.GetEpochByID(testService.ctx, "1")
		a.NoError(err)
		a.Equal(expectedEpoch, epoch)
	})

	t.Run("returns error", func(t *testing.T) {
		a := assert.New(t)
		testService := getTestService(t)
		defer testService.Finish()

		testService.epochStore.EXPECT().GetEpochByID("1").Return(nil, fmt.Errorf("something went wrong")).Times(1)

		epoch, err := testService.GetEpochByID(testService.ctx, "1")
		a.EqualError(err, "something went wrong")
		a.Nil(epoch)
	})
}

func (t *testService) Finish() {
	t.cancel()
	_ = t.log.Sync()
	t.ctrl.Finish()
}
