package dehistory_test

import (
	"context"
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

	mostRecentSegment := &v2.HistorySegment{
		FromHeight:               1001,
		ToHeight:                 2000,
		ChainId:                  "testchainid",
		HistorySegmentId:         "segment2",
		PreviousHistorySegmentId: "segment1",
	}

	cfg.MinimumBlockCount = 1500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(mostRecentSegment, map[string]*v2.HistorySegment{"peer1": mostRecentSegment}, nil)

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

	service.EXPECT().LoadAllAvailableHistoryIntoDatanode(gomock.Any(), sqlstore.EmbedMigrations).Times(1)

	dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
}

func TestInitialiseNonEmptyDataNode(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := dehistory.NewDefaultInitializationConfig()

	cfg.MinimumBlockCount = 2000

	ctrl := gomock.NewController(t)
	service := mocks.NewMockDeHistory(ctrl)
	ctx := context.Background()

	mostRecentSegment := &v2.HistorySegment{
		FromHeight:               3001,
		ToHeight:                 4000,
		ChainId:                  "testchainid",
		HistorySegmentId:         "segment4",
		PreviousHistorySegmentId: "segment3",
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(mostRecentSegment, map[string]*v2.HistorySegment{"peer1": mostRecentSegment}, nil)

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

	service.EXPECT().LoadAllAvailableHistoryIntoDatanode(gomock.Any(), sqlstore.EmbedMigrations).Times(1)

	dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{
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

	mostRecentSegment := &v2.HistorySegment{
		FromHeight:               3001,
		ToHeight:                 4000,
		ChainId:                  "testchainid",
		HistorySegmentId:         "segment4",
		PreviousHistorySegmentId: "segment3",
	}

	cfg.MinimumBlockCount = 500
	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(mostRecentSegment, map[string]*v2.HistorySegment{"peer1": mostRecentSegment}, nil)

	assert.Nil(t, dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{
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

	mostRecentSegment := &v2.HistorySegment{
		FromHeight:               1001,
		ToHeight:                 2000,
		ChainId:                  "testchainid",
		HistorySegmentId:         "segment2",
		PreviousHistorySegmentId: "segment1",
	}

	service.EXPECT().GetMostRecentHistorySegmentFromPeers(gomock.Any(), []int{}).Times(1).
		Return(mostRecentSegment, map[string]*v2.HistorySegment{"peer1": mostRecentSegment}, nil)

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

	service.EXPECT().LoadAllAvailableHistoryIntoDatanode(gomock.Any(), sqlstore.EmbedMigrations).Times(1)

	dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
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

	service.EXPECT().LoadAllAvailableHistoryIntoDatanode(gomock.Any(), sqlstore.EmbedMigrations).Times(1)

	dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{})
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
		Return(nil, nil, dehistory.ErrNoActivePeersFound)

	assert.Equal(t, dehistory.ErrDeHistoryNotAvailable,
		dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{}))
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
		Return(nil, map[string]*v2.HistorySegment{}, nil)

	assert.Equal(t, dehistory.ErrDeHistoryNotAvailable,
		dehistory.DatanodeFromDeHistory(ctx, cfg, log, service, sqlstore.DatanodeBlockSpan{}, []int{}))
}

func TestSelectRootSegment(t *testing.T) {
	segments := map[string]*v2.HistorySegment{
		"1": {FromHeight: 1001, ToHeight: 2000},
		"2": {FromHeight: 1001, ToHeight: 3000},
		"3": {FromHeight: 1001, ToHeight: 4000},
		"4": {FromHeight: 1001, ToHeight: 4000},
		"5": {FromHeight: 1001, ToHeight: 3000},
		"6": {FromHeight: 1001, ToHeight: 2000},
	}

	rootSegment := dehistory.SelectRootSegment(segments)
	assert.Equal(t, int64(4000), rootSegment.ToHeight)
}

func TestSelectRootSegmentWithOneSegment(t *testing.T) {
	segments := map[string]*v2.HistorySegment{
		"1": {FromHeight: 1001, ToHeight: 2000},
	}

	rootSegment := dehistory.SelectRootSegment(segments)
	assert.Equal(t, int64(2000), rootSegment.ToHeight)
}

func TestSelectRootSegmentWithZeroSegment(t *testing.T) {
	segments := map[string]*v2.HistorySegment{}

	rootSegment := dehistory.SelectRootSegment(segments)
	assert.Nil(t, rootSegment)
}
