package dehistory_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/mocks"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestInitialiseEmptyDataNode(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	peerResponse := &dehistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentDeHistorySegmentResponse{
			Segment: &v2.HistorySegment{
				FromHeight:               1001,
				ToHeight:                 2000,
				ChainId:                  "testchainid",
				HistorySegmentId:         "segment2",
				PreviousHistorySegmentId: "segment1",
			},
			SwarmKey: "",
		},
	}

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentDeHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               1001,
			HeightTo:                 2001,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "segment1",
		},
		HistorySegmentID: "segment2",
	}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               0,
			HeightTo:                 1000,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "",
		},
		HistorySegmentID: "segment1",
	}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().LoadDeHistoryIntoDatanode(gomock.Any()).Times(1)

	dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
}

func TestInitialiseNonEmptyDataNode(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	peerResponse := &dehistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentDeHistorySegmentResponse{
			Segment: &v2.HistorySegment{
				FromHeight:               3001,
				ToHeight:                 4000,
				ChainId:                  "testchainid",
				HistorySegmentId:         "segment4",
				PreviousHistorySegmentId: "segment3",
			},
			SwarmKey: "",
		},
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentDeHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment4").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               3001,
			HeightTo:                 4000,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "segment3",
		},
		HistorySegmentID: "segment4",
	}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment3").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               2001,
			HeightTo:                 3000,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "segment2",
		},
		HistorySegmentID: "segment3",
	}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().LoadDeHistoryIntoDatanode(gomock.Any()).Times(1)

	dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{
		FromHeight: 0,
		ToHeight:   2243,
		HasData:    true,
	}, []int{})
}

func TestLoadingHistoryWithinDatanodeCurrentSpanDoesNothing(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	peerResponse := &dehistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentDeHistorySegmentResponse{
			Segment: &v2.HistorySegment{
				FromHeight:               3001,
				ToHeight:                 4000,
				ChainId:                  "testchainid",
				HistorySegmentId:         "segment4",
				PreviousHistorySegmentId: "segment3",
			},
			SwarmKey: "",
		},
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentDeHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	assert.Nil(t, dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{
		FromHeight: 0,
		ToHeight:   5000,
		HasData:    true,
	}, []int{}))
}

func TestWhenMinimumBlockCountExceedsAvailableHistory(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 5000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	peerResponse := &dehistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentDeHistorySegmentResponse{
			Segment: &v2.HistorySegment{
				FromHeight:               1001,
				ToHeight:                 2000,
				ChainId:                  "testchainid",
				HistorySegmentId:         "segment2",
				PreviousHistorySegmentId: "segment1",
			},
			SwarmKey: "",
		},
	}

	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentDeHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               1001,
			HeightTo:                 2001,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "segment1",
		},
		HistorySegmentID: "segment2",
	}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               0,
			HeightTo:                 1000,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "",
		},
		HistorySegmentID: "segment1",
	}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().LoadDeHistoryIntoDatanode(gomock.Any()).Times(1)

	dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
}

func TestInitialiseToASpecifiedSegment(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000
	cfg.ToSegment = "segment1"

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom:               0,
			HeightTo:                 1000,
			ChainID:                  "testchainid",
			PreviousHistorySegmentID: "",
		},
		HistorySegmentID: "segment1",
	}, nil)

	service.EXPECT().LoadDeHistoryIntoDatanode(gomock.Any()).Times(1)

	dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
}

func TestAutoInitialiseWhenNoActivePeers(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(nil, nil, errors.New("no peers found"))

	assert.NotNil(t, dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{}))
}

func TestAutoInitialiseWhenNoHistoryAvailableFromPeers(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(nil, map[string]*v2.GetMostRecentDeHistorySegmentResponse{}, nil)

	assert.NotNil(t, dehistory.InitialiseDatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{}))
}

func TestSelectRootSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentDeHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: ""},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: ""},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: ""},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: ""},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
	}

	selectedResponse := dehistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(4000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithSwarmKey(t *testing.T) {
	responses := map[string]*v2.GetMostRecentDeHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: "A"},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: "A"},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: "B"},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: "D"},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: "A"},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: "B"},
	}

	selectedResponse := dehistory.SelectMostRecentHistorySegmentResponse(responses, "A")
	assert.Equal(t, int64(3000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithOneSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentDeHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
	}

	selectedResponse := dehistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(2000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithZeroSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentDeHistorySegmentResponse{}

	rootSegment := dehistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Nil(t, rootSegment)
}
