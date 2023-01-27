package networkhistory_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/mocks"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestInitialiseEmptyDataNode(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	peerResponse := &networkhistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentNetworkHistorySegmentResponse{
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
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

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

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{})
}

func TestInitialiseNonEmptyDataNode(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	peerResponse := &networkhistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentNetworkHistorySegmentResponse{
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
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

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

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{
		FromHeight: 0,
		ToHeight:   2243,
		HasData:    true,
	}, nil)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{})
}

func TestLoadingHistoryWithinDatanodeCurrentSpanDoesNothing(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	peerResponse := &networkhistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentNetworkHistorySegmentResponse{
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
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{
		FromHeight: 0,
		ToHeight:   5000,
		HasData:    true,
	}, nil)

	assert.Nil(t, networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{}))
}

func TestWhenMinimumBlockCountExceedsAvailableHistory(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 5000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	peerResponse := &networkhistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentNetworkHistorySegmentResponse{
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
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

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

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{})
}

func TestInitialiseToASpecifiedSegment(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000
	cfg.ToSegment = "segment1"

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
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

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{})
}

func TestAutoInitialiseWhenNoActivePeers(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(nil, nil, errors.New("no peers found"))
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	assert.NotNil(t, networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{}))
}

func TestAutoInitialiseWhenNoHistoryAvailableFromPeers(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(nil, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{}, nil)
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	assert.NotNil(t, networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{}))
}

func TestSelectRootSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: ""},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: ""},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: ""},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: ""},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(4000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithSwarmKey(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: "A"},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: "A"},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: "B"},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKey: "D"},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKey: "A"},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: "B"},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "A")
	assert.Equal(t, int64(3000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithOneSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKey: ""},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(2000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithZeroSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{}

	rootSegment := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Nil(t, rootSegment)
}
