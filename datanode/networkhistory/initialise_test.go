package networkhistory_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/mocks"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/stretchr/testify/assert"
)

type TestSegment struct {
	HeightFrom               int64
	HeightTo                 int64
	PreviousHistorySegmentId string
	HistorySegmentId         string
}

func (m TestSegment) GetPreviousHistorySegmentId() string {
	return m.PreviousHistorySegmentId
}

func (m TestSegment) GetHistorySegmentId() string {
	return m.HistorySegmentId
}

func (m TestSegment) GetFromHeight() int64 {
	return m.HeightFrom
}

func (m TestSegment) GetToHeight() int64 {
	return m.HeightTo
}

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
				HistorySegmentId:         "segment2",
				PreviousHistorySegmentId: "segment1",
			},
			SwarmKeySeed: "",
		},
	}

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(
		TestSegment{HeightFrom: 1001, HeightTo: 2000, PreviousHistorySegmentId: "segment1", HistorySegmentId: "segment2"}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(
		TestSegment{HeightFrom: 0, HeightTo: 1000, PreviousHistorySegmentId: "", HistorySegmentId: "segment1"}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return([]networkhistory.Segment{
		TestSegment{HeightFrom: 0, HeightTo: 1000},
		TestSegment{HeightFrom: 1001, HeightTo: 2000},
	}, nil)

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), networkhistory.ContiguousHistory{
		HeightFrom: 0,
		HeightTo:   2000,
		SegmentsOldestFirst: []networkhistory.Segment{
			TestSegment{HeightFrom: 0, HeightTo: 1000},
			TestSegment{HeightFrom: 1001, HeightTo: 2000},
		},
	}, gomock.Any(), false).Times(1)

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
				HistorySegmentId:         "segment4",
				PreviousHistorySegmentId: "segment3",
			},
			SwarmKeySeed: "",
		},
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment4").Times(1).Return(
		TestSegment{HeightFrom: 3001, HeightTo: 4000, PreviousHistorySegmentId: "segment3", HistorySegmentId: "segment4"}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment3").Times(1).Return(
		TestSegment{HeightFrom: 2001, HeightTo: 3000, PreviousHistorySegmentId: "segment2", HistorySegmentId: "segment3"}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().ListAllHistorySegments().Times(1).Return(
		[]networkhistory.Segment{
			TestSegment{HeightFrom: 2001, HeightTo: 3000},
			TestSegment{HeightFrom: 3001, HeightTo: 4000},
		}, nil)

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), networkhistory.ContiguousHistory{
		HeightFrom: 2001,
		HeightTo:   4000,
		SegmentsOldestFirst: []networkhistory.Segment{
			TestSegment{HeightFrom: 2001, HeightTo: 3000},
			TestSegment{HeightFrom: 3001, HeightTo: 4000},
		},
	}, gomock.Any(), true).Times(1)

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
				HistorySegmentId:         "segment4",
				PreviousHistorySegmentId: "segment3",
			},
			SwarmKeySeed: "",
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
				HistorySegmentId:         "segment2",
				PreviousHistorySegmentId: "segment1",
			},
			SwarmKeySeed: "",
		},
	}

	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(
		TestSegment{HeightFrom: 1001, HeightTo: 2000, PreviousHistorySegmentId: "segment1", HistorySegmentId: "segment2"}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(
		TestSegment{HeightFrom: 0, HeightTo: 1000, PreviousHistorySegmentId: "", HistorySegmentId: "segment1"}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	service.EXPECT().ListAllHistorySegments().Times(1).Return(
		[]networkhistory.Segment{
			TestSegment{HeightFrom: 0, HeightTo: 1000},
			TestSegment{HeightFrom: 1001, HeightTo: 2000},
		}, nil)

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), networkhistory.ContiguousHistory{
		HeightFrom: 0,
		HeightTo:   2000,
		SegmentsOldestFirst: []networkhistory.Segment{
			TestSegment{HeightFrom: 0, HeightTo: 1000},
			TestSegment{HeightFrom: 1001, HeightTo: 2000},
		},
	}, gomock.Any(), false).Times(1)

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

	service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(
		TestSegment{HeightFrom: 0, HeightTo: 1000, PreviousHistorySegmentId: "", HistorySegmentId: "segment1"}, nil)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)

	service.EXPECT().ListAllHistorySegments().Times(1).Return(
		[]networkhistory.Segment{
			TestSegment{HeightFrom: 0, HeightTo: 1000},
		}, nil)

	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), networkhistory.ContiguousHistory{
		HeightFrom: 0,
		HeightTo:   1000,
		SegmentsOldestFirst: []networkhistory.Segment{
			TestSegment{HeightFrom: 0, HeightTo: 1000},
		},
	}, gomock.Any(), false).Times(1)

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

func TestInitialiseEmptyDataNodeWhenMultipleContiguousHistories(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	peerResponse := &networkhistory.PeerResponse{
		PeerAddr: "",
		Response: &v2.GetMostRecentNetworkHistorySegmentResponse{
			Segment: &v2.HistorySegment{
				FromHeight:               6001,
				ToHeight:                 7000,
				HistorySegmentId:         "segment2",
				PreviousHistorySegmentId: "segment1",
			},
			SwarmKeySeed: "",
		},
	}

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(
		TestSegment{HeightFrom: 6001, HeightTo: 7000, PreviousHistorySegmentId: "segment1", HistorySegmentId: "segment2"}, nil)

	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(
		TestSegment{HeightFrom: 5001, HeightTo: 6000, PreviousHistorySegmentId: "", HistorySegmentId: "segment1"}, nil)

	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return(
		[]networkhistory.Segment{
			TestSegment{HeightFrom: 2001, HeightTo: 3000},
			TestSegment{HeightFrom: 3001, HeightTo: 4000},

			TestSegment{HeightFrom: 5001, HeightTo: 6000},
			TestSegment{HeightFrom: 6001, HeightTo: 7000},
		},
		nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(),
		networkhistory.ContiguousHistory{
			HeightFrom: 5001,
			HeightTo:   7000,
			SegmentsOldestFirst: []networkhistory.Segment{
				TestSegment{HeightFrom: 5001, HeightTo: 6000},
				TestSegment{HeightFrom: 6001, HeightTo: 7000},
			},
		}, gomock.Any(), false).Times(1)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{})
}

func TestSelectRootSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKeySeed: ""},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKeySeed: ""},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKeySeed: ""},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKeySeed: ""},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKeySeed: ""},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKeySeed: ""},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(4000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithSwarmKey(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKeySeed: "A"},
		"2": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKeySeed: "A"},
		"3": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKeySeed: "B"},
		"4": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 4000}, SwarmKeySeed: "D"},
		"5": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 3000}, SwarmKeySeed: "A"},
		"6": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKeySeed: "B"},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "A")
	assert.Equal(t, int64(3000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithOneSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{
		"1": {Segment: &v2.HistorySegment{FromHeight: 1001, ToHeight: 2000}, SwarmKeySeed: ""},
	}

	selectedResponse := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Equal(t, int64(2000), selectedResponse.Response.Segment.ToHeight)
}

func TestSelectRootSegmentWithZeroSegment(t *testing.T) {
	responses := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{}

	rootSegment := networkhistory.SelectMostRecentHistorySegmentResponse(responses, "")
	assert.Nil(t, rootSegment)
}
