package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/dehistory"

	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"google.golang.org/grpc/codes"
)

type deHistoryService struct {
	v2.UnimplementedDeHistoryServiceServer
	config           dehistory.Config
	deHistoryService DeHistoryService
}

func (d *deHistoryService) GetMostRecentDeHistorySegment(context.Context, *v2.GetMostRecentDeHistorySegmentRequest) (*v2.GetMostRecentDeHistorySegmentResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentDeHistorySegment")()

	segment, err := d.deHistoryService.GetHighestBlockHeightHistorySegment()
	if err != nil {
		if errors.Is(err, store.ErrSegmentNotFound) {
			return &v2.GetMostRecentDeHistorySegmentResponse{
				Segment: nil,
			}, nil
		}

		return nil, apiError(codes.Internal, ErrGetMostRecentHistorySegment, err)
	}

	return &v2.GetMostRecentDeHistorySegmentResponse{
		Segment:  toHistorySegment(segment),
		SwarmKey: d.deHistoryService.GetSwarmKey(),
	}, nil
}

func (d *deHistoryService) ListAllDeHistorySegments(context.Context, *v2.ListAllDeHistorySegmentsRequest) (*v2.ListAllDeHistorySegmentsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllDeHistorySegments")()
	if d.deHistoryService == nil {
		return nil, apiError(codes.Internal, ErrDeHistoryNotEnabled, fmt.Errorf("dehistory is not enabled"))
	}
	segments, err := d.deHistoryService.ListAllHistorySegments()
	if err != nil {
		return nil, apiError(codes.Internal, ErrListAllDeHistorySegment, err)
	}

	historySegments := make([]*v2.HistorySegment, 0, len(segments))
	for _, segment := range segments {
		historySegments = append(historySegments, toHistorySegment(segment))
	}

	return &v2.ListAllDeHistorySegmentsResponse{
		Segments: historySegments,
	}, nil
}

func (d *deHistoryService) FetchDeHistorySegment(ctx context.Context, req *v2.FetchDeHistorySegmentRequest) (*v2.FetchDeHistorySegmentResponse, error) {
	if !d.config.AllowFetchSegments {
		return nil, apiError(codes.PermissionDenied, ErrFetchDeHistorySegment, fmt.Errorf("fetching segments is not allowed"))
	}

	defer metrics.StartAPIRequestAndTimeGRPC("FetchDeHistorySegment'")()
	if d.deHistoryService == nil {
		return nil, apiError(codes.Internal, ErrDeHistoryNotEnabled, fmt.Errorf("dehistory is not enabled"))
	}
	segment, err := d.deHistoryService.FetchHistorySegment(ctx, req.HistorySegmentId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrFetchDeHistorySegment, err)
	}

	return &v2.FetchDeHistorySegmentResponse{
		Segment: toHistorySegment(segment),
	}, nil
}

func toHistorySegment(segment store.SegmentIndexEntry) *v2.HistorySegment {
	return &v2.HistorySegment{
		FromHeight:               segment.HeightFrom,
		ToHeight:                 segment.HeightTo,
		ChainId:                  segment.ChainID,
		HistorySegmentId:         segment.HistorySegmentID,
		PreviousHistorySegmentId: segment.PreviousHistorySegmentID,
	}
}

func (d *deHistoryService) GetActiveDeHistoryPeerAddresses(_ context.Context, _ *v2.GetActiveDeHistoryPeerAddressesRequest) (*v2.GetActiveDeHistoryPeerAddressesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentHistorySegmentFromPeers")()
	if d.deHistoryService == nil {
		return nil, apiError(codes.Internal, ErrDeHistoryNotEnabled, fmt.Errorf("dehistory is not enabled"))
	}
	addresses := d.deHistoryService.GetActivePeerAddresses()

	return &v2.GetActiveDeHistoryPeerAddressesResponse{
		IpAddresses: addresses,
	}, nil
}

func (d *deHistoryService) CopyHistorySegmentToFile(ctx context.Context, req *v2.CopyHistorySegmentToFileRequest) (*v2.CopyHistorySegmentToFileResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("CopyHistorySegmentToFile")()
	if d.deHistoryService == nil {
		return nil, apiError(codes.Internal, ErrDeHistoryNotEnabled, fmt.Errorf("dehistory is not enabled"))
	}

	err := d.deHistoryService.CopyHistorySegmentToFile(ctx, req.HistorySegmentId, req.TargetFile)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCopyHistorySegmentToFile, err)
	}

	return &v2.CopyHistorySegmentToFileResponse{}, nil
}

func (d *deHistoryService) Ping(context.Context, *v2.PingRequest) (*v2.PingResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Ping")()
	return &v2.PingResponse{}, nil
}
