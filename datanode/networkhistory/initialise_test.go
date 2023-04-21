package networkhistory_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/mocks"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
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

func (m TestSegment) GetDatabaseVersion() int64 {
	return 1
}

func (m TestSegment) GetChainId() string {
	return "test-chain-id"
}

func makeFullSegment(from, to int64, previous, id string) segment.Full {
	return segment.Full{
		MetaData: segment.MetaData{
			Base: segment.Base{
				HeightFrom: from,
				HeightTo:   to,
			},
			PreviousHistorySegmentID: previous,
		},
		HistorySegmentID: id,
	}
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
				PreviousHistorySegmentId: "segment1",
				HistorySegmentId:         "segment2",
			},
			SwarmKeySeed: "",
		},
	}

	segment1 := makeFullSegment(0, 1000, "", "segment1")
	segment2 := makeFullSegment(1001, 2000, "segment1", "segment2")
	segments := []segment.Full{segment1, segment2}
	chunk := segment.ContiguousHistory[segment.Full]{
		HeightFrom: 0,
		HeightTo:   2000,
		Segments:   segments,
	}

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(segment2, nil)
	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(segment1, nil)

	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return(segments, nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), chunk, gomock.Any(), false, false).Times(1)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{}, false)
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

	segment4 := makeFullSegment(3001, 4000, "segment3", "segment4")
	segment3 := makeFullSegment(2001, 3000, "segment2", "segment3")
	segments := []segment.Full{segment3, segment4}
	chunk := segment.ContiguousHistory[segment.Full]{
		HeightFrom: 2001,
		HeightTo:   4000,
		Segments:   segments,
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment4").Times(1).Return(segment4, nil)
	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment3").Times(1).Return(segment3, nil)
	gomock.InOrder(first, second)

	service.EXPECT().ListAllHistorySegments().Times(1).Return(segments, nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), chunk, gomock.Any(), true, false).Times(1)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{
		FromHeight: 0,
		ToHeight:   2243,
		HasData:    true,
	}, nil)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{}, false)
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

	assert.Nil(t, networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{}, false))
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

	segment1 := makeFullSegment(0, 1000, "", "segment1")
	segment2 := makeFullSegment(1001, 2000, "segment1", "segment2")
	segments := []segment.Full{segment1, segment2}
	chunk := segment.ContiguousHistory[segment.Full]{
		HeightFrom: 0,
		HeightTo:   2000,
		Segments:   segments,
	}

	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment2").Times(1).Return(segment2, nil)
	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(segment1, nil)
	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return(segments, nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), chunk, gomock.Any(), false, false).Times(1)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{}, false)
}

func TestInitialiseToASpecifiedSegment(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := networkhistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000
	cfg.ToSegment = "segment1"

	ctrl := gomock.NewController(t)
	service := mocks.NewMockNetworkHistory(ctrl)
	ctx := context.Background()

	segment1 := makeFullSegment(0, 1000, "", "segment1")
	segments := []segment.Full{segment1}
	chunk := segment.ContiguousHistory[segment.Full]{
		HeightFrom: 0,
		HeightTo:   1000,
		Segments:   segments,
	}

	service.EXPECT().FetchHistorySegment(gomock.Any(), "segment1").Times(1).Return(segment1, nil)
	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return(segments, nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), chunk, gomock.Any(), false, false).Times(1)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig,
		service, []int{}, false)
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
		service, []int{}, false))
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
		service, []int{}, false))
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
				HistorySegmentId:         "segment7",
				PreviousHistorySegmentId: "segment6",
			},
			SwarmKeySeed: "",
		},
	}

	cfg.MinimumBlockCount = 1500

	segment3 := makeFullSegment(2001, 3000, "", "segment3")
	segment4 := makeFullSegment(3001, 4000, "segment3", "segment4")

	segment6 := makeFullSegment(5001, 6000, "", "segment6")
	segment7 := makeFullSegment(6001, 7000, "segment6", "segment7")
	allSegments := []segment.Full{segment3, segment4, segment6, segment7}
	lastSegments := []segment.Full{segment6, segment7}
	chunk := segment.ContiguousHistory[segment.Full]{
		HeightFrom: 5001,
		HeightTo:   7000,
		Segments:   lastSegments,
	}

	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(peerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{"peer1": peerResponse.Response}, nil)

	first := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment7").Times(1).Return(segment7, nil)
	second := service.EXPECT().FetchHistorySegment(gomock.Any(), "segment6").Times(1).Return(segment6, nil)

	gomock.InOrder(first, second)

	service.EXPECT().GetDatanodeBlockSpan(gomock.Any()).Times(1).Return(sqlstore.DatanodeBlockSpan{}, nil)
	service.EXPECT().ListAllHistorySegments().Times(1).Return(allSegments, nil)
	service.EXPECT().LoadNetworkHistoryIntoDatanode(gomock.Any(), chunk, gomock.Any(), false, false).Times(1)

	networkhistory.InitialiseDatanodeFromNetworkHistory(ctx, cfg, log, sqlstore.NewDefaultConfig().ConnectionConfig, service, []int{}, false)
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
